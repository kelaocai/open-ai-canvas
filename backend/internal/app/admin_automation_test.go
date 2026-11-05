package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yingce/backend/internal/model"
)

func TestAutomationPriceScopeAndAtomicAudit(t *testing.T) {
	s, db, admin := seedRepricingModels(t)
	admin.Status = model.UserStatusActive
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 48)
	file := filepath.Join(t.TempDir(), "grants.json")
	t.Setenv("RIGHTMOVIE_ADMIN_GRANTS", file)
	grant := automationGrant{Name: "test", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(token))), ActorID: admin.ID, Channels: []string{"channel"}}
	save := func() {
		b, err := json.Marshal([]automationGrant{grant})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(file, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	save()
	price := int64(47_000_000)
	req := AdminAutomationRequest{Action: "apply", ChannelID: "channel", Reason: "核价", Changes: []AutomationPrice{{ModelID: "a", TierID: "tier-a", ModelVersion: 1, TierVersion: 1, Cost: model.CreditCostPricing{Configured: true, UnitPriceMicrocredits: 35_000_000}, Prices: map[string]*int64{"unitPriceMicrocredits": &price}}}}
	rejected := func() {
		t.Helper()
		if _, err := s.AdminAutomation(token, req); err == nil {
			t.Fatal("expected rejection", req.Action, req.ChannelID)
		}
	}
	rejected()
	grant.Write = true
	save()
	req.Action = "preview"
	req.ChannelID = "other"
	rejected()
	req.ChannelID = "channel"
	if _, err := s.AdminAutomation(token, req); err != nil {
		t.Fatal(err)
	}
	req.Action = "apply"
	// Missing audit table forces a late failure. Prices must roll back with it.
	rejected()
	if err := db.AutoMigrate(&model.AdminAuditEvent{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AdminAutomation(token, req); err != nil {
		t.Fatal(err)
	}
	rejected()
	item, err := s.repo.ChannelModelByID("channel", "a")
	if err != nil || item.PriceVersion != 2 || item.PriceTiers[0].CostPricing.UnitPriceMicrocredits != 35_000_000 || item.PriceTiers[0].UnitPriceMicrocredits != price {
		t.Fatal(item, err)
	}
	_, count, err := s.repo.AdminAuditEvents("mcp.channel", "channel", 50, 0)
	if err != nil || count != 1 {
		t.Fatal(count, err)
	}
}
