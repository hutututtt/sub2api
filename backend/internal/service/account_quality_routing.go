package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"

	"math"
	rand "math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AccountQualityRequestScope struct {
	Model       string `json:"model"`
	Effort      string `json:"effort"`
	ServiceTier string `json:"service_tier"`
	Endpoint    string `json:"endpoint"`
}
type accountQualityScopeKey struct{}
type accountQualityOriginalScopeKey struct{}

// Copy only non-secret quality conditions into the detached usage worker.
func CopyAccountQualityRequestScope(dst, src context.Context) context.Context {
	if src != nil {
		if scope, ok := src.Value(accountQualityScopeKey{}).(AccountQualityRequestScope); ok {
			return WithAccountQualityRequestScope(dst, scope)
		}
	}
	return dst
}

// Capture the original request before any group policy rewrites. Later attempts
// retain this identity so success, failure and routing use the same scope.
func WithAccountQualityOriginalBodyScope(ctx context.Context, model string, body []byte, path string) context.Context {
	ctx = WithAccountQualityBodyScope(ctx, model, body, path)
	return context.WithValue(ctx, accountQualityOriginalScopeKey{}, true)
}

func WithAccountQualityRequestScope(ctx context.Context, s AccountQualityRequestScope) context.Context {
	return context.WithValue(ctx, accountQualityScopeKey{}, s)
}
func WithAccountQualityBodyScope(ctx context.Context, model string, body []byte, path string) context.Context {
	if captured, _ := ctx.Value(accountQualityOriginalScopeKey{}).(bool); captured {
		return ctx
	}
	view := openAIRequestPayloadView(body)
	effort := ""
	for _, field := range []string{"reasoning.effort", "reasoning_effort", "output_config.effort", "generationConfig.thinkingConfig.thinkingLevel"} {
		if v := view.Get(field); v.Exists() {
			effort = strings.ToLower(strings.TrimSpace(v.String()))
			break
		}
	}
	if effort == "" {
		if budget := view.Get("thinking.budget_tokens"); budget.Exists() {
			effort = "budget:" + strconv.FormatInt(budget.Int(), 10)
		}
	}
	tier := view.Get("service_tier").String()
	if tier == "fast" {
		tier = "priority"
	}
	if tier == "default" {
		tier = ""
	}
	endpoint := ""
	switch {
	case strings.HasSuffix(path, "/responses"):
		endpoint = "responses"
	case strings.HasSuffix(path, "/chat/completions"):
		endpoint = "chat_completions"
	case strings.HasSuffix(path, "/messages"):
		endpoint = "messages"
	case strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent"):
		endpoint = "generate_content"
	}
	return WithAccountQualityRequestScope(ctx, AccountQualityRequestScope{Model: model, Effort: effort, ServiceTier: tier, Endpoint: endpoint})
}

type accountQualityRoutingEntry struct {
	ConfigVersion int64    `json:"-"`
	Multiplier    float64  `json:"multiplier"`
	ConfirmedBad  bool     `json:"confirmed_bad"`
	Score         *float64 `json:"score"`
}

const accountQualityMinMultiplier = .1

func accountQualityConfirmedBad(r AccountQualityRow, now time.Time) bool {
	if r.CapabilityFailed {
		return true
	}
	if r.LastTestAt == nil || r.LastTestAt.After(now) || now.Sub(*r.LastTestAt) >= time.Hour {
		return false
	}
	return r.CandyStatus == "incorrect" || r.CandyStatus == "inconsistent" ||
		(r.QualitySamples >= 3 && float64(r.QualityPassed)/float64(r.QualitySamples) <= .25)
}

type accountQualityRoutingGroup struct {
	Policy  AccountQualityPolicy
	Weights map[int64]accountQualityRoutingEntry
}
type accountQualityRoutingSnapshot struct {
	LoadedAt time.Time
	Groups   map[int64]accountQualityRoutingGroup
}

