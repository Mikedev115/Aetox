package main

// The Settings page's picture section, and the one tool it configures:
// `image_make` (internal/skill/image_make.go).
//
// The exact shape of voice.go's top half, deliberately — internal/imagegen
// holds a catalog and one interface, and these bindings are the wiring:
// enumerate the catalog for the vendor picker, persist the picks, and answer
// whether the picked vendor can actually run right now. Nothing here is engine
// work, and nothing here knows a vendor's name.
//
// The one piece of judgement that lives here rather than in internal/imagegen
// is which MODELS to offer, and it is here for the same reason the voice
// page's language default is: this file can see the installed model catalog and
// the package deliberately cannot. See imageModelNames.

import (
	"fmt"
	"strings"

	"github.com/Mikedev115/Aetox/internal/imagegen"
	"github.com/Mikedev115/Aetox/internal/model"
)

// imageOptions is the one translation from config to internal/imagegen options,
// so the bindings and the engine bootstrap cannot drift.
func (a *App) imageOptions() imagegen.Options {
	cfg := a.cur().cfg
	return imagegen.Options{
		Engine: strings.TrimSpace(cfg.ImageEngine),
		Model:  strings.TrimSpace(cfg.ImageModelName),
	}
}

// ListImageEngines enumerates the picture vendors the catalog knows, in the
// same row shape the two voice pickers use. Never nil (ARCHITECTURE.md §34).
func (a *App) ListImageEngines() []VoiceEngineInfo {
	cfg := a.cur().cfg
	return engineRows(imageDescriptors(), cfg.ImageEngine, cfg.ImageModelName)
}

// SetImageEngine picks the picture vendor. Empty goes back to the catalog
// default. The model pick is cleared with it, the same rule the voice page
// follows: a model name is one vendor's private vocabulary, and keeping it
// across a switch would pin the new vendor to a name it has never heard of.
func (a *App) SetImageEngine(id string) error {
	id = strings.TrimSpace(id)
	if _, ok := imagegen.Lookup(id); !ok {
		return fmt.Errorf("ไม่รู้จัก engine สร้างภาพชื่อ %q", id)
	}
	next := a.cfg
	next.ImageEngine = id
	next.ImageModelName = ""
	a.applyConfig(a.cur(), next)
	return nil
}

// SetImageModelName pins a NAMED model on the active vendor. Empty is a real
// choice — back to the vendor's first.
func (a *App) SetImageModelName(name string) error {
	name = strings.TrimSpace(name)
	if err := validateNamedModel(imageDescriptors(), a.cur().cfg.ImageEngine, name); err != nil {
		return err
	}
	next := a.cfg
	next.ImageModelName = name
	a.applyConfig(a.cur(), next)
	return nil
}

// ImageStatus is the picked vendor's own reason it cannot run right now, in the
// user's language, or "" when it is ready. Built by asking internal/imagegen to
// construct the engine and reporting what it says — a missing API key is the
// answer this exists for, and the package already words it as an instruction
// ("ใส่ได้ที่ ตั้งค่า > โมเดล > OpenAI") rather than as a fault.
//
// Constructing an engine reaches nothing: no key check is a network call, and
// no engine here opens a connection until Generate is called.
func (a *App) ImageStatus() string {
	if _, err := imagegen.New(a.imageOptions()); err != nil {
		return err.Error()
	}
	return ""
}

// imageDescriptors flattens internal/imagegen's catalog into the shape
// engineRows renders — the same flattening voice.go does for the two speech
// catalogs, and here for the same reason: the packages do not import each other
// and none of them should know what a settings row looks like.
func imageDescriptors() []descriptorRow {
	descs := imagegen.Catalog()
	out := make([]descriptorRow, 0, len(descs))
	for _, d := range descs {
		out = append(out, descriptorRow{
			id:         d.ID,
			label:      d.Label,
			install:    d.Install,
			cmd:        d.InstallCommand,
			modelNames: imageModelNames(d),
			def:        d.Default,
			// No picture vendor reads a model FILE off the disk. HasModels is
			// the speech page's question about a file picker, and answering it
			// true here would draw a control over nothing.
			models: false,
		})
	}
	return out
}

// imageModelNames is the roster the picker offers for one vendor: the curated
// list the descriptor carries, then whatever else the installed model catalog
// reports as producing a picture for that same provider.
//
// **Curated first, and that is the whole decision.** The catalog's list is the
// current one — it knows names this build has never heard of, which is the
// point of consulting it — but it is a third party's index of what a provider
// SERVES, not of what this app's wire format can DRIVE. An image model filed
// under `openai` that only answers on a different endpoint would otherwise
// become the default simply by sorting first. So the descriptor's own list
// decides the default and the catalog's extras are offered after it, where
// picking one is the user's deliberate act.
//
// A vendor the catalog has never heard of, or a machine that has no catalog at
// all, keeps exactly the curated list and loses nothing.
func imageModelNames(d imagegen.Descriptor) []string {
	out := append([]string{}, d.Models...)
	seen := map[string]bool{}
	for _, m := range out {
		seen[strings.ToLower(m)] = true
	}
	// Keyed by the imagegen row id, which is deliberately the same string the
	// provider catalog uses ("openai", "xai", "gemini") — that shared id is
	// what lets one key and one base URL serve both sides.
	for _, m := range model.ImageModelsFor(d.ID) {
		if key := strings.ToLower(m); !seen[key] {
			seen[key] = true
			out = append(out, m)
		}
	}
	return out
}
