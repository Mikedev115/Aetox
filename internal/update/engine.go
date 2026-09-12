package update

// The engine for another machine (§248 phase 3). A Windows screen that
// puts an engine on a Linux host needs the Linux build of cmd/aetox-engine
// for THIS version — a file that does not ship in the Windows package
// (decision 4: fetched on demand, not carried in every .msix) and cannot be
// pinned in internal/capability's manifest, whose hashes are written into
// the source before the release that would contain them is built.
//
// So it comes from the release this build was cut from, by tag, under the
// same trust as an update: checksums.txt signed by the release key says
// which bytes are ours, and only then does the hash say the download is
// those bytes. A cached copy is re-hashed against the same checksums on
// every use — it sat in a user-writable directory in between.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// EngineAssetName is the release asset for the engine on goos/goarch, the
// name release.yml gives it: aetox-engine-linux-amd64.
func EngineAssetName(goos, goarch string) string {
	return "aetox-engine-" + goos + "-" + goarch
}

// releaseDownloadBase is where a release's files are, by tag. A var for
// the package's tests, which serve a release of their own.
var releaseDownloadBase = "https://github.com/" + repoOwner + "/" + repoName + "/releases/download/"

// FetchEngine is the engine binary for goos/goarch of this version, on
// disk and verified, downloading it when it is not already here.
func FetchEngine(ctx context.Context, version, goos, goarch string, progress Progress) (string, error) {
	name := EngineAssetName(goos, goarch)
	dir, err := stagingDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "engine", version)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	sumsPath := filepath.Join(dir, checksumsName)

	// Already here: the checksums beside it were verified when they were
	// written, and the file is checked against them again now.
	if sums, err := os.ReadFile(sumsPath); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			if verifySHA256(path, name, sums) == nil {
				return path, nil
			}
		}
	}

	base := releaseDownloadBase + "v" + version + "/"
	sums, err := downloadSmall(ctx, base+checksumsName)
	if err != nil {
		return "", fmt.Errorf("ดาวน์โหลด %s ของรุ่น %s ไม่สำเร็จ: %w", checksumsName, version, err)
	}
	sig, err := downloadSmall(ctx, base+signatureName)
	if err != nil {
		return "", fmt.Errorf("ดาวน์โหลด %s ของรุ่น %s ไม่สำเร็จ: %w", signatureName, version, err)
	}
	// Origin first, then integrity — the order Stage uses, for its reason.
	if err := verifySignature(sums, sig); err != nil {
		return "", err
	}
	if _, ok := expectedSum(sums, name); !ok {
		return "", fmt.Errorf("รุ่น %s ไม่มี %s ให้ดาวน์โหลด — release นี้ยังไม่ได้แนบเครื่องยนต์สำหรับ Linux", version, name)
	}
	tmp := path + ".part"
	if err := download(ctx, base+name, tmp, progress); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("ดาวน์โหลด %s ไม่สำเร็จ: %w", name, err)
	}
	if err := verifySHA256(tmp, name, sums); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.WriteFile(sumsPath, sums, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
