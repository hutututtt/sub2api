package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const accountQualityCandyBenchmark = "candy-manxue-v1"

// Only this fixed origin receives the account's upstream credential. Redirects are disabled.
var defaultManxueCandyClient = newManxueCandyClient(http.DefaultTransport)

type manxueCandyRequest struct {
	Benchmark       string `json:"benchmark"`
	BaseURL         string `json:"base_url"`
	APIKey          string `json:"api_key"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
	ServiceTier     string `json:"service_tier,omitempty"`
}

type manxueCandyTask struct {
	ID        string `json:"id"`
	Benchmark string `json:"benchmark"`
	Status    string `json:"status"`
	Candy     *struct {
		Status     string `json:"status"`
		Answer     string `json:"answer"`
		DurationMs int64  `json:"duration_ms"`
	} `json:"candy"`
}

type manxueCandyClient struct {
	http         *http.Client
	pollInterval time.Duration
}

func newManxueCandyClient(transport http.RoundTripper) *manxueCandyClient {
	return &manxueCandyClient{
		http:         &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		pollInterval: 3 * time.Second,
	}
}

func validateManxueCandyPolicy(p AccountQualityPolicy) error {
	if p.Endpoint != "responses" {
		return fmt.Errorf("满血糖果 API 仅支持 Responses 协议 / Manxue candy requires Responses")
	}
	switch p.Effort {
	case "low", "medium", "high", "xhigh", "max", "ultra":
	default:
		return fmt.Errorf("满血糖果 API 推理程度须为 low/medium/high/xhigh/max/ultra")
	}
	if p.ServiceTier != "" && p.ServiceTier != "priority" && p.ServiceTier != "ultrafast" {
		return fmt.Errorf("满血糖果 API 服务档位须为默认、priority 或 ultrafast")
	}
	return nil
}

func qualityPublicHTTPSURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("public HTTPS URL required")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	ip := net.ParseIP(host)
	if (ip == nil && !strings.Contains(host, ".")) || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || (ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())) {
		return nil, fmt.Errorf("public HTTPS URL required")
	}
	return u, nil
}
func qualityCandyAccountSupported(platform, kind string) bool {
	return kind == AccountTypeAPIKey && (platform == PlatformOpenAI || platform == PlatformKimi || platform == PlatformZhipu || platform == PlatformDeepseek || platform == PlatformOpenCodeGo)
}

func manxueCandyRequestForAccount(a *Account, p AccountQualityPolicy) (manxueCandyRequest, string) {
	if a == nil {
		return manxueCandyRequest{}, "account_unavailable"
	}
	if !qualityCandyAccountSupported(a.Platform, a.Type) || a.IsSyntheticUITest() {
		return manxueCandyRequest{}, "manxue_unsupported_account"
	}
	if !a.IsSchedulable() {
		return manxueCandyRequest{}, "account_not_schedulable"
	}
	model := a.GetMappedModel(p.Model)
	if validateManxueCandyPolicy(p) != nil || strings.TrimSpace(model) == "" || qualityMediaModel(p.Model) || qualityMediaModel(model) {
		return manxueCandyRequest{}, "manxue_unsupported_parameters"
	}
	apiKey := strings.TrimSpace(a.GetOpenAIProtocolAPIKey())
	if apiKey == "" {
		return manxueCandyRequest{}, "missing_api_key"
	}
	base := a.GetOpenAIBaseURL()
	if a.UsesNativeCNResponses() && a.IsAdaptiveAPIProtocol() {
		base = a.GetCNProtocolBaseURL(APIProtocolResponses)
	}
	u, err := qualityPublicHTTPSURL(base)
	if err != nil {
		return manxueCandyRequest{}, "manxue_public_https_required"
	}
	// Use the same Responses URL rules as the account adapter, then give Manxue
	// its base URL: Manxue appends /responses itself. No local relay is involved.
	endpoint := buildOpenAIResponsesURLForPlatform(a.Platform, strings.TrimRight(u.String(), "/"))
	base = strings.TrimSuffix(strings.TrimRight(endpoint, "/"), "/responses")
	return manxueCandyRequest{Benchmark: "candy", BaseURL: base, APIKey: apiKey, Model: model, ReasoningEffort: p.Effort, ServiceTier: p.ServiceTier}, ""
}

func (s *AccountTestService) RunCandyProbe(ctx context.Context, id int64, p AccountQualityPolicy, key string) AccountQualityResult {
	a, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return AccountQualityResult{Verdict: "error", Detail: "account_unavailable"}
	}
	input, detail := manxueCandyRequestForAccount(a, p)
	if detail != "" {
		return AccountQualityResult{Verdict: "error", Detail: detail}
	}
	client := s.qualityCandyClient
	if client == nil {
		client = defaultManxueCandyClient
	}
	result := client.run(ctx, input, key)
	var evidence map[string]any
	_ = json.Unmarshal(result.Evidence, &evidence)
	if evidence == nil {
		evidence = map[string]any{}
	}
	evidence["request_model"] = p.Model
	evidence["upstream_model"] = input.Model
	evidence["transport"] = "account_upstream"
	result.Evidence, _ = json.Marshal(evidence)
	return result
}

var manxueTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)

func (c *manxueCandyClient) run(parent context.Context, input manxueCandyRequest, key string) (result AccountQualityResult) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute+30*time.Second)
	defer cancel()
	result.Verdict = "error"
	evidence := map[string]any{"provider": "manxue", "request_model": input.Model}
	defer func() {
		result.DurationMs = time.Since(start).Milliseconds()
		result.Evidence, _ = json.Marshal(evidence)
		result.Evidence = qualityResponseEvidence(result.Evidence, result.Answer)
	}()
	if ctx.Err() != nil {
		result.Detail = "cancelled"
		return
	}
	body, _ := json.Marshal(input)
	result.RequestsUsed = 1 // Budget one remote test; GET/DELETE do not use model quota.
	var task manxueCandyTask
	result.Detail = c.request(ctx, http.MethodPost, "", body, key, &task)
	if result.Detail != "" {
		return
	}
	if !manxueTaskIDPattern.MatchString(task.ID) || task.Benchmark != "candy" {
		result.Detail = "manxue_invalid_response"
		return
	}
	id := task.ID
	terminal := false
	defer func() {
		if !terminal {
			// Cancellation also reaches the remote worker after local cancellation,
			// timeouts or a lost poll, so it does not keep consuming upstream quota.
			cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			_ = c.request(cleanup, http.MethodDelete, id, nil, "", nil)
		}
	}()
	for {
		evidence["remote_status"] = task.Status
		switch task.Status {
		case "succeeded":
			terminal = true
			if task.Candy == nil {
				result.Detail = "manxue_invalid_response"
				return
			}
			evidence["candy_status"] = task.Candy.Status
			evidence["remote_duration_ms"] = task.Candy.DurationMs
			// Remote completion is not a pass. Only the remote candy verdict counts.
			switch task.Candy.Status {
			case "passed", "incorrect":
				result.Verdict = task.Candy.Status
			case "error":
				result.Detail = "manxue_evaluation_error"
			default:
				result.Detail = "manxue_invalid_response"
			}
			result.Answer = task.Candy.Answer
			if input.APIKey != "" {
				result.Answer = strings.ReplaceAll(result.Answer, input.APIKey, "[redacted]")
			}
			return
		case "failed", "cancelled":
			terminal = true
			result.Detail = "manxue_task_failed"
			if task.Status == "cancelled" {
				result.Detail = "cancelled"
			}
			return
		case "queued", "running":
		default:
			result.Detail = "manxue_invalid_response"
			return
		}
		if !manxueWait(ctx, c.pollInterval) {
			result.Detail = manxueContextFailure(ctx)
			return
		}
		var next manxueCandyTask
		result.Detail = c.request(ctx, http.MethodGet, id, nil, "", &next)
		if result.Detail != "" {
			return
		}
		if next.ID != id || next.Benchmark != "candy" {
			result.Detail = "manxue_invalid_response"
			return
		}
		task = next
	}
}

func manxueContextFailure(ctx context.Context) string {
	if ctx.Err() == context.Canceled {
		return "cancelled"
	}
	return "timeout"
}

func manxueWait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c *manxueCandyClient) request(ctx context.Context, method, id string, body []byte, key string, output *manxueCandyTask) string {
	endpoint := "https://manxue.ai/api/v1/tests"
	if id != "" {
		endpoint += "/" + id
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
		if err != nil {
			return "manxue_request_failed"
		}
		req.Header.Set("Accept", "application/json")
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", key)
		}
		response, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return manxueContextFailure(ctx)
			}
			return "manxue_transport_error"
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
		response.Body.Close()
		if (response.StatusCode == 429 || response.StatusCode == 503) && attempt < 2 {
			delay := 3 * time.Second
			if seconds, e := strconv.Atoi(response.Header.Get("Retry-After")); e == nil && seconds >= 0 {
				delay = time.Duration(min(seconds, 660)) * time.Second
			} else if until, e := http.ParseTime(response.Header.Get("Retry-After")); e == nil {
				delay = time.Until(until)
			}
			if delay < time.Second {
				delay = time.Second
			}
			if !manxueWait(ctx, delay) {
				return manxueContextFailure(ctx)
			}
			continue // The same idempotency key prevents duplicate POST tasks.
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Sprintf("manxue_http_%d", response.StatusCode)
		}
		if output == nil {
			return ""
		}
		if readErr != nil || len(data) > 1024*1024 || json.Unmarshal(data, output) != nil {
			return "manxue_invalid_response"
		}
		return ""
	}
	return "manxue_request_failed"
}
