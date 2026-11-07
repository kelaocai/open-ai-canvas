package protocol

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestModelverseImagesUsesMultipartReferences(t *testing.T) {
	adapter := officialPackageAdapter(t, "modelverse-images.yingce-plugin", "modelverse-image")
	for _, edit := range []bool{false, true} {
		request := GenerationRequest{Model: "gpt-image-2", Prompt: "add title"}
		if edit {
			request.Images = []MediaReference{
				{DataURL: "data:image/png;base64,aW1hZ2U=", Role: "edit_source", Order: 0},
				{URL: "https://cdn.example/second.png", Role: "edit_source", Order: 1},
				{URL: "https://cdn.example/mask.png", Role: "mask", Order: 2},
			}
		}
		spec, err := adapter.BuildCreate(context.Background(), RequestContext{Request: request})
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(spec.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Images []string        `json:"images"`
			Mask   json.RawMessage `json:"mask"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Mask) != 0 {
			t.Fatal("mask must not be sent")
		}
		if edit {
			if spec.Path != "/v1/images/edits" || spec.ContentType != "multipart/form-data" || len(spec.Files) != 2 || len(body.Images) != 0 {
				t.Fatalf("edit payload: %s %s", spec.Path, data)
			}
			if !reflect.DeepEqual([]string{spec.Files[0].Name, spec.Files[1].Name}, []string{"image", "image"}) || spec.Files[0].Reference.DataURL != request.Images[0].DataURL || spec.Files[1].Reference.URL != request.Images[1].URL {
				t.Fatalf("unexpected files: %#v", spec.Files)
			}
		} else if spec.Path != "/v1/images/generations" || spec.ContentType != "application/json" || len(spec.Files) != 0 || len(body.Images) != 0 {
			t.Fatalf("generation payload: %s %s", spec.Path, data)
		}
	}
}
