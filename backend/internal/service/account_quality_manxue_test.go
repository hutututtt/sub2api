package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type manxueMockTransport func(*http.Request) (*http.Response, error)

func (f manxueMockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func manxueMockResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func manxueFixtureRequest() manxueCandyRequest {
	return manxueCandyRequest{Benchmark: "candy", BaseURL: "https://upstream.example/v1", APIKey: "fixture-private-key", Model: "mapped-model", ReasoningEffort: "high", ServiceTier: "priority"}
}

func manxueFixtureAccount() *Account {
	return &Account{ID: 7, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{"base_url": "https://upstream.example/v1", "api_key": "fixture-private-key", "model_mapping": map[string]any{"public-model": "mapped-model", "mapped-model": "must-not-map-twice"}}}
}

func TestManxueUsesSavedAccountCredentialsAndModelMapping(t *testing.T) {
	account := manxueFixtureAccount()
	policy := DefaultAccountQualityPolicy()
	policy.Model = "public-model"
	posts := 0
	client := newManxueCandyClient(manxueMockTransport(func(req *http.Request) (*http.Response, error) {
		posts++
		var input manxueCandyRequest
		if req.Method != http.MethodPost || json.NewDecoder(req.Body).Decode(&input) != nil {
			t.Fatal("invalid direct inspection submission")
		}
		if input.BaseURL != "https://upstream.example/v1" || input.APIKey != "fixture-private-key" || input.Model != "mapped-model" {
			t.Fatal("saved account upstream, key or model mapping was not used")
		}
		return manxueMockResponse(202, `{"id":"mock_task","benchmark":"candy","status":"succeeded","candy":{"status":"passed","answer":"fixture-private-key"}}`), nil
	}))
	tester := &AccountTestService{accountRepo: &qualityAccountRepo{account: account}, qualityCandyClient: client}
	result := tester.RunCandyProbe(context.Background(), account.ID, policy, "fixture-idempotency")
	if posts != 1 || result.Verdict != "passed" || result.RequestsUsed != 1 {
		t.Fatalf("direct inspection failed: %+v", result)
	}
	var evidence map[string]any
	_ = json.Unmarshal(result.Evidence, &evidence)
	if evidence["request_model"] != "public-model" || evidence["upstream_model"] != "mapped-model" || evidence["transport"] != "account_upstream" {
		t.Fatal("model evidence is not tied to the account mapping")
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "fixture-private-key") || account.GetCredential("api_key") != "fixture-private-key" || account.GetCredential("base_url") != "https://upstream.example/v1" {
		t.Fatal("inspection leaked credentials or changed account configuration")
	}
}

func TestManxueAccountUpstreamURLCompatibility(t *testing.T) {
	policy := DefaultAccountQualityPolicy()
	policy.Model = "public-model"
	for raw, want := range map[string]string{
		"":                                  "https://api.openai.com/v1",
		"https://codekey.example/":          "https://codekey.example/v1",
		"https://upstream.example/v1":       "https://upstream.example/v1",
		"https://upstream.example/proxy/v4": "https://upstream.example/proxy/v4",
		"https://upstream.example/proxy/v1/responses/": "https://upstream.example/proxy/v1",
	} {
		a := manxueFixtureAccount()
		a.Credentials["base_url"] = raw
		input, detail := manxueCandyRequestForAccount(a, policy)
		if detail != "" || input.BaseURL != want {
			t.Fatalf("unexpected upstream URL for %s: %s (%s)", raw, input.BaseURL, detail)
		}
	}
	a := manxueFixtureAccount()
	a.Platform = PlatformDeepseek
	a.Credentials["base_url"] = "https://deepseek.example"
	input, detail := manxueCandyRequestForAccount(a, policy)
	if detail != "" || input.BaseURL != "https://deepseek.example" {
		t.Fatalf("native Responses path changed: %s (%s)", input.BaseURL, detail)
	}
}

func TestManxueRejectsOAuthAndInvalidUpstreamsBeforeSubmission(t *testing.T) {
	policy := DefaultAccountQualityPolicy()
	policy.Model = "public-model"
	for _, tc := range []struct {
		name, detail string
		modify       func(*Account, *AccountQualityPolicy)
	}{
		{"oauth", "manxue_unsupported_account", func(a *Account, _ *AccountQualityPolicy) {
			a.Type = AccountTypeOAuth
			a.Credentials["access_token"] = "fixture-oauth-token"
		}},
		{"missing key", "missing_api_key", func(a *Account, _ *AccountQualityPolicy) { delete(a.Credentials, "api_key") }},
		{"local URL", "manxue_public_https_required", func(a *Account, _ *AccountQualityPolicy) { a.Credentials["base_url"] = "https://127.0.0.1/v1" }},
		{"private URL", "manxue_public_https_required", func(a *Account, _ *AccountQualityPolicy) { a.Credentials["base_url"] = "https://10.0.0.1/v1" }},
		{"query credential", "manxue_public_https_required", func(a *Account, _ *AccountQualityPolicy) {
			a.Credentials["base_url"] = "https://upstream.example/v1?key=private"
		}},
		{"disabled", "account_not_schedulable", func(a *Account, _ *AccountQualityPolicy) { a.Schedulable = false }},
		{"protocol", "manxue_unsupported_parameters", func(_ *Account, p *AccountQualityPolicy) { p.Endpoint = "chat_completions" }},
		{"mapped media", "manxue_unsupported_parameters", func(a *Account, _ *AccountQualityPolicy) {
			a.Credentials["model_mapping"] = map[string]any{"public-model": "gpt-image-1"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, p := manxueFixtureAccount(), policy
			tc.modify(a, &p)
			client := newManxueCandyClient(manxueMockTransport(func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid account submitted to Manxue")
				return nil, nil
			}))
			tester := &AccountTestService{accountRepo: &qualityAccountRepo{account: a}, qualityCandyClient: client}
			result := tester.RunCandyProbe(context.Background(), a.ID, p, "fixture-idempotency")
			if result.Verdict != "error" || result.Detail != tc.detail || result.RequestsUsed != 0 {
				t.Fatalf("invalid account preflight: %+v", result)
			}
		})
	}
}

func TestManxueCandyContractAndRemoteVerdict(t *testing.T) {
	for _, tc := range []struct{ status, answer, verdict, detail string }{
		{"passed", "999", "passed", ""}, // No local answer/oracle may override Manxue.
		{"incorrect", "21", "incorrect", ""},
		{"error", "21", "error", "manxue_evaluation_error"},
		{"unexpected", "21", "error", "manxue_invalid_response"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			posts, polls := 0, 0
			c := newManxueCandyClient(manxueMockTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Scheme != "https" || r.URL.Host != "manxue.ai" {
					t.Fatal("credentials left fixed Manxue origin")
				}
				if r.Method == http.MethodPost {
					posts++
					var body map[string]any
					if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 6 || body["benchmark"] != "candy" || body["api_key"] != "fixture-private-key" || body["model"] != "mapped-model" || body["base_url"] != "https://upstream.example/v1" || body["reasoning_effort"] != "high" || body["service_tier"] != "priority" {
						t.Fatalf("invalid remote contract: %+v", body)
					}
					if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Idempotency-Key") != "fixture-key" {
						t.Fatal("missing content type or idempotency")
					}
					return manxueMockResponse(202, `{"id":"test_123","benchmark":"candy","status":"running"}`), nil
				}
				polls++
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tests/test_123" || r.Header.Get("Authorization") != "" {
					t.Fatal("invalid poll")
				}
				body, _ := json.Marshal(map[string]any{"id": "test_123", "benchmark": "candy", "status": "succeeded", "candy": map[string]any{"status": tc.status, "answer": tc.answer, "duration_ms": 1000}})
				return manxueMockResponse(200, string(body)), nil
			}))
			c.pollInterval = time.Millisecond
			r := c.run(context.Background(), manxueFixtureRequest(), "fixture-key")
			if r.Verdict != tc.verdict || r.Detail != tc.detail || r.Answer != tc.answer || r.RequestsUsed != 1 || posts != 1 || polls != 1 {
				t.Fatalf("unexpected result: %+v, posts=%d polls=%d", r, posts, polls)
			}
			if strings.Contains(string(r.Evidence), "fixture-private-key") || strings.Contains(string(r.Evidence), "test_123") || !strings.Contains(string(r.Evidence), `"provider":"manxue"`) {
				t.Fatal("unsafe or missing provider evidence")
			}
		})
	}
}

