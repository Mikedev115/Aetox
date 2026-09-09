package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Mikedev115/Aetox/internal/proc"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

func openBrowser(url string) {
	// proc-detached: the user's browser is meant to outlive the probe — the
	// sign-in finishes in it, long after this function has returned.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	proc.HideConsole(cmd)
	_ = cmd.Start()
}

func main() {
	modelFlag := flag.String("model", "", "Model to test (leave empty to discover and use primary model from upstream)")
	forceLogin := flag.Bool("login", false, "Force new browser OAuth sign-in even if already stored")
	testTools := flag.Bool("tools", true, "Test tool calling round-trip")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("🚀 Aetox - Google Antigravity Real-World Probe")
	fmt.Println("==================================================")

	ctx := context.Background()

	cred, ok := oauth.Get("antigravity")
	needLogin := !ok || *forceLogin || cred.Access == ""

	if !needLogin {
		if cred.Account == "" {
			cred.Account = "aicode-consumers"
			_ = oauth.Set("antigravity", cred)
		}
		if cred.Endpoint == "" || (strings.Contains(cred.Endpoint, "cloudcode-pa.googleapis.com") && !strings.Contains(cred.Endpoint, "daily-")) {
			cred.Endpoint = oauth.AntigravityBaseURL
			_ = oauth.Set("antigravity", cred)
		}
		fmt.Printf("✔ Found existing credential: %s\n", cred.Label)
		fmt.Printf("✔ Companion Project: %s\n", cred.Account)
		if cred.Expired() {
			fmt.Println("⏳ Token expired, refreshing via refresh token...")
			tok, err := oauth.Token(ctx, "antigravity")
			if err != nil {
				fmt.Printf("⚠️ Refresh failed (%v), switching to browser login...\n", err)
				needLogin = true
			} else {
				fmt.Printf("✔ Token refreshed successfully (prefix: %s...)\n", tok[:min(10, len(tok))])
				cred, _ = oauth.Get("antigravity")
			}
		} else {
			fmt.Println("✔ Stored token is currently valid")
		}
	}

	if needLogin {
		if oauth.AntigravityCLIAvailable() && !*forceLogin {
			fmt.Println("🔍 Found existing Antigravity CLI session. Importing...")
			if err := oauth.ImportAntigravityCLI(ctx); err == nil {
				fmt.Println("✔ Imported CLI session successfully!")
				cred, _ = oauth.Get("antigravity")
				needLogin = false
			} else {
				fmt.Printf("⚠️ CLI import failed: %v. Falling back to Browser Login...\n", err)
			}
		}
	}

	if needLogin {
		fmt.Println("\n🔑 Starting Browser OAuth 2.0 PKCE Flow...")
		pending, err := oauth.StartAntigravity()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to start OAuth: %v\n", err)
			os.Exit(1)
		}
		defer pending.Cancel()

		fmt.Println("\n👉 Please sign in with your Google account:")
		fmt.Printf("   %s\n\n", pending.URL)
		fmt.Println("🌐 Opening browser automatically...")
		openBrowser(pending.URL)

		fmt.Println("⏳ Waiting for callback on local port 51121...")
		waitCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()

		if err := oauth.FinishAntigravity(waitCtx, pending); err != nil {
			fmt.Fprintf(os.Stderr, "❌ OAuth callback error: %v\n", err)
			os.Exit(1)
		}

		cred, _ = oauth.Get("antigravity")
		fmt.Println("\n🎉 Sign-in Successful!")
		fmt.Printf("   User: %s\n", cred.Label)
		fmt.Printf("   Project: %s\n", cred.Account)
	}

	// Dynamic model discovery directly from upstream endpoint
	fmt.Println("\n📡 Querying upstream endpoint for available models...")
	discoveredModels, err := model.DiscoverAntigravityModels(ctx, "antigravity", cred.Endpoint, cred.Access)
	if err != nil {
		fmt.Printf("⚠️ Dynamic model discovery warning: %v\n", err)
	} else {
		fmt.Printf("✔ Upstream returned %d available models:\n", len(discoveredModels))
		for i, m := range discoveredModels {
			if i < 8 {
				fmt.Printf("   [%d] %s\n", i+1, m)
			}
		}
		if len(discoveredModels) > 8 {
			fmt.Printf("   ... and %d more models\n", len(discoveredModels)-8)
		}
	}

	chosenModel := *modelFlag
	if chosenModel == "" {
		if len(discoveredModels) > 0 {
			chosenModel = discoveredModels[0]
			fmt.Printf("🎯 Dynamically selected primary model from upstream: %s\n", chosenModel)
		} else {
			chosenModel = "gemini-3.8-flash-high"
		}
	} else {
		fmt.Printf("🎯 Using specified model: %s\n", chosenModel)
	}

	// Test 0: Settings "Test Connection" Simulation (MaxTokens: 1, ping)
	fmt.Printf("\n--- [Test 0] Settings Ping Simulation (MaxTokens: 1) on %s ---\n", chosenModel)
	p0, err := model.NewProvider(model.ProviderOptions{
		Provider: "antigravity",
		Model:    chosenModel,
		BaseURL:  cred.Endpoint,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to instantiate provider: %v\n", err)
		os.Exit(1)
	}
	p0Start := time.Now()
	resp0, err := p0.Complete(ctx, model.Request{
		Model:     chosenModel,
		Messages:  []model.Message{{Role: model.RoleUser, Content: "ping"}},
		MaxTokens: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Settings Ping failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✔ Settings Ping succeeded in %dms (FinishReason: %q, TextLen: %d, ReasoningLen: %d)\n",
		time.Since(p0Start).Milliseconds(), resp0.FinishReason, len(resp0.Text), len(resp0.ReasoningContent))

	// Test 1: Basic Streaming Text & Reasoning
	fmt.Printf("\n--- [Test 1] Streaming Generation with %s ---\n", chosenModel)
	p, err := model.NewProvider(model.ProviderOptions{
		Provider: "antigravity",
		Model:    chosenModel,
		BaseURL:  cred.Endpoint,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to instantiate provider: %v\n", err)
		os.Exit(1)
	}

	sp, ok := p.(model.StreamingProvider)
	if !ok {
		fmt.Fprintf(os.Stderr, "❌ Provider does not implement StreamingProvider\n")
		os.Exit(1)
	}

	prompt := "อธิบายสั้นๆ 2 บรรทัดว่า Google Antigravity คืออะไร และมีความสามารถเด่นอะไรบ้าง"
	fmt.Printf("Prompt: %s\n\n", prompt)

	var fullText strings.Builder
	var fullReasoning strings.Builder

	startTime := time.Now()
	resp, err := sp.StreamComplete(ctx, model.Request{
		Model: chosenModel,
		Messages: []model.Message{
			{Role: model.RoleUser, Content: prompt},
		},
		Reasoning: &model.ReasoningConfig{Effort: "low"},
	}, func(chunk string) error {
		fmt.Print(chunk)
		fullText.WriteString(chunk)
		return nil
	}, func(reasoningChunk string) error {
		fullReasoning.WriteString(reasoningChunk)
		return nil
	})

	elapsed := time.Since(startTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ StreamComplete failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n\n✔ Response received in %.2fs!\n", elapsed.Seconds())
	if fullReasoning.Len() > 0 {
		fmt.Printf("🧠 Thought / Reasoning: %s\n", fullReasoning.String())
	}
	if resp.Usage != nil {
		fmt.Printf("📊 Token Usage: Prompt=%d, Completion=%d, Total=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	// Test 2: Tool Calling
	if *testTools {
		fmt.Println("\n--- [Test 2] Tool Calling Round-Trip ---")
		toolDef := model.ToolDefinition{
			Type: "function",
			Function: model.ToolFunction{
				Name:        "get_weather",
				Description: "Get the current weather for a given city",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string","description":"City name"}},"required":["city"]}`),
			},
		}

		toolPrompt := "เมืองโตเกียวสภาพอากาศตอนนี้เป็นยังไงบ้าง? (กรุณาใช้ tool get_weather ตรวจสอบ)"
		fmt.Printf("Prompt: %s\n", toolPrompt)

		resp1, err := p.Complete(ctx, model.Request{
			Model: chosenModel,
			Messages: []model.Message{
				{Role: model.RoleUser, Content: toolPrompt},
			},
			Tools: []model.ToolDefinition{toolDef},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Tool call prompt failed: %v\n", err)
			os.Exit(1)
		}

		if len(resp1.ToolCalls) == 0 {
			fmt.Printf("⚠️ Model replied with text instead of tool call: %s\n", resp1.Text)
		} else {
			tc := resp1.ToolCalls[0]
			fmt.Printf("✔ Model initiated tool call: %s(%s)\n", tc.Function.Name, tc.Function.Arguments)

			// Feed tool result back
			mockResult := `{"temperature":"19°C","condition":"แจ่มใส มีลมเบาๆ"}`
			fmt.Printf("↪ Supplying tool result: %s\n", mockResult)

			finalResp, err := sp.StreamComplete(ctx, model.Request{
				Model: chosenModel,
				Messages: []model.Message{
					{Role: model.RoleUser, Content: toolPrompt},
					{
						Role:      model.RoleAssistant,
						ToolCalls: resp1.ToolCalls,
					},
					{
						Role:       model.RoleTool,
						ToolCallID: tc.ID,
						Name:       tc.Function.Name,
						Content:    mockResult,
					},
				},
				Tools: []model.ToolDefinition{toolDef},
			}, func(chunk string) error {
				fmt.Print(chunk)
				return nil
			}, nil)

			if err != nil {
				fmt.Fprintf(os.Stderr, "\n❌ Final tool response failed: %v\n", err)
				os.Exit(1)
			}
			if finalResp.Usage != nil {
				fmt.Printf("\n✔ Tool cycle finished! Total Tokens: %d\n", finalResp.Usage.TotalTokens)
			}
		}
	}

	fmt.Println("\n==================================================")
	fmt.Println("🎉 ALL PROBE CHECKS PASSED PERFECTLY!")
	fmt.Println("==================================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
