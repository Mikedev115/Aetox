package main

// The engine on another machine (§248 phase 3, design doc §5): the
// supervisor in engine_local.go pointed at a host, with remote.Driver
// doing the ssh. The window is the same window; the wire is the same
// wire; what changes is that spawn opens a tunnel instead of a child, and
// that the project, the database, the permissions and the memory are the
// host's (§2.4 — per host in v1, and the Settings page says so).
//
// The bindings here are the Settings page's "เครื่องระยะไกล" section:
// the hosts, connect, disconnect, the engine's log tail. They are the
// screen's own, not forwarded to the engine — a window that cannot reach
// its engine still has to be able to say "use this machine instead".

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/engine/remote"
	"github.com/Mikedev115/Aetox/internal/update"
	"github.com/Mikedev115/Aetox/internal/version"
)

// spawnRemote is spawn for a host: the whole road (remote.Driver.Connect),
// with each step on the status chip, and the tunnel as the process the
// supervisor watches. What Connect learned — the token above all — goes
// back into screen.json before the wire is dialed, so a window that dies
// right after can still reach the engine it started.
func (e *localEngine) spawnRemote(ctx context.Context, host remote.Host) (*engineProcess, error) {
	tun, err := e.driver.Connect(ctx, &host, version.Current, func(s remote.Step) {
		e.setStatus(engineStarting, stepWords(host, s))
	})
	if err != nil {
		return nil, err
	}
	if err := updateHost(host.Name, func(h *remote.Host) {
		h.Token, h.Version, h.Arch, h.LastUsed = host.Token, host.Version, host.Arch, host.LastUsed
	}); err != nil {
		debuglog.Msg("engine: screen.json: %v", err)
	}
	e.mu.Lock()
	if e.target.mode == modeRemote && e.target.host.Name == host.Name {
		e.target.host = host
	}
	e.mu.Unlock()
	e.token = host.Token
	return &engineProcess{
		network: "tcp",
		address: tun.Local,
		exited:  tun.Exited(),
		leave:   tun.Close,
		remote:  host.Label(),
	}, nil
}

// stepWords is the chip's sentence for a step of the road.
func stepWords(host remote.Host, s remote.Step) string {
	name := host.Label()
	switch s.Name {
	case "probe":
		return "กำลังติดต่อ " + name
	case "download":
		if s.Total > 0 {
			return fmt.Sprintf("กำลังดาวน์โหลดเครื่องยนต์สำหรับ Linux (%d%%)", s.Done*100/s.Total)
		}
		return "กำลังดาวน์โหลดเครื่องยนต์สำหรับ Linux"
	case "install":
		if s.Total > 0 {
			return fmt.Sprintf("กำลังส่งเครื่องยนต์ไปที่ %s (%d%%)", name, s.Done*100/s.Total)
		}
		return "กำลังส่งเครื่องยนต์ไปที่ " + name
	case "start":
		return "กำลังเริ่มเครื่องยนต์บน " + name
	case "tunnel":
		return "กำลังเปิดอุโมงค์ ssh ไป " + name
	}
	return s.Name
}