func (s *AccountQualityService) refreshRouting(ctx context.Context) error {
	s.routingMu.Lock()
	defer s.routingMu.Unlock()
	generation := s.routingGeneration.Load()
	groups, err := s.repo.Groups(ctx)
	if err != nil {
		return err
	}
	next := &accountQualityRoutingSnapshot{LoadedAt: time.Now(), Groups: map[int64]accountQualityRoutingGroup{}}
	for _, g := range groups {
		p := g.Policy
		if p.Mode != "auto" || p.Model == "" || g.ID <= 0 {
			continue
		}
		rows, err := s.repo.Rows(ctx, g.ID, p)
		if err != nil {
			return err
		}
		s.applyReliabilityCoverage(ctx, rows)
		ScoreAccountQualityWithPolicy(rows, p)
		old, err := s.repo.LoadWeights(ctx, g.ID, p.ScopeKey())
		if err != nil {
			return err
		}
		weights := map[int64]AccountQualityStoredWeight{}
		entries := map[int64]accountQualityRoutingEntry{}
		now := time.Now()
		for _, r := range rows {
			if r.Status != "active" || !r.Schedulable {
				continue
			}
			target := 1.0
			if r.Score != nil {
				target = .25 + 1.75*math.Pow(*r.Score/100, 2)
			}
			bad := accountQualityConfirmedBad(r, now)
			if bad {
				target = accountQualityMinMultiplier
			}
			prior := old[r.ID]
			if prior.Multiplier <= 0 || prior.ConfigVersion != r.ConfigVersion {
				prior.Multiplier = 1
				prior.UpdatedAt = now.Add(-time.Minute)
			}
			elapsed := math.Max(0, now.Sub(prior.UpdatedAt).Seconds())
			step := .15 * math.Min(1, elapsed/60)
			m := math.Max(accountQualityMinMultiplier, math.Min(2, prior.Multiplier+math.Max(-step, math.Min(step, target-prior.Multiplier))))
			if bad {
				m = accountQualityMinMultiplier
			}
			weights[r.ID] = AccountQualityStoredWeight{Multiplier: m, UpdatedAt: now, ConfigVersion: r.ConfigVersion}
			entries[r.ID] = accountQualityRoutingEntry{Multiplier: m, ConfirmedBad: bad, Score: r.Score, ConfigVersion: r.ConfigVersion}
		}
		if err = s.repo.SaveWeights(ctx, g.ID, p.ScopeKey(), weights); err != nil {
			return err
		}
		next.Groups[g.ID] = accountQualityRoutingGroup{Policy: p, Weights: entries}
	}
	if s.routingGeneration.Load() == generation {
		s.routing.Store(next)
	}
	return nil
}
func (s *AccountQualityService) invalidateRoutingGroup(id int64) {
	if s == nil {
		return
	}
	s.routingGeneration.Add(1)
	old := s.routing.Load()
	if old == nil {
		return
	}
	next := &accountQualityRoutingSnapshot{LoadedAt: old.LoadedAt, Groups: map[int64]accountQualityRoutingGroup{}}
	for g, v := range old.Groups {
		if g != id {
			next.Groups[g] = v
		}
	}
	s.routing.Store(next)
}
func (s *AccountQualityService) matchRouting(ctx context.Context, g *int64, model string) (accountQualityRoutingGroup, bool) {
	if s == nil || ctx == nil || g == nil || *g <= 0 {
		return accountQualityRoutingGroup{}, false
	}
	state := s.routing.Load()
	if state == nil || time.Since(state.LoadedAt) > 2*time.Minute {
		return accountQualityRoutingGroup{}, false
	}
	group, ok := state.Groups[*g]
	if !ok {
		return group, false
	}
	scope, ok := ctx.Value(accountQualityScopeKey{}).(AccountQualityRequestScope)
	if !ok || scope.Endpoint == "" || scope.Model == "" || model == "" {
		return group, false
	}
	p := group.Policy
	effort := scope.Effort
	if effort == "" {
		effort = "unspecified"
	}
	return group, p.Mode == "auto" && p.Model == scope.Model && p.Effort == effort && p.ServiceTier == scope.ServiceTier && p.Endpoint == scope.Endpoint
}
func (s *AccountQualityService) effectiveMultiplier(_ context.Context, g int64, p AccountQualityPolicy, id, version int64) (float64, bool) {
	ctx := WithAccountQualityRequestScope(context.Background(), AccountQualityRequestScope{Model: p.Model, Effort: p.Effort, ServiceTier: p.ServiceTier, Endpoint: p.Endpoint})
	group, ok := s.matchRouting(ctx, &g, p.Model)
	if !ok {
		return 0, false
	}
	w, ok := group.Weights[id]
	return w.Multiplier, ok && version > 0 && w.ConfigVersion == version
}
func (s *AccountQualityService) shouldEscapeSticky(ctx context.Context, g *int64, model string, id int64) bool {
	group, ok := s.matchRouting(ctx, g, model)
	w := group.Weights[id]
	if !ok || !w.ConfirmedBad || w.ConfigVersion <= 0 {
		return false
	}
	version, err := s.repo.ConfigVersion(ctx, id)
	return err == nil && version == w.ConfigVersion
}

