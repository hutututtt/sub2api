package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type qualityRetryProbe struct {
	qualityFakeProbe
	results []AccountQualityResult
	keys    map[string]bool
	after   func()
}

func (p *qualityRetryProbe) RunCandyProbe(_ context.Context, id int64, _ AccountQualityPolicy, key string) AccountQualityResult {
	index := len(p.ids)
	p.ids = append(p.ids, id)
	if p.keys == nil {
		p.keys = map[string]bool{}
	}
	p.keys[key] = true
	if p.after != nil {
		p.after()
	}
	return p.results[min(index, len(p.results)-1)]
}

func TestAccountQualityCandyRetriesErrorsAndStopsAtConclusiveVerdict(t *testing.T) {
	failure := AccountQualityResult{Verdict: "error", Detail: "manxue_evaluation_error", RequestsUsed: 1}
	passed := AccountQualityResult{Verdict: "passed", RequestsUsed: 1}
	incorrect := AccountQualityResult{Verdict: "incorrect", RequestsUsed: 1}
	for _, tc := range []struct {
		name    string
		results []AccountQualityResult
		calls   int
		verdict string
	}{
		{"three retries then pass", []AccountQualityResult{failure, failure, failure, passed}, 4, "passed"},
		{"retry limit", []AccountQualityResult{failure}, 4, "error"},
		{"failure then incorrect", []AccountQualityResult{failure, incorrect}, 2, "incorrect"},
		{"first pass", []AccountQualityResult{passed}, 1, "passed"},
		{"first incorrect", []AccountQualityResult{incorrect}, 1, "incorrect"},
		{"preflight error", []AccountQualityResult{{Verdict: "error", Detail: "missing_api_key"}}, 1, "error"},
		{"remote cancellation", []AccountQualityResult{{Verdict: "error", Detail: "cancelled", RequestsUsed: 1}}, 1, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &qualityCaptureRepo{}
			probe := &qualityRetryProbe{results: tc.results}
			svc := NewAccountQualityService(repo, nil)
			svc.tester = probe
			svc.probeCandy(context.Background(), &AccountQualityRun{ID: 5, GroupID: 3}, DefaultAccountQualityPolicy(), 7, 1)
			if len(probe.ids) != tc.calls || len(repo.results) != tc.calls || len(probe.keys) != tc.calls {
				t.Fatalf("incorrect number of independent attempts: requests=%d results=%d keys=%d", len(probe.ids), len(repo.results), len(probe.keys))
			}
			if repo.results[tc.calls-1].Verdict != tc.verdict {
				t.Fatal("valid verdict lost")
			}
			for _, result := range repo.results {
				if result.AccountID != 7 || result.GroupID != 3 || result.ConfigVersion != 1 {
					t.Fatal("retry escaped its original account or scope")
				}
			}
		})
	}
}

type qualityResultFailureRepo struct{ qualityCaptureRepo }

func (*qualityResultFailureRepo) AddResult(context.Context, *AccountQualityResult) error {
	return errors.New("fixture persistence failure")
}

func TestAccountQualityCandyRetriesStopOnCancellationEditOrStorageFailure(t *testing.T) {
	for _, reason := range []string{"cancelled", "edited", "storage"} {
		t.Run(reason, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repo := &qualityVersionRepo{version: 1}
			probe := &qualityRetryProbe{results: []AccountQualityResult{{Verdict: "error", Detail: "manxue_transport_error", RequestsUsed: 1}}}
			svc := NewAccountQualityService(repo, nil)
			svc.tester = probe
			switch reason {
			case "cancelled":
				probe.after = cancel
			case "edited":
				probe.after = func() { repo.version++ }
			case "storage":
				svc.repo = &qualityResultFailureRepo{}
			}
			svc.probeCandy(ctx, &AccountQualityRun{ID: 5, GroupID: 3}, DefaultAccountQualityPolicy(), 7, 1)
			if len(probe.ids) != 1 {
				t.Fatal("unsafe retry after " + reason)
			}
		})
	}
}

type qualityImmediateRepo struct{ qualityWeightRepo }

func (*qualityImmediateRepo) ConfigVersion(context.Context, int64) (int64, error) { return 1, nil }
func (*qualityImmediateRepo) Canceled(context.Context, int64) bool                { return false }
func (*qualityImmediateRepo) RecentResult(context.Context, int64, AccountQualityPolicy, string) (*AccountQualityResult, error) {
	return nil, nil
}
func (r *qualityImmediateRepo) AddResult(_ context.Context, result *AccountQualityResult) error {
	now := time.Now()
	r.rows[0].LastTestAt = &now
	r.rows[0].CandyStatus = result.Verdict
	r.rows[0].QualitySamples++
	r.rows[0].CapabilityFailed = result.Verdict == "incorrect"
	return nil
}

func TestAccountQualitySingleIncorrectImmediatelyPublishesMinimumWeight(t *testing.T) {
	p := DefaultAccountQualityPolicy()
	p.Model, p.Mode = "fixture-model", "auto"
	// A confirmed failure overrides the scoring sample threshold and component weights.
	p.ScoreWeights = AccountQualityScoreWeights{Cache: 100}
	repo := &qualityImmediateRepo{qualityWeightRepo{
		groups:  []AccountQualityGroup{{ID: 3, Policy: p}},
		rows:    []AccountQualityRow{{ID: 7, ConfigVersion: 1, Status: "active", Schedulable: true}},
		weights: map[int64]AccountQualityStoredWeight{7: {Multiplier: 2, ConfigVersion: 1, UpdatedAt: time.Now()}},
	}}
	svc := NewAccountQualityService(repo, nil)
	svc.tester = &qualityFakeProbe{answer: "incorrect answer"}
	svc.probe(context.Background(), &AccountQualityRun{ID: 5, GroupID: 3}, p, 7)
	snapshot := svc.routing.Load()
	if snapshot == nil || snapshot.Groups[3].Weights[7].Multiplier != .1 || !snapshot.Groups[3].Weights[7].ConfirmedBad || repo.weights[7].Multiplier != .1 {
		t.Fatal("one failed probe did not immediately publish and persist minimum weight")
	}
}

func TestAccountQualityConfirmedFailureIgnoresErrorsAndStaleEvidence(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	future := now.Add(time.Minute)
	for _, tc := range []struct {
		row AccountQualityRow
		bad bool
	}{
		{AccountQualityRow{CandyStatus: "incorrect", LastTestAt: &now, QualitySamples: 1}, true},
		{AccountQualityRow{CandyStatus: "inconsistent", LastTestAt: &now}, true},
		{AccountQualityRow{CandyStatus: "error", LastTestAt: &now}, false},
		{AccountQualityRow{CandyStatus: "error", LastTestAt: &now, CapabilityFailed: true}, true},
		{AccountQualityRow{CandyStatus: "incorrect", LastTestAt: &old}, false},
		{AccountQualityRow{CandyStatus: "incorrect", LastTestAt: &future}, false},
	} {
		if got := accountQualityConfirmedBad(tc.row, now); got != tc.bad {
			t.Fatalf("unexpected capability decision for %+v: %v", tc.row, got)
		}
	}
}
