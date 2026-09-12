package main

// screen.json — the screen's own file under DataRoot (design doc §2.4):
// the hosts it can put an engine on and the token each running engine was
// given. Written by this process only, never by the engine, and wrapped
// with atrest like credentials.json: a token in it admits whoever holds it
// to an engine that runs shell commands on a host.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/atrest"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/engine/remote"
)

type screenConfig struct {
	// ActiveHost is the host the window last connected to by choice, and
	// starts on next time; empty is this machine.
	ActiveHost string        `json:"active_host,omitempty"`
	Hosts      []remote.Host `json:"hosts,omitempty"`
}

// screenMu serializes every load-modify-save, for the reason
// credentials.mu exists.
var screenMu sync.Mutex

func screenConfigPath() (string, error) {
	root, err := config.DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "screen.json"), nil
}

func loadScreenConfig() (screenConfig, error) {
	var c screenConfig
	path, err := screenConfigPath()
	if err != nil {
		return c, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(atrest.Unprotect(raw), &c); err != nil {
		return c, err
	}
	for _, h := range c.Hosts {
		if h.Token != "" {
			debuglog.Redact(h.Token)
		}
	}
	return c, nil
}

func saveScreenConfig(c screenConfig) error {
	path, err := screenConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	for _, h := range c.Hosts {
		if h.Token != "" {
			debuglog.Redact(h.Token)
		}
	}
	payload, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, atrest.Protect(payload), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (c *screenConfig) host(name string) (remote.Host, bool) {
	for _, h := range c.Hosts {
		if h.Name == name {
			return h, true
		}
	}
	return remote.Host{}, false
}

// put stores h under its name, replacing the entry of that name.
func (c *screenConfig) put(h remote.Host) {
	h.Name = strings.TrimSpace(h.Name)
	for i, old := range c.Hosts {
		if old.Name == h.Name {
			c.Hosts[i] = h
			return
		}
	}
	c.Hosts = append(c.Hosts, h)
}

func (c *screenConfig) remove(name string) {
	kept := c.Hosts[:0]
	for _, h := range c.Hosts {
		if h.Name != name {
			kept = append(kept, h)
		}
	}
	c.Hosts = kept
	if c.ActiveHost == name {
		c.ActiveHost = ""
	}
}

// updateHost is load, change one entry, save — under the lock.
func updateHost(name string, change func(h *remote.Host)) error {
	screenMu.Lock()
	defer screenMu.Unlock()
	c, err := loadScreenConfig()
	if err != nil {
		return err
	}
	h, ok := c.host(name)
	if !ok {
		h = remote.Host{Name: name}
	}
	change(&h)
	c.put(h)
	return saveScreenConfig(c)
}