// Weighted permutations retain at least 5% uniform exploration mass.
func qualityWeightedPermutation(weights []float64, random func() float64) []int {
	pool := make([]int, len(weights))
	for i := range pool {
		pool[i] = i
	}
	out := make([]int, 0, len(pool))
	for len(pool) > 0 {
		total := 0.0
		for _, i := range pool {
			total += math.Max(.0001, weights[i])
		}
		target := random()
		acc := 0.0
		chosen := len(pool) - 1
		for j, i := range pool {
			acc += .95*math.Max(.0001, weights[i])/total + .05/float64(len(pool))
			if target < acc {
				chosen = j
				break
			}
		}
		out = append(out, pool[chosen])
		pool = append(pool[:chosen], pool[chosen+1:]...)
	}
	return out
}
func (s *AccountQualityService) orderAccounts(ctx context.Context, g *int64, model string, accounts []*Account) []*Account {
	group, ok := s.matchRouting(ctx, g, model)
	if !ok || len(accounts) < 2 {
		return accounts
	}
	out := append([]*Account(nil), accounts...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority < out[j].Priority })
	for begin := 0; begin < len(out); {
		end := begin + 1
		for end < len(out) && out[end].Priority == out[begin].Priority {
			end++
		}
		chunk := append([]*Account(nil), out[begin:end]...)
		weights := make([]float64, len(chunk))
		for i, a := range chunk {
			weights[i] = qualityAccountMultiplier(group.Weights[a.ID], a)
			if weights[i] <= 0 {
				weights[i] = 1
			}
		}
		order := qualityWeightedPermutation(weights, rand.Float64)
		for i, j := range order {
			out[begin+i] = chunk[j]
		}
		begin = end
	}
	return out
}
func (s *AccountQualityService) firstEligible(ctx context.Context, g *int64, model string, accounts []*Account) *Account {
	if _, ok := s.matchRouting(ctx, g, model); !ok {
		return nil
	}
	ordered := s.orderAccounts(ctx, g, model, accounts)
	if len(ordered) == 0 {
		return nil
	}
	return ordered[0]
}
func (s *GatewayService) SetAccountQualityService(q *AccountQualityService) { s.accountQuality = q }
func (s *OpenAIGatewayService) SetAccountQualityService(q *AccountQualityService) {
	s.accountQuality = q
}

type AccountQualityRoutingEvent struct {
	GroupID    int64                      `json:"group_id"`
	AccountID  int64                      `json:"account_id"`
	RequestID  string                     `json:"request_id"`
	Scope      AccountQualityRequestScope `json:"scope"`
	Multiplier float64                    `json:"multiplier"`
	Reason     string                     `json:"reason"`
	CreatedAt  time.Time                  `json:"created_at"`
}