// engineBinaryFor is the engine to send to a Linux host of this
// architecture: a file named by AETOX_ENGINE_LINUX_DIR or lying beside
// this program (a development tree, a build of one's own), else the
// release's, fetched and verified by internal/update.
func engineBinaryFor(ctx context.Context, arch string, progress func(done, total int64)) (string, error) {
	name := update.EngineAssetName("linux", arch)
	if dir := strings.TrimSpace(os.Getenv("AETOX_ENGINE_LINUX_DIR")); dir != "" {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("AETOX_ENGINE_LINUX_DIR: ไม่มี %s ใน %s", name, dir)
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return update.FetchEngine(ctx, version.Current, "linux", arch, progress)
}

// reloadWindow starts the frontend over: after a switch of engine every
// store it holds is about the engine before. WindowReload — a plain
// location.reload() of whatever the webview shows — and not
// WindowReloadApp, which navigates to the start URL and left the window
// black on the first real switch (2026-09-12, wails dev on Windows).
func (a *App) reloadWindow() {
	if a.reload != nil {
		a.reload()
		return
	}
	if a.ctx != nil {
		wailsruntime.WindowReload(a.ctx)
	}
}

// RemoteHostView is one host as the Settings page sees it — no token.
type RemoteHostView struct {
	Name     string `json:"name"`
	Target   string `json:"target"`
	Root     string `json:"root"`
	Version  string `json:"version"`
	Arch     string `json:"arch"`
	LastUsed string `json:"lastUsed"`
}

// RemoteHostsView is the Settings page's picture: the hosts, which one the
// window is on, and whether ssh is there to reach any.
type RemoteHostsView struct {
	Active string           `json:"active"`
	Hosts  []RemoteHostView `json:"hosts"`
	// SSH is the ssh program that would be used, or empty with SSHError
	// saying why none.
	SSH      string `json:"ssh"`
	SSHError string `json:"sshError"`
	// Engine is where this build's engine for Linux would come from: a
	// file beside the program, or the release.
	Engine string `json:"engine"`
}

// RemoteHosts is the Settings page's binding.
func (a *App) RemoteHosts() RemoteHostsView {
	screenMu.Lock()
	c, err := loadScreenConfig()
	screenMu.Unlock()
	if err != nil {
		debuglog.Msg("engine: screen.json: %v", err)
	}
	v := RemoteHostsView{Active: c.ActiveHost, Hosts: []RemoteHostView{}}
	for _, h := range c.Hosts {
		row := RemoteHostView{Name: h.Name, Target: h.Target, Root: h.Root, Version: h.Version, Arch: h.Arch}
		if !h.LastUsed.IsZero() {
			row.LastUsed = h.LastUsed.Format(time.RFC3339)
		}
		v.Hosts = append(v.Hosts, row)
	}
	if a.engine != nil {
		if p, err := a.engine.driver.SSHPath(); err != nil {
			v.SSHError = err.Error()
		} else {
			v.SSH = p
		}
	}
	if exe, err := os.Executable(); err == nil {
		if _, err := os.Stat(filepath.Join(filepath.Dir(exe), update.EngineAssetName("linux", "amd64"))); err == nil {
			v.Engine = "beside"
		}
	}
	if v.Engine == "" {
		v.Engine = "release"
	}
	return v
}

// SaveRemoteHost adds or edits a host. The name is the key; a rename is a
// remove and an add.
func (a *App) SaveRemoteHost(name, target, root string) error {
	name, target, root = strings.TrimSpace(name), strings.TrimSpace(target), strings.TrimSpace(root)
	if err := remote.CheckTarget(target); err != nil {
		return err
	}
	if name == "" {
		name = target
	}
	if strings.ContainsAny(name, "/\\") {
		return errors.New("ชื่อเครื่องมี / ไม่ได้")
	}
	return updateHost(name, func(h *remote.Host) {
		if h.Target != target {
			// Another machine under the same name: the token was the old
			// machine's engine's.
			h.Token, h.Version, h.Arch = "", "", ""
		}
		h.Target, h.Root = target, root
	})
}

// ForgetRemoteHost removes a host from the list. The engine on it, if any,
// is left to its idle clock — forgetting a host is not reaching it.
func (a *App) ForgetRemoteHost(name string) error {
	screenMu.Lock()
	defer screenMu.Unlock()
	c, err := loadScreenConfig()
	if err != nil {
		return err
	}
	if c.ActiveHost == name {
		return errors.New("ตัดการเชื่อมต่อก่อนแล้วค่อยลบ")
	}
	c.remove(name)
	return saveScreenConfig(c)
}

// ConnectRemote points the window at a host. It returns at once; the road
// is walked by the supervisor and reported on engine:status, and the
// window reloads when the engine there answers.
func (a *App) ConnectRemote(name string) error {
	if a.engine == nil {
		return errors.New("no supervisor")
	}
	screenMu.Lock()
	c, err := loadScreenConfig()
	if err == nil {
		if _, ok := c.host(name); !ok {
			err = fmt.Errorf("ไม่มีเครื่องชื่อ %q", name)
		} else {
			c.ActiveHost = name
			err = saveScreenConfig(c)
		}
	}
	h, _ := c.host(name)
	screenMu.Unlock()
	if err != nil {
		return err
	}
	debuglog.Msg("engine: switching to host %s (%s)", h.Label(), h.Target)
	a.engine.retarget(engineTarget{mode: modeRemote, host: h})
	return nil
}

// DisconnectRemote brings the window back to this machine's engine. The
// engine on the host keeps running until its idle clock ends it.
func (a *App) DisconnectRemote() error {
	if a.engine == nil {
		return errors.New("no supervisor")
	}
	screenMu.Lock()
	c, err := loadScreenConfig()
	if err == nil && c.ActiveHost != "" {
		c.ActiveHost = ""
		err = saveScreenConfig(c)
	}
	screenMu.Unlock()
	if err != nil {
		return err
	}
	debuglog.Msg("engine: switching back to this machine")
	a.engine.retarget(engineTarget{mode: modeLocal})
	return nil
}

// StopRemoteEngine ends the engine on a host now rather than at its idle
// clock — the user's "stop it" on the Settings row. A window on that host
// comes back to this machine first.
func (a *App) StopRemoteEngine(name string) error {
	h, err := a.remoteHost(name)
	if err != nil {
		return err
	}
	if a.engine != nil {
		a.engine.mu.Lock()
		onIt := a.engine.target.mode == modeRemote && a.engine.target.host.Name == name
		a.engine.mu.Unlock()
		if onIt {
			if err := a.DisconnectRemote(); err != nil {
				return err
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.engine.driver.Stop(ctx, h); err != nil {
		return err
	}
	return updateHost(name, func(h *remote.Host) { h.Token = "" })
}

// RemoteEngineLog is the engine's stderr and newest log on the host, for
// the Settings row's "ดู log".
func (a *App) RemoteEngineLog(name string) (string, error) {
	h, err := a.remoteHost(name)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.engine.driver.Tail(ctx, h)
}

func (a *App) remoteHost(name string) (remote.Host, error) {
	if a.engine == nil {
		return remote.Host{}, errors.New("no supervisor")
	}
	screenMu.Lock()
	c, err := loadScreenConfig()
	screenMu.Unlock()
	if err != nil {
		return remote.Host{}, err
	}
	h, ok := c.host(name)
	if !ok {
		return remote.Host{}, fmt.Errorf("ไม่มีเครื่องชื่อ %q", name)
	}
	return h, nil
}