func TestManxueCandyRetryKeepsIdempotencyKey(t *testing.T) {
	posts := 0
	c := newManxueCandyClient(manxueMockTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.Header.Get("Idempotency-Key") != "stable-key" {
			t.Fatal("retry changed task key")
		}
		posts++
		if posts == 1 {
			response := manxueMockResponse(429, `{"error":"limited"}`)
			response.Header.Set("Retry-After", "1")
			return response, nil
		}
		return manxueMockResponse(202, `{"id":"test","benchmark":"candy","status":"succeeded","candy":{"status":"passed"}}`), nil
	}))
	start := time.Now()
	r := c.run(context.Background(), manxueFixtureRequest(), "stable-key")
	if posts != 2 || r.Verdict != "passed" || time.Since(start) < time.Second {
		t.Fatalf("retry contract failed: %+v", r)
	}
}

func TestManxueCandyCancellationDeletesRemoteTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	deletes := 0
	c := newManxueCandyClient(manxueMockTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodDelete {
			deletes++
			if r.Context().Err() != nil || r.URL.Path != "/api/v1/tests/private_task" {
				t.Fatal("remote cancellation inherited dead context")
			}
			return manxueMockResponse(200, `{}`), nil
		}
		cancel()
		return manxueMockResponse(202, `{"id":"private_task","benchmark":"candy","status":"running"}`), nil
	}))
	r := c.run(ctx, manxueFixtureRequest(), "cancel-key")
	if r.Detail != "cancelled" || deletes != 1 {
		t.Fatalf("remote job not cancelled: %+v, deletes=%d", r, deletes)
	}
}

