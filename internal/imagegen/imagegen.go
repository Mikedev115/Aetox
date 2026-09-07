package imagegen

// Package imagegen is the third of the media packages, and it is deliberately
// the same shape as the two that came before it (ARCHITECTURE.md §33):
// internal/stt turns sound into text, internal/tts turns text into sound, and
// this turns text into a picture. A catalog describes what exists, New()
// switches on it and hands back one interface, and callers never name a
// concrete vendor. Adding a vendor is a Descriptor plus a file — the settings
// picker renders from Catalog() and needs no change.
//
// **Why this is not a model in the model picker.** A picture-making model is
// not a chat model wearing a different hat: `openai/gpt-image-1` has no context
// window and cannot call a tool, and both of the catalog's chat filters
// (internal/model's chatCandidates and usableFirst) require Produces("text")
// for good reasons of their own. Threading image output through the engine
// would mean loosening the filters that keep an un-callable model out of the
// chat picker, to serve a job the chat loop does not do. Speech was kept out
// for exactly this reason and this follows it.
//
// Engines disagree about everything below this line, as usual. Pollinations is
// a GET with the prompt in the path and no key at all; the OpenAI images
// endpoint is a POST that answers with base64 in JSON; Gemini asks for an image
// modality on an otherwise ordinary generateContent call; a local
// stable-diffusion.cpp is a binary with weight files. Nothing above this
// package is allowed to care. An Engine takes a prompt and a path, writes a
// picture there, and that is the whole contract.
//
// Policy this package does NOT hold: where the file goes, what it is called,
// and whether the caller was allowed to write there. Those are the sandbox's
// business and they live with the skill (internal/skill/image_make.go), which
// is also where the §31 rule lands — a cloud vendor is only ever reached
// because the user picked one by name in settings, and every cloud row's
// Install text says outright that the prompt leaves the machine.

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Engine turns a prompt into an image file. Implementations own nothing: the
// caller picks the path, shows or ships the file, and deletes it.
type Engine interface {
	// ID matches the Descriptor it was built from.
	ID() string
	// Generate writes one picture for prompt at outPath. The caller has
	// already decided outPath is writable and has given it the extension Ext
	// reports; an engine that would rather have written something else must
	// convert rather than rename.
	Generate(ctx context.Context, prompt string, req Request, outPath string) error
	// Ext is the file extension this engine writes, leading dot included
	// (".png", ".jpg"). The caller names the file before it exists, so this
	// cannot be discovered from the bytes afterwards.
	Ext() string
}

// Request is what varies per picture, as opposed to per engine. The zero value
// is valid and means "the engine's own default size, unseeded".
type Request struct {
	// Width and Height in pixels. Zero on either lets the engine choose, and
	// an engine that only serves fixed sizes rounds to its nearest.
	Width, Height int
	// Seed makes a picture reproducible on engines that accept one. Zero means
	// unseeded — a different picture every call.
	Seed int
}

// Descriptor is what the settings UI renders and what New() builds from — one
// entry per supported vendor, data only, no behavior. Same fields as
// tts.Descriptor wherever the question is the same one, so desktop's engine-row
// builder can serve both pickers without a second shape to learn.
type Descriptor struct {
	ID    string // stable config value, e.g. "pollinations"
	Label string // shown to the user
	// Binaries are the PATH candidates that count as this vendor being
	// installed, in preference order. Empty for a vendor that is an HTTP call.
	Binaries []string
	// Install is ONE SHORT LINE, shown under the vendor picker. It carries only
	// what the label cannot: what this vendor needs, its one real quirk, and —
	// never omitted on a cloud row — where the prompt goes.
	//
	// It used to be a paragraph each, stacked under a second paragraph of
	// generic advice: two walls of prose over a dropdown whose label already
	// said the important half (owner, 7 ก.ย.: "เขียนรายละเอียดสะยาวเลย"). The
	// generic half is gone from the page and this half is a line.
	Install string
	// InstallCommand is the exact argv the settings page's ติดตั้ง button runs,
	// and also the command it displays — one value serving both, so the command
	// on screen can never differ from the command that runs. Empty for a vendor
	// with no runnable install.
	InstallCommand []string
	// Models are the NAMED models this vendor accepts, first = the default.
	// One entry means there is no real choice and the page draws no picker;
	// empty means the vendor has no model concept.
	//
	// This is the FALLBACK roster, not the authority. Where the installed
	// model catalog knows the vendor, the picker prefers what it reports for
	// Produces("image") — see desktop/image.go — because a picture model's
	// name moves (gpt-image-1 will not be the last of its line) and a list
	// typed in here goes stale silently. This list is what a machine with no
	// catalog, or an offline one, still gets to choose from.
	Models []string
	// Default marks the vendor chosen when config says nothing.
	Default bool
}

// Options is the per-call configuration. The zero value is valid and resolves
// to the default vendor running its default model.
type Options struct {
	// Engine is a Descriptor.ID. Empty picks the catalog default.
	Engine string
	// Model pins one of the vendor's named models. Empty takes the first.
	Model string
}

