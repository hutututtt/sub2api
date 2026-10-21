package admin

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Expose only the fields used by the existing capacity cell, never credentials.
type accountQualityCapacity struct {
	Platform                string   `json:"platform"`
	Type                    string   `json:"type"`
	Concurrency             int      `json:"concurrency"`
	CurrentConcurrency      *int     `json:"current_concurrency"`
	WindowCostLimit         *float64 `json:"window_cost_limit,omitempty"`
	WindowCostStickyReserve *float64 `json:"window_cost_sticky_reserve,omitempty"`
	CurrentWindowCost       *float64 `json:"current_window_cost"`
	MaxSessions             *int     `json:"max_sessions,omitempty"`
	SessionIdleTimeoutMin   *int     `json:"session_idle_timeout_minutes,omitempty"`
	ActiveSessions          *int     `json:"active_sessions"`
	BaseRPM                 *int     `json:"base_rpm,omitempty"`
	RPMStrategy             *string  `json:"rpm_strategy,omitempty"`
	RPMStickyBuffer         *int     `json:"rpm_sticky_buffer,omitempty"`
	CurrentRPM              *int     `json:"current_rpm"`
	QuotaLimit              *float64 `json:"quota_limit,omitempty"`
	QuotaUsed               *float64 `json:"quota_used,omitempty"`
	QuotaDailyLimit         *float64 `json:"quota_daily_limit,omitempty"`
	QuotaDailyUsed          *float64 `json:"quota_daily_used,omitempty"`
	QuotaWeeklyLimit        *float64 `json:"quota_weekly_limit,omitempty"`
	QuotaWeeklyUsed         *float64 `json:"quota_weekly_used,omitempty"`
}

type accountQualityPageItem struct {
	service.AccountQualityRow
	Capacity                *accountQualityCapacity `json:"capacity"`
	HomepageURL             string                  `json:"homepage_url,omitempty"`
	ErrorMessage            string                  `json:"error_message"`
	TempUnschedulableUntil  *time.Time              `json:"temp_unschedulable_until"`
	TempUnschedulableReason string                  `json:"temp_unschedulable_reason"`
	RateLimitResetAt        *time.Time              `json:"rate_limit_reset_at"`
	OverloadUntil           *time.Time              `json:"overload_until"`
}
type accountQualityCapacityPage struct {
	service.AccountQualityPage
	Items []accountQualityPageItem `json:"items"`
}

func qualityAccountHomepageURL(account *service.Account) string {
	if account == nil || account.Type != service.AccountTypeAPIKey {
		return ""
	}
	baseURL, ok := account.Credentials["base_url"].(string)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Hostname() == "" || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return ""
	}
	// Return only the upstream origin, without URL credentials, paths or query parameters.
	return (&url.URL{Scheme: strings.ToLower(parsed.Scheme), Host: parsed.Host}).String()
}

func qualityCapacityFromAccount(account *service.Account) *accountQualityCapacity {
	a := dto.AccountFromServiceShallow(account)
	if a == nil {
		return nil
	}
	return &accountQualityCapacity{
		Platform: a.Platform, Type: a.Type, Concurrency: a.Concurrency,
		WindowCostLimit: a.WindowCostLimit, WindowCostStickyReserve: a.WindowCostStickyReserve,
		MaxSessions: a.MaxSessions, SessionIdleTimeoutMin: a.SessionIdleTimeoutMin,
		BaseRPM: a.BaseRPM, RPMStrategy: a.RPMStrategy, RPMStickyBuffer: a.RPMStickyBuffer,
		QuotaLimit: a.QuotaLimit, QuotaUsed: a.QuotaUsed,
		QuotaDailyLimit: a.QuotaDailyLimit, QuotaDailyUsed: a.QuotaDailyUsed,
		QuotaWeeklyLimit: a.QuotaWeeklyLimit, QuotaWeeklyUsed: a.QuotaWeeklyUsed,
	}
}

