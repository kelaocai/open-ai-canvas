package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"yingce/backend/internal/model"
)

func TestDramaBatchVideoStaleOverride(t *testing.T) {
	for _, tc := range []struct {
		name   string
		choice *DramaModelChoice
	}{
		{"use displayed model", &fakeDramaModels.Video},
		{"clear override", &DramaModelChoice{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, fake := newDramaTestService(t)
			video := dramaVideoFixture()
			video.Resolutions, video.DefaultResolution = []string{"720p"}, "720p"
			setDramaVideoCapability(t, svc, fake, video)
			if err := fake.db.Create(&model.ModelChannel{ID: "ch", Name: "测试视频渠道", Scope: model.ChannelScopeSystem, Enabled: true}).Error; err != nil {
				t.Fatal(err)
			}
			if err := fake.db.Create(&model.ChannelModelPriceTier{ID: "drama-price", ChannelModelID: "drama-video", SelectorKey: "{}", SelectorJSON: "{}", Resolution: "*", ProviderModelKey: "video", BillingMode: "fixed_request", Enabled: true, PriceConfigured: true}).Error; err != nil {
				t.Fatal(err)
			}
			project := createTestDrama(t, svc, DramaReviewAuto).Project.ID
			runDramaTicks(t, svc, project, 24)
			state, err := svc.DramaSummary("usr-drama", project)
			if err != nil {
				t.Fatal(err)
			}
			episode := state.Episodes[0].ID
			rows, err := svc.repo.DramaSegments(episode)
			if err != nil || len(rows) != 2 {
				t.Fatalf("segments: %d %v", len(rows), err)
			}
			old, err := json.Marshal(DramaModelChoice{ChannelID: "ch", ModelKey: "removed", Label: "历史模型"})
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.repo.UpdateDramaSegmentFields(rows[1].ID, map[string]any{"video_model_json": string(old)}); err != nil {
				t.Fatal(err)
			}
			req := GenerateDramaSegmentsRequest{Segments: []int{0, 1}}
			_, err = svc.GenerateDramaSegments("usr-drama", project, episode, req)
			var public *AppError
			if !errors.As(err, &public) || public.Status != http.StatusBadRequest || !strings.Contains(public.Message, "第 2 段") || !strings.Contains(public.Message, "历史模型") {
				t.Fatalf("unhelpful error: %v", err)
			}
			unchanged, err := svc.repo.DramaSegment(project, rows[0].ID)
			if err != nil || unchanged.Revision != rows[0].Revision {
				t.Fatal("invalid batch partially queued")
			}
			req.VideoModel = tc.choice
			result, err := svc.GenerateDramaSegments("usr-drama", project, episode, req)
			if err != nil || result["queued"] != 2 {
				t.Fatalf("retry: %v %v", result, err)
			}
			updated, err := svc.repo.DramaSegment(project, rows[1].ID)
			if err != nil || strings.Contains(updated.VideoModelJSON, "removed") || updated.VideoStatus != "queued" {
				t.Fatalf("override not replaced: %+v %v", updated, err)
			}
		})
	}
}
