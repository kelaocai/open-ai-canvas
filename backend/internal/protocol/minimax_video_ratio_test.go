package protocol

import (
	"context"
	"strings"
	"testing"
)

// 首尾帧与多模态参考互斥；普通衔接参考图使用作品比例。
func TestMiniMaxVideoRatioWithFirstFrameAndReferences(t *testing.T) {
	adapter := officialPackageAdapter(t, "minimax-hailuo-video-v2.yingce-plugin", "minimax-video")
	firstFrame := MediaReference{Kind: "image", Role: "first_frame", URL: "https://cdn.example/first.png"}
	reference := MediaReference{Kind: "image", Role: "reference_image", URL: "https://cdn.example/cast.png", Order: 1}
	for _, tc := range []struct {
		name   string
		images []MediaReference
		ratio  string
		want   string
	}{
		{"continuity reference keeps requested ratio", []MediaReference{reference, {Kind: "image", Role: "reference_image", URL: firstFrame.URL, Order: 2}}, "9:16", "9:16"},
		{"first frame only follows the frame", []MediaReference{firstFrame}, "9:16", "adaptive"},
		{"references only", []MediaReference{reference}, "9:16", "9:16"},
		{"text only defaults to landscape", nil, "", "16:9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{Model: "MiniMax-H3", Prompt: "test", Images: tc.images, AspectRatio: tc.ratio, Duration: 15, Resolution: "768P"}})
			if err != nil {
				t.Fatal(err)
			}
			if got := manifestTestBody(t, spec)["ratio"]; got != tc.want {
				t.Fatalf("ratio = %v, want %s", got, tc.want)
			}
		})
	}
}

func TestMiniMaxVideoRejectsMixedFrameAndReferences(t *testing.T) {
	adapter := officialPackageAdapter(t, "minimax-hailuo-video-v2.yingce-plugin", "minimax-video")
	for _, kind := range []string{"image", "video", "audio"} {
		t.Run(kind, func(t *testing.T) {
			request := GenerationRequest{Model: "MiniMax-H3", Prompt: "test", Duration: 14, Images: []MediaReference{{Role: "first_frame", URL: "https://cdn.example/first.png"}}}
			switch kind {
			case "image":
				request.Images = append(request.Images, MediaReference{Role: "reference_image", URL: "https://cdn.example/cast.png"})
			case "video":
				request.Videos = []MediaReference{{Role: "reference_video", URL: "https://cdn.example/video.mp4"}}
			case "audio":
				request.Audios = []MediaReference{{Role: "reference_audio", URL: "https://cdn.example/audio.mp3"}}
			}
			_, err := adapter.BuildCreate(context.Background(), RequestContext{Request: request})
			if err == nil || !strings.Contains(err.Error(), "首尾帧不能与") {
				t.Fatalf("mixed %s error = %v", kind, err)
			}
		})
	}
}