func (h *AccountQualityHandler) pageWithCapacity(ctx context.Context, page service.AccountQualityPage) accountQualityCapacityPage {
	result := accountQualityCapacityPage{AccountQualityPage: page, Items: make([]accountQualityPageItem, len(page.Items))}
	ids := make([]int64, 0, len(page.Items))
	for i, row := range page.Items {
		result.Items[i].AccountQualityRow = row
		ids = append(ids, row.ID)
	}
	if len(ids) == 0 || h.accounts == nil || h.accounts.adminService == nil {
		return result
	}
	accounts, err := h.accounts.adminService.GetAccountsByIDs(ctx, ids)
	if err != nil {
		slog.Warn("quality capacity accounts unavailable", "error", err)
		return result
	}
	capacities := make(map[int64]*accountQualityCapacity, len(accounts))
	homepages := make(map[int64]string, len(accounts))
	accountsByID := make(map[int64]*service.Account, len(accounts))
	windowAccounts := make([]*service.Account, 0)
	sessionIDs := make([]int64, 0)
	rpmIDs := make([]int64, 0)
	idleTimeouts := make(map[int64]time.Duration)
	for _, a := range accounts {
		if a == nil {
			continue
		}
		capacities[a.ID] = qualityCapacityFromAccount(a)
		accountsByID[a.ID] = a
		homepages[a.ID] = qualityAccountHomepageURL(a)
		if !a.IsAnthropicOAuthOrSetupToken() {
			continue
		}
		if a.GetWindowCostLimit() > 0 {
			windowAccounts = append(windowAccounts, a)
		}
		if a.GetMaxSessions() > 0 {
			sessionIDs = append(sessionIDs, a.ID)
			idleTimeouts[a.ID] = time.Duration(a.GetSessionIdleTimeoutMinutes()) * time.Minute
		}
		if a.GetBaseRPM() > 0 {
			rpmIDs = append(rpmIDs, a.ID)
		}
	}
	if h.accounts.concurrencyService != nil {
		if counts, e := h.accounts.concurrencyService.GetAccountConcurrencyBatch(ctx, ids); e == nil {
			for id, c := range capacities {
				if n, ok := counts[id]; ok {
					value := n
					c.CurrentConcurrency = &value
				}
			}
		}
	}
	if len(sessionIDs) > 0 && h.accounts.sessionLimitCache != nil {
		if counts, e := h.accounts.sessionLimitCache.GetActiveSessionCountBatch(ctx, sessionIDs, idleTimeouts); e == nil {
			for id, n := range counts {
				if c := capacities[id]; c != nil {
					value := n
					c.ActiveSessions = &value
				}
			}
		}
	}
	if len(rpmIDs) > 0 && h.accounts.rpmCache != nil {
		if counts, e := h.accounts.rpmCache.GetRPMBatch(ctx, rpmIDs); e == nil {
			for id, n := range counts {
				if c := capacities[id]; c != nil {
					value := n
					c.CurrentRPM = &value
				}
			}
		}
	}
	if h.accounts.accountUsageService != nil {
		for _, a := range windowAccounts {
			if stats, e := h.accounts.accountUsageService.GetAccountWindowStats(ctx, a.ID, a.GetCurrentWindowStartTime()); e == nil && stats != nil {
				value := stats.StandardCost
				capacities[a.ID].CurrentWindowCost = &value
			}
		}
	}
	for i := range result.Items {
		result.Items[i].Capacity = capacities[result.Items[i].ID]
		result.Items[i].HomepageURL = homepages[result.Items[i].ID]
		if a := accountsByID[result.Items[i].ID]; a != nil {
			item := &result.Items[i]
			item.Status, item.Schedulable = a.Status, a.Schedulable
			item.ErrorMessage = a.ErrorMessage
			item.TempUnschedulableUntil = a.TempUnschedulableUntil
			item.TempUnschedulableReason = a.TempUnschedulableReason
			item.RateLimitResetAt = a.RateLimitResetAt
			item.OverloadUntil = a.OverloadUntil
		}
	}
	return result
}
