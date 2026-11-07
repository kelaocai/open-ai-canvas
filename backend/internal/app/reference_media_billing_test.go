package app

import (
	"encoding/json"
	"math"
	"testing"

	"yingce/backend/internal/model"
)

func TestReferenceMediaAmount(t *testing.T) {
	for _, tc := range []struct{ images, millis, bps, want int64 }{
		{2, 2500, 10000, 221500000}, {0, 1, 10000, 67000}, {1, 0, 15000, 40500000}, {0, 0, 10000, 0},
	} {
		got, err := referenceMediaAmount(27000000, 67000000, tc.images, tc.millis, tc.bps)
		if err != nil || got != tc.want {
			t.Fatalf("amount=%d err=%v want=%d", got, err, tc.want)
		}
	}
	if _, err := referenceMediaAmount(math.MaxInt64, 1, 128, 1, 10000); err == nil {
		t.Fatal("overflow accepted")
	}
	if err := validateReferencePrices("video", "token", 1, 0); err == nil {
		t.Fatal("token plus reference prices accepted")
	}
}

func TestReferenceMediaQuoteReservationSettlementAndRefund(t *testing.T) {
	svc, db, channel, item, _ := newVideoTokenQuoteFixture(t)
	if err := db.Model(&model.ChannelModelPriceTier{}).Where("channel_model_id = ?", item.ID).Updates(map[string]any{
		"billing_mode": "per_second", "unit_price_microcredits": 67000000, "reference_image_price_microcredits": 27000000, "reference_video_price_microcredits": 67000000,
		"cost_configured": true, "cost_unit_price_microcredits": 50000000, "cost_reference_image_price_microcredits": 20000000, "cost_reference_video_price_microcredits": 50000000,
	}).Error; err != nil {
		t.Fatal(err)
	}
	intent := ModelRequestIntent{Capability: "video", Inputs: map[string]int{"image": 2}, Options: map[string]any{"vquality": "720p", "size": "16:9", "videoSeconds": 5}}
	quote, err := svc.QuoteChannelModel("quote-user", ChannelModelQuoteRequest{ChannelID: channel.ID, ModelKey: item.ModelKey, Intent: intent})
	if err != nil {
		t.Fatal(err)
	}
	if quote.AmountMicrocredits != 389000000 || quote.ReferenceAmountMicrocredits != 54000000 || quote.Estimated {
		t.Fatalf("quote=%#v", quote)
	}
	input := quoteInput(intent, item.ModelKey)
	input["config"].(map[string]any)["channelId"] = channel.ID
	resolved, err := svc.resolveSystemChannelModelSelection(testGroupAccess(svc), input, "canvas_video", "")
	if err != nil {
		t.Fatal(err)
	}
	order, err := svc.taskBillingOrder("quote-user", &model.Task{ID: "reference-task", Type: "canvas_video"}, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMicrocredits != quote.AmountMicrocredits || order.ReferenceImageCount != 2 || order.CostReferenceAmountMicrocredits != 40000000 {
		t.Fatalf("order=%#v", order)
	}
	if err := db.Model(&model.CreditAccount{}).Where("user_id = ?", "quote-user").Updates(map[string]any{"available_microcredits": 2000000000, "reserved_microcredits": 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.ReserveBillingOrder(order); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.SettleBillingOrder(order.ID, ""); err != nil {
		t.Fatal(err)
	}
	var settled model.BillingOrder
	if err := db.First(&settled, "id = ?", order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if settled.ActualAmountMicrocredits != 389000000 {
		t.Fatalf("settled=%#v", settled)
	}
	cost, err := billingCreditCost(settled)
	if err != nil || cost == nil || *cost != 290000000 {
		t.Fatalf("cost=%v err=%v", cost, err)
	}
	// 修改当前价格后，历史订单仍使用下单时的销售与成本快照。
	db.Model(&model.ChannelModelPriceTier{}).Where("channel_model_id = ?", item.ID).Update("cost_reference_image_price_microcredits", 0)
	cost, err = billingCreditCost(settled)
	if err != nil || *cost != 290000000 {
		t.Fatal("historical cost changed")
	}
	next := *order
	next.ID = "reference-failed"
	next.IdempotencyKey = "reference-failed"
	next.TaskID = "reference-failed"
	if err := svc.repo.ReserveBillingOrder(&next); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.RefundBillingOrder(next.ID, "upstream failed"); err != nil {
		t.Fatal(err)
	}
	var refunded model.BillingOrder
	db.First(&refunded, "id = ?", next.ID)
	if refunded.RefundedAmountMicrocredits != 389000000 {
		t.Fatalf("refund=%#v", refunded)
	}
}

func TestReferenceVideoUsageRejectsForgedDurationAndOwner(t *testing.T) {
	requireFFmpegTools(t)
	svc, db := newMediaRecoveryTestService(t)
	r := seedLocalVideoResource(t, svc, "owner", 64, 64, nil)
	profile := DefaultModelCapabilityConfigForModel("minimax-video", "MiniMax-H3")
	profile.Video.References.MaxVideoDuration = 15
	profile.Video.References.MaxVideoBytes = 50 * 1024 * 1024
	raw, _ := json.Marshal(profile)
	item := &model.ChannelModel{CapabilityConfigJSON: string(raw)}
	media := map[string]any{"storageKey": "resource:" + r.ID, "durationMs": 1}
	input := map[string]any{"referenceImages": []any{nil}, "referenceVideos": []any{media}}
	images, millis, estimated, err := svc.referenceBillingUsage("owner", "task", item, input)
	if err != nil || images != 1 || millis < 1000 || millis > 1100 || estimated {
		t.Fatalf("usage=%d %d %v %v", images, millis, estimated, err)
	}
	if _, _, _, err := svc.referenceBillingUsage("other", "task", item, input); err == nil {
		t.Fatal("foreign resource accepted")
	}
	input["referenceVideos"] = []any{map[string]any{"url": "https://cdn.example/test.mp4", "durationMs": 1}}
	if _, _, _, err := svc.referenceBillingUsage("owner", "task", item, input); err == nil {
		t.Fatal("unverified external duration accepted")
	}
	input["referenceVideos"] = []any{nil, nil}
	_, millis, estimated, err = svc.referenceBillingUsage("owner", "", item, input)
	if err != nil || millis != 30000 || !estimated {
		t.Fatalf("quote usage=%d %v %v", millis, estimated, err)
	}
	_ = db
}
