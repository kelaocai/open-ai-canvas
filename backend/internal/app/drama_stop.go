package app

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"yingce/backend/internal/model"
	"yingce/backend/internal/repository"
)

// StopDramaSegmentVideo binds the stop to the generation attempt shown in the UI.
// Cancel the slot, job and task together so a worker cannot claim the old queue entry after success.
func (s *Service) StopDramaSegmentVideo(userID, projectID, episodeID, segmentID, jobID string, revision int64) (string, error) {
	if _, _, err := s.dramaForUser(userID, projectID); err != nil {
		return "", err
	}
	message := "已停止，本段已退出生成队列"
	var stoppedTask *model.Task
	var upstream bool
	err := s.repo.Transaction(func(repo *repository.Repository) error {
		segment, err := repo.DramaSegment(projectID, segmentID)
		if err != nil {
			return err
		}
		if segment.EpisodeID != episodeID || segment.Revision != revision {
			return BadAuthRequest("剧情段已变化，请刷新后再停止")
		}
		if segment.VideoStatus == model.DramaMediaStopped {
			return nil
		}
		ok, err := repo.StopDramaVideoSlot(projectID, episodeID, segmentID, revision)
		if err != nil {
			return err
		}
		if !ok {
			return BadAuthRequest("本段当前没有可停止的生成任务")
		}
		jobs, err := repo.ActiveDramaVideoJobs(projectID, episodeID, segmentID)
		if err != nil {
			return err
		}
		if len(jobs) == 0 && jobID == "" {
			return nil
		}
		if len(jobs) != 1 || jobs[0].ID != jobID {
			return BadAuthRequest("生成任务已变化，请刷新后再停止")
		}
		job := &jobs[0]
		now := time.Now()
		previousStatus := job.Status
		if job.TaskID != "" {
			task, err := repo.TaskForUser(userID, job.TaskID)
			if err != nil {
				return err
			}
			if task.ProjectID != projectID || task.Type != "canvas_video" {
				return BadAuthRequest("作业绑定的视频生成任务不属于本作品")
			}
			if task.Status != model.TaskStatusQueued && task.Status != model.TaskStatusRunning {
				return BadAuthRequest("生成任务已结束，请刷新查看结果")
			}
			if task.ProviderRequestID == "" && task.BillingOrderID != "" {
				order, err := repo.BillingOrder(task.BillingOrderID)
				if err != nil {
					return err
				}
				task.ProviderRequestID = order.ProviderRequestID
			}
			if task.ProviderRequestID == "" {
				id, err := repo.LatestProviderRequestIDForTask(task.ID)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				task.ProviderRequestID = id
			}
			submitted := taskUpstreamSubmitted(task) || task.Status == model.TaskStatusRunning
			recordTaskDiagnostic(task, "cancellation", "task_cancelled", taskSubmissionOutcome(task, task.Status == model.TaskStatusQueued), "reconcile_cancellation", nil)
			ok, err := repo.CancelTaskIfStatus(userID, task.ID, task.Status, now, model.TaskCancellationIntent{Source: model.TaskCancellationDramaStop, ActorID: userID, RequestedAt: now, Diagnostic: *task.Diagnostic})
			if err != nil {
				return err
			}
			if !ok {
				return BadAuthRequest("任务已结束或刚刚开始执行，请刷新后重试")
			}
			task.Status, task.CancellationSource = model.TaskStatusCancelled, model.TaskCancellationDramaStop
			stoppedTask, upstream = task, submitted
			if submitted {
				message = "已停止本地排队；原上游取消状态和费用待核对"
				if err := repo.UpdateTaskProviderState(task.ID, task.ProviderRequestID, task.PollStage, nil); err != nil {
					return err
				}
				if err := repo.ClaimTaskProviderCancellation(userID, task.ID, now); err != nil {
					return err
				}
			}
			if task.BillingOrderID != "" {
				if submitted {
					err = repo.MarkBillingUncertain(task.BillingOrderID, message)
				} else {
					err = repo.RefundBillingOrder(task.BillingOrderID, message)
				}
				if err != nil {
					return err
				}
			}
			if !dramaAttemptRecorded(job, task.ID) {
				var cost int64
				if task.BillingOrderID != "" {
					order, err := repo.BillingOrder(task.BillingOrderID)
					if err != nil {
						return err
					}
					charged, held, _ := dramaOrderSpend(*order)
					cost = charged + held
				}
				recordDramaAttempt(job, task.ID, string(task.Status), message, cost)
				job.CostMicrocredits = cost
			}
		}
		job.Status, job.FinishedAt, job.Error = model.DramaJobCanceled, &now, message
		return repo.BindDramaJobTask(job, previousStatus, job.TaskID)
	})
	if err != nil {
		return "", err
	}
	if stoppedTask != nil {
		s.cancelActiveTask(stoppedTask.ID)
		_ = s.log(userID, stoppedTask.ID, "warn", message, "")
		if upstream {
			run := func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := s.sendClaimedProviderCancellation(ctx, stoppedTask); err != nil {
					_ = s.log(userID, stoppedTask.ID, "warn", "上游取消状态待核对", safeProviderLogError(err))
				}
			}
			if !s.runWorkerTask(run) {
				run()
			}
		}
	}
	return message, nil
}
