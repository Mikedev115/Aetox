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
)

// Google Antigravity sign-in.
//
// Signs in through Google OAuth to access Antigravity model quotas (Gemini 3 Pro,
// 3.8 Flash, Claude models served via Antigravity).
//
// Uses PKCE with a loopback listener (default registered port 51121 with dynamic
// fallback). The project id is resolved via Cloud Code backend (:loadCodeAssist)
// and carried on Credential.Account.
const (
	antigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityClientSecret = "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"
	antigravityAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	antigravityTokenURL     = "https://oauth2.googleapis.com/token"
	antigravityUserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
	antigravityRedirectHost = "localhost"
	antigravityRedirectPort = 51121
	antigravityRedirectPath = "/oauth-callback"
	antigravityScope        = "openid https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/cloud-platform"

	// AntigravityBaseURL is the Google Antigravity service endpoint.
	AntigravityBaseURL = "https://daily-cloudcode-pa.googleapis.com/v1internal"
	antigravityDefaultProject = "aicode-consumers"
	antigravityUserAgent      = "antigravity/2.12.2"
)

// StartAntigravity opens the Google consent screen and the loopback listener.
func StartAntigravity() (*Pending, error) {
	verifier, challenge, err := NewPKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomString(32)
	if err != nil {
		return nil, err
	}

	lb, err := StartLoopbackAs(antigravityRedirectHost, antigravityRedirectPort, antigravityRedirectPath)
	if err != nil {
		// Port 51121 in use; fall back to any available loopback port.
		lb, err = StartLoopback(0, antigravityRedirectPath)
		if err != nil {
			return nil, fmt.Errorf("Google Antigravity sign-in failed to start callback listener: %w", err)
		}
	}

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", antigravityClientID)
	q.Set("redirect_uri", lb.RedirectURI)
	q.Set("scope", antigravityScope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")

	return &Pending{
		URL:      antigravityAuthorizeURL + "?" + q.Encode(),
		Verifier: verifier,
		State:    state,
		provider: "antigravity",
		lb:       lb,
	}, nil
}

// FinishAntigravity catches the callback, exchanges the code, and saves the login.
func FinishAntigravity(ctx context.Context, pending *Pending) error {
	if pending == nil || pending.lb == nil {
		return errors.New("no sign-in in progress")
	}
	defer pending.Cancel()

	code, state, err := pending.lb.Wait(ctx)
	if err != nil {
		return err
	}
	if state != "" && state != pending.State {
		return errors.New("authorization code does not match this sign-in — start again")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", pending.lb.RedirectURI)
	form.Set("client_id", antigravityClientID)
	form.Set("client_secret", antigravityClientSecret)
	form.Set("code_verifier", pending.Verifier)

	tokens, err := antigravityToken(ctx, form)
	if err != nil {
		return err
	}

	cred := Credential{
		Type:      "oauth",
		Access:    tokens.AccessToken,
		Refresh:   tokens.RefreshToken,
		ExpiresAt: tokens.expiresAt(),
		Endpoint:  AntigravityBaseURL,
		Label:     "Google Antigravity",
		Account:   antigravityDefaultProject,
	}

	email := antigravityEmail(ctx, tokens.AccessToken)
	if email != "" {
		cred.Label = "Google Antigravity · " + email
	}

	project, err := resolveAntigravityProject(ctx, tokens.AccessToken)
	if err == nil && project != "" {
		cred.Account = project
	}

	return Set("antigravity", cred)
}

func refreshAntigravity(ctx context.Context, cred Credential) (Credential, error) {
	if strings.TrimSpace(cred.Refresh) == "" {
		return Credential{}, errors.New("Antigravity sign-in has no refresh token — sign in again")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.Refresh)
	form.Set("client_id", antigravityClientID)
	form.Set("client_secret", antigravityClientSecret)

	tokens, err := antigravityToken(ctx, form)
	if err != nil {
		return Credential{}, err
	}

	next := cred
	next.Access = tokens.AccessToken
	if tokens.RefreshToken != "" {
		next.Refresh = tokens.RefreshToken
	}
	next.ExpiresAt = tokens.expiresAt()
	return next, nil
}

func antigravityToken(ctx context.Context, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, antigravityTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", antigravityUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	var tokens tokenResponse
	if err := readJSON(resp, &tokens); err != nil {
		return tokenResponse{}, fmt.Errorf("Google Antigravity sign-in failed: %w", err)
	}
	if tokens.AccessToken == "" {
		return tokenResponse{}, errors.New("Google Antigravity sign-in returned no access token")
	}
	return tokens, nil
}

func antigravityEmail(ctx context.Context, accessToken string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, antigravityUserInfoURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", antigravityUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	var out struct {
		Email string `json:"email"`
	}
	if err := readJSON(resp, &out); err != nil {
		return ""
	}
	return out.Email
}

// resolveAntigravityProject queries the backend for the companion project ID.
func resolveAntigravityProject(ctx context.Context, accessToken string) (string, error) {
	type caMetadata struct {
		IDEType  string `json:"ideType"`
		Platform string `json:"platform"`
	}
	load := struct {
		Metadata caMetadata `json:"metadata"`
	}{
		Metadata: caMetadata{
			IDEType:  "ANTIGRAVITY",
			Platform: "PLATFORM_UNSPECIFIED",
		},
	}
	payload, err := json.Marshal(load)
	if err != nil {
		return antigravityDefaultProject, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AntigravityBaseURL+":loadCodeAssist", strings.NewReader(string(payload)))
	if err != nil {
		return antigravityDefaultProject, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", antigravityUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return antigravityDefaultProject, err
	}

	var loaded struct {
		CloudAICompanionProject string `json:"cloudaicompanionProject"`
	}
	if err := readJSON(resp, &loaded); err != nil {
		return antigravityDefaultProject, err
	}
	if loaded.CloudAICompanionProject != "" {
		return loaded.CloudAICompanionProject, nil
	}
	return antigravityDefaultProject, nil
}

// ---------------------------------------------------------------------------
// Importing existing Antigravity CLI session
// ---------------------------------------------------------------------------

// ImportAntigravityCLI adopts an existing Antigravity CLI session.
func ImportAntigravityCLI(ctx context.Context) error {
	path := antigravityCLICredsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no Antigravity CLI session file found at %s — please use Browser Sign-in", path)
		}
		return err
	}

	var file struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiryDate   int64  `json:"expiry_date"`
		Project      string `json:"project"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("could not read Antigravity CLI session at %s: %w", path, err)
	}
	if file.AccessToken == "" && file.RefreshToken == "" {
		return fmt.Errorf("the Antigravity CLI session at %s is not signed in", path)
	}

	cred := Credential{
		Type:      "oauth",
		Access:    file.AccessToken,
		Refresh:   file.RefreshToken,
		ExpiresAt: file.ExpiryDate,
		Endpoint:  AntigravityBaseURL,
		Label:     "Google Antigravity (from CLI)",
		Account:   file.Project,
	}

	if cred.Expired() && cred.Refresh != "" {
		refreshed, err := refreshAntigravity(ctx, cred)
		if err == nil {
			cred = refreshed
		}
	}
	if email := antigravityEmail(ctx, cred.Access); email != "" {
		cred.Label = "Google Antigravity · " + email
	}
	if cred.Account == "" {
		project, err := resolveAntigravityProject(ctx, cred.Access)
		if err == nil && project != "" {
			cred.Account = project
		}
	}

	return Set("antigravity", cred)
}

// AntigravityCLIAvailable reports whether a local Antigravity CLI session exists to import.
func AntigravityCLIAvailable() bool {
	_, err := os.Stat(antigravityCLICredsPath())
	return err == nil
}

func antigravityCLICredsPath() string {
	if home := strings.TrimSpace(os.Getenv("ANTIGRAVITY_HOME")); home != "" {
		return filepath.Join(home, "auth.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".gemini", "antigravity-cli", "auth.json")
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "auth.json")
}
