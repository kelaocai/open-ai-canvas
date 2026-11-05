package app

import (
	"context"
	"testing"
	"yingce/backend/internal/protocol"
)

func TestProtocolImagesCollapseRepeatedInlineOutputs(t *testing.T) {
	first := protocol.MediaReference{DataURL: "data:image/png;base64,aGVsbG8="}
	second := protocol.MediaReference{DataURL: "data:image/png;base64,d29ybGQ="}
	result, err := finishProtocolResult(context.Background(), providerConfig{}, "image", "", &protocol.Result{Images: []protocol.MediaReference{first, first, second}}, videoPollPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	images := result["images"].([]interface{})
	if len(images) != 2 {
		t.Fatalf("got %d outputs, want two distinct images", len(images))
	}
	if images[0].(map[string]interface{})["dataUrl"] == images[1].(map[string]interface{})["dataUrl"] {
		t.Fatal("distinct outputs were lost")
	}
}
