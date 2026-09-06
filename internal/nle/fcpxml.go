package nle

// Aetox writes no editing engine, and this package is not one. It is a
// document writer: a cut list in, a file another program opens out — the same
// job internal/ooxml does for a Word document, aimed at a different room.
//
// **Why it exists.** The editor agent could already cut and render, and what
// came back was an mp4: the answer to "what did you decide" in the one form
// nobody can argue with. Move one cut and the whole thing is rebuilt from a
// sentence, minutes at a time. kinocut's Cutfile fixed half of that — a text
// file a person can edit and re-render — and this fixes the other half, for
// the person whose editor is DaVinci Resolve rather than a text editor.
//
// **Why FCPXML and not the obvious alternatives.** kinocut ships
// video_otio_export, and it is not this: read multipliers/otio_io.py and its
// clips carry no media reference and no source range, so nothing can open the
// result — it is the Timeline IR wearing OTIO's name, for round-tripping back
// into kinocut. CapCut's draft_content.json is closed and reverse-engineered,
// and breaks on their release schedule. EDL carries cuts and nothing else.
// FCPXML is documented, is plain XML, and is imported by Resolve (free),
// Premiere Pro and Final Cut — which is the whole point of handing someone a
// project instead of a video.
//
// **Every time in this file is exact.** FCPXML states time as a rational
// number of seconds, and an importer is entitled to reject a value that is not
// a whole number of frames. So seconds never survive into the output: they are
// converted to a frame count once, at the edge, and everything after that is
// integer arithmetic over one denominator. A cut half a frame from where the
// agent asked for it is a cut on the wrong side of a word, and floating point
// is how that happens silently.

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"strings"
)

// Rate is a frame rate as the exact rational ffprobe reports it — 30/1,
// 30000/1001 — never a float. 29.97 is not a number; 30000/1001 is.
type Rate struct {
	Num int
	Den int
}

// Valid reports whether the rate can be divided by at all.
func (r Rate) Valid() bool { return r.Num > 0 && r.Den > 0 }

// FPS is for reports and messages only. Nothing in the output is derived from
// it.
func (r Rate) FPS() float64 {
	if !r.Valid() {
		return 0
	}
	return float64(r.Num) / float64(r.Den)
}

// frameDuration is one frame's length in seconds, as the numerator and
// denominator FCPXML writes it with.
//
// Scaled up until the denominator reaches four figures, because that is the
// shape Final Cut's own exports use (100/3000s for 30fps, not the equal and
// equally legal 1/30s) and an importer that special-cases what it has seen
// before should be handed what it has seen before.
func (r Rate) frameDuration() (num, den int) {
	num, den = r.Den, r.Num
	for den < 1000 {
		num *= 10
		den *= 10
	}
	return num, den
}

// frames converts seconds to a whole frame count, once, at the edge.
func (r Rate) frames(seconds float64) int64 {
	if !r.Valid() || seconds <= 0 {
		return 0
	}
	return int64(math.Round(seconds * r.FPS()))
}

// Source is one file on this machine that clips are taken from.
type Source struct {
	ID            string
	Name          string
	Path          string // absolute, on this machine
	Seconds       float64
	Width         int
	Height        int
	Rate          Rate
	HasAudio      bool
	AudioChannels int
}

// Clip is one piece of a Source, in the order it plays.
//
// In and Out are source time, not timeline time — where the piece is taken
// from, not where it lands. Where it lands is the sum of everything before it,
// which is the spine's business and is computed here rather than asked for.
type Clip struct {
	Source string // Source.ID
	Name   string
	In     float64
	Out    float64
}

// Timeline is one sequence: the sources it draws on and the clips in order.
type Timeline struct {
	Name    string
	Event   string
	Width   int
	Height  int
	Rate    Rate
	Sources []Source
	Clips   []Clip
}

// PlacedClip is one clip after the frame arithmetic, which is the only form of
// it that is exactly true. Callers report these rather than the seconds they
// asked for: a cut lands on a frame, and saying otherwise to a user who then
// measures it is a small lie that costs a bug report.
type PlacedClip struct {
	Name      string
	Source    string
	InFrames  int64
	LenFrames int64
	AtFrames  int64
}

// Seconds renders a frame count back, for a report a person reads.
func (t Timeline) Seconds(frames int64) float64 {
	if !t.Rate.Valid() {
		return 0
	}
	return float64(frames) / t.Rate.FPS()
}

