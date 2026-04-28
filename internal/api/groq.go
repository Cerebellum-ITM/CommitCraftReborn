package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Choice struct {
	Message Message `json:"message"`
}

// Usage mirrors the `usage` block returned by Groq on every chat completion.
// Token counts are integers; timing fields are seconds (float).
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	QueueTime        float64 `json:"queue_time"`
	PromptTime       float64 `json:"prompt_time"`
	CompletionTime   float64 `json:"completion_time"`
	TotalTime        float64 `json:"total_time"`
}
type ResponseBody struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// RateLimits captures the x-ratelimit-* headers Groq sends back on every
// response. Reset fields are server-side wait durations until the bucket
// refills; we keep them as parsed time.Duration. A zero value means the
// header was absent or unparseable. RequestsParsed/TokensParsed flag
// whether both halves of the corresponding bucket (limit + remaining)
// were actually present in the response — without those flags a missing
// `remaining-*` header silently degrades to 0 and the render code can't
// tell "fully consumed" from "no data".
type RateLimits struct {
	LimitRequests     int
	RemainingRequests int
	ResetRequests     time.Duration
	LimitTokens       int
	RemainingTokens   int
	ResetTokens       time.Duration
	CapturedAt        time.Time
	RequestsParsed    bool
	TokensParsed      bool
	// RequestsToday / RequestsDay are the locally-tracked daily counter
	// for requests. Groq's `remaining-requests` header is unreliable for
	// the daily bucket on the free tier, so we count ourselves. The
	// counter resets when RequestsDay (YYYY-MM-DD UTC) changes — that
	// matches Groq's UTC midnight bucket reset.
	RequestsToday int
	RequestsDay   string
}

// CallStats bundles every per-call metric we surface in the UI and
// persist to the ai_calls table.
type CallStats struct {
	Model            string
	RequestID        string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	QueueTime        time.Duration
	PromptTime       time.Duration
	CompletionTime   time.Duration
	TotalTime        time.Duration
	RateLimits       RateLimits
}

// GroqModel mirrors a single entry from GET /openai/v1/models. Only the
// fields we currently surface are decoded; extra fields are ignored.
type GroqModel struct {
	ID            string `json:"id"`
	OwnedBy       string `json:"owned_by"`
	Active        bool   `json:"active"`
	ContextWindow int    `json:"context_window"`
}

type modelsListResponse struct {
	Object string      `json:"object"`
	Data   []GroqModel `json:"data"`
}

// ListGroqModels fetches the catalogue of models the API key can address.
// The endpoint does not flag free-tier vs paid models; callers filter the
// result via the curated allowlist in internal/config.
func ListGroqModels(apiKey string) ([]GroqModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Groq API key was not provided")
	}

	req, err := http.NewRequest("GET", "https://api.groq.com/openai/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"API returned a non-success status: %d, %s",
			resp.StatusCode, string(body),
		)
	}

	var parsed modelsListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error decoding response JSON: %w", err)
	}
	return parsed.Data, nil
}

// GetGroqChatCompletion is a generic function to interact with the Groq Chat API.
// It returns the assistant message content, the parsed call stats (tokens,
// timing, rate-limit headers) and any error. Callers that don't need the
// stats can ignore the second return value.
func GetGroqChatCompletion(
	apiKey, modelName string,
	messages []Message,
) (string, *CallStats, error) {
	if apiKey == "" {
		return "", nil, fmt.Errorf("Groq API key was not provided")
	}
	if modelName == "" {
		return "", nil, fmt.Errorf("model name was not provided")
	}
	if len(messages) == 0 {
		return "", nil, fmt.Errorf("at least one message is required")
	}

	requestData := RequestBody{
		Model:    modelName,
		Messages: messages,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", nil, fmt.Errorf("error encoding JSON: %w", err)
	}

	url := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf(
			"API returned a non-success status: %d, %s",
			resp.StatusCode,
			string(body),
		)
	}

	var responseBody ResponseBody
	if err := json.Unmarshal(body, &responseBody); err != nil {
		return "", nil, fmt.Errorf("error decoding response JSON: %w", err)
	}

	stats := &CallStats{
		Model:            responseBody.Model,
		RequestID:        resp.Header.Get("x-request-id"),
		PromptTokens:     responseBody.Usage.PromptTokens,
		CompletionTokens: responseBody.Usage.CompletionTokens,
		TotalTokens:      responseBody.Usage.TotalTokens,
		QueueTime:        secondsToDuration(responseBody.Usage.QueueTime),
		PromptTime:       secondsToDuration(responseBody.Usage.PromptTime),
		CompletionTime:   secondsToDuration(responseBody.Usage.CompletionTime),
		TotalTime:        secondsToDuration(responseBody.Usage.TotalTime),
		RateLimits:       parseRateLimitHeaders(resp.Header),
	}
	if stats.Model == "" {
		stats.Model = modelName
	}

	if len(responseBody.Choices) > 0 && responseBody.Choices[0].Message.Content != "" {
		return responseBody.Choices[0].Message.Content, stats, nil
	}

	return "", stats, fmt.Errorf("API response did not contain a valid choice")
}

// secondsToDuration converts a Groq "seconds as float64" timing field into
// a time.Duration without losing sub-millisecond precision.
func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// parseRateLimitHeaders pulls the 6 x-ratelimit-* headers from a Groq
// response. Missing or unparseable headers leave the corresponding field
// at its zero value. RequestsParsed and TokensParsed are set to true
// only when *both* the limit and remaining headers for that bucket were
// present and parsed to ints; this lets renderers distinguish "no data"
// from "0 remaining" instead of falling back to a 100%-full bar.
func parseRateLimitHeaders(h http.Header) RateLimits {
	rl := RateLimits{CapturedAt: time.Now()}
	limitReqOK := false
	remainingReqOK := false
	limitTokOK := false
	remainingTokOK := false

	if v := h.Get("x-ratelimit-limit-requests"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.LimitRequests = n
			limitReqOK = true
		}
	}
	if v := h.Get("x-ratelimit-remaining-requests"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.RemainingRequests = n
			remainingReqOK = true
		}
	}
	if v := h.Get("x-ratelimit-reset-requests"); v != "" {
		rl.ResetRequests = parseGroqResetDuration(v)
	}
	if v := h.Get("x-ratelimit-limit-tokens"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.LimitTokens = n
			limitTokOK = true
		}
	}
	if v := h.Get("x-ratelimit-remaining-tokens"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.RemainingTokens = n
			remainingTokOK = true
		}
	}
	if v := h.Get("x-ratelimit-reset-tokens"); v != "" {
		rl.ResetTokens = parseGroqResetDuration(v)
	}
	rl.RequestsParsed = limitReqOK && remainingReqOK
	rl.TokensParsed = limitTokOK && remainingTokOK
	return rl
}

// parseGroqResetDuration accepts either Go-style durations (e.g. "2m59.56s")
// or plain seconds ("5.4") that Groq mixes across endpoints, returning 0
// when the value is unrecognised.
func parseGroqResetDuration(v string) time.Duration {
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return secondsToDuration(f)
	}
	return 0
}
