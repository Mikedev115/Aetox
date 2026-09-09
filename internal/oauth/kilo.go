package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Kilo Code sign-in and session management.
//
// Reaches models across 500+ frontier and open-weight models via the Kilo AI Gateway
// (https://api.kilo.ai/api/gateway).
const (
	// KiloGatewayBaseURL is the unified OpenAI-compatible endpoint.
	KiloGatewayBaseURL = "https://api.kilo.ai/api/gateway"
)

// ImportKiloCLI adopts an existing Kilo CLI session or environment key.
func ImportKiloCLI(ctx context.Context) error {
	token, account := findLocalKiloToken()
	if token == "" {
		return errors.New("no local Kilo Code session found — please set KILO_API_KEY or use Kilo CLI login")
	}

	cred := Credential{
		Type:     "api",
		Key:      token,
		Access:   token,
		Endpoint: KiloGatewayBaseURL,
		Label:    "Kilo Code",
	}
	if account != "" {
		cred.Account = account
		cred.Label = "Kilo Code · " + account
	}

	return Set("kilo", cred)
}

// KiloCLIAvailable reports whether local Kilo Code credentials exist.
func KiloCLIAvailable() bool {
	token, _ := findLocalKiloToken()
	return token != ""
}

func findLocalKiloToken() (string, string) {
	if envKey := strings.TrimSpace(os.Getenv("KILO_API_KEY")); envKey != "" {
		return envKey, "env"
	}

	var candidates []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".config", "kilo", "auth.json"),
			filepath.Join(home, ".config", "kilocode", "auth.json"),
			filepath.Join(home, ".kilo", "auth.json"),
			filepath.Join(home, ".kilocode", "auth.json"),
			filepath.Join(home, ".config", "kilo", "config.json"),
		)
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		candidates = append(candidates,
			filepath.Join(appdata, "kilo", "auth.json"),
			filepath.Join(appdata, "kilocode", "auth.json"),
		)
	}

	for _, p := range candidates {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var parsed struct {
			APIKey      string `json:"apiKey"`
			Key         string `json:"key"`
			Token       string `json:"token"`
			AccessToken string `json:"access_token"`
			Email       string `json:"email"`
			User        string `json:"user"`
		}
		if err := json.Unmarshal(raw, &parsed); err == nil {
			token := firstNonEmpty(parsed.APIKey, parsed.Key, parsed.Token, parsed.AccessToken)
			if token != "" {
				user := firstNonEmpty(parsed.Email, parsed.User)
				return token, user
			}
		}
	}

	return "", ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
