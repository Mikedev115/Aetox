package rpc

// A file from the screen's machine, put where the engine can read it
// (§248 phase 4, design doc §6 — "uploads through /file/attach").
//
// Every dialog on the screen answers with a path on the screen's machine,
// and every engine binding that takes a path reads it where the engine
// runs. With the engine a child of this process the two are one disk and
// nothing needs saying. With the engine on a host over ssh they are two
// machines, and a Windows path handed to a Linux engine is a file that does
// not exist. So the screen streams the bytes up first — PUT here, through
// the same tunnel and behind the same token as /rpc and /file/ — and the
// engine answers with where they landed: <DataRoot>/inbox/<id>/<name>, a
// path on ITS disk, which then goes to the binding the dialog was always
// going to call. The bindings do not learn about the wire; the door does.
//
// The inbox is the screen's to clear: DELETE /upload/<id> once the binding
// has taken what it needed. A screen that died between the two leaves a
// folder behind, and the server sweeps anything older than a day when it
// starts, so a host is never slowly filled by a window's crashes.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadPath is where the engine's listener takes a file from the screen.
const UploadPath = "/upload/"

// inboxDir is the folder under DataRoot the uploads land in.
const inboxDir = "inbox"

// inboxKeep is how long a forgotten upload lives before the sweep takes it.
const inboxKeep = 24 * time.Hour

// Uploaded is the engine's answer to a PUT: where the file is on its disk.
type Uploaded struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// upload is the handler: PUT /upload/?name=<file name> with the bytes as the
// body lands them; DELETE /upload/<id> removes that landing.
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.uploadPut(w, r)
	case http.MethodDelete:
		s.uploadDelete(w, r)
	default:
		http.Error(w, "PUT or DELETE", http.StatusMethodNotAllowed)
	}
}

func (s *Server) uploadPut(w http.ResponseWriter, r *http.Request) {
	name := safeUploadName(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "a file name is needed", http.StatusBadRequest)
		return
	}
	if s.dataRoot == "" {
		http.Error(w, "the engine has no data root", http.StatusInternalServerError)
		return
	}
	id, err := newUploadID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dir := filepath.Join(s.dataRoot, inboxDir, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dest := filepath.Join(dir, name)
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		os.RemoveAll(dir)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Streamed: a clip is gigabytes and the body is read once, as it comes
	// through the tunnel. A body that ends early — the tunnel dropped — is a
	// landing removed whole, never a half file with a whole file's name.
	n, err := io.Copy(f, r.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && r.ContentLength >= 0 && n != r.ContentLength {
		err = fmt.Errorf("the body ended after %d of %d bytes", n, r.ContentLength)
	}
	if err != nil {
		os.RemoveAll(dir)
		http.Error(w, "upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Uploaded{ID: id, Path: dest})
}

func (s *Server) uploadDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, UploadPath)
	if !validUploadID(id) || s.dataRoot == "" {
		http.Error(w, "no such upload", http.StatusNotFound)
		return
	}
	if err := os.RemoveAll(filepath.Join(s.dataRoot, inboxDir, id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// newUploadID is 8 random bytes as hex: the folder one upload lands in.
func newUploadID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// validUploadID is what newUploadID makes and nothing else — in particular
// nothing with a separator in it, since the id becomes a path segment.
func validUploadID(id string) bool {
	if len(id) != 16 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// safeUploadName keeps the file's own name — the extension is what every
// binding downstream decides by — and nothing of the folders it came from.
// A name that is only separators, or empty, is refused rather than guessed.
func safeUploadName(name string) string {
	name = strings.TrimSpace(name)
	// Both separators, whatever machine the name was typed on.
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if name == "" || name == "." || name == ".." {
		return ""
	}
	return name
}

// sweepInbox removes landings older than inboxKeep — a screen that never
// came back for them. Called once, when the server is built; errors are
// nobody's problem here, the next start tries again.
func sweepInbox(dataRoot string) {
	if dataRoot == "" {
		return
	}
	entries, err := os.ReadDir(filepath.Join(dataRoot, inboxDir))
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-inboxKeep)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		os.RemoveAll(filepath.Join(dataRoot, inboxDir, e.Name()))
	}
}

// ---------------------------------------------------------------- the screen's half

// Upload sends one file to the engine at endpoint and answers with where it
// landed there. size is the body's length, so the engine can tell a body
// that ended early from one that was short to begin with.
func Upload(ctx context.Context, endpoint Endpoint, name string, body io.Reader, size int64) (Uploaded, error) {
	network, address, token, ok := endpoint()
	if !ok {
		return Uploaded{}, errNoEngine
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://engine"+UploadPath+"?name="+url.QueryEscape(name), body)
	if err != nil {
		return Uploaded{}, err
	}
	req.ContentLength = size
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := uploadClient(network, address).Do(req)
	if err != nil {
		return Uploaded{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return Uploaded{}, fmt.Errorf("upload: %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	var up Uploaded
	if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
		return Uploaded{}, err
	}
	if up.Path == "" {
		return Uploaded{}, errors.New("upload: the engine answered without a path")
	}
	return up, nil
}

// Discard removes an upload the binding has taken what it needed from. Best
// effort by design: the sweep is behind it.
func Discard(ctx context.Context, endpoint Endpoint, id string) error {
	network, address, token, ok := endpoint()
	if !ok {
		return errNoEngine
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "http://engine"+UploadPath+id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := uploadClient(network, address).Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discard: %s", resp.Status)
	}
	return nil
}

// uploadClient dials the engine's address whatever host the URL names, the
// way FileProxy's transport does. No response-header timeout: the engine
// answers a PUT only after the last byte, and a clip through a tunnel takes
// as long as it takes — the caller's ctx is the bound.
func uploadClient(network, address string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		d.Timeout = 10 * time.Second
		return d.DialContext(ctx, network, address)
	}
	return &http.Client{Transport: transport}
}

// bodySize is the length of what an *os.File will stream, for Upload's size.
func bodySize(f *os.File) (int64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// UploadFile is Upload for a file on the screen's disk, named as it is there.
func UploadFile(ctx context.Context, endpoint Endpoint, path string) (Uploaded, error) {
	f, err := os.Open(path)
	if err != nil {
		return Uploaded{}, err
	}
	defer f.Close()
	size, err := bodySize(f)
	if err != nil {
		return Uploaded{}, err
	}
	return Upload(ctx, endpoint, filepath.Base(path), f, size)
}
