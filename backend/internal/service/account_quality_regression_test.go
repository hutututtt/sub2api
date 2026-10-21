package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAccountQualityRequestedEffortSurvivesNormalization(t *testing.T) {
	body := []byte(`{"model":"gpt-5","reasoning":{"effort":"max"}}`)
	ctx := WithAccountQualityBodyScope(context.Background(), "gpt-5", body, "/v1/responses")
	scope := ctx.Value(accountQualityScopeKey{}).(AccountQualityRequestScope)
	requested := CanonicalRequestedReasoningEffort(body, "gpt-5")
	effective := extractOpenAIReasoningEffortFromBody(body, "gpt-5")
	if requested == nil || effective == nil {
		t.Fatal("missing effort")
	}
	if scope.Effort != "max" || *requested != "max" || *effective != "xhigh" {
		t.Fatalf("unexpected fixture: %+v %s %s", scope, *requested, *effective)
	}
	stored := requestedQualityReasoningEffort(ctx, requested, effective)
	if stored == nil || *stored != scope.Effort {
		t.Fatal("success and failure scopes diverged")
	}
}

func TestAccountQualityOriginalScopeSurvivesPolicyRewrite(t *testing.T) {
	for _, effort := range []string{"max", ""} {
		body := []byte(`{"model":"gpt-5","reasoning":{"effort":"` + effort + `"}}`)
		ctx := WithAccountQualityOriginalBodyScope(context.Background(), "gpt-5", body, "/v1/responses")
		ctx = WithAccountQualityBodyScope(ctx, "gpt-5", []byte(`{"reasoning":{"effort":"low"}}`), "/v1/responses")
		entry := &OpsInsertErrorLogInput{}
		AttachAccountQualityErrorScope(ctx, entry)
		forwarded := "low"
		stored := requestedQualityReasoningEffort(ctx, &forwarded, &forwarded)
		if effort == "" {
			if stored != nil || entry.QualityReasoningEffort != "unspecified" {
				t.Fatal("missing client effort became a model default")
			}
		} else if stored == nil || *stored != effort || entry.QualityReasoningEffort != effort {
			t.Fatal("group rewrite replaced original request scope")
		}
	}
}

func TestAccountQualityPolicyRequiresCurrentConsent(t *testing.T) {
	p := DefaultAccountQualityPolicy()
	p.Enabled = true
	p.Model = "fixture-model"
	for _, version := range []string{"", "local-v1", "manxue-v0"} {
		p.CandyConsentVersion = version
		if p.Validate() == nil {
			t.Fatal("unapproved external schedule accepted")
		}
	}
	p.CandyConsentVersion = AccountQualityCandyConsentVersion
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Enabled = false
	p.CandyConsentVersion = ""
	if err := p.Validate(); err != nil {
		t.Fatal("disabling must not require consent", err)
	}
}

type qualityVersionRepo struct {
	qualityCaptureRepo
	version int64
}

func (r *qualityVersionRepo) ConfigVersion(context.Context, int64) (int64, error) {
	return r.version, nil
}

type qualityChangingProbe struct {
	qualityFakeProbe
	repo *qualityVersionRepo
}

func (p *qualityChangingProbe) RunCandyProbe(ctx context.Context, id int64, policy AccountQualityPolicy, key string) AccountQualityResult {
	p.repo.version++
	return p.qualityFakeProbe.RunCandyProbe(ctx, id, policy, key)
}

func TestAccountQualityProbeKeepsStartingVersionAcrossEdit(t *testing.T) {
	repo := &qualityVersionRepo{version: 1}
	svc := NewAccountQualityService(repo, nil)
	svc.tester = &qualityChangingProbe{repo: repo}
	svc.probe(context.Background(), &AccountQualityRun{ID: 1, GroupID: 1}, DefaultAccountQualityPolicy(), 7)
	if len(repo.results) != 1 || repo.results[0].ConfigVersion != 1 || repo.version != 2 {
		t.Fatal("in-flight evidence acquired the new configuration version")
	}
}

func TestAccountQualityRoutingRejectsChangedConfiguration(t *testing.T) {
	s, ctx, g := qualityRoutingFixture()
	s.repo = &qualityVersionRepo{version: 2}
	if s.shouldEscapeSticky(ctx, &g, "fixture-model", 2) {
		t.Fatal("old failure escaped sticky for new upstream")
	}
	w := accountQualityRoutingEntry{Multiplier: 2, ConfigVersion: 1}
	if qualityAccountMultiplier(w, &Account{QualityConfigVersion: 2}) != 1 || qualityAccountMultiplier(w, &Account{}) != 1 {
		t.Fatal("old or missing version inherited weights")
	}
	if qualityAccountMultiplier(w, &Account{QualityConfigVersion: 1}) != 2 {
		t.Fatal("matching version lost weights")
	}
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{accountQuality: s}}
	plan := openAIAccountLoadPlan{candidates: []openAIAccountCandidateScore{{account: &Account{ID: 1, QualityConfigVersion: 2}, score: 1}, {account: &Account{ID: 2, QualityConfigVersion: 2}, score: 1}}}
	scheduler.applyAccountQuality(ctx, OpenAIAccountScheduleRequest{GroupID: &g, RequestedModel: "fixture-model"}, &plan)
	if plan.candidates[0].qualityWeight != plan.candidates[1].qualityWeight {
		t.Fatal("modern scheduler retained old quality bias")
	}
}

func TestAccountQualityChangedConfigurationResetsWeightRamp(t *testing.T) {
	p := DefaultAccountQualityPolicy()
	p.Model = "fixture-model"
	p.Mode = "auto"
	repo := &qualityWeightRepo{groups: []AccountQualityGroup{{ID: 1, Policy: p}}, rows: []AccountQualityRow{{ID: 1, ConfigVersion: 2, Status: "active", Schedulable: true}}, weights: map[int64]AccountQualityStoredWeight{1: {Multiplier: 2, ConfigVersion: 1, UpdatedAt: time.Now().Add(-time.Minute)}}}
	s := NewAccountQualityService(repo, nil)
	if err := s.refreshRouting(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w := repo.weights[1]; w.Multiplier != 1 || w.ConfigVersion != 2 {
		t.Fatalf("old ramp survived upstream edit: %+v", w)
	}
}

func TestAccountQualityLegacyLocalInspectionDoesNotStartRemoteCandyWithoutNewConsent(t *testing.T) {
	p := DefaultAccountQualityPolicy()
	// Configuration written by the committed local-only version, with no third-party opt-in field.
	if err := json.Unmarshal([]byte(`{"enabled":true,"model":"fixture-model","effort":"low","endpoint":"responses","mode":"observe"}`), &p); err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(-time.Minute)
	repo := &qualityScheduledRepo{policy: p, groups: []AccountQualityGroup{{ID: 1, Policy: p, NextRunAt: &due}}, finished: make(chan string, 1)}
	probe := &qualityFakeProbe{answer: "fixture"}
	svc := NewAccountQualityService(repo, nil)
	svc.tester = probe
	defer svc.Stop()
	svc.tick(context.Background())
	select {
	case <-repo.finished:
		if len(probe.ids) > 0 {
			t.Errorf("legacy local policy automatically invoked the new remote candy runner %d time(s)", len(probe.ids))
		}
	case <-time.After(time.Second):
	}
}