// catalog is the single list of vendors Aetox knows how to draw with. A new
// vendor is an entry here plus its constructor in newEngine.
//
// Pollinations is the default for the same reason `edge` is the free row on the
// speech side: it needs no key, no install and no GPU, so the first picture a
// user asks for works on a machine that has only just been unzipped. What it
// costs is stated in its Install text rather than buried — the prompt goes to
// somebody else's server.
var catalog = []Descriptor{
	{
		ID:    "pollinations",
		Label: "Pollinations (คลาวด์, ฟรี ไม่ใช้ key)",
		// No Binaries and no InstallCommand: it is one HTTP GET, so there is
		// nothing to install.
		Install: "ฟรี ไม่ต้องตั้งอะไร · คำสั่งวาดส่งไปที่เซิร์ฟเวอร์ของ Pollinations",
		Models:  []string{"flux", "turbo"},
		Default: true,
	},
	// The three below need no new credential from the user: each reads the key
	// already entered for that same provider on ตั้งค่า > โมเดล
	// (config.ProviderAPIKey), which is the whole reason they are worth having
	// as separate rows rather than as one "bring your own endpoint" row.
	{
		ID:      "openai",
		Label:   "OpenAI (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · เปลี่ยน Base URL ชี้ไปเซิร์ฟเวอร์ในบ้านได้ · คำสั่งวาดส่งไปที่ OpenAI",
		Models:  []string{"gpt-image-1", "dall-e-3"},
	},
	{
		ID:      "xai",
		Label:   "xAI Grok (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · กำหนดขนาดภาพไม่ได้ · คำสั่งวาดส่งไปที่ xAI",
		Models:  []string{"grok-2-image"},
	},
	// Three more that this build already holds a key and a base URL for
	// (internal/provider/catalog.go), all three declaring
	// RuntimeOpenAICompatible — so they cost one row each and no new code.
	//
	// Their sizes are deliberately NOT declared supported: DashScope writes a
	// size as "1024*1024" rather than "1024x1024", and ModelScope's roster
	// moves. An omitted size gets the vendor's own default, which works; a
	// wrongly-formatted one is a 400 on a request that would have succeeded.
	{
		ID:      "alibaba",
		Label:   "Alibaba Qwen (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · คำสั่งวาดส่งไปที่ Alibaba Cloud",
		Models:  []string{"wan2.2-t2i-flash", "qwen-image"},
	},
	{
		ID:      "zai",
		Label:   "Z.ai CogView (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · คำสั่งวาดส่งไปที่ Z.ai",
		Models:  []string{"cogview-4", "cogview-3-flash"},
	},
	{
		ID:      "modelscope",
		Label:   "ModelScope (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · คำสั่งวาดส่งไปที่ ModelScope",
		Models:  []string{"MusePublic/FLUX.1-Krea-dev"},
	},
	{
		ID:      "gemini",
		Label:   "Gemini (คลาวด์, ใช้ API key เดิม)",
		Install: "ใช้ API key เดิมจากหน้าโมเดล · ขนาดฝากไปกับคำบรรยาย · คำสั่งวาดส่งไปที่ Google",
		Models:  []string{"gemini-2.5-flash-image", "gemini-3-pro-image"},
	},
}

// Catalog returns every known vendor — default first, then alphabetical — for
// the settings picker to enumerate.
func Catalog() []Descriptor {
	out := make([]Descriptor, len(catalog))
	copy(out, catalog)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Default != out[j].Default {
			return out[i].Default
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Lookup finds a vendor descriptor by ID. An empty id resolves to the default.
func Lookup(id string) (Descriptor, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, d := range catalog {
		if id == "" && d.Default {
			return d, true
		}
		if d.ID == id {
			return d, true
		}
	}
	return Descriptor{}, false
}

// New resolves opts into a ready engine, or returns an actionable Thai error —
// a wrong vendor name, or one this machine cannot run. Callers hand that error
// straight to the user.
func New(opts Options) (Engine, error) {
	desc, ok := Lookup(opts.Engine)
	if !ok {
		return nil, fmt.Errorf("ไม่รู้จัก engine สร้างภาพชื่อ %q — ที่รองรับตอนนี้: %s", opts.Engine, strings.Join(engineIDs(), ", "))
	}
	return newEngine(desc, opts)
}

// newEngine is the one switch that maps a descriptor to a concrete runtime.
func newEngine(desc Descriptor, opts Options) (Engine, error) {
	switch desc.ID {
	case "pollinations":
		return newPollinations(desc, opts)
	// Two rows, one runtime: these disagree about their models and their sizes
	// and about nothing else (openai_api.go).
	case "openai", "xai", "alibaba", "zai", "modelscope":
		return newAPIImages(desc, opts)
	case "gemini":
		return newGeminiImages(desc, opts)
	default:
		return nil, fmt.Errorf("engine %q อยู่ในรายการแต่ยังไม่มีตัวรัน", desc.ID)
	}
}

// resolveNamedModel picks the named model an engine runs: the pinned one when
// the roster knows it, the roster's first when nothing is pinned. An unknown
// pin is a loud error, not a silent fallback — a setting that quietly stops
// applying is a setting that lies. Same rule and same wording as tts.
func resolveNamedModel(desc Descriptor, pinned string) (string, error) {
	pinned = strings.TrimSpace(pinned)
	if len(desc.Models) == 0 {
		if pinned != "" {
			return "", fmt.Errorf("%s ไม่มีให้เลือกโมเดล แต่ตั้งค่าไว้เป็น %q", desc.Label, pinned)
		}
		return "", nil
	}
	if pinned == "" {
		return desc.Models[0], nil
	}
	for _, m := range desc.Models {
		if m == pinned {
			return m, nil
		}
	}
	// Not in the built-in roster is not by itself wrong: the picker prefers
	// the installed model catalog (see Descriptor.Models), which knows names
	// this list has never heard of. What would be wrong is refusing a name the
	// user picked from a list the app itself drew.
	return pinned, nil
}

func engineIDs() []string {
	out := make([]string, 0, len(catalog))
	for _, d := range catalog {
		out = append(out, d.ID)
	}
	sort.Strings(out)
	return out
}
