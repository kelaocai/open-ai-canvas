package app

import (
	"testing"

	"yingce/backend/internal/model"
)

func TestStopDramaSegmentVideoOwnsAttemptAndClearsQueue(t *testing.T) {
	svc, fake := newDramaTestService(t)
	project := createTestDrama(t, svc, DramaReviewSample).Project.ID
	segment := model.DramaSegment{ID: "stop-segment", ProjectID: project, EpisodeID: "stop-episode", Revision: 3, VideoStatus: model.DramaMediaRunning}
	task := model.Task{ID: "stop-task", UserID: "usr-drama", ProjectID: project, Type: "canvas_video", Status: model.TaskStatusRunning, Stage: "上游排队中", ProviderRequestID: "old-job", InputJSON: "{}"}
	job := model.DramaJob{ID: "stop-job", ProjectID: project, EpisodeID: segment.EpisodeID, ShotID: segment.ID, Kind: model.DramaJobSegmentVideo, Status: model.DramaJobRunning, TaskID: task.ID}
	for _, row := range []any{&segment, &task, &job} {
		if err := fake.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	stop := func(user, jobID string) error {
		_, err := svc.StopDramaSegmentVideo(user, project, segment.EpisodeID, segment.ID, jobID, 3)
		return err
	}
	if stop("other", job.ID) == nil || stop(task.UserID, "stale-job") == nil {
		t.Fatal("scope or attempt check bypassed")
	}
	got, err := svc.repo.DramaSegment(project, segment.ID)
	if err != nil || got.VideoStatus != model.DramaMediaRunning {
		t.Fatalf("rejected request changed segment: %+v %v", got, err)
	}
	if err := stop(task.UserID, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := stop(task.UserID, job.ID); err != nil {
		t.Fatal(err)
	}
	got, err = svc.repo.DramaSegment(project, segment.ID)
	if err != nil || got.VideoStatus != model.DramaMediaStopped {
		t.Fatalf("slot %+v %v", got, err)
	}
	stored, err := svc.repo.TaskForUser(task.UserID, task.ID)
	if err != nil || stored.Status != model.TaskStatusCancelled || stored.CancellationSource != model.TaskCancellationDramaStop {
		t.Fatalf("task %+v %v", stored, err)
	}
	j, err := svc.repo.DramaJob(project, job.ID)
	if err != nil || j.Status != model.DramaJobCanceled || !dramaAttemptRecorded(j, task.ID) {
		t.Fatalf("job %+v %v", j, err)
	}
}

func TestStopDramaSegmentBeforeDispatch(t *testing.T) {
	svc, fake := newDramaTestService(t)
	project := createTestDrama(t, svc, DramaReviewSample).Project.ID
	seg := model.DramaSegment{ID: "waiting-segment", ProjectID: project, EpisodeID: "waiting-episode", Revision: 4, VideoStatus: model.DramaMediaQueued}
	if err := fake.db.Create(&seg).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StopDramaSegmentVideo("usr-drama", project, seg.EpisodeID, seg.ID, "", 3); err == nil {
		t.Fatal("stale stop allowed")
	}
	if _, err := svc.StopDramaSegmentVideo("usr-drama", project, seg.EpisodeID, seg.ID, "", 4); err != nil {
		t.Fatal(err)
	}
	got, err := svc.repo.DramaSegment(project, seg.ID)
	if err != nil || got.VideoStatus != model.DramaMediaStopped {
		t.Fatalf("slot %+v %v", got, err)
	}
}
