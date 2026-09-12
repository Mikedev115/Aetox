package assetlib

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Stills are read with Go's own decoders, so a shelf of two thousand PNG
	// icons never starts two thousand processes.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/Mikedev115/Aetox/internal/proc"
)

// probeTimeout bounds one ffprobe. A file it cannot read in this long is a
// file it will not read, and the scan should move on rather than hang on it.
const probeTimeout = 20 * time.Second

// batchTimeout bounds one ffmpeg reading a whole batch of headers.
const batchTimeout = 2 * time.Minute

// FFTools reads sound and video with the ffmpeg pair every video job already
// depends on (desktop/videotooling.go keeps them beside each other because
// they ship together), and reads stills itself. Either path may be empty:
// with no ffmpeg the scan falls back to one ffprobe per file, and with
// neither the sound and clips are catalogued from their names alone.
//
// Found by the caller so this package needs no opinion about where tools live.
type FFTools struct {
	FFmpeg  string
	FFprobe string
}

// Probe reads one file.
func (t FFTools) Probe(ctx context.Context, path string) (Probe, error) {
	if mediaExt[strings.ToLower(filepath.Ext(path))] == "image" {
		return probeStill(path)
	}
	if strings.TrimSpace(t.FFprobe) == "" {
		return Probe{}, errors.New("ไม่มี ffprobe")
	}
	return probeFF(ctx, t.FFprobe, path)
}

// ProbeMany reads a batch: stills through the decoders, everything else
// through ONE ffmpeg fed every path as an input. ffmpeg with inputs and no
// output opens each, prints its header, and exits complaining that no output
// was named — which is exactly the work wanted and one process start for the
// lot (BatchProber says why that matters).
//
// A batch ffmpeg refuses as a whole — one input it cannot open aborts the
// run — falls back to single probes, so one broken file costs its batch a
// slower pass and not its answers.
func (t FFTools) ProbeMany(ctx context.Context, paths []string) ([]Probe, []error) {
	probes := make([]Probe, len(paths))
	errs := make([]error, len(paths))

	var media []int
	for i, p := range paths {
		if mediaExt[strings.ToLower(filepath.Ext(p))] == "image" {
			probes[i], errs[i] = probeStill(p)
			continue
		}
		media = append(media, i)
	}
	if len(media) == 0 {
		return probes, errs
	}
	if strings.TrimSpace(t.FFmpeg) != "" {
		in := make([]string, len(media))
		for j, i := range media {
			in[j] = paths[i]
		}
		if got, err := probeBatch(ctx, t.FFmpeg, in); err == nil {
			for j, i := range media {
				probes[i], errs[i] = got[j], nil
			}
			return probes, errs
		}
	}
	for _, i := range media {
		probes[i], errs[i] = t.Probe(ctx, paths[i])
	}
	return probes, errs
}

// FFProbe is the single-file reader kept for callers with only ffprobe.
func FFProbe(exe string) Prober { return FFTools{FFprobe: exe} }

// ffOut is the slice of ffprobe's JSON this package reads.
type ffOut struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		PixFmt    string `json:"pix_fmt"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func probeFF(ctx context.Context, exe, path string) (Probe, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe,
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type,width,height,pix_fmt",
		"-of", "json",
		path,
	)
	proc.HideConsole(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return Probe{}, fmt.Errorf("ffprobe: %s", msg)
	}
	var out ffOut
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return Probe{}, fmt.Errorf("ffprobe: %w", err)
	}
	var p Probe
	if d, err := strconv.ParseFloat(strings.TrimSpace(out.Format.Duration), 64); err == nil {
		p.Duration = d
	}
	for _, s := range out.Streams {
		if s.CodecType != "video" {
			continue
		}
		p.Width, p.Height = s.Width, s.Height
		p.HasAlpha = pixFmtHasAlpha(s.PixFmt)
		break
	}
	return p, nil
}

// probeBatch runs one ffmpeg over every path and reads its banner.
//
// The banner is stable text ffmpeg has printed for fifteen years:
//
//	Input #0, wav, from 'x.wav':
//	  Duration: 00:00:03.40, bitrate: 1411 kb/s
//	  Stream #0:0: Audio: pcm_s16le ...
//	Input #1, gif, from 'y.gif':
//	  Duration: 00:00:05.03, start: 0.000000, bitrate: 797 kb/s
//	  Stream #1:0: Video: gif, bgra, 400x400, 25 fps ...
//	At least one output file must be specified
//
// Inputs are matched by their number, not by path, so a path with a quote in
// it costs nothing. An input ffmpeg could not open makes it exit before the
// banner is complete, and that is the error the caller falls back on.
func probeBatch(ctx context.Context, exe string, paths []string) ([]Probe, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()
	args := []string{"-hide_banner", "-nostdin", "-v", "info"}
	for _, p := range paths {
		args = append(args, "-i", p)
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	proc.HideConsole(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run() // exits non-zero on purpose: no output was asked for
	out := parseBanner(stderr.String(), len(paths))
	if out == nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[len(msg)-300:]
		}
		return nil, fmt.Errorf("ffmpeg banner incomplete: %s", msg)
	}
	return out, nil
}

var (
	bannerInput    = regexp.MustCompile(`^Input #(\d+), `)
	bannerDuration = regexp.MustCompile(`^\s*Duration: (\d+):(\d+):(\d+)\.(\d+)`)
	bannerVideo    = regexp.MustCompile(`^\s*Stream #(\d+):\d+.*?: Video: (.*)$`)
	bannerSize     = regexp.MustCompile(`\b(\d{2,5})x(\d{2,5})\b`)
)