func (s *AccountQualityService) recordRouting(ctx context.Context, g *int64, model string, account *Account, reason string) {
	group, ok := s.matchRouting(ctx, g, model)
	if !ok {
		return
	}
	scope, _ := ctx.Value(accountQualityScopeKey{}).(AccountQualityRequestScope)
	rid, _ := ctx.Value(ctxkey.ClientRequestID).(string)
	m := qualityAccountMultiplier(group.Weights[account.ID], account)
	if m <= 0 {
		m = 1
	}
	e := AccountQualityRoutingEvent{GroupID: *g, AccountID: account.ID, RequestID: rid, Scope: scope, Multiplier: m, Reason: reason, CreatedAt: time.Now()}
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	if len(s.routeEvents) >= 512 {
		copy(s.routeEvents, s.routeEvents[1:])
		s.routeEvents = s.routeEvents[:511]
	}
	s.routeEvents = append(s.routeEvents, e)
}
func (s *AccountQualityService) RoutingEvents(g int64, requestID string) []AccountQualityRoutingEvent {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	out := []AccountQualityRoutingEvent{}
	for i := len(s.routeEvents) - 1; i >= 0 && len(out) < 100; i-- {
		e := s.routeEvents[i]
		if (g == 0 || e.GroupID == g) && (requestID == "" || requestID == e.RequestID) {
			out = append(out, e)
		}
	}
	return out
}
func (s *AccountQualityService) RoutingPreview(ctx context.Context, g int64, scope AccountQualityRequestScope) (map[string]any, error) {
	p, err := s.repo.Policy(ctx, g)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"configured_mode": p.Mode, "policy_model": p.Model, "policy_effort": p.Effort, "policy_service_tier": p.ServiceTier, "policy_endpoint": p.Endpoint, "applied": false, "reason": "not_automatic", "weights": map[int64]accountQualityRoutingEntry{}}
	if p.Mode != "auto" {
		return out, nil
	}
	state := s.routing.Load()
	if state == nil || time.Since(state.LoadedAt) > 2*time.Minute {
		out["reason"] = "expired_snapshot"
		return out, nil
	}
	ctx = WithAccountQualityRequestScope(ctx, scope)
	group, ok := s.matchRouting(ctx, &g, scope.Model)
	if !ok {
		out["reason"] = "scope_mismatch"
		return out, nil
	}
	out["applied"] = true
	out["reason"] = "ready"
	rows, err := s.repo.Rows(ctx, g, p)
	if err != nil {
		return nil, err
	}
	weights := make(map[int64]accountQualityRoutingEntry)
	for _, row := range rows {
		if w, exists := group.Weights[row.ID]; exists {
			if row.ConfigVersion <= 0 || row.ConfigVersion != w.ConfigVersion {
				w = accountQualityRoutingEntry{Multiplier: 1, ConfigVersion: row.ConfigVersion}
			}
			weights[row.ID] = w
		}
	}
	out["weights"] = weights
	out["snapshot_at"] = state.LoadedAt
	out["exploration_fraction"] = .05
	return out, nil
}

func (s *AccountQualityService) pickAccountLoad(ctx context.Context, g *int64, model string, accounts []accountWithLoad) *accountWithLoad {
	group, ok := s.matchRouting(ctx, g, model)
	if !ok || len(accounts) == 0 {
		return nil
	}
	weights := make([]float64, len(accounts))
	for i, a := range accounts {
		m := qualityAccountMultiplier(group.Weights[a.account.ID], a.account)
		if m <= 0 {
			m = 1
		}
		load := 0.0
		queue := 0
		if a.loadInfo != nil {
			load = math.Max(0, float64(a.loadInfo.LoadRate))
			queue = max(0, a.loadInfo.WaitingCount)
		}
		weights[i] = m / (1 + load/25 + float64(queue))
	}
	indices := qualityWeightedPermutation(weights, rand.Float64)
	return &accounts[indices[0]]
}
func (s *AccountQualityService) orderAccountLoads(ctx context.Context, g *int64, model string, accounts []accountWithLoad) []accountWithLoad {
	if _, ok := s.matchRouting(ctx, g, model); !ok {
		return accounts
	}
	pool := append([]accountWithLoad(nil), accounts...)
	out := make([]accountWithLoad, 0, len(pool))
	for len(pool) > 0 {
		tier := filterByMinPriority(pool)
		picked := s.pickAccountLoad(ctx, g, model, tier)
		if picked == nil {
			return accounts
		}
		out = append(out, *picked)
		next := pool[:0]
		for _, a := range pool {
			if a.account.ID != picked.account.ID {
				next = append(next, a)
			}
		}
		pool = next
	}
	return out
}

func AttachAccountQualityErrorScope(ctx context.Context, entry *OpsInsertErrorLogInput) {
	if ctx == nil || entry == nil {
		return
	}
	scope, ok := ctx.Value(accountQualityScopeKey{}).(AccountQualityRequestScope)
	if !ok || scope.Endpoint == "" {
		return
	}
	entry.QualityReasoningEffort = scope.Effort
	if entry.QualityReasoningEffort == "" {
		entry.QualityReasoningEffort = "unspecified"
	}
	entry.QualityServiceTier = scope.ServiceTier
	entry.QualityScopeKnown = true
}

func qualityAccountMultiplier(w accountQualityRoutingEntry, a *Account) float64 {
	if a == nil || a.QualityConfigVersion <= 0 || w.ConfigVersion != a.QualityConfigVersion || w.Multiplier <= 0 {
		return 1
	}
	return w.Multiplier
}
