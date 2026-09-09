package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHub Copilot sign-in.
//
// Two tokens, not one. The device flow yields a long-lived GitHub OAuth token,
// which is only good for asking /copilot_internal/v2/token for a short-lived
// Copilot token (~25 minutes) — that second one is what api.githubcopilot.com
// accepts. So the GitHub token is stored in the Refresh slot: refreshing here
// means re-minting, not an OAuth refresh_token grant.
const (
	copilotClientID    = "Iv1.b507a08c87ecfe98" // the editor plugin's public client
	copilotDeviceURL   = "https://github.com/login/device/code"
	copilotAccessURL   = "https://github.com/login/oauth/access_token"
	copilotTokenURL    = "https://api.github.com/copilot_internal/v2/token"
	copilotUserURL     = "https://api.github.com/user"
	copilotIntegration = "vscode-chat"

	// These identify Aetox to Copilot as an editor client. They are pinned
	// constants rather than anything read off this machine — the values a
	// developer happens to have installed must never leak into what we send.
	copilotUserAgent     = "GitHubCopilotChat/0.32.4"
	copilotEditorVersion = "vscode/1.105.1"
	copilotPluginVersion = "copilot-chat/0.32.4"

	// CopilotBaseURL is GitHub Copilot's OpenAI-compatible completions endpoint.
	CopilotBaseURL = "https://api.githubcopilot.com"
)

// ErrCopilotNotAvailable means the GitHub account signed in fine but has no
// Copilot entitlement — a different problem from a failed login, and one the
// user fixes on github.com rather than by retrying.
var ErrCopilotNotAvailable = errors.New("this GitHub account has no active Copilot subscription")

// CopilotHeaders are required on every Copilot request, including the token
// mint. Without Copilot-Integration-Id the API answers 400.
func CopilotHeaders() map[string]string {
	return map[string]string{
		"User-Agent":             copilotUserAgent,
		"Editor-Version":         copilotEditorVersion,
		"Editor-Plugin-Version":  copilotPluginVersion,
		"Copilot-Integration-Id": copilotIntegration,
	}
}

// StartCopilot asks GitHub for a device code. The user types Pending.UserCode
// into Pending.VerificationURI; nothing is stored until FinishCopilot returns.
func StartCopilot(ctx context.Context) (*Pending, error) {
	form := url.Values{}
	form.Set("client_id", copilotClientID)
	form.Set("scope", "read:user")

	var out struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := copilotForm(ctx, copilotDeviceURL, form, &out); err != nil {
		return nil, fmt.Errorf("github sign-in failed: %w", err)
	}
	if out.DeviceCode == "" {
		return nil, errors.New("github returned no device code")
	}
	return &Pending{
		URL:             out.VerificationURI,
		UserCode:        out.UserCode,
		VerificationURI: out.VerificationURI,
		deviceCode:      out.DeviceCode,
		interval:        out.Interval,
		expiresIn:       out.ExpiresIn,
		provider:        "github-copilot",
	}, nil
}

// FinishCopilot blocks until the user authorizes in the browser, then stores
// the login. Cancel ctx to give up.
func FinishCopilot(ctx context.Context, pending *Pending) error {
	if pending == nil || pending.deviceCode == "" {
		return errors.New("no sign-in in progress")
	}
	tokens, err := pollDeviceCode(ctx, pending, copilotAccessURL, url.Values{
		"client_id":   {copilotClientID},
		"device_code": {pending.deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}, copilotForm)
	if err != nil {
		return err
	}
	githubToken := tokens.AccessToken

	cred, err := mintCopilotToken(ctx, githubToken)
	if err != nil {
		return err
	}
	cred.Label = "GitHub Copilot"
	if login := copilotLogin(ctx, githubToken); login != "" {
		cred.Account = login
		cred.Label = "GitHub Copilot · " + login
	}
	return Set("github-copilot", cred)
}

func refreshCopilot(ctx context.Context, cred Credential) (Credential, error) {
	if strings.TrimSpace(cred.Refresh) == "" {
		return Credential{}, errors.New("copilot sign-in is incomplete — sign in again")
	}
	next, err := mintCopilotToken(ctx, cred.Refresh)
	if err != nil {
		return Credential{}, err
	}
	// Keep the identity fields: minting a token says nothing about who owns it.
	next.Account = cred.Account
	next.Label = cred.Label
	return next, nil
}

func mintCopilotToken(ctx context.Context, githubToken string) (Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilotTokenURL, nil)
	if err != nil {
		return Credential{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+githubToken)
	for k, v := range CopilotHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return Credential{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		return Credential{}, ErrCopilotNotAvailable
	}
	var out struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"` // unix seconds
	}
	if err := readJSON(resp, &out); err != nil {
		return Credential{}, fmt.Errorf("copilot token request failed: %w", err)
	}
	if out.Token == "" {
		return Credential{}, errors.New("copilot returned no token")
	}
	expires := out.ExpiresAt * 1000
	if out.ExpiresAt == 0 {
		// Undocumented but observed: some responses omit expires_at. Assume the
		// documented 25 minutes rather than treating the token as eternal.
		expires = time.Now().Add(25 * time.Minute).UnixMilli()
	}
	return Credential{
		Type:      "oauth",
		Access:    out.Token,
		Refresh:   githubToken,
		ExpiresAt: expires,
		Endpoint:  CopilotBaseURL,
	}, nil
}

// copilotLogin is best-effort decoration for the Settings row. A failure here
// must not fail the sign-in.
func copilotLogin(ctx context.Context, githubToken string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilotUserURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+githubToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var out struct {
		Login string `json:"login"`
	}
	if err := readJSON(resp, &out); err != nil {
		return ""
	}
	return out.Login
}

func copilotForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", copilotUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readJSON(resp, out)
}

// ---------------------------------------------------------------------------
// Importing existing Copilot CLI / editor session
// ---------------------------------------------------------------------------

// ImportCopilotCLI adopts an existing GitHub Copilot token from local configs.
func ImportCopilotCLI(ctx context.Context) error {
	token, user := findLocalCopilotToken()
	if token == "" {
		return errors.New("no local GitHub Copilot session found — please use Device Sign-in")
	}

	cred, err := mintCopilotToken(ctx, token)
	if err != nil {
		return fmt.Errorf("could not activate imported Copilot session: %w", err)
	}

	cred.Label = "GitHub Copilot (from CLI)"
	if user != "" {
		cred.Account = user
		cred.Label = "GitHub Copilot · " + user
	} else if login := copilotLogin(ctx, token); login != "" {
		cred.Account = login
		cred.Label = "GitHub Copilot · " + login
	}

	return Set("github-copilot", cred)
}

// CopilotCLIAvailable reports whether local Copilot credentials exist on disk.
func CopilotCLIAvailable() bool {
	token, _ := findLocalCopilotToken()
	return token != ""
}

func findLocalCopilotToken() (token string, user string) {
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", "github-copilot", "hosts.json"),
			filepath.Join(home, ".config", "github-copilot", "apps.json"),
		)
	}
	if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
		paths = append(paths, filepath.Join(appdata, "github-copilot", "hosts.json"))
	}

	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var hosts map[string]struct {
			User       string `json:"user"`
			OAuthToken string `json:"oauth_token"`
		}
		if err := json.Unmarshal(raw, &hosts); err == nil {
			for _, entry := range hosts {
				if entry.OAuthToken != "" {
					return entry.OAuthToken, entry.User
				}
			}
		}
	}
	return "", ""
}
