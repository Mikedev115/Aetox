package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// The role field of the agent editor has three roads in besides typing
// (DECISIONS §256.5, owner 13 ก.ย.: "เอาแบบเปิดไฟล์ + วางลิงก์แล้วดึง +
// เทมเพลต ลงเลย"): a file on this machine, a link to a file somebody keeps
// on GitHub or Google Drive, and a template. The first two are here; the
// templates are the frontend's (agentTemplates.ts), being words.
//
// Both return the TEXT and nothing else. The field is what the person edits
// and saves; the file or link is only where the words came from, and is not
// remembered — an agent whose brief lived at a URL would be an agent that
// changes when somebody else edits a document.

const (
	// briefMaxBytes caps what either road hands back. A brief is a page or
	// ten of prose; a megabyte of it is a mistake (the wrong file, a binary),
	// and the cap is what turns that into a sentence instead of a frozen form.
	briefMaxBytes = 512 << 10
	briefTimeout  = 20 * time.Second
)

// PickAgentBrief asks for a text file and returns its content. "" with no
// error when the picker was dismissed — cancelling is not a failure.
func (a *App) PickAgentBrief() (string, error) {
	p, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เลือกไฟล์บทบาทของเอเจน",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "ข้อความ (*.md, *.txt)", Pattern: "*.md;*.markdown;*.txt"},
			{DisplayName: "ทุกไฟล์", Pattern: "*.*"},
		},
	})
	if err != nil || strings.TrimSpace(p) == "" {
		return "", err
	}
	return readBriefFile(p)
}

func readBriefFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("เปิดไฟล์ไม่ได้: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, briefMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("อ่านไฟล์ไม่ได้: %w", err)
	}
	return briefText(raw, path.Base(p))
}

// FetchAgentBrief reads a brief from a link. It understands the links people
// actually paste — a GitHub file page, a gist, a Google Drive share, a Google
// Docs link — and takes any other http(s) URL as it is, as long as what
// comes back is text.
func (a *App) FetchAgentBrief(link string) (string, error) {
	ctx, cancel := context.WithTimeout(a.ctx, briefTimeout)
	defer cancel()
	return fetchBrief(ctx, http.DefaultClient, link)
}

func fetchBrief(ctx context.Context, client *http.Client, link string) (string, error) {
	u, err := briefURL(link)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("ลิงก์ใช้ไม่ได้: %w", err)
	}
	req.Header.Set("User-Agent", "Aetox")
	req.Header.Set("Accept", "text/plain, text/markdown, text/*;q=0.9, */*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ดึงไม่ได้: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return "", errors.New("ดึงไม่ได้ (404) — ตรวจว่าลิงก์ถูก และไฟล์เปิดให้ทุกคนที่มีลิงก์ดูได้")
		case http.StatusForbidden, http.StatusUnauthorized:
			return "", fmt.Errorf("ดึงไม่ได้ (%d) — ไฟล์นี้ไม่ได้เปิดสาธารณะ ตั้งแชร์เป็น ทุกคนที่มีลิงก์ แล้วลองใหม่", resp.StatusCode)
		}
		return "", fmt.Errorf("ดึงไม่ได้ (%d)", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, briefMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("ดึงไม่ได้: %w", err)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/html") || looksLikeHTML(raw) {
		// Drive answers a sign-in or a virus-scan page in HTML; a GitHub
		// page pasted from the wrong tab is the repository's chrome. Neither
		// is the file.
		return "", errors.New("ลิงก์นี้ตอบกลับมาเป็นหน้าเว็บ ไม่ใช่ตัวไฟล์ — ใช้ลิงก์ของไฟล์ .md/.txt โดยตรง หรือลิงก์แชร์ของ Google Docs/Drive ที่เปิดให้ทุกคนที่มีลิงก์")
	}
	return briefText(raw, path.Base(u))
}

// briefURL turns the link somebody pasted into the one that answers with the
// file's bytes. Anything it does not recognise goes through as it is, so a
// plain raw URL needs no rule here.
func briefURL(link string) (string, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", errors.New("วางลิงก์ก่อน")
	}
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("ลิงก์ต้องขึ้นต้นด้วย http:// หรือ https://")
	}
	host := strings.ToLower(u.Host)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch host {
	case "github.com", "www.github.com":
		// /owner/repo/blob/ref/path → raw.githubusercontent.com/owner/repo/ref/path
		if len(parts) >= 5 && (parts[2] == "blob" || parts[2] == "raw") {
			return "https://raw.githubusercontent.com/" + strings.Join(append(parts[:2:2], parts[3:]...), "/"), nil
		}
	case "gist.github.com":
		// /user/id → /user/id/raw (GitHub redirects to the single file)
		if len(parts) >= 2 && !strings.HasSuffix(u.Path, "/raw") {
			return "https://gist.github.com/" + parts[0] + "/" + parts[1] + "/raw", nil
		}
	case "drive.google.com":
		// /file/d/<id>/view, /open?id=<id>, /uc?id=<id>
		id := ""
		if len(parts) >= 3 && parts[0] == "file" && parts[1] == "d" {
			id = parts[2]
		} else if q := u.Query().Get("id"); q != "" {
			id = q
		}
		if id != "" {
			return "https://drive.google.com/uc?export=download&id=" + url.QueryEscape(id), nil
		}
	case "docs.google.com":
		// /document/d/<id>/edit → the document as plain text
		if len(parts) >= 3 && parts[0] == "document" && parts[1] == "d" {
			return "https://docs.google.com/document/d/" + url.PathEscape(parts[2]) + "/export?format=txt", nil
		}
	}
	return u.String(), nil
}

func looksLikeHTML(raw []byte) bool {
	head := strings.ToLower(strings.TrimSpace(string(raw[:min(len(raw), 512)])))
	return strings.HasPrefix(head, "<!doctype html") || strings.HasPrefix(head, "<html")
}

// briefText is the one check both roads share: text, in UTF-8, under the cap.
func briefText(raw []byte, name string) (string, error) {
	if len(raw) > briefMaxBytes {
		return "", fmt.Errorf("%s ใหญ่เกิน %d KB — บทบาทควรเป็นข้อความสั้น ๆ ไม่ใช่ทั้งคู่มือ", name, briefMaxBytes>>10)
	}
	// A UTF-8 BOM is what Notepad writes; it is not part of the words.
	raw = []byte(strings.TrimPrefix(string(raw), "\xEF\xBB\xBF"))
	if !utf8.Valid(raw) || strings.ContainsRune(string(raw), 0) {
		return "", fmt.Errorf("%s ไม่ใช่ไฟล์ข้อความ — เลือกไฟล์ .md หรือ .txt", name)
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s ว่างเปล่า", name)
	}
	return strings.TrimRight(text, "\n") + "\n", nil
}