// Place runs the frame arithmetic without writing anything, so a caller can
// report what it is about to write, and so the numbers in the report and the
// numbers in the file cannot drift apart.
func (t Timeline) Place() ([]PlacedClip, error) {
	if err := t.validate(); err != nil {
		return nil, err
	}
	placed := make([]PlacedClip, 0, len(t.Clips))
	var at int64
	for i, c := range t.Clips {
		src := t.source(c.Source)
		in := src.Rate.frames(c.In)
		length := src.Rate.frames(c.Out) - in
		if length <= 0 {
			return nil, fmt.Errorf("clip %d (%s) is shorter than one frame: %.3f to %.3f at %g fps",
				i+1, c.Name, c.In, c.Out, src.Rate.FPS())
		}
		placed = append(placed, PlacedClip{
			Name: c.Name, Source: c.Source,
			InFrames: in, LenFrames: length, AtFrames: at,
		})
		at += length
	}
	return placed, nil
}

func (t Timeline) source(id string) Source {
	for _, s := range t.Sources {
		if s.ID == id {
			return s
		}
	}
	return Source{}
}

func (t Timeline) validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("timeline has no name")
	}
	if !t.Rate.Valid() {
		return fmt.Errorf("timeline %q has no frame rate", t.Name)
	}
	if t.Width <= 0 || t.Height <= 0 {
		return fmt.Errorf("timeline %q has no frame size", t.Name)
	}
	if len(t.Clips) == 0 {
		return fmt.Errorf("timeline %q has no clips, which is not an edit", t.Name)
	}
	seen := map[string]bool{}
	for _, s := range t.Sources {
		if s.ID == "" {
			return fmt.Errorf("a source has no id")
		}
		if seen[s.ID] {
			return fmt.Errorf("two sources share the id %q", s.ID)
		}
		seen[s.ID] = true
		if !s.Rate.Valid() {
			return fmt.Errorf("source %q has no frame rate", s.ID)
		}
		if !filepath.IsAbs(s.Path) {
			return fmt.Errorf("source %q path is not absolute: %s", s.ID, s.Path)
		}
	}
	for i, c := range t.Clips {
		if !seen[c.Source] {
			return fmt.Errorf("clip %d refers to source %q, which is not declared", i+1, c.Source)
		}
		if c.Out <= c.In {
			return fmt.Errorf("clip %d (%s) ends at or before it starts: %.3f to %.3f", i+1, c.Name, c.In, c.Out)
		}
	}
	return nil
}

// fileURL is the address FCPXML gives a media file. Windows is the case that
// makes this a function: a path with a drive letter and a space has to arrive
// percent-encoded behind three slashes, and a string joined by hand gets the
// space wrong every time.
func fileURL(abs string) string {
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + (&url.URL{Path: p}).EscapedPath()
}

// FCPXML renders the timeline as a Final Cut Pro X XML document.
func (t Timeline) FCPXML() ([]byte, error) {
	placed, err := t.Place()
	if err != nil {
		return nil, err
	}
	fdNum, fdDen := t.Rate.frameDuration()
	// One time renderer for the whole document, closed over the sequence's
	// timebase: two denominators in one file is how a clip lands a frame out.
	at := func(frames int64) string {
		if frames == 0 {
			return "0s"
		}
		return fmt.Sprintf("%d/%ds", frames*int64(fdNum), fdDen)
	}

	doc := fcpxml{Version: "1.10"}
	const seqFormat = "r1"
	doc.Resources.Formats = append(doc.Resources.Formats, xformat{
		ID:            seqFormat,
		Name:          fmt.Sprintf("AetoxVideoFormat%dp", t.Height),
		FrameDuration: fmt.Sprintf("%d/%ds", fdNum, fdDen),
		Width:         t.Width,
		Height:        t.Height,
		ColorSpace:    "1-1-1 (Rec. 709)",
	})

	// A source that is not the sequence's shape gets a format of its own —
	// declaring a 4K phone clip as 1080p is how an importer decides to scale
	// something nobody asked it to scale.
	formatIDs := map[string]string{}
	assetIDs := map[string]string{}
	next := 2
	used := map[string]bool{}
	for _, c := range t.Clips {
		used[c.Source] = true
	}
	for _, s := range t.Sources {
		if !used[s.ID] {
			continue
		}
		sfNum, sfDen := s.Rate.frameDuration()
		key := fmt.Sprintf("%dx%d@%d/%d", s.Width, s.Height, sfNum, sfDen)
		id, ok := formatIDs[key]
		if !ok {
			if s.Width == t.Width && s.Height == t.Height && sfNum == fdNum && sfDen == fdDen {
				id = seqFormat
			} else {
				id = fmt.Sprintf("r%d", next)
				next++
				doc.Resources.Formats = append(doc.Resources.Formats, xformat{
					ID:            id,
					Name:          fmt.Sprintf("AetoxSourceFormat%dp", s.Height),
					FrameDuration: fmt.Sprintf("%d/%ds", sfNum, sfDen),
					Width:         s.Width,
					Height:        s.Height,
					ColorSpace:    "1-1-1 (Rec. 709)",
				})
			}
			formatIDs[key] = id
		}
		assetID := fmt.Sprintf("r%d", next)
		next++
		assetIDs[s.ID] = assetID
		a := xasset{
			ID:       assetID,
			Name:     s.Name,
			Start:    "0s",
			Duration: fmt.Sprintf("%d/%ds", s.Rate.frames(s.Seconds)*int64(sfNum), sfDen),
			HasVideo: "1",
			Format:   id,
			MediaRep: xmediaRep{Kind: "original-media", Src: fileURL(s.Path)},
		}
		if s.HasAudio {
			a.HasAudio = "1"
			a.AudioSources = "1"
			channels := s.AudioChannels
			if channels <= 0 {
				channels = 2
			}
			a.AudioChannels = fmt.Sprintf("%d", channels)
		}
		doc.Resources.Assets = append(doc.Resources.Assets, a)
	}

	var total int64
	for _, p := range placed {
		total = p.AtFrames + p.LenFrames
	}
	spine := xspine{}
	for _, p := range placed {
		spine.Clips = append(spine.Clips, xassetClip{
			Ref:      assetIDs[p.Source],
			Name:     p.Name,
			Offset:   at(p.AtFrames),
			Start:    at(p.InFrames),
			Duration: at(p.LenFrames),
			TCFormat: "NDF",
		})
	}
	event := t.Event
	if strings.TrimSpace(event) == "" {
		event = "Aetox"
	}
	doc.Library.Event = xevent{
		Name: event,
		Project: xproject{
			Name: t.Name,
			Sequence: xsequence{
				Format:      seqFormat,
				Duration:    at(total),
				TCStart:     "0s",
				TCFormat:    "NDF",
				AudioLayout: "stereo",
				AudioRate:   "48k",
				Spine:       spine,
			},
		},
	}

	body, err := xml.MarshalIndent(doc, "", "    ")
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(xml.Header)
	out.WriteString("<!DOCTYPE fcpxml>\n")
	out.Write(body)
	out.WriteString("\n")
	return out.Bytes(), nil
}

