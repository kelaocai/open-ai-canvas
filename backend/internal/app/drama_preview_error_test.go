package app

import (
	"encoding/json"
	"testing"

	"yingce/backend/internal/model"
)

func TestDramaPreviewModelErrorKeepsArrayContract(t *testing.T) {
	svc, _ := newDramaTestService(t)
	request, _, _, err := svc.prepareDramaSegment(DramaSettings{}, dramaSegmentDraft{}, model.DramaSegment{}, nil, nil)
	if err == nil {
		t.Fatal("missing model must remain an explicit error")
	}
	request.Warning = err.Error()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if refs, ok := payload["references"].([]any); !ok || len(refs) != 0 {
		t.Fatalf("failed preview must serialize references as [], got %s", data)
	}
	if request.AudioReferences == nil || request.Warning == "" {
		t.Fatalf("failed preview contract: %+v", request)
	}
}
