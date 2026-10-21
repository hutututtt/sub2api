package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/qualitytrace"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Opt-in verification uses a private schema in the local DB; it never reads or changes public account data.
func TestAccountQualityPostgresContracts(t *testing.T) {
	dsn := os.Getenv("PGQUALITY_TEST_DSN")
	if dsn == "" {
		t.Skip("set PGQUALITY_TEST_DSN to run isolated local PostgreSQL verification")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("verification is restricted to loopback PostgreSQL")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal("cannot initialize local verification DB")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	schema := "quality_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	exec := func(query string, args ...any) {
		t.Helper()
		if _, e := db.ExecContext(ctx, query, args...); e != nil {
			t.Fatalf("fixture SQL failed: %v", e)
		}
	}
	exec("CREATE SCHEMA " + schema)
	adminDB := db
	defer adminDB.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	db, err = sql.Open("postgres", dsn+" options='-c search_path="+schema+"'")
	if err != nil {
		t.Fatal("cannot initialize schema-bound verification pool")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec(`CREATE TABLE groups(id BIGINT PRIMARY KEY,name TEXT,platform TEXT,deleted_at TIMESTAMPTZ);
 CREATE TABLE accounts(id BIGINT PRIMARY KEY,name TEXT,platform TEXT,type TEXT,status TEXT,schedulable BOOLEAN,priority INT,concurrency INT,deleted_at TIMESTAMPTZ,credentials JSONB NOT NULL DEFAULT '{}',proxy_id BIGINT,parent_account_id BIGINT);
 CREATE TABLE account_groups(account_id BIGINT,group_id BIGINT);
 CREATE TABLE usage_logs(id BIGSERIAL PRIMARY KEY,account_id BIGINT,group_id BIGINT,request_id TEXT,model TEXT,requested_model TEXT,reasoning_effort TEXT,requested_reasoning_effort TEXT,service_tier TEXT,inbound_endpoint TEXT,first_token_ms INT,input_tokens INT,cache_creation_tokens INT,cache_read_tokens INT,created_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE ops_error_logs(id BIGSERIAL PRIMARY KEY,account_id BIGINT,group_id BIGINT,request_id TEXT,model TEXT,requested_model TEXT,request_path TEXT,error_owner TEXT,is_business_limited BOOLEAN DEFAULT false,created_at TIMESTAMPTZ DEFAULT NOW());`)
	for _, name := range []string{"246_account_quality.sql", "247_account_quality_evidence.sql", "248_account_quality_error_scope.sql", "249_account_quality_relay.sql", "250_account_quality_config_version.sql"} {
		raw, e := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if e != nil {
			t.Fatal(e)
		}
		exec(string(raw))
	}
	exec("INSERT INTO groups VALUES(1,'fixture A','openai',NULL),(2,'fixture B','openai',NULL)")
	exec("INSERT INTO accounts(id,name,platform,type,status,schedulable,priority,concurrency,deleted_at) VALUES(1,'fixture account','openai','apikey','active',true,1,2,NULL),(2,'second fixture','openai','apikey','active',true,1,2,NULL)")
	exec("INSERT INTO account_groups VALUES(1,1),(2,1),(1,2)")
	for i := 0; i < 35; i++ {
		age, input, creation, read := 2, 5, 5, 90
		if i < 5 {
			age, input, creation, read = 4, 1000, 1000, 0
		}
		exec("INSERT INTO usage_logs(account_id,group_id,request_id,model,requested_model,reasoning_effort,requested_reasoning_effort,inbound_endpoint,first_token_ms,input_tokens,cache_creation_tokens,cache_read_tokens,created_at) VALUES(1,1,$1,'fixture-model','fixture-model','low','low','/v1/responses',$2,$3,$4,$5,NOW()-$6*interval '1 hour')", fmt.Sprint("success-", i), 1000+i, input, creation, read, age)
	}
	exec(`INSERT INTO usage_logs(account_id,group_id,request_id,model,requested_model,requested_reasoning_effort,inbound_endpoint,first_token_ms,input_tokens,cache_creation_tokens,cache_read_tokens,created_at) VALUES
 (1,1,'outside-day','fixture-model','fixture-model','low','/v1/responses',2000,10000,0,0,NOW()-interval '25 hours'),
 (2,1,'older-cache','fixture-model','fixture-model','low','/v1/responses',2000,10,0,90,NOW()-interval '4 hours')`)
	exec(`INSERT INTO ops_error_logs(account_id,group_id,request_id,model,request_path,error_owner,is_business_limited,quality_reasoning_effort,quality_scope_known) VALUES
 (1,1,'failed','fixture-model','/v1/responses','upstream',false,'low',true),
 (1,1,'limited','fixture-model','/v1/responses','client',true,NULL,false),
 (1,1,'success-0','fixture-model','/v1/responses','upstream',false,'low',true)`)

	repo := &accountQualityRepo{db: db}
	p := service.DefaultAccountQualityPolicy()
	p.Model = "fixture-model"
	if err = repo.SavePolicy(ctx, 1, p); err != nil {
		t.Fatal(err)
	}
	groups, err := repo.Groups(ctx)
	if err != nil || len(groups) != 2 || groups[0].Total != 2 || groups[1].Total != 1 {
		t.Fatalf("group membership mismatch: %+v / %v", groups, err)
	}
	rows, err := repo.Rows(ctx, 1, p)
	if err != nil {
		t.Fatal(err)
	}
	var a service.AccountQualityRow
	for _, r := range rows {
		if r.ID == 1 {
			a = r
		}
	}
	expectedCache := 100.0 * 2700 / 13000
	if a.Requests != 35 || a.TTFTSamples != 35 || a.Failures != 1 || a.CacheSamples != 35 || a.CacheRate == nil || math.Abs(*a.CacheRate-expectedCache) > .000001 {
		t.Fatalf("bad business aggregation: %+v", a)
	}
	for _, r := range rows {
		if r.ID == 2 && (r.Requests != 1 || r.TTFTSamples != 1 || r.CacheSamples != 1 || r.CacheRate == nil || *r.CacheRate != 90) {
			t.Fatalf("four-hour-old traffic must contribute to 24-hour cache metrics: %+v", r)
		}
	}
	t.Run("CacheWithoutInputUsageStaysUnknown", func(t *testing.T) {
		exec("UPDATE usage_logs SET input_tokens=0,cache_read_tokens=0 WHERE request_id='older-cache'")
		defer exec("UPDATE usage_logs SET input_tokens=10,cache_read_tokens=90 WHERE request_id='older-cache'")
		exec(`INSERT INTO usage_logs(account_id,group_id,request_id,model,requested_model,requested_reasoning_effort,inbound_endpoint,input_tokens,cache_creation_tokens,cache_read_tokens) VALUES(2,1,'zero-input','fixture-model','fixture-model','low','/v1/responses',0,0,0)`)
		defer exec("DELETE FROM usage_logs WHERE request_id='zero-input'")
		zeroRows, err := repo.Rows(ctx, 1, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range zeroRows {
			if r.ID == 2 && (r.CacheSamples != 2 || r.CacheRate != nil) {
				t.Fatalf("zero input usage must not report a zero cache rate: %+v", r)
			}
		}
	})
	other := p
	other.Endpoint = "chat_completions"
	scoped, err := repo.Rows(ctx, 1, other)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range scoped {
		if r.Requests != 0 || r.Failures != 0 {
			t.Fatalf("protocol scopes mixed: %+v", r)
		}
	}
	t.Run("BusinessStatisticsIndependentOfInspection", func(t *testing.T) {
		exec(`INSERT INTO usage_logs(account_id,group_id,request_id,model,requested_model,requested_reasoning_effort,service_tier,inbound_endpoint,first_token_ms,input_tokens,cache_creation_tokens,cache_read_tokens,created_at) VALUES
		 (1,1,'business-other','other-model','other-model','xhigh','priority','/v1/chat/completions',2500,9000,0,1000,NOW()-interval '4 hours'),
		 (1,1,'business-effort','fixture-model','fixture-model','high','flex','/v1/responses',0,0,0,100,NOW()-interval '23 hours'),
		 (1,1,'business-legacy','other-model','other-model',NULL,NULL,NULL,1500,0,0,900,NOW()-interval '2 hours'),
		 (1,2,'business-other-group','other-model','other-model','xhigh','priority','/v1/responses',1500,100000,0,0,NOW()),
		 (1,1,'business-outside','other-model','other-model','xhigh','priority','/v1/responses',1500,100000,0,0,NOW()-interval '25 hours')`)
		defer exec("DELETE FROM usage_logs WHERE request_id LIKE 'business-%'")
		exec(`INSERT INTO ops_error_logs(account_id,group_id,request_id,model,request_path,error_owner,quality_scope_known) VALUES
		 (1,1,'business-error','other-model','','upstream',false),
		 (1,1,'business-error','other-model','','upstream',false),
		 (1,1,'business-unknown',NULL,'','upstream',false),
		 (1,1,'business-other','other-model','','upstream',false),
		 (1,1,'business-client','other-model','','client',false)`)
		defer exec("DELETE FROM ops_error_logs WHERE request_id LIKE 'business-%'")
		businessRows, e := repo.Rows(ctx, 1, p, service.AccountQualityObservationScope{})
		if e != nil {
			t.Fatal(e)
		}
		business := businessRows[0]
		if business.Requests != 38 || business.TTFTSamples != 37 || business.CacheSamples != 38 || business.Failures != 3 || business.UnknownFailureScope != 0 || business.CacheRate == nil || math.Abs(*business.CacheRate-100.0*4700/24000) > .000001 {
			t.Fatalf("business statistics dropped model/effort/tier/protocol samples or averaged percentages: %+v", business)
		}
		filtered, e := repo.Rows(ctx, 1, p, service.AccountQualityObservationScope{Model: "other-model"})
		if e != nil {
			t.Fatal(e)
		}
		if got := filtered[0]; got.Requests != 2 || got.CacheSamples != 2 || got.Failures != 1 || got.UnknownFailureScope != 1 || got.CacheRate == nil || math.Abs(*got.CacheRate-100.0*1900/10900) > .000001 {
			t.Fatalf("independent model filter or unknown error coverage incorrect: %+v", got)
		}
		exec("UPDATE accounts SET quality_config_changed_at=NOW()-interval '1 hour' WHERE id=1")
		defer exec("UPDATE accounts SET quality_config_changed_at='1970-01-01' WHERE id=1")
		businessRows, e = repo.Rows(ctx, 1, p, service.AccountQualityObservationScope{})
		if e != nil || businessRows[0].Requests != 38 {
			t.Fatalf("business history disappeared after configuration change: %+v / %v", businessRows, e)
		}
		routingRows, e := repo.Rows(ctx, 1, p)
		if e != nil || routingRows[0].Requests != 0 {
			t.Fatalf("routing must still reject performance from the previous configuration: %+v / %v", routingRows, e)
		}
	})
	run, err := repo.StartRun(ctx, 1, p, 3, "modeltrace", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Heartbeat(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	evidence := json.RawMessage(`{"bank_revision":"fixture-hash","expected_model":"fixture-model","attribution":{"prediction":"fixture-model","probability":0.9}}`)
	result := &service.AccountQualityResult{ConfigVersion: 1, RunID: run.ID, GroupID: 1, AccountID: 1, Model: p.Model, Effort: p.Effort, Endpoint: p.Endpoint, Benchmark: "modeltrace-v1", Verdict: "consistent", Evidence: evidence, RequestsUsed: 3}
	if err = repo.AddResult(ctx, result); err != nil {
		t.Fatal(err)
	}
	history, err := repo.Results(ctx, 1, 1)
	if err != nil || len(history) != 1 || history[0].RequestsUsed != 3 || !json.Valid(history[0].Evidence) {
		t.Fatalf("evidence round-trip failed: %+v / %v", history, err)
	}
	runs, err := repo.Runs(ctx, 1)
	if err != nil || runs[0].Completed != 3 || runs[0].Benchmark != "modeltrace" {
		t.Fatalf("request progress is not measured in calls: %+v / %v", runs, err)
	}
	if err = repo.FinishRun(ctx, run.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	second, err := repo.StartRun(ctx, 1, p, 3, "candy", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.CancelRun(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	result.RunID = second.ID
	if err = repo.AddResult(ctx, result); err == nil {
		t.Fatal("cancelled job accepted a result")
	}
	if err = repo.FinishRun(ctx, second.ID, "cancelled"); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err = repo.SaveWeights(ctx, 1, p.ScopeKey(), map[int64]service.AccountQualityStoredWeight{1: {Multiplier: .7, UpdatedAt: now, ConfigVersion: 1}}); err != nil {
		t.Fatal(err)
	}
	weights, err := repo.LoadWeights(ctx, 1, p.ScopeKey())
	if err != nil || weights[1].Multiplier != .7 {
		t.Fatal("routing state not persisted", err)
	}
	weights, err = repo.LoadWeights(ctx, 2, p.ScopeKey())
	if err != nil || len(weights) != 0 {
		t.Fatal("weights leaked across groups", err)
	}
	t.Run("RealRunnerAgainstLoopbackMock", func(t *testing.T) {
		var calls atomic.Int64
		var block atomic.Bool
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if block.Load() {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(3 * time.Second):
				}
				return
			}
			raw, _ := io.ReadAll(io.LimitReader(r.Body, 200000))
			prompt := string(raw)
			answer := "<answer>21</answer>"
			if strings.Contains(prompt, "1 到 355") {
				answer = strings.Repeat("42,", 330)
			}
			if strings.Contains(prompt, "只查询 record_") {
				match := regexp.MustCompile(`record_[a-f0-9]{16} = ([0-9]{9});`).FindStringSubmatch(prompt)
				if match == nil {
					t.Error("synthetic context target not found")
				} else {
					answer = "<answer>" + match[1] + "</answer>"
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\ndata: {\"type\":\"response.completed\"}\n\n", answer)
		}))
		defer upstream.Close()
		accountRepo := &qualityLoopbackAccountRepo{account: &service.Account{ID: 1, Name: "local fixture", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: "active", Schedulable: true, Concurrency: 2, Credentials: map[string]any{"api_key": "fixture-only", "base_url": upstream.URL}}}
		cfg := &config.Config{}
		cfg.Security.URLAllowlist.AllowInsecureHTTP = true
		tester := service.NewAccountTestService(accountRepo, nil, nil, nil, nil, &qualityLoopbackTransport{}, cfg, nil)
		runner := service.NewAccountQualityService(repo, tester)
		runner.Start()
		defer runner.Stop()
		p.DailyBudget = 30
		p.ReferenceModel = qualitytrace.Models()[0]
		if err := runner.SavePolicy(ctx, 1, p); err != nil {
			t.Fatal(err)
		}
		wait := func(id int64) service.AccountQualityRun {
			t.Helper()
			deadline := time.Now().Add(6 * time.Second)
			for time.Now().Before(deadline) {
				runs, err := repo.Runs(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range runs {
					if r.ID == id && r.Status != "queued" && r.Status != "running" {
						return r
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			t.Fatal("local inspection did not reach a terminal state")
			return service.AccountQualityRun{}
		}
		preflight, err := runner.Run(ctx, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if finished := wait(preflight.ID); finished.Completed != 0 {
			t.Fatal("invalid upstream consumed quota")
		}
		if calls.Load() != 0 {
			t.Fatal("private upstream submitted to Manxue")
		}
		preflightResults, err := repo.Results(ctx, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		preflightFound := false
		for _, result := range preflightResults {
			if result.RunID == preflight.ID {
				preflightFound = result.Detail == "manxue_public_https_required" && result.RequestsUsed == 0 && result.Verdict == "error"
			}
		}
		if !preflightFound {
			t.Fatal("upstream preflight error not persisted")
		}
		fixtureRun, err := repo.StartRun(ctx, 1, p, 5, "candy", false)
		if err != nil {
			t.Fatal(err)
		}
		for _, benchmark := range []string{"candy-manxue-v1", "candy-manxue-v1", "candy-manxue-v1", "candy-v1", "candy-variant-v1"} {
			verdict := "passed"
			if benchmark != "candy-manxue-v1" {
				verdict = "incorrect"
			}
			x := &service.AccountQualityResult{ConfigVersion: 1, RunID: fixtureRun.ID, GroupID: 1, AccountID: 1, Model: p.Model, Effort: p.Effort, Endpoint: p.Endpoint, Benchmark: benchmark, Verdict: verdict, Evidence: json.RawMessage(`{"provider":"manxue"}`), RequestsUsed: 1}
			if err := repo.AddResult(ctx, x); err != nil {
				t.Fatal(err)
			}
		}
		if err := repo.FinishRun(ctx, fixtureRun.ID, "completed"); err != nil {
			t.Fatal(err)
		}
		recent, err := repo.RecentResult(ctx, 1, p, "candy-manxue-v1")
		if err != nil || recent == nil || !strings.Contains(string(recent.Evidence), "manxue") {
			t.Fatal("remote evidence lost on reuse", err)
		}
		page, err := runner.Page(ctx, 1, 1, 8, "", "", "", "score")
		if err != nil {
			t.Fatal(err)
		}
		if page.Items[0].CandyStatus != "passed" || page.Items[0].QualitySamples != 3 || page.Items[0].QualityPassed != 3 {
			t.Fatalf("legacy local candy affected remote scoring: %+v", page.Items)
		}
		trace, err := runner.RunBenchmark(ctx, 1, 1, "modeltrace")
		if err != nil {
			t.Fatal(err)
		}
		traceDone := wait(trace.ID)
		if traceDone.Completed != 3 || calls.Load() != 3 {
			t.Fatalf("three trace requests not counted: %+v calls=%d", traceDone, calls.Load())
		}
		history, err := repo.Results(ctx, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range history {
			if r.RunID == trace.ID && r.Benchmark == "modeltrace-v1" {
				found = true
				if r.Verdict == "error" || len(r.Evidence) < 100 || r.RequestsUsed != 3 {
					t.Fatalf("trace evidence missing: %+v", r)
				}
			}
		}
		if !found {
			t.Fatal("fingerprint result missing")
		}
		contextRun, err := runner.RunBenchmark(ctx, 1, 1, "context")
		if err != nil {
			t.Fatal(err)
		}
		if wait(contextRun.ID).Completed != 3 || calls.Load() != 6 {
			t.Fatal("short/medium/long probes were not executed")
		}
		verifiedPage, verifyErr := runner.Page(ctx, 1, 1, 8, "", "", "", "score")
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
		if verifiedPage.Items[0].CandyStatus != "passed" {
			t.Fatal("successful context checks changed remote candy verdict")
		}
		block.Store(true)
		cancelRun, err := runner.RunBenchmark(ctx, 1, 1, "modeltrace")
		if err != nil {
			t.Fatal(err)
		}
		if err = runner.Cancel(ctx, cancelRun.ID); err != nil {
			t.Fatal(err)
		}
		if wait(cancelRun.ID).Status != "cancelled" {
			t.Fatal("cancellation did not stop local job")
		}
		p.Mode = "auto"
		if err = runner.SavePolicy(ctx, 1, p); err != nil {
			t.Fatal(err)
		}
		preview, err := runner.RoutingPreview(ctx, 1, service.AccountQualityRequestScope{Model: p.Model, Effort: p.Effort, Endpoint: p.Endpoint})
		if err != nil || preview["applied"] != true {
			t.Fatalf("automatic routing snapshot not available: %v / %v", preview, err)
		}
		p.Mode = "manual"
		if err = runner.SavePolicy(ctx, 1, p); err != nil {
			t.Fatal(err)
		}
		preview, err = runner.RoutingPreview(ctx, 1, service.AccountQualityRequestScope{Model: p.Model, Effort: p.Effort, Endpoint: p.Endpoint})
		if err != nil || preview["applied"] != false {
			t.Fatal("manual rollback left quality routing enabled", err)
		}
	})

}

type qualityLoopbackAccountRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *qualityLoopbackAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if id != r.account.ID {
		return nil, fmt.Errorf("unexpected fixture account")
	}
	return r.account, nil
}

type qualityLoopbackTransport struct{ service.HTTPUpstream }

func (t *qualityLoopbackTransport) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if !strings.HasPrefix(req.URL.Host, "127.0.0.1:") {
		return nil, fmt.Errorf("fixture refuses non-loopback upstream")
	}
	return http.DefaultClient.Do(req)
}
