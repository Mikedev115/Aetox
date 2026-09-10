package main

// The pull requests of the focused project, as a room.
//
// The `pr` tool (internal/skill/pr_pack.go) answers the model in sentences.
// This answers the window in rows, from the same fetcher (internal/github),
// because two fetchers is how two surfaces start disagreeing about the same
// pull request — the rule GitPane already follows for the working tree
// (DECISIONS §161.4).
//
// What the room adds that the tool cannot: seeing without asking. A CI result
// that is on screen while you work is a different thing from one you have to
// remember to go and request.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gh "github.com/Mikedev115/Aetox/internal/github"
	"github.com/Mikedev115/Aetox/internal/model"
)

// prClient is the room's client, and the one seam a test needs: prAPI and
// prHTTP are empty in every real build, which is exactly the zero PRClient
// pointing at github.com.
func (a *App) prClient() *gh.PRClient {
	return &gh.PRClient{BaseURL: a.prAPI, HTTP: a.prHTTP}
}

// maxRoomPRs is what one list shows. A repository with sixty open pull
// requests does not need all sixty drawn before anybody has scrolled.
const maxRoomPRs = 30

// PRSuggestion holds AI-generated title and description for opening a pull request.
type PRSuggestion struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// PRRoom is the whole answer for the pane: the list, and — when there is no
// list — why.
//
// One binding rather than a list plus a separate status call, because an empty
// list has four different meanings (no account, not a GitHub repo, no project,
// genuinely none open) and a pane that has to guess which will guess wrong. The
// reason travels with the emptiness that needs it.
type PRRoom struct {
	// Repo is owner/name, "" when there is none to name.
	Repo string `json:"repo"`
	// Reason is why the list is empty, "" when nothing is wrong. It is a
	// sentence for a person, already in the language the engine speaks.
	Reason string `json:"reason"`
	// Connected reports whether a GitHub account is attached at all, so the
	// pane can offer the way to attach one rather than only saying it is not.
	Connected bool             `json:"connected"`
	Items     []gh.PullRequest `json:"items"`
}

// PullRequests lists the open pull requests of the project this chat is focused on.
func (a *App) PullRequests() PRRoom {
	return a.PullRequestsState("open")
}

// PullRequestsState lists pull requests matching the requested state ("open", "closed", "all").
func (a *App) PullRequestsState(state string) PRRoom {
	room := PRRoom{Items: []gh.PullRequest{}, Connected: gh.Token() != ""} // never nil: §34
	repo, ok := a.roomRepo(&room)
	if !ok {
		return room
	}
	s := strings.TrimSpace(state)
	if s == "" {
		s = "open"
	}
	items, err := a.prClient().List(context.Background(), repo, s, maxRoomPRs)
	if err != nil {
		room.Reason = err.Error()
		return room
	}
	room.Items = items
	return room
}

