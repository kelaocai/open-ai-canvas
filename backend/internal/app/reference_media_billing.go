package app

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"

	"yingce/backend/internal/model"
)

func validateReferencePrices(capability, mode string, imagePrice, videoPrice int64) error {
	if imagePrice < 0 || videoPrice < 0 || imagePrice > 1_000_000_000_000 || videoPrice > 1_000_000_000_000 {
		return BadAuthRequest("参考素材单价必须在 0 到 1000000 之间")
	}
	if (imagePrice != 0 || videoPrice != 0) && (capability != "video" || mode != "per_second") {
		return BadAuthRequest("参考素材附加计费仅适用于按秒计费的视频模型")
	}
	return nil
}

// 图片按张、输入视频按毫秒计算，合计后仅向上取整一次。
func referenceMediaAmount(imagePrice, videoPrice, images, millis, multiplier int64) (int64, error) {
	if imagePrice < 0 || videoPrice < 0 || images < 0 || millis < 0 || multiplier <= 0 {
		return 0, BadAuthRequest("参考素材计费参数无效")
	}
	n := new(big.Int).Mul(big.NewInt(imagePrice), big.NewInt(images))
	n.Mul(n, big.NewInt(1000))
	n.Add(n, new(big.Int).Mul(big.NewInt(videoPrice), big.NewInt(millis)))
	n.Mul(n, big.NewInt(multiplier))
	const denominator = int64(1000 * 10000)
	n.Add(n, big.NewInt(denominator-1)).Div(n, big.NewInt(denominator))
	if !n.IsInt64() {
		return 0, BadAuthRequest("参考素材计费金额溢出")
	}
	return n.Int64(), nil
}

// 报价只有数量时使用模型允许的单段最大时长，并明确标记预估。
// 正式下单只接受归属校验后的已上传视频，客户端 durationMs 不作为账务依据。
func (s *Service) referenceBillingUsage(userID, taskID string, item *model.ChannelModel, input map[string]any) (images, millis int64, estimated bool, err error) {
	imageItems, _ := input["referenceImages"].([]any)
	videoItems, _ := input["referenceVideos"].([]any)
	if len(imageItems) > 128 || len(videoItems) > 128 {
		return 0, 0, false, BadAuthRequest("参考素材数量超出计费上限")
	}
	images = int64(len(imageItems))
	if len(videoItems) == 0 {
		return images, 0, false, nil
	}
	profile, decodeErr := DecodeModelCapabilityConfig(item.CapabilityConfigJSON)
	if decodeErr != nil || profile == nil || profile.Video == nil || profile.Video.References.MaxVideoDuration <= 0 {
		return 0, 0, false, BadAuthRequest("参考视频计费缺少有效的时长限制")
	}
	maxMillis := int64(profile.Video.References.MaxVideoDuration) * 1000
	for _, raw := range videoItems {
		media, _ := raw.(map[string]any)
		var duration int64
		if taskID == "" {
			duration, _ = strconv.ParseInt(fmt.Sprint(media["durationMs"]), 10, 64)
			if duration <= 0 {
				duration, estimated = maxMillis, true
			}
		} else {
			key := strings.TrimSpace(stringValue(media["storageKey"]))
			if !strings.HasPrefix(key, "resource:") {
				return 0, 0, false, BadAuthRequest("按时长计费的参考视频请先上传到素材库，不能使用未核验的外部地址")
			}
			resource, resourceErr := s.repo.ResourceForUser(userID, strings.TrimPrefix(key, "resource:"))
			if resourceErr != nil || resource.Status != "ready" || !strings.HasPrefix(strings.ToLower(resource.MimeType), "video/") {
				return 0, 0, false, BadAuthRequest("参考视频未就绪或缺少服务端核验时长")
			}
			duration, err = s.measuredReferenceVideoMillis(userID, resource.ID, profile.Video.References.MaxVideoBytes)
			if err != nil {
				return 0, 0, false, err
			}
			media["durationMs"] = duration
		}
		if duration <= 0 || duration > maxMillis || duration > math.MaxInt64-millis {
			return 0, 0, false, BadAuthRequest("参考视频时长超出模型允许范围")
		}
		millis += duration
	}
	return images, millis, estimated, nil
}

func (s *Service) measuredReferenceVideoMillis(userID, resourceID string, maxBytes int64) (int64, error) {
	if maxBytes <= 0 || maxBytes > 256*1024*1024 {
		return 0, BadAuthRequest("参考视频文件上限无效")
	}
	_, body, err := s.OpenResource(userID, resourceID)
	if err != nil {
		return 0, err
	}
	defer body.Close()
	file, err := os.CreateTemp("", "reference-billing-*.mp4")
	if err != nil {
		return 0, err
	}
	defer os.Remove(file.Name())
	n, copyErr := io.Copy(file, io.LimitReader(body, maxBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || n > maxBytes {
		return 0, BadAuthRequest("参考视频读取失败或超过模型文件上限")
	}
	info, err := probeMediaFile(context.Background(), file.Name())
	if err != nil || info.DurationMs <= 0 || info.Width <= 0 || info.Height <= 0 {
		return 0, BadAuthRequest("无法核验参考视频时长，请检查视频文件及服务器 ffprobe")
	}
	return info.DurationMs, nil
}
