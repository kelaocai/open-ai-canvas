package protocol

import (
	"context"
	"encoding/json"
	"testing"
)

func TestModelverseH3MultimodalContract(t *testing.T) {
	a := officialPackageAdapter(t, "modelverse-h3.yingce-plugin", "modelverse-h3-video")
	ctx := context.Background()
	request := GenerationRequest{Model: "MiniMax-H3", Prompt: "test", Duration: 5, Resolution: "1440p", AspectRatio: "16:9", Images: []MediaReference{{URL: "https://cdn.example/image.png", Role: "reference_image"}}, Videos: []MediaReference{{URL: "https://cdn.example/video.mp4", Role: "reference_video"}}}
	spec, err := a.BuildCreate(ctx, RequestContext{Request: request})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(spec.Body)
	var body struct {
		Model string
		Input struct {
			Content []struct {
				Type, Role string
				ImageURL   map[string]string `json:"image_url"`
			}
		}
		Parameters struct {
			Resolution, Ratio string
			Duration          int
		}
	}
	if err = json.Unmarshal(b, &body); err != nil {
		t.Fatal(err)
	}
	if spec.Path != "/v1/tasks/submit" || body.Parameters.Resolution != "2K" || body.Parameters.Duration != 5 || body.Parameters.Ratio != "16:9" || len(body.Input.Content) != 3 || body.Input.Content[1].Role != "reference_image" || body.Input.Content[1].ImageURL["url"] != "https://cdn.example/image.png" {
		t.Fatalf("wrong request: %s", b)
	}
	request.Images[0].Role = "first_frame"
	request.Videos = nil
	request.Resolution = "768p"
	spec, err = a.BuildCreate(ctx, RequestContext{Request: request})
	if err != nil {
		t.Fatal(err)
	}
	b, _ = json.Marshal(spec.Body)
	_ = json.Unmarshal(b, &body)
	if body.Parameters.Ratio != "adaptive" || body.Parameters.Resolution != "768P" {
		t.Fatalf("wrong frame request: %s", b)
	}
	created, err := a.ParseCreate(ctx, []byte(`{"output":{"task_id":"h3-123"},"request_id":"req-1"}`))
	if err != nil || created.TaskID != "h3-123" || created.Status != StatusPending {
		t.Fatalf("create=%#v %v", created, err)
	}
	poll, err := a.BuildPoll(ctx, PollContext{TaskID: created.TaskID})
	if err != nil {
		t.Fatal(err)
	}
	if poll.Path != "/v1/tasks/status" || len(poll.Query["task_id"]) != 1 || poll.Query["task_id"][0] != "h3-123" {
		t.Fatalf("poll=%#v", poll)
	}
	for _, tc := range []struct {
		body   string
		status Status
	}{
		{`{"output":{"task_status":"Running"}}`, StatusProcessing},
		{`{"output":{"task_status":"Success","urls":["https://cdn.example/result.mp4"]}}`, StatusSucceeded},
		{`{"output":{"task_status":"Failure","error_message":"invalid dimensions"}}`, StatusFailed},
	} {
		state, e := a.ParsePoll(ctx, PollContext{TaskID: created.TaskID}, []byte(tc.body))
		if e != nil {
			t.Fatal(e)
		}
		if state.Status != tc.status {
			t.Fatalf("state=%#v", state)
		}
		if tc.status == StatusSucceeded && (state.Result == nil || len(state.Result.Videos) != 1) {
			t.Fatalf("missing video: %#v", state)
		}
		if tc.status == StatusFailed && state.Message != "invalid dimensions" {
			t.Fatalf("missing error: %#v", state)
		}
	}
}

func TestModelverseH3RejectsInvalidModes(t *testing.T) {
	a := officialPackageAdapter(t, "modelverse-h3.yingce-plugin", "modelverse-h3-video")
	base := GenerationRequest{Model: "MiniMax-H3", Prompt: "test", Duration: 5, Resolution: "768p", AspectRatio: "16:9"}
	for _, change := range []func(*GenerationRequest){
		func(r *GenerationRequest) { r.Duration = 30 },
		func(r *GenerationRequest) { r.Resolution = "1080p" },
		func(r *GenerationRequest) { r.Model = "MiniMax-H3-Max" },
		func(r *GenerationRequest) {
			r.Audios = []MediaReference{{URL: "https://cdn.example/a.mp3", Role: "reference_audio"}}
		},
		func(r *GenerationRequest) {
			r.Images = []MediaReference{{URL: "https://cdn.example/a.png", Role: "first_frame"}, {URL: "https://cdn.example/b.png", Role: "reference_image"}}
		},
	} {
		r := base
		change(&r)
		if _, err := a.BuildCreate(context.Background(), RequestContext{Request: r}); err == nil {
			t.Fatalf("accepted invalid request: %#v", r)
		}
	}
}