// SuggestPRDetails uses AI to draft a PR title and description based on commits between base and head.
func (a *App) SuggestPRDetails(head, base string) (PRSuggestion, error) {
	root, ok := a.gitRoot()
	if !ok {
		return PRSuggestion{}, errors.New("no git repository focused")
	}

	ctx, cancel := a.gitContext()
	defer cancel()

	h := strings.TrimSpace(head)
	if h == "" {
		cur, _ := gitOut(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
		h = strings.TrimSpace(cur)
	}
	if h == "" {
		return PRSuggestion{}, errors.New("cannot determine current branch")
	}

	b := strings.TrimSpace(base)
	if b == "" {
		var room PRRoom
		if repo, ok := a.roomRepo(&room); ok {
			if def, err := a.prClient().DefaultBranch(ctx, repo); err == nil && def != "" {
				b = def
			}
		}
		if b == "" {
			b = "main"
		}
	}

	logOut, _ := gitOut(ctx, root, "log", b+".."+h, "--oneline")
	diffStat, _ := gitOut(ctx, root, "diff", "--stat", b+".."+h)
	diffHunk, _ := gitOut(ctx, root, "diff", b+".."+h)
	diffLines := strings.Split(diffHunk, "\n")
	if len(diffLines) > 60 {
		diffHunk = strings.Join(diffLines[:60], "\n") + "\n... (truncated)"
	}

	fallbackTitle := "Update " + h
	lines := strings.Split(strings.TrimSpace(logOut), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		parts := strings.SplitN(lines[0], " ", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			fallbackTitle = parts[1]
		}
	}
	fallbackBody := "## Changes\n" + strings.TrimSpace(logOut)

	p, modelName, err := a.oneShotProvider()
	if err != nil {
		return PRSuggestion{Title: fallbackTitle, Body: fallbackBody}, nil
	}

	prompt := fmt.Sprintf(`You are a GitHub Pull Request assistant.
Analyze the branch changes below and generate a Pull Request Title and Description.
Branch: %s -> %s
Commits:
%s

Diff Summary:
%s

Diff Preview:
%s

Format your response as a JSON object with two fields:
{
  "title": "A clear, concise Conventional Commit style title (e.g. feat(auth): add token refresh)",
  "body": "A structured Markdown description including ## Summary, ## Changes, and ## Testing"
}
Output ONLY raw JSON, with no code fences or explanations.`, h, b, strings.TrimSpace(logOut), strings.TrimSpace(diffStat), diffHunk)

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleUser, Content: prompt},
		},
		Temperature: 0.2,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return PRSuggestion{Title: fallbackTitle, Body: fallbackBody}, nil
	}

	text := strings.TrimSpace(resp.Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var sug PRSuggestion
	if err := json.Unmarshal([]byte(text), &sug); err == nil && sug.Title != "" {
		return sug, nil
	}

	// Fallback parsing if model returned plain text
	textLines := strings.Split(text, "\n")
	if len(textLines) > 0 && strings.TrimSpace(textLines[0]) != "" {
		sug.Title = strings.TrimSpace(textLines[0])
		if len(textLines) > 1 {
			sug.Body = strings.TrimSpace(strings.Join(textLines[1:], "\n"))
		}
		return sug, nil
	}

	return PRSuggestion{Title: fallbackTitle, Body: fallbackBody}, nil
}

// ReviewPullRequest uses AI to review the code changes and diffs of a pull request.
func (a *App) ReviewPullRequest(number int) (string, error) {
	if number <= 0 {
		return "", errors.New("invalid pull request number")
	}

	files := a.PullRequestFiles(number)
	if len(files) == 0 {
		return "", errors.New("no changed files found for this pull request")
	}

	var diffBuilder strings.Builder
	totalLines := 0
	for _, f := range files {
		if f.Patch == "" {
			continue
		}
		diffBuilder.WriteString(fmt.Sprintf("\n### File: %s (%s)\n```diff\n", f.Path, f.Status))
		lines := strings.Split(f.Patch, "\n")
		for _, l := range lines {
			diffBuilder.WriteString(l + "\n")
			totalLines++
			if totalLines > 120 {
				diffBuilder.WriteString("... (truncated for review)\n")
				break
			}
		}
		diffBuilder.WriteString("```\n")
		if totalLines > 120 {
			break
		}
	}

	p, modelName, err := a.oneShotProvider()
	if err != nil {
		return "", fmt.Errorf("provider unavailable: %w", err)
	}

	ctx, cancel := a.gitContext()
	defer cancel()

	prompt := fmt.Sprintf(`You are a senior software engineer conducting a code review for GitHub Pull Request #%d.
Review the code changes below and provide an actionable, constructive review in clean Markdown.
Changes:
%s

Please structure your review as follows:
### 📋 Overview & Highlights
Brief summary of what this PR accomplishes.

### 🔍 Potential Issues & Edge Cases
Any bugs, missing validations, unhandled edge cases, or potential regressions. (If none, note that everything looks clean).

### 💡 Suggestions & Code Quality
Any suggestions for readability, maintainability, or testing.

### 🏁 Recommendation
State one of: **LGTM (Looks Good to Me)**, **LGTM with Suggestions**, or **Changes Requested**.`, number, diffBuilder.String())

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleUser, Content: prompt},
		},
		Temperature: 0.2,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ai review failed: %w", err)
	}

	return strings.TrimSpace(resp.Text), nil
}

