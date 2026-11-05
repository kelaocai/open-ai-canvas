package protocol

import (
	"context"
	"testing"
)

func TestWalkingWithAIH3Contract(t *testing.T) {
	a := officialPackageAdapter(t, "walkingwithai-h3.yingce-plugin", "walkingwithai-h3")
	media := MediaReference{Kind: "image", MIMEType: "image/png", Role: "reference_image", Metadata: map[string]any{MediaUploadIDKey: "asset-1"}}
	uploader := a.(MediaUploadAdapter)
	upload, err := uploader.BuildUpload(context.Background(), RequestContext{}, media)
	if err != nil || upload.Path != "/api/v1/assets" || upload.ContentType != "multipart/form-data" {
		t.Fatalf("upload: %+v %v", upload, err)
	}
	id, err := uploader.ParseUpload(context.Background(), []byte(`{"asset_id":"asset-1"}`))
	if err != nil || id != "asset-1" {
		t.Fatalf("upload ID: %q %v", id, err)
	}
	spec, err := a.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{Model: "minimax-h3", Prompt: "test", Duration: 5, Resolution: "480P", AspectRatio: "9:16", Images: []MediaReference{media}}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, spec)
	if spec.Path != "/api/v1/jobs" || body["mode"] != "r2v" {
		t.Fatalf("request: %+v", spec)
	}
	inputs := body["inputs"].(map[string]any)
	images := inputs["reference_images"].([]any)
	if len(images) != 1 || images[0] != "asset-1" {
		t.Fatalf("reference assets: %+v", inputs)
	}
	first := media
	first.Role = "first_frame"
	spec, err = a.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{Model: "minimax-h3", Prompt: "continue", Images: []MediaReference{first}}})
	if err != nil {
		t.Fatal(err)
	}
	body = manifestTestBody(t, spec)
	inputs = body["inputs"].(map[string]any)
	if body["mode"] != "i2v" || inputs["first_frame"] != "asset-1" || inputs["reference_images"] != nil {
		t.Fatalf("first-frame request: %+v", body)
	}
	if _, err := a.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{Images: []MediaReference{first, media}}}); err == nil {
		t.Fatal("mixed first-frame/reference request must be rejected")
	}
	download, err := a.(ResultAdapter).BuildResult(context.Background(), PollContext{TaskID: "job-1"})
	if err != nil || download.Path != "/api/v1/jobs/job-1/outputs" {
		t.Fatalf("download: %+v %v", download, err)
	}
	outputs, err := a.ParsePoll(context.Background(), PollContext{TaskID: "job-1"}, []byte(`{"job_id":"job-1","outputs":[{"url":"/view?filename=result.mp4&subfolder=video&type=output"}]}`))
	if err != nil || outputs.Result == nil || len(outputs.Result.Videos) != 1 || outputs.Result.Videos[0].URL != "/view?filename=result.mp4&subfolder=video&type=output" {
		t.Fatalf("outputs: %+v %v", outputs, err)
	}
	created, err := a.ParseCreate(context.Background(), []byte(`{"job_id":"job-1","status":"queued"}`))
	if err != nil || created.TaskID != "job-1" {
		t.Fatalf("create: %+v %v", created, err)
	}
	result, err := a.ParsePoll(context.Background(), PollContext{TaskID: "job-1"}, []byte(`{"job_id":"job-1","status":"completed"}`))
	if err != nil || result.Status != StatusSucceeded {
		t.Fatalf("poll: %+v %v", result, err)
	}
}
