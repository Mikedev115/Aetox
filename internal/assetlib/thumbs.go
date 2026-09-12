package assetlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/proc"
)

// Posters: a small still of every clip and picture on the shelf, rendered
// once and kept, so the browser can SHOW a shelf without asking the webview
// to open sixty files.
//
// Three things used to be one thing on the page: drawing the grid, deciding
// what a tile looks like, and fetching the file. The first draft gave every
// clip a <video> and let the webview sort it out — which meant a page of
// sixty tiles opened sixty files, decoded sixty first frames, and drew black
// for every one WebView2 has no decoder for (ProRes .mov, the whole of the
// CinePacks and FILM BURN folders). Owner, 12 ก.ย. 2569: *"มันควรแยกส่วนเลย
// แสดงผล เรนเดอร์ และ โหลดไฟล์"*.
//
// So: the grid draws from the catalogue alone (display). A poster is made by
// ffmpeg, which decodes everything, at 256px, once, into <DataRoot>/studio/
// thumbs (render). The real file is fetched only when someone hovers a tile
// whose format the webview can actually play (load).
//
// One ffmpeg per batch, for the reason ProbeMany gives: starting the program
// costs seconds on the owner's machine and the frame costs milliseconds.

// ThumbDir is where posters live under the data root.
const ThumbDir = "studio/thumbs"

// thumbSize is the poster's longer side. Tiles are ~150px wide on a 1x
// screen and twice that on a 2x one; 256 covers both without a 4K frame's
// worth of bytes behind it.
const thumbSize = 256

// thumbBatch is how many posters one ffmpeg renders. Each input is a
// separate decode, so this is smaller than the probe batch.
const thumbBatch = 16

// thumbTimeout bounds one batch. A 4K ProRes frame is a real decode; sixteen
// of them still finish in seconds, and a hang here is a hang the browser
// waits on.
const thumbTimeout = 90 * time.Second

// ThumbPath is where an asset's poster is, or would be. PNG for anything
// with transparency so the checkerboard behind it stays honest; JPEG for the
// rest, which is a quarter of the bytes.
func ThumbPath(dataRoot string, a Asset) string {
	ext := ".jpg"
	if a.HasAlpha {
		ext = ".png"
	}
	return filepath.Join(dataRoot, filepath.FromSlash(ThumbDir), a.ID+ext)
}

// HasPoster reports whether a kind gets a poster at all. Sounds do not.
func HasPoster(a Asset) bool {
	switch mediaExt[strings.ToLower(filepath.Ext(a.Path))] {
	case "video", "image":
		return true
	}
	return false
}

// Thumber renders posters. Owns the ffmpeg path, like FFTools.
type Thumber struct {
	FFmpeg   string
	DataRoot string
}

// Missing returns the assets among the given that have a poster kind and no
// poster on disk yet.
func (t Thumber) Missing(libs []*Library, ids []string) []Asset {
	var out []Asset
	for _, id := range ids {
		_, a, ok := Find(libs, id)
		if !ok || !HasPoster(*a) {
			continue
		}
		if _, err := os.Stat(ThumbPath(t.DataRoot, *a)); err == nil {
			continue
		}
		out = append(out, *a)
	}
	return out
}

// Render makes posters for the assets given, in batches, and returns the ids
// it managed to render. Assets whose poster already exists are skipped.
//
// A batch ffmpeg refuses as a whole falls back to one process per asset, so
// a single unreadable file costs its batch a slower pass rather than fifteen
// blank tiles. An asset that fails alone is left without a poster and the
// tile shows its kind's icon instead — silently, because a broken file on
// the user's shelf is not a fault of this program's.
func (t Thumber) Render(ctx context.Context, libs []*Library, assets []Asset) []string {
	if strings.TrimSpace(t.FFmpeg) == "" || len(assets) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(t.DataRoot, filepath.FromSlash(ThumbDir)), 0o755); err != nil {
		return nil
	}
	var done []string
	for start := 0; start < len(assets); start += thumbBatch {
		end := start + thumbBatch
		if end > len(assets) {
			end = len(assets)
		}
		batch := assets[start:end]
		if ctx.Err() != nil {
			break
		}
		if err := t.renderBatch(ctx, libs, batch); err == nil {
			for _, a := range batch {
				if _, err := os.Stat(ThumbPath(t.DataRoot, a)); err == nil {
					done = append(done, a.ID)
				}
			}
			continue
		}
		for _, a := range batch {
			if ctx.Err() != nil {
				break
			}
			if err := t.renderBatch(ctx, libs, []Asset{a}); err == nil {
				done = append(done, a.ID)
			}
		}
	}
	return done
}

// renderBatch runs one ffmpeg over the batch: every asset an input with its
// own seek, every poster an output with its own map and scale.
//
// The frame is taken a little way in rather than at zero — a clip's first
// frame is black or empty more often than not (a fade-in, a meme's title
// card) — at a third of the way through, capped at two seconds so a long
// background loop does not seek deep into a 4K file for its poster. Stills
// have no time axis and get no seek.
func (t Thumber) renderBatch(ctx context.Context, libs []*Library, batch []Asset) error {
	ctx, cancel := context.WithTimeout(ctx, thumbTimeout)
	defer cancel()
	args := []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y"}
	for _, a := range batch {
		lib, _, ok := Find(libs, a.ID)
		if !ok {
			return errors.New("asset not on any shelf")
		}
		if seek := posterSeek(a); seek > 0 {
			args = append(args, "-ss", strconv.FormatFloat(seek, 'f', 2, 64))
		}
		args = append(args, "-i", lib.FullPath(a))
	}
	for i, a := range batch {
		// scale keeps the aspect, longer side to thumbSize, never upscales
		// (a 64px icon stays 64px); -2 keeps even dimensions for the encoder.
		vf := fmt.Sprintf("scale='if(gt(iw,ih),min(iw,%d),-2)':'if(gt(iw,ih),-2,min(ih,%d))'", thumbSize, thumbSize)
		out := ThumbPath(t.DataRoot, a)
		args = append(args, "-map", fmt.Sprintf("%d:v:0", i), "-frames:v", "1", "-vf", vf)
		if !a.HasAlpha {
			args = append(args, "-q:v", "4")
		}
		args = append(args, out)
	}
	cmd := exec.CommandContext(ctx, t.FFmpeg, args...)
	proc.HideConsole(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[len(msg)-300:]
		}
		// Half-written posters from a failed run would be served as posters.
		for _, a := range batch {
			if st, err := os.Stat(ThumbPath(t.DataRoot, a)); err == nil && st.Size() == 0 {
				os.Remove(ThumbPath(t.DataRoot, a))
			}
		}
		return fmt.Errorf("ffmpeg: %s", msg)
	}
	return nil
}

func posterSeek(a Asset) float64 {
	if mediaExt[strings.ToLower(filepath.Ext(a.Path))] != "video" || a.Duration <= 0 {
		return 0
	}
	seek := a.Duration / 3
	if seek > 2 {
		seek = 2
	}
	// Under a quarter of a second there is nothing to seek past.
	if seek < 0.25 {
		return 0
	}
	return seek
}