// PullRequestFiles is one pull request's files, each with GitHub's own unified
// diff. Fetched when a row is expanded, never before: a pull request of forty
// files is ordinary, and drawing none of them costs nothing.
func (a *App) PullRequestFiles(number int) []gh.PRFile {
	files := []gh.PRFile{} // never nil: §34
	var room PRRoom
	repo, ok := a.roomRepo(&room)
	if !ok || number <= 0 {
		return files
	}
	got, err := a.prClient().Files(context.Background(), repo, number, maxPRRoomFiles)
	if err != nil {
		return files
	}
	return got
}

// maxPRRoomFiles caps one expanded row. Past this the change is too big to read
// in a side panel, and the pull request's own page is the right place for it.
const maxPRRoomFiles = 60

// PullRequestChecks is the CI runs for one commit.
//
// By SHA rather than by pull request number, and the reason is the row it
// draws: the list already carries every head SHA, so a badge per row is one
// call each. Taking the number would mean fetching the pull request again just
// to learn a string the caller is already holding.
func (a *App) PullRequestChecks(sha string) []gh.CheckRun {
	runs := []gh.CheckRun{} // never nil: §34
	var room PRRoom
	repo, ok := a.roomRepo(&room)
	if !ok || strings.TrimSpace(sha) == "" {
		return runs
	}
	got, err := a.prClient().Checks(context.Background(), repo, sha)
	if err != nil {
		return runs
	}
	return got
}

// PRCreated is what opening one from the room answers with: the pull request,
// or the sentence explaining why there is none.
//
// An error rather than a Go error, because this one is READ by the pane and
// shown in the form the user is standing in. GitHub's refusals here are almost
// always actionable — "No commits between main and feature" means push first,
// "A pull request already exists" means it is already open — and a dialog that
// said "failed" would throw away the only useful part.
type PRCreated struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
	Base   string `json:"base"`
	Error  string `json:"error"`
}

// CreatePullRequest opens one for the focused project.
//
// No approval gate, and that is not an oversight: the gate exists for acts the
// MODEL performs (safety.AssessCommand marks `pr_create` high-risk for exactly
// that), and a person pressing a button in a form they filled in has already
// given the only approval there is. The same reason saving from the editor does
// not ask.
//
// base empty means the repository's own default branch, which internal/github
// looks up — the form leaves it blank far more often than it fills it in.
func (a *App) CreatePullRequest(title, head, base, body string, draft bool) PRCreated {
	var room PRRoom
	repo, ok := a.roomRepo(&room)
	if !ok {
		return PRCreated{Error: room.Reason}
	}
	created, err := a.prClient().Create(context.Background(), repo, gh.NewPR{
		Title: strings.TrimSpace(title),
		Head:  strings.TrimSpace(head),
		Base:  strings.TrimSpace(base),
		Body:  body,
		Draft: draft,
	})
	if err != nil {
		return PRCreated{Error: err.Error()}
	}
	return PRCreated{Number: created.Number, URL: created.URL, Base: created.BaseRef}
}

// roomRepo is the three checks every call here shares, in the order that gives
// the most useful answer: no account beats no repository, because connecting
// one is the thing to do either way.
func (a *App) roomRepo(room *PRRoom) (gh.Repo, bool) {
	if gh.Token() == "" {
		room.Reason = "no GitHub account is connected"
		return gh.Repo{}, false
	}
	root := a.cur().cfg.SandboxRoot
	if !a.projectFocused || root == "" {
		room.Reason = "no project is open"
		return gh.Repo{}, false
	}
	repo, err := gh.OriginRepo(root)
	if err != nil {
		room.Reason = err.Error()
		return gh.Repo{}, false
	}
	room.Repo = repo.String()
	return repo, true
}
