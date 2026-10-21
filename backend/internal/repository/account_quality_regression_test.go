package repository

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func TestAccountQualityPostgresRegressions(t *testing.T) {
	dsn := os.Getenv("PGQUALITY_TEST_DSN")
	if dsn == "" {
		t.Skip("set PGQUALITY_TEST_DSN for isolated local PostgreSQL")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("loopback required")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "quality_regression_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	db, err := sql.Open("postgres", dsn+" options='-c search_path="+schema+"'")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := db.ExecContext(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`CREATE TABLE groups(id BIGINT PRIMARY KEY,name TEXT,platform TEXT,deleted_at TIMESTAMPTZ);
 CREATE TABLE accounts(id BIGINT PRIMARY KEY,name TEXT,platform TEXT,type TEXT,status TEXT,schedulable BOOLEAN,priority INT,concurrency INT,deleted_at TIMESTAMPTZ,credentials JSONB,proxy_id BIGINT,parent_account_id BIGINT);
 CREATE TABLE account_groups(account_id BIGINT,group_id BIGINT);
 CREATE TABLE usage_logs(id BIGSERIAL PRIMARY KEY,account_id BIGINT,group_id BIGINT,request_id TEXT,model TEXT,requested_model TEXT,reasoning_effort TEXT,requested_reasoning_effort TEXT,service_tier TEXT,inbound_endpoint TEXT,first_token_ms INT,input_tokens INT,cache_creation_tokens INT,cache_read_tokens INT,created_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE ops_error_logs(id BIGSERIAL PRIMARY KEY,account_id BIGINT,group_id BIGINT,request_id TEXT,model TEXT,requested_model TEXT,request_path TEXT,error_owner TEXT,is_business_limited BOOLEAN DEFAULT false,created_at TIMESTAMPTZ DEFAULT NOW());`)
	for _, f := range []string{"246_account_quality.sql", "247_account_quality_evidence.sql", "248_account_quality_error_scope.sql", "249_account_quality_relay.sql"} {
		raw, e := os.ReadFile("../../migrations/" + f)
		if e != nil {
			t.Fatal(e)
		}
		exec(string(raw))
	}
	exec(`INSERT INTO groups VALUES (1,'A','openai',NULL),(2,'B','openai',NULL),(3,'not a member','openai',NULL);
 INSERT INTO accounts(id,name,platform,type,status,schedulable,priority,concurrency,credentials) VALUES (1,'fixture','openai','apikey','active',true,1,10,'{"base_url":"https://old.example/v1","model_mapping":{"public-model":"old-model"}}');
 INSERT INTO account_groups VALUES (1,1),(1,2);
 INSERT INTO account_quality_policies(group_id,config,next_run_at) VALUES (1,'{"enabled":true}',NOW()),(2,'{"enabled":true,"candy_consent_version":"manxue-v1"}',NOW());`)
	raw, e := os.ReadFile("../../migrations/250_account_quality_config_version.sql")
	if e != nil {
		t.Fatal(e)
	}
	exec(string(raw))
	repo := &accountQualityRepo{db: db}
	p := service.DefaultAccountQualityPolicy()
	p.Model = "public-model"
	rows := func(group int64, policy service.AccountQualityPolicy) []service.AccountQualityRow {
		t.Helper()
		rows, e := repo.Rows(ctx, group, policy)
		if e != nil {
			t.Fatal(e)
		}
		return rows
	}
	seed := func() {
		exec(`DELETE FROM account_quality_results;
 INSERT INTO account_quality_results(run_id,group_id,account_id,model,effort,benchmark,verdict,evidence,config_version)
 SELECT 1,1,1,'public-model','low','candy-manxue-v1','passed','{"upstream_model":"fixture"}'::jsonb,quality_config_version FROM accounts CROSS JOIN generate_series(1,3) WHERE accounts.id=1`)
	}
	exec(`INSERT INTO account_quality_runs(id,group_id,spec,status) VALUES(1,1,'{}','completed')`)
	t.Run("migration_requires_new_consent", func(t *testing.T) {
		legacy, e := repo.Policy(ctx, 1)
		if e != nil {
			t.Fatal(e)
		}
		current, e := repo.Policy(ctx, 2)
		if e != nil {
			t.Fatal(e)
		}
		if legacy.Enabled || !current.Enabled || !current.CandyAuthorized() {
			t.Fatal("consent migration did not distinguish old and authorized policies")
		}
		var next *time.Time
		if e = db.QueryRowContext(ctx, "SELECT next_run_at FROM account_quality_policies WHERE group_id=1").Scan(&next); e != nil || next != nil {
			t.Fatal("legacy schedule retained", e)
		}
	})
	t.Run("requested_effort_keeps_success_and_failure_together", func(t *testing.T) {
		exec(`INSERT INTO usage_logs(account_id,group_id,request_id,model,requested_model,reasoning_effort,requested_reasoning_effort,inbound_endpoint,first_token_ms,input_tokens,cache_creation_tokens,cache_read_tokens) VALUES(1,1,'mapped-success','public-model','public-model','xhigh','max','/v1/responses',1000,10,0,90)`)
		exec(`INSERT INTO ops_error_logs(account_id,group_id,request_id,model,requested_model,request_path,error_owner,quality_reasoning_effort,quality_scope_known) VALUES(1,1,'mapped-failure','public-model','public-model','/v1/responses','upstream','max',true)`)
		requested := p
		requested.Effort = "max"
		r := rows(1, requested)[0]
		if r.Requests != 1 || r.Failures != 1 {
			t.Fatalf("requested scope: successes=%d failures=%d", r.Requests, r.Failures)
		}
		effective := p
		effective.Effort = "xhigh"
		if r = rows(1, effective)[0]; r.Requests != 0 || r.Failures != 0 {
			t.Fatal("effective effort inherited requested samples")
		}
	})
	t.Run("upstream_change_invalidates_evidence_cache_and_weights", func(t *testing.T) {
		seed()
		before := rows(1, p)
		service.ScoreAccountQualityWithPolicy(before, p)
		if before[0].Score == nil || *before[0].Score != 100 {
			t.Fatal("fixture must first score 100")
		}
		version, e := repo.ConfigVersion(ctx, 1)
		if e != nil {
			t.Fatal(e)
		}
		if e = repo.SaveWeights(ctx, 1, p.ScopeKey(), map[int64]service.AccountQualityStoredWeight{1: {Multiplier: 2, UpdatedAt: time.Now(), ConfigVersion: version}}); e != nil {
			t.Fatal(e)
		}
		exec(`UPDATE accounts SET credentials='{"base_url":"https://new.example/v1","model_mapping":{"public-model":"new-model"}}' WHERE id=1`)
		after := rows(1, p)
		service.ScoreAccountQualityWithPolicy(after, p)
		if after[0].ConfigVersion != version+1 || after[0].QualitySamples != 0 || after[0].Score != nil || after[0].CandyStatus != "untested" {
			t.Fatalf("old evidence leaked: %+v", after[0])
		}
		if recent, e := repo.RecentResult(ctx, 1, p, "candy-manxue-v1"); e != nil || recent != nil {
			t.Fatal("reused old upstream result", e)
		}
		if weights, e := repo.LoadWeights(ctx, 1, p.ScopeKey()); e != nil || len(weights) != 0 {
			t.Fatal("inherited old weights", e)
		}
		requested := p
		requested.Effort = "max"
		if r := rows(1, requested)[0]; r.Requests != 0 || r.Failures != 0 {
			t.Fatal("old performance affected new configuration")
		}
		// An old in-flight probe finishing after the edit must remain ineligible.
		exec(`INSERT INTO account_quality_results(run_id,group_id,account_id,model,effort,benchmark,verdict,config_version) VALUES(1,1,1,'public-model','low','candy-manxue-v1','passed',$1)`, version)
		if rows(1, p)[0].QualitySamples != 0 {
			t.Fatal("in-flight old probe certified new configuration")
		}
		exec(`UPDATE accounts SET name='renamed',schedulable=false WHERE id=1`)
		if v, e := repo.ConfigVersion(ctx, 1); e != nil || v != version+1 {
			t.Fatal("display/scheduling edit invalidated evidence", e)
		}
		exec(`UPDATE accounts SET schedulable=true WHERE id=1`)
	})
	t.Run("credential_changes_invalidate_linked_accounts", func(t *testing.T) {
		exec(`INSERT INTO accounts(id,name,platform,type,status,schedulable,priority,concurrency,credentials,parent_account_id) VALUES(2,'shadow','openai','oauth','active',true,1,1,'{}',1)`)
		before, err := repo.ConfigVersion(ctx, 2)
		if err != nil {
			t.Fatal(err)
		}
		exec(`UPDATE accounts SET credentials=credentials||'{"api_key":"replacement-fixture-key"}'::jsonb WHERE id=1`)
		after, err := repo.ConfigVersion(ctx, 2)
		if err != nil || after != before+1 {
			t.Fatal("linked account kept old configuration version", err)
		}
	})
	t.Run("cross_group_status_opens_actual_evidence", func(t *testing.T) {
		seed()
		r := rows(2, p)[0]
		if r.CandyStatus != "passed" || len(r.EvidenceIDs) == 0 {
			t.Fatal("fixture missing shared evidence")
		}
		results, e := repo.Results(ctx, 2, 1, r.EvidenceIDs...)
		if e != nil || len(results) != 3 {
			t.Fatalf("shared evidence unavailable: %d %v", len(results), e)
		}
		for _, x := range results {
			if x.GroupID != 1 {
				t.Fatal("source group lost")
			}
		}
		if result, e := repo.Results(ctx, 3, 1, r.EvidenceIDs...); e != nil || len(result) != 0 {
			t.Fatal("evidence crossed unrelated group", e)
		}
	})
	t.Run("expired_result_is_stale_and_retains_timestamp", func(t *testing.T) {
		seed()
		exec(`UPDATE account_quality_results SET created_at=NOW()-interval '2 hours'`)
		r := rows(1, p)[0]
		if r.CandyStatus != "stale" || r.QualitySamples != 0 || r.LastTestAt == nil || len(r.EvidenceIDs) == 0 {
			t.Fatalf("expiry state incorrect: %+v", r)
		}
	})
	t.Run("single_failure_survives_technical_errors_but_not_pass_or_expiry", func(t *testing.T) {
		seed()
		insert := func(verdict string) {
			exec(`INSERT INTO account_quality_results(run_id,group_id,account_id,model,effort,benchmark,verdict,config_version)
 SELECT 1,1,1,'public-model','low','candy-manxue-v1',$1,quality_config_version FROM accounts WHERE id=1`, verdict)
		}
		insert("incorrect")
		if !rows(1, p)[0].CapabilityFailed {
			t.Fatal("single confirmed failure was ignored")
		}
		insert("error")
		if r := rows(1, p)[0]; r.CandyStatus != "error" || !r.CapabilityFailed {
			t.Fatal("technical failure hid recent conclusive failure")
		}
		insert("passed")
		if rows(1, p)[0].CapabilityFailed {
			t.Fatal("new passing verdict did not supersede failure")
		}
		insert("incorrect")
		exec(`UPDATE account_quality_results SET created_at=NOW()-interval '2 hours'`)
		if rows(1, p)[0].CapabilityFailed {
			t.Fatal("expired capability failure remained active")
		}
	})
}
