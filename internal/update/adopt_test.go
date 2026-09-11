package update

// "Later" on the installer channel: the download the user did not restart into
// has to be there on the next launch, verified again, and gone the moment it
// is either installed or overtaken. And the hand-off's outcome has to reach the
// card in words — the waiter used to swallow it, and a declined UAC prompt
// looked exactly like the update never having happened.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stageInstallerFixture writes what Stage leaves behind on the installer
// channel — the file, checksums.txt signed with a throwaway key, and the
// manifest — straight into the staging dir, and returns the restore func for
// the key.
func stageInstallerFixture(t *testing.T, version string, body []byte) func() {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	orig := releasePublicKey
	releasePublicKey = base64.StdEncoding.EncodeToString(pub)

	dir, err := stagingDir()
	if err != nil {
		t.Fatal(err)
	}
	const name = "aetox-amd64-installer.exe"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	sums := []byte(hex.EncodeToString(sum[:]) + "  " + name + "\n")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sums)) + "\n")
	staged := Staged{Version: version, Channel: ChannelInstaller, installer: path}
	if err := writeManifest(dir, staged, sums, sig); err != nil {
		t.Fatal(err)
	}
	return func() { releasePublicKey = orig }
}

func TestAdoptPicksUpAVerifiedInstallerFromThePreviousRun(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	defer stageInstallerFixture(t, "0.9.7", []byte("installer bytes"))()

	s, ok := adoptOn("0.9.6", ChannelInstaller)
	if !ok {
		t.Fatal("a staged, signed, matching installer was not adopted")
	}
	if s.Version != "0.9.7" || s.Channel != ChannelInstaller {
		t.Errorf("adopted %+v", s)
	}
	if !strings.HasSuffix(s.installer, "aetox-amd64-installer.exe") {
		t.Errorf("installer path = %q, want the staged file", s.installer)
	}
	if !s.Ready() {
		t.Error("adopted update reports not ready")
	}
}

// The morning after a successful update: the running build IS the staged
// version, so there is nothing to adopt and RemoveLeftovers may sweep.
func TestAdoptIgnoresAnInstallerThatIsNoLongerNewer(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	defer stageInstallerFixture(t, "0.9.7", []byte("installer bytes"))()

	if _, ok := adoptOn("0.9.7", ChannelInstaller); ok {
		t.Error("adopted the version already running")
	}
	if _, ok := adoptOn("0.9.8", ChannelInstaller); ok {
		t.Error("adopted a version older than the one running")
	}
}

// The file sat in a user-writable directory between runs. The same chain a
// fresh download passes — signature over checksums, checksum over file — has
// to pass again, or the elevated hand-off would run whatever is there now.
func TestAdoptRefusesATamperedInstaller(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	defer stageInstallerFixture(t, "0.9.7", []byte("installer bytes"))()

	dir, _ := stagingDir()
	if err := os.WriteFile(filepath.Join(dir, "aetox-amd64-installer.exe"), []byte("something else"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := adoptOn("0.9.6", ChannelInstaller); ok {
		t.Error("adopted a file whose hash no longer matches the signed checksums")
	}
}

// Only the installer channel stages a file for later; a portable install
// swapped its exe in place. And a data root shared between two copies of the
// app must not hand a portable copy the installer's download.
func TestAdoptIsInstallerChannelOnly(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	defer stageInstallerFixture(t, "0.9.7", []byte("installer bytes"))()

	for _, ch := range []Channel{ChannelPortable, ChannelScoop, ChannelStore, ChannelUnknown} {
		if _, ok := adoptOn("0.9.6", ch); ok {
			t.Errorf("%s adopted an installer", ch)
		}
	}
}

func TestAdoptWithNothingStagedIsQuiet(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if _, ok := adoptOn("0.9.6", ChannelInstaller); ok {
		t.Error("adopted from an empty data root")
	}
}

// The sweep is what used to delete a staged download on every launch — which
// is why "close the app and open it later" ended in the same download again.
func TestRemoveLeftoversKeepsAStagedInstallerOnlyWhenAsked(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	defer stageInstallerFixture(t, "0.9.7", []byte("installer bytes"))()
	dir, _ := stagingDir()

	RemoveLeftovers(true)
	if _, err := os.Stat(filepath.Join(dir, manifestName)); err != nil {
		t.Fatalf("keepStaged swept the staging dir: %v", err)
	}
	if _, ok := adoptOn("0.9.6", ChannelInstaller); !ok {
		t.Error("staged installer no longer adoptable after a keeping sweep")
	}

	RemoveLeftovers(false)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("staging dir still there after a full sweep: %v", err)
	}
}

// The waiter's one line, read once and gone: a failure from last week must not
// be shown under an update that has since installed fine.
func TestRestartLogIsReadOnce(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", root)

	if got := ReadRestartLog(); got != "" {
		t.Errorf("ReadRestartLog() = %q with no log, want empty", got)
	}
	path, _ := restartLogPath()
	if err := os.WriteFile(path, []byte("exit=2\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ReadRestartLog(); got != "exit=2" {
		t.Errorf("ReadRestartLog() = %q, want exit=2", got)
	}
	if got := ReadRestartLog(); got != "" {
		t.Errorf("second read = %q, want empty — the line is consumed", got)
	}
}

func TestInstallFailureWordsTheOutcome(t *testing.T) {
	cases := []struct {
		line string
		want string // a substring, or "" for "no failure"
	}{
		{"", ""},
		{"exit=0", ""},
		{"exit=2", "รหัส 2"},
		{"error=This command cannot be run due to the error: The operation was canceled by the user.", "UAC"},
		{"error=The system cannot find the file specified.", "cannot find the file"},
		{"something the waiter never writes", ""},
	}
	for _, c := range cases {
		got := InstallFailure(c.line)
		if c.want == "" && got != "" {
			t.Errorf("InstallFailure(%q) = %q, want no failure", c.line, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("InstallFailure(%q) = %q, want it to mention %q", c.line, got, c.want)
		}
	}
}

// A connection that goes quiet mid-file used to leave the card at "กำลัง
// ดาวน์โหลด" forever, with Stage stuck in Read. The guard is what ends it.
func TestDownloadGivesUpWhenTheBytesStop(t *testing.T) {
	orig := stallTimeout
	stallTimeout = 200 * time.Millisecond
	defer func() { stallTimeout = orig }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		fmt.Fprint(w, "a few bytes")
		w.(http.Flusher).Flush()
		<-r.Context().Done() // then nothing, until the client gives up
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "file")
	start := time.Now()
	err := download(context.Background(), srv.URL, dest, nil)
	if err == nil {
		t.Fatal("download returned nil from a stream that never ended")
	}
	if !strings.Contains(err.Error(), "ไม่ได้รับข้อมูล") {
		t.Errorf("err = %v, want the stall sentence", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("took %s to give up — the guard did not fire", time.Since(start))
	}
}
