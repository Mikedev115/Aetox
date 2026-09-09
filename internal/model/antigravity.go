package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The Antigravity runtime — Google's Cloud Code / Antigravity service.
//
// Reaches models like gemini-3.8-flash, gemini-3-pro, and claude-opus-4-5-thinking
// using Google OAuth tokens and the companion project ID.

type AntigravityConfig struct {
	Provider    string
	Model       string
	BaseURL     string
	Project     string
	Timeout     time.Duration
	TokenSource func(context.Context) (string, error)
}

type AntigravityProvider struct {
	provider    string
	model       string
	baseURL     string
	project     string
	tokenSource func(context.Context) (string, error)
	httpClient  *http.Client
}

func NewAntigravityProvider(cfg AntigravityConfig) (*AntigravityProvider, error) {
	provider := strings.TrimSpace(strings.ToLower(cfg.Provider))
	if provider == "" {
		provider = "antigravity"
	}
	model := resolveAntigravityModel(strings.TrimSpace(cfg.Model), nil)
	if model == "" {
		return nil, ErrMissingModel
	}
	if cfg.TokenSource == nil {
		return nil, ErrMissingAPIKey
	}
	baseURL := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" || (strings.Contains(baseURL, "cloudcode-pa.googleapis.com") && !strings.Contains(baseURL, "daily-")) {
		baseURL = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	}

	project := strings.TrimSpace(cfg.Project)
	if project == "" {
		project = "aicode-consumers"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &AntigravityProvider{
		provider:    provider,
		model:       model,
		baseURL:     baseURL,
		project:     project,
		tokenSource: cfg.TokenSource,
		httpClient:  newModelHTTPClient(timeout, baseURL),
	}, nil
}

func resolveAntigravityModel(model string, reasoning *ReasoningConfig) string {
	m := strings.TrimSpace(strings.ToLower(model))
	switch m {
	case "gemini-3.8-flash":
		if reasoning != nil {
			switch strings.ToLower(reasoning.Effort) {
			case "low":
				return "gemini-3.8-flash-low"
			case "medium":
				return "gemini-3.8-flash-medium"
			}
		}
		return "gemini-3.8-flash-high"
	case "gemini-3.7-flash":
		if reasoning != nil {
			switch strings.ToLower(reasoning.Effort) {
			case "low":
				return "gemini-3.7-flash-low"
			case "medium":
				return "gemini-3.7-flash-medium"
			}
		}
		return "gemini-3.7-flash-high"
	case "gemini-3.6-flash":
		if reasoning != nil {
			switch strings.ToLower(reasoning.Effort) {
			case "low":
				return "gemini-3.6-flash-low"
			case "medium":
				return "gemini-3.6-flash-medium"
			}
		}
		return "gemini-3.6-flash-high"
	case "gemini-3-pro", "gemini-3.1-pro", "gemini-pro":
		return "gemini-3.1-pro-low"
	case "claude-opus-4-5-thinking", "claude-opus":
		return "claude-opus-4-6-thinking"
	case "claude-sonnet-4-5", "claude-sonnet":
		return "claude-sonnet-4-6"
	default:
		return model
	}
}

func (p *AntigravityProvider) Name() string { return p.provider }

func (p *AntigravityProvider) SupportsToolCalling() bool { return true }

func (p *AntigravityProvider) SupportsReasoning() bool { return true }

// ---------------------------------------------------------------------------
// Wire types
// ---------------------------------------------------------------------------

type caRequest struct {
	Model   string      `json:"model"`
	Project string      `json:"project,omitempty"`
	Request geminiInner `json:"request"`
}

type geminiInner struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	Tools             []geminiTools    `json:"tools,omitempty"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string              `json:"text,omitempty"`
	Thought          bool                `json:"thought,omitempty"`
	ThoughtSignature string              `json:"thoughtSignature,omitempty"`
	InlineData       *geminiInlineData   `json:"inlineData,omitempty"`
	FunctionCall     *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResp `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

type geminiFunctionResp struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTools struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type geminiGenConfig struct {
	Temperature     float64            `json:"temperature,omitempty"`
	MaxOutputTokens int                `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *geminiThinkConfig `json:"thinkingConfig,omitempty"`
}

type geminiThinkConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"`
}