type fcpxml struct {
	XMLName   xml.Name   `xml:"fcpxml"`
	Version   string     `xml:"version,attr"`
	Resources xresources `xml:"resources"`
	Library   xlibrary   `xml:"library"`
}

type xresources struct {
	Formats []xformat `xml:"format"`
	Assets  []xasset  `xml:"asset"`
}

type xformat struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr,omitempty"`
	FrameDuration string `xml:"frameDuration,attr"`
	Width         int    `xml:"width,attr"`
	Height        int    `xml:"height,attr"`
	ColorSpace    string `xml:"colorSpace,attr,omitempty"`
}

type xasset struct {
	ID            string    `xml:"id,attr"`
	Name          string    `xml:"name,attr"`
	Start         string    `xml:"start,attr"`
	Duration      string    `xml:"duration,attr"`
	HasVideo      string    `xml:"hasVideo,attr,omitempty"`
	Format        string    `xml:"format,attr,omitempty"`
	HasAudio      string    `xml:"hasAudio,attr,omitempty"`
	AudioSources  string    `xml:"audioSources,attr,omitempty"`
	AudioChannels string    `xml:"audioChannels,attr,omitempty"`
	MediaRep      xmediaRep `xml:"media-rep"`
}

type xmediaRep struct {
	Kind string `xml:"kind,attr"`
	Src  string `xml:"src,attr"`
}

type xlibrary struct {
	Event xevent `xml:"event"`
}

type xevent struct {
	Name    string   `xml:"name,attr"`
	Project xproject `xml:"project"`
}

type xproject struct {
	Name     string    `xml:"name,attr"`
	Sequence xsequence `xml:"sequence"`
}

type xsequence struct {
	Format      string `xml:"format,attr"`
	Duration    string `xml:"duration,attr"`
	TCStart     string `xml:"tcStart,attr"`
	TCFormat    string `xml:"tcFormat,attr"`
	AudioLayout string `xml:"audioLayout,attr,omitempty"`
	AudioRate   string `xml:"audioRate,attr,omitempty"`
	Spine       xspine `xml:"spine"`
}

type xspine struct {
	XMLName xml.Name     `xml:"spine"`
	Clips   []xassetClip `xml:"asset-clip"`
}

type xassetClip struct {
	Ref      string `xml:"ref,attr"`
	Name     string `xml:"name,attr"`
	Offset   string `xml:"offset,attr"`
	Start    string `xml:"start,attr"`
	Duration string `xml:"duration,attr"`
	TCFormat string `xml:"tcFormat,attr,omitempty"`
}