// parseBanner reads n inputs out of ffmpeg's stderr. Returns nil unless every
// one of the n inputs was seen — a banner short of one is a run that aborted.
func parseBanner(text string, n int) []Probe {
	out := make([]Probe, n)
	seen := make([]bool, n)
	cur := -1
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if m := bannerInput.FindStringSubmatch(line); m != nil {
			i, _ := strconv.Atoi(m[1])
			if i < 0 || i >= n {
				cur = -1
				continue
			}
			cur = i
			seen[i] = true
			continue
		}
		if cur < 0 {
			continue
		}
		if m := bannerDuration.FindStringSubmatch(line); m != nil {
			h, _ := strconv.Atoi(m[1])
			mi, _ := strconv.Atoi(m[2])
			s, _ := strconv.Atoi(m[3])
			frac, _ := strconv.ParseFloat("0."+m[4], 64)
			out[cur].Duration = float64(h*3600+mi*60+s) + frac
			continue
		}
		if m := bannerVideo.FindStringSubmatch(line); m != nil && out[cur].Width == 0 {
			// "gif, bgra, 400x400, 25 fps" — codec, then the pixel format
			// (sometimes with a parenthesis of colour details), then the size.
			fields := strings.Split(m[2], ", ")
			if len(fields) > 1 {
				pix := fields[1]
				if i := strings.Index(pix, "("); i >= 0 {
					pix = pix[:i]
				}
				out[cur].HasAlpha = pixFmtHasAlpha(strings.TrimSpace(pix))
			}
			if sm := bannerSize.FindStringSubmatch(m[2]); sm != nil {
				out[cur].Width, _ = strconv.Atoi(sm[1])
				out[cur].Height, _ = strconv.Atoi(sm[2])
			}
		}
	}
	for _, ok := range seen {
		if !ok {
			return nil
		}
	}
	return out
}

// pixFmtHasAlpha reads transparency off a pixel format name. The formats with
// an alpha plane all say so in the name — yuva420p, rgba, argb, bgra, gbrap,
// ya8 — and the opaque ones never do. An animated GIF reports pal8 or bgra
// depending on the demuxer, so it is read the honest way: from what the probe
// said, not from the extension.
func pixFmtHasAlpha(pixFmt string) bool {
	f := strings.ToLower(pixFmt)
	switch {
	case strings.HasPrefix(f, "yuva"), strings.HasPrefix(f, "gbrap"), strings.HasPrefix(f, "ya"):
		return true
	case strings.Contains(f, "rgba"), strings.Contains(f, "argb"), strings.Contains(f, "bgra"), strings.Contains(f, "abgr"):
		return true
	}
	return false
}

// probeStill reads a picture's size and whether its colour model can carry
// transparency. SVG and WebP have no decoder here and are catalogued from
// their names alone, which is enough to find them and not enough to size
// them; the asset says so by carrying no dimensions.
func probeStill(path string) (Probe, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".svg" {
		// Vector, so no size and — as a format — always able to be transparent.
		return Probe{HasAlpha: true}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Probe{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		if ext == ".webp" || ext == ".tif" || ext == ".tiff" || ext == ".bmp" {
			return Probe{}, nil // no decoder linked; not the file's fault
		}
		return Probe{}, err
	}
	return Probe{Width: cfg.Width, Height: cfg.Height, HasAlpha: modelHasAlpha(cfg.ColorModel)}, nil
}

func modelHasAlpha(m color.Model) bool {
	switch m {
	case color.RGBAModel, color.RGBA64Model, color.NRGBAModel, color.NRGBA64Model, color.AlphaModel, color.Alpha16Model:
		return true
	}
	// A palette with any transparent entry is an alpha picture; a GIF's
	// transparent index arrives this way.
	if p, ok := m.(color.Palette); ok {
		for _, c := range p {
			if _, _, _, a := c.RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}
