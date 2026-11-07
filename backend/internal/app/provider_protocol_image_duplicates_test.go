package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
	"yingce/backend/internal/protocol"
)

func protocolTestPNG(t *testing.T, pixel color.NRGBA, metadataType, metadata string) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, pixel)
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	data := out.Bytes()
	if metadataType == "" {
		return data
	}
	chunk := append([]byte(metadataType), []byte(metadata)...)
	var encoded bytes.Buffer
	_ = binary.Write(&encoded, binary.BigEndian, uint32(len(metadata)))
	_, _ = encoded.Write(chunk)
	_ = binary.Write(&encoded, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	result := append([]byte{}, data[:len(data)-12]...)
	result = append(result, encoded.Bytes()...)
	return append(result, data[len(data)-12:]...)
}

func TestProtocolImagesCollapseSamePixelsWithDifferentProvenance(t *testing.T) {
	first := protocolTestPNG(t, color.NRGBA{R: 180, A: 255}, "caBX", "google-provenance-one")
	second := protocolTestPNG(t, color.NRGBA{R: 180, A: 255}, "caBX", "google-provenance-two")
	distinct := protocolTestPNG(t, color.NRGBA{B: 180, A: 255}, "", "")
	result, err := finishProtocolResult(context.Background(), providerConfig{}, "image", "", &protocol.Result{Images: []protocol.MediaReference{
		{DataURL: dataURL("image/png", first)}, {DataURL: dataURL("image/png", second)}, {DataURL: dataURL("image/png", distinct)},
	}}, videoPollPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	images := result["images"].([]interface{})
	if len(images) != 2 {
		t.Fatalf("got %d images, want the first image and the distinct image", len(images))
	}
	if images[0].(map[string]interface{})["dataUrl"] != dataURL("image/png", first) {
		t.Fatal("deduplication must retain the original first image and its provenance")
	}
}

func TestProtocolImageIdentityPreservesRenderingAndAnimationDifferences(t *testing.T) {
	for _, metadataType := range []string{"sRGB", "eXIf", "acTL"} {
		t.Run(metadataType, func(t *testing.T) {
			first := protocolTestPNG(t, color.NRGBA{R: 80, A: 255}, metadataType, "\x00")
			second := protocolTestPNG(t, color.NRGBA{R: 80, A: 255}, metadataType, "\x01")
			if protocolImageIdentity(first) == protocolImageIdentity(second) {
				t.Fatal("rendering or animation metadata differences must not be collapsed")
			}
		})
	}
}

func TestGeminiImagesCollapseSamePixelsWithDifferentProvenance(t *testing.T) {
	parts := make([]interface{}, 0, 2)
	for _, marker := range []string{"one", "two"} {
		data := protocolTestPNG(t, color.NRGBA{G: 150, A: 255}, "caBX", marker)
		parts = append(parts, map[string]interface{}{"inlineData": map[string]interface{}{"mimeType": "image/png", "data": base64.StdEncoding.EncodeToString(data)}})
	}
	images, err := geminiImageDataURLs(map[string]interface{}{"candidates": []interface{}{map[string]interface{}{"content": map[string]interface{}{"parts": parts}}}})
	if err != nil || len(images) != 1 {
		t.Fatalf("got %d images, err=%v; want one image", len(images), err)
	}
}

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
