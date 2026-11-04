package app

import (
	"strings"
	"testing"

	"yingce/backend/internal/model"
)

func TestDramaMiniMaxContinuityUsesOrdinaryReference(t *testing.T) {
	svc, fake := newDramaTestService(t)
	if err := fake.db.Model(&model.ChannelModel{}).Where("channel_id = ? AND model_key = ?", "ch", "video").Update("protocol", model.ChannelInterfaceMiniMaxVideo).Error; err != nil {
		t.Fatal(err)
	}
	profile, err := svc.dramaVideoProfile(fakeDramaModels.Video)
	if err != nil || !profile.ContinuityAsReference {
		t.Fatalf("profile = %+v, error = %v", profile, err)
	}
	if operation, _ := dramaSegmentOperation(DramaSettings{}, profile); operation != "reference_to_video" {
		t.Fatalf("quote operation = %s", operation)
	}
	project := createTestDrama(t, svc, DramaReviewSample).Project.ID
	runDramaTicks(t, svc, project, 24)
	requests := dramaVideoRequests(fake)
	if len(requests) < 2 {
		t.Fatalf("video requests = %d", len(requests))
	}
	request := requests[1]
	if request.Operation != "reference_to_video" || request.Input["metadata"].(map[string]any)["videoStartFrameNodeId"] != nil {
		t.Fatalf("continuity must use reference mode: %+v", request)
	}
	refs := request.Input["referenceImages"].([]any)
	if len(refs) < 2 || refs[len(refs)-1].(map[string]any)["role"] != "reference_image" || refs[len(refs)-1].(map[string]any)["id"] != "first-frame" {
		t.Fatalf("tail must follow cast references: %+v", refs)
	}
	for _, raw := range refs {
		if raw.(map[string]any)["role"] == "first_frame" {
			t.Fatal("first_frame must not be sent with references")
		}
	}
	if !strings.Contains(request.Prompt, "最后一张输入图片") || strings.Contains(request.Prompt, "首帧图为上一段视频的最后一帧，不占参考图编号") {
		t.Fatalf("wrong continuity description: %s", request.Prompt)
	}
}
