package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"yingce/backend/internal/model"
	"yingce/backend/internal/repository"
)

// A deployment grants specific channels to an existing administrator. Re-read
// the mounted file for each request so rotation and revocation take effect online.
type automationGrant struct {
	Name     string
	SHA256   string
	ActorID  string
	Channels []string
	Write    bool
}

func (g automationGrant) allows(channel string) bool {
	for _, id := range g.Channels {
		if id == "*" || id == channel {
			return true
		}
	}
	return false
}

type AutomationPrice struct {
	ModelID      string
	TierID       string
	ModelVersion int64
	TierVersion  int64
	Cost         model.CreditCostPricing
	Prices       map[string]*int64
}
type AdminAutomationRequest struct {
	ChannelID string
	Action    string
	Reason    string
	Changes   []AutomationPrice
	ModelKey  string
	Intent    ModelRequestIntent
}

func (s *Service) AdminAutomation(token string, req AdminAutomationRequest) (any, error) {
	raw, err := os.ReadFile(os.Getenv("RIGHTMOVIE_ADMIN_GRANTS"))
	var grants []automationGrant
	if err != nil || json.Unmarshal(raw, &grants) != nil || len(token) < 32 {
		return nil, Unauthorized("管理凭证无效")
	}
	sum := sha256.Sum256([]byte(token))
	for _, g := range grants {
		digest, err := hex.DecodeString(g.SHA256)
		if err != nil || subtle.ConstantTimeCompare(sum[:], digest) != 1 {
			continue
		}
		actor, err := s.repo.User(g.ActorID)
		if err != nil || actor.Status != model.UserStatusActive || s.RequireAdmin(actor) != nil {
			return nil, Forbidden("管理身份不可用")
		}
		if req.Action == "channels" {
			channels, err := s.repo.SystemChannels(true)
			if err != nil {
				return nil, err
			}
			out := []map[string]any{}
			for _, ch := range channels {
				if g.allows(ch.ID) {
					out = append(out, map[string]any{"id": ch.ID, "name": ch.Name, "enabled": ch.Enabled})
				}
			}
			return out, nil
		}
		if req.ChannelID == "" || !g.allows(req.ChannelID) {
			return nil, Forbidden("渠道未授权")
		}
		switch req.Action {
		case "tiers":
			models, err := s.AdminChannelModels(actor, req.ChannelID)
			if err != nil {
				return nil, err
			}
			out := []map[string]any{}
			access, err := s.userGroupAccessForUser(actor)
			if err != nil {
				return nil, err
			}
			for _, m := range models {
				pricing, err := access.pricing(m.PriceCurrency, m.ModelKey)
				if err != nil {
					return nil, err
				}
				for _, t := range m.PriceTiers {
					out = append(out, map[string]any{"modelId": m.ID, "modelKey": m.ModelKey, "modelVersion": m.PriceVersion, "tier": t, "cost": t.CostPricing, "currency": m.PriceCurrency, "rateMicrocredits": pricing.RateMicrocredits, "capability": m.Capability})
				}
			}
			return out, nil
		case "quote":
			return s.QuoteChannelModel(actor.ID, ChannelModelQuoteRequest{ChannelID: req.ChannelID, ModelKey: req.ModelKey, Intent: req.Intent})
		case "audit":
			events, _, err := s.repo.AdminAuditEvents("mcp.channel", req.ChannelID, 50, 0)
			return events, err
		case "preview", "apply":
			if req.Action == "apply" && !g.Write {
				return nil, Forbidden("凭证仅允许读取")
			}
			return s.automationReprice(actor, g.Name, req)
		default:
			return nil, BadAuthRequest("未知管理操作")
		}
	}
	return nil, Unauthorized("管理凭证无效")
}

func (s *Service) automationReprice(actor *model.User, agent string, req AdminAutomationRequest) (any, error) {
	if len(req.Changes) == 0 || len(req.Changes) > 100 || strings.TrimSpace(req.Reason) == "" || len(req.Reason) > 500 {
		return nil, BadAuthRequest("需要变更原因和 1 至 100 个价格档")
	}
	diff := []map[string]any{}
	err := s.repo.Transaction(func(tx *repository.Repository) error {
		if _, err := tx.AdminSystemChannel(req.ChannelID); err != nil {
			return err
		}
		seen := map[string]bool{}
		models, err := tx.ChannelModels(req.ChannelID, true)
		if err != nil {
			return err
		}
		var updates []model.ChannelModel
		for _, m := range models {
			changed := false
			for _, c := range req.Changes {
				if c.ModelID != m.ID {
					continue
				}
				for i := range m.PriceTiers {
					t := &m.PriceTiers[i]
					if t.ID != c.TierID {
						continue
					}
					if seen[t.ID] || !t.Enabled || m.PriceVersion != c.ModelVersion || t.PriceVersion != c.TierVersion {
						return BadAuthRequest("规格重复、停用或版本冲突，请重新读取")
					}
					before := *t
					t.CostPricing = c.Cost
					t.PriceConfigured = true
					if !c.Cost.Configured {
						return BadAuthRequest("必须明确配置成本")
					}
					if err := applyChannelModelTierPrices(t, m.Capability, m.Protocol, c.Prices); err != nil {
						return err
					}
					diff = append(diff, map[string]any{"before": before, "beforeCost": before.CostPricing, "after": *t, "afterCost": t.CostPricing})
					seen[t.ID] = true
					changed = true
				}
			}
			if changed {
				s.applyChannelModelPriceTierSummary(&m, m.PriceTiers)
				updates = append(updates, m)
			}
		}
		if len(seen) != len(req.Changes) {
			return BadAuthRequest("价格档不存在或不属于该渠道")
		}
		if req.Action == "preview" {
			return nil
		}
		if err := tx.UpdateChannelModelSalePrices(req.ChannelID, updates); err != nil {
			return err
		}
		for _, c := range req.Changes {
			if err := tx.SetAutomationTierCost(c.ModelID, c.TierID, c.Cost); err != nil {
				return err
			}
		}
		event, err := newAdminAuditEvent(actor, "mcp.reprice", "mcp.channel", req.ChannelID, req.Reason, map[string]any{"agent": agent, "changes": diff})
		if err != nil {
			return err
		}
		return tx.AppendAdminAudit(event)
	})
	if err != nil {
		return nil, err
	}
	if req.Action == "apply" {
		s.invalidateRouteCatalog()
	}
	return diff, nil
}