type caResponse struct {
	Response struct {
		Candidates []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata *struct {
			PromptTokenCount        int `json:"promptTokenCount"`
			CandidatesTokenCount    int `json:"candidatesTokenCount"`
			TotalTokenCount         int `json:"totalTokenCount"`
			CachedContentTokenCount int `json:"cachedContentTokenCount"`
			ThoughtsTokenCount      int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
		ModelVersion string `json:"modelVersion"`
	} `json:"response"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// ---------------------------------------------------------------------------
// Request building
// ---------------------------------------------------------------------------

func buildAntigravityRequest(model, project string, req Request) (caRequest, error) {
	model = resolveAntigravityModel(model, req.Reasoning)
	system, contents := convertMessagesToGemini(req.Messages)
	if len(contents) == 0 {
		return caRequest{}, ErrNoMessages
	}

	inner := geminiInner{
		Contents: contents,
		Tools:    convertToolsToGemini(req.Tools),
	}
	if system != "" {
		inner.SystemInstruction = &geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: system}},
		}
	}
	if req.Temperature > 0 || req.MaxTokens > 0 || req.Reasoning != nil {
		inner.GenerationConfig = &geminiGenConfig{}
		if req.Temperature > 0 {
			inner.GenerationConfig.Temperature = req.Temperature
		}
		if req.MaxTokens > 0 {
			inner.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Reasoning != nil && req.Reasoning.Effort != "" {
			inner.GenerationConfig.ThinkingConfig = &geminiThinkConfig{IncludeThoughts: true}
		}
	}

	return caRequest{
		Model:   model,
		Project: project,
		Request: inner,
	}, nil
}

func convertMessagesToGemini(messages []Message) (system string, contents []geminiContent) {
	for _, m := range messages {
		if m.Role == RoleSystem {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}

		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}

		var parts []geminiPart
		if m.Content != "" {
			parts = append(parts, geminiPart{Text: m.Content})
		}
		for _, img := range m.Images {
			if len(img.Data) > 0 {
				parts = append(parts, geminiPart{
					InlineData: &geminiInlineData{
						MimeType: img.MediaType,
						Data:     base64.StdEncoding.EncodeToString(img.Data),
					},
				})
			}
		}
		for _, tc := range m.ToolCalls {
			args := json.RawMessage(tc.Function.Arguments)
			if len(args) == 0 {
				args = json.RawMessage("{}")
			}
			gp := geminiPart{
				FunctionCall: &geminiFunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: args,
				},
			}
			if len(tc.ExtraContent) > 0 {
				var sig string
				if err := json.Unmarshal(tc.ExtraContent, &sig); err == nil && sig != "" {
					gp.ThoughtSignature = sig
				} else {
					gp.ThoughtSignature = strings.Trim(string(tc.ExtraContent), `"`)
				}
			}
			parts = append(parts, gp)
		}
		if m.Role == RoleTool {
			name := m.Name
			if name == "" {
				name = "tool_result"
			}
			var respMap map[string]any
			if err := json.Unmarshal([]byte(m.Content), &respMap); err != nil {
				respMap = map[string]any{"output": m.Content}
			}
			parts = append(parts, geminiPart{
				FunctionResponse: &geminiFunctionResp{
					ID:       m.ToolCallID,
					Name:     name,
					Response: respMap,
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}
	return system, contents
}

func convertToolsToGemini(tools []ToolDefinition) []geminiTools {
	if len(tools) == 0 {
		return nil
	}
	decls := make([]geminiFunctionDecl, 0, len(tools))
	for _, tool := range tools {
		decls = append(decls, geminiFunctionDecl{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}
	return []geminiTools{{FunctionDeclarations: decls}}
}

// ---------------------------------------------------------------------------
// Streaming & Completion
// ---------------------------------------------------------------------------

func (p *AntigravityProvider) Complete(ctx context.Context, req Request) (Response, error) {
	return p.StreamComplete(ctx, req, nil, nil)
}

func (p *AntigravityProvider) StreamComplete(ctx context.Context, req Request, onChunk StreamChunkHandler, onReasoningChunk StreamChunkHandler) (Response, error) {
	if len(req.Messages) == 0 {
		return Response{}, ErrNoMessages
	}

	model := req.Model
	if model == "" {
		model = p.model
	}
	model = resolveAntigravityModel(model, req.Reasoning)

	payload, err := buildAntigravityRequest(model, p.project, req)
	if err != nil {
		return Response{}, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}

	token, err := p.tokenSource(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("%s sign-in: %w", p.provider, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+":streamGenerateContent?alt=sse", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("User-Agent", "antigravity/2.12.2")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return Response{}, p.statusError(httpResp)
	}

	var text, reasoning strings.Builder
	var toolCalls []ToolCall
	progress := newToolProgressTracker(req.OnToolCallProgress)
	respModel := model
	var usage Usage
	var finishReason string
	var streamErr error

	var lastThoughtSig string

	err = scanSSE(httpResp.Body, func(data string) (bool, error) {
		var chunk caResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return false, nil
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			streamErr = fmt.Errorf("%s: %s", p.provider, chunk.Error.Message)
			return true, nil
		}
		if chunk.Response.ModelVersion != "" {
			respModel = chunk.Response.ModelVersion
		}
		if u := chunk.Response.UsageMetadata; u != nil {
			usage = Usage{
				PromptTokens:       u.PromptTokenCount,
				CachedPromptTokens: u.CachedContentTokenCount,
				CacheReported:      true,
				CompletionTokens:   u.CandidatesTokenCount + u.ThoughtsTokenCount,
				TotalTokens:        u.TotalTokenCount,
			}
		}

		for _, candidate := range chunk.Response.Candidates {
			if candidate.FinishReason != "" {
				finishReason = candidate.FinishReason
			}
			for _, part := range candidate.Content.Parts {
				if part.ThoughtSignature != "" {
					lastThoughtSig = part.ThoughtSignature
				}
				switch {
				case part.FunctionCall != nil:
					args := strings.TrimSpace(string(part.FunctionCall.Args))
					if args == "" {
						args = "{}"
					}
					id := part.FunctionCall.ID
					if id == "" {
						id = "call_" + strconv.Itoa(len(toolCalls)+1)
					}
					sig := part.ThoughtSignature
					if sig == "" {
						sig = lastThoughtSig
					}
					tc := ToolCall{
						ID:       id,
						Type:     "function",
						Function: FunctionCall{Name: part.FunctionCall.Name, Arguments: args},
					}
					if sig != "" {
						tc.ExtraContent = json.RawMessage(strconv.Quote(sig))
					}
					toolCalls = append(toolCalls, tc)
					progress.report(len(toolCalls)-1, id, part.FunctionCall.Name, args)

				case part.Thought:
					reasoning.WriteString(part.Text)
					if onReasoningChunk != nil && part.Text != "" {
						if err := onReasoningChunk(part.Text); err != nil {
							return true, err
						}
					}

				case part.Text != "":
					text.WriteString(part.Text)
					if onChunk != nil {
						if err := onChunk(part.Text); err != nil {
							return true, err
						}
					}
				}
			}
		}
		return false, nil
	})

	if streamErr != nil {
		return Response{}, streamErr
	}
	if err != nil {
		return Response{}, err
	}

	if strings.EqualFold(finishReason, "max_tokens") || (req.MaxTokens > 0 && usage.CompletionTokens >= req.MaxTokens) {
		finishReason = FinishReasonLength
	}

	textOut := strings.TrimSpace(text.String())
	reasoningOut := strings.TrimSpace(reasoning.String())
	if err := errEmptyCompletion(p.provider, finishReason, textOut, reasoningOut, len(toolCalls)); err != nil {
		return Response{}, err
	}

	return Response{
		Provider:         p.Name(),
		Model:            modelOr(respModel, model),
		Text:             textOut,
		ReasoningContent: reasoningOut,
		ToolCalls:        toolCalls,
		FinishReason:     finishReason,
		Usage:            normalizeUsage(usage),
	}, nil
}

func (p *AntigravityProvider) statusError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	detail := strings.TrimSpace(string(body))
	if len(detail) > 500 {
		detail = detail[:500] + "…"
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%s rejected the sign-in. Sign in again. (401: %s)", p.provider, detail)
	case http.StatusForbidden:
		return fmt.Errorf("%s refused this account (403). Ensure this account has Antigravity access: %s", p.provider, detail)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%s rate limit reached (429): %s", p.provider, detail)
	default:
		return fmt.Errorf("%s answered %s: %s", p.provider, resp.Status, detail)
	}
}

// DiscoverAntigravityModels dynamically queries the backend endpoint
// (:fetchAvailableModels) to discover the models permitted for the authenticated account,
// preserving the recommended ordering provided by the server.
func DiscoverAntigravityModels(ctx context.Context, providerName, baseURL, token string) ([]string, error) {
	endpoint := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if endpoint == "" || (strings.Contains(endpoint, "cloudcode-pa.googleapis.com") && !strings.Contains(endpoint, "daily-")) {
		endpoint = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpoint+":fetchAvailableModels", strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "antigravity/2.12.2")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s model list failed with status %d: %s",
			providerName, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		DefaultAgentModelID string `json:"defaultAgentModelId"`
		AgentModelSorts     []struct {
			DisplayName string `json:"displayName"`
			Groups      []struct {
				ModelIDs []string `json:"modelIds"`
			} `json:"groups"`
		} `json:"agentModelSorts"`
		Models map[string]struct {
			DisplayName string `json:"displayName"`
		} `json:"models"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%s model list parse failed: %w", providerName, err)
	}

	seen := make(map[string]bool)
	var result []string

	// 1. Put recommended models first in server-specified order
	for _, sort := range parsed.AgentModelSorts {
		for _, group := range sort.Groups {
			for _, id := range group.ModelIDs {
				id = strings.TrimSpace(id)
				if id != "" && !seen[id] {
					seen[id] = true
					result = append(result, id)
				}
			}
		}
	}

	// 2. Add defaultAgentModelID if not already present
	if def := strings.TrimSpace(parsed.DefaultAgentModelID); def != "" && !seen[def] {
		seen[def] = true
		result = append([]string{def}, result...)
	}

	// 3. Add any other models in parsed.Models that aren't already included
	for id := range parsed.Models {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result, nil
}