func TestManxueCandyFailuresNeverPassOrLeakCredentials(t *testing.T) {
	for _, tc := range []struct {
		body, detail string
		code         int
	}{
		{`{"error":"fixture-private-key"}`, "manxue_http_400", 400},
		{`{"error":"fixture-private-key"}`, "manxue_http_404", 404},
		{`{"id":"t","benchmark":"candy","status":"succeeded"}`, "manxue_invalid_response", 200},
		{`{"id":"t","benchmark":"candy","status":"failed","error":"fixture-private-key"}`, "manxue_task_failed", 200},
		{`invalid fixture-private-key`, "manxue_invalid_response", 200},
		{`{"id":"t","benchmark":"candy","status":"succeeded","candy":{"status":"error","answer":"fixture-private-key"}}`, "manxue_evaluation_error", 200},
	} {
		c := newManxueCandyClient(manxueMockTransport(func(*http.Request) (*http.Response, error) { return manxueMockResponse(tc.code, tc.body), nil }))
		r := c.run(context.Background(), manxueFixtureRequest(), "failure-key")
		raw, _ := json.Marshal(r)
		if r.Verdict != "error" || r.Detail != tc.detail || strings.Contains(string(raw), "fixture-private-key") {
			t.Fatalf("unsafe result: %s", raw)
		}
	}
}

func TestManxueCandyDoesNotFollowRedirects(t *testing.T) {
	calls := 0
	c := newManxueCandyClient(manxueMockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "manxue.ai" {
			t.Fatal("credentials sent to redirect target")
		}
		response := manxueMockResponse(307, "")
		response.Header.Set("Location", "https://other.example/collect")
		return response, nil
	}))
	r := c.run(context.Background(), manxueFixtureRequest(), "redirect-key")
	if calls != 1 || r.Detail != "manxue_http_307" {
		t.Fatalf("redirect followed: %+v", r)
	}
}
