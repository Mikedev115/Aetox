//go:build windows

package main

// UI Automation — the hand that reaches windows Aetox did not draw.
//
// Direction: docs/architecture/computer-use-2026-09-07.md §5 row 1. That file
// picked IUIAutomation over the two alternatives on purpose, and the reason is
// worth keeping next to the code: an accessibility element carries a name AND a
// rectangle AND an invoke verb, so a click is aimed at a thing rather than at a
// pixel. The tool this replaces (internal/skill/computer.go, 099f4bd, removed
// 50584e5) aimed at pixels read out of a screenshot by OCR, and OCR returns
// letters with no coordinates — runTesseract throws the TSV's boxes away. That
// loop could read the screen and could never point at what it read, which is a
// dead end independent of the runtime bug that actually killed it.
//
// Two things in here are load-bearing and both were learned from that failure:
//
//   - **Its own thread, in MTA.** COM interface pointers are apartment-bound.
//     The app's other COM lives in the WebView2 STA thread (browser_windows.go),
//     which runs a message pump and must not be blocked; UIA is happy in a
//     multithreaded apartment and needs no pump at all, so this is a plain
//     goroutine with a channel of closures rather than a second PostThreadMessage
//     queue. Every pointer created here is used here and released here.
//
//   - **The DPI context is set from the app's main window, not inherited.**
//     desktop/build/windows/wails.exe.manifest declares per-monitor v2; a raw
//     goroutine thread starts on the process default, which is not the same
//     thing. BoundingRectangle comes back in physical pixels either way, but
//     anything that later hands a rectangle to a Win32 call has to agree with
//     the rest of the app about what a pixel is. browser_windows.go:688 does
//     this for the same reason, one line at a time.
//
// Nothing in this file decides whether a reach is allowed. That is
// computer_guard.go, and this file is called only after it says yes.

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

var (
	oleaut32            = syscall.NewLazyDLL("oleaut32.dll")
	procSysAllocString  = oleaut32.NewProc("SysAllocString")
	procSysFreeString   = oleaut32.NewProc("SysFreeString")
	procCoCreateInst    = ole32.NewProc("CoCreateInstance")
	procCoUninitialize  = ole32.NewProc("CoUninitialize")
	procSafeArrayGetLBd = oleaut32.NewProc("SafeArrayGetLBound")
	procSafeArrayGetUBd = oleaut32.NewProc("SafeArrayGetUBound")
	procSafeArrayGetEl  = oleaut32.NewProc("SafeArrayGetElement")
	procSafeArrayDestry = oleaut32.NewProc("SafeArrayDestroy")

	procGetDesktopWindow = user32.NewProc("GetDesktopWindow")
)

// desktopRoot is the one window handle that is always there. Used to reach the
// automation root without asking for a specific application, which is what
// list_apps needs and what the binding's own test walks.
func desktopRoot() uintptr {
	h, _, _ := procGetDesktopWindow.Call()
	return h
}

// coinitMultithreaded is the apartment this file's thread joins. Deliberately
// not coinitApartmentThreaded (browser_windows.go's, 0x2): an STA without a
// message pump deadlocks the moment a COM call needs to marshal, and this
// thread has no pump by design.
const coinitMultithreaded = 0x0

// comProc is one vtable slot. Same shape as the vendored WebView2 bridge's
// ComProc (third_party/go-webview2/pkg/edge/corewebview2.go) — reimplemented
// rather than imported because that one belongs to a vendored upstream we do
// not edit, and because syscall.SyscallN is the form the unsafe.Pointer rules
// actually exempt.
type comProc uintptr

func (p comProc) call(a ...uintptr) uintptr {
	hr, _, _ := syscall.SyscallN(uintptr(p), a...)
	return hr
}

type iUnknownVtbl struct {
	QueryInterface comProc
	AddRef         comProc
	Release        comProc
}

// hresult and the HRESULT constants live in computer_errors.go: the code is the
// input to a decision made there, and that file compiles on every platform so
// the decision can be tested on every platform.

func hrOK(hr uintptr) bool { return int32(uint32(hr)) >= 0 }

func hrErr(what string, hr uintptr) error {
	if hrOK(hr) {
		return nil
	}
	return hresult{code: uint32(hr), what: what}
}

// ---------------------------------------------------------------------------
// GUIDs
// ---------------------------------------------------------------------------

type comGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidCUIAutomation = comGUID{0xff48dba4, 0x60ef, 0x4201, [8]byte{0xaa, 0x87, 0x54, 0x10, 0x3e, 0xef, 0x59, 0x4e}}
	iidIUIAutomation   = comGUID{0x30cbe57d, 0xd9d0, 0x452a, [8]byte{0xab, 0x13, 0x7a, 0xc5, 0xac, 0x48, 0x25, 0xee}}

	iidInvokePattern = comGUID{0xfb377fbe, 0x8ea6, 0x46d5, [8]byte{0x9c, 0x73, 0x64, 0x99, 0x64, 0x2d, 0x30, 0x59}}
	iidValuePattern  = comGUID{0xa94cd8b1, 0x0844, 0x4cd6, [8]byte{0x9d, 0x2d, 0x64, 0x05, 0x37, 0xab, 0x39, 0xe9}}
	iidTogglePattern = comGUID{0x94cf8058, 0x9b8d, 0x4ab9, [8]byte{0x8b, 0xfd, 0x4c, 0xd0, 0xa3, 0x3c, 0x8c, 0x70}}
	iidSelItemPatt   = comGUID{0xa8efa66a, 0x0fda, 0x421a, [8]byte{0x91, 0x94, 0x38, 0x02, 0x1f, 0x35, 0x78, 0xea}}
	iidLegacyPattern = comGUID{0x828055ad, 0x355b, 0x4435, [8]byte{0x86, 0xd5, 0x3b, 0x51, 0xc1, 0x4a, 0x9b, 0x1b}}
)

// Pattern ids — UIAutomationClient.h. Only the five this project acts through.
const (
	patternInvoke        = 10000
	patternValue         = 10002
	patternSelectionItem = 10010
	patternToggle        = 10015
	patternLegacy        = 10018
)

// TreeScope bits.
const (
	scopeElement     = 0x1
	scopeChildren    = 0x2
	scopeDescendants = 0x4
	scopeSubtree     = 0x7
)

const clsctxInprocServer = 0x1

// ---------------------------------------------------------------------------
// IUIAutomation
// ---------------------------------------------------------------------------

// The vtable is declared in full, in IDL order, up to the last slot this file
// calls. Order is the whole contract — a missing or reordered field silently
// calls the wrong function, which is the one COM mistake that does not announce
// itself. Slots past the last one used are omitted rather than guessed.
type iUIAutomationVtbl struct {
	iUnknownVtbl
	CompareElements             comProc
	CompareRuntimeIds           comProc
	GetRootElement              comProc
	ElementFromHandle           comProc
	ElementFromPoint            comProc
	GetFocusedElement           comProc
	GetRootElementBuildCache    comProc
	ElementFromHandleBuildCache comProc
	ElementFromPointBuildCache  comProc
	GetFocusedElementBuildCache comProc
	CreateTreeWalker            comProc
	GetControlViewWalker        comProc
	GetContentViewWalker        comProc
	GetRawViewWalker            comProc
	GetRawViewCondition         comProc
	GetControlViewCondition     comProc
	GetContentViewCondition     comProc
	CreateCacheRequest          comProc
	CreateTrueCondition         comProc
	CreateFalseCondition        comProc
	CreatePropertyCondition     comProc
}

type iUIAutomation struct{ vtbl *iUIAutomationVtbl }

func (a *iUIAutomation) release() {
	a.vtbl.Release.call(uintptr(unsafe.Pointer(a)))
}

func (a *iUIAutomation) elementFromHandle(hwnd uintptr) (*iUIAutomationElement, error) {
	var el *iUIAutomationElement
	hr := a.vtbl.ElementFromHandle.call(
		uintptr(unsafe.Pointer(a)), hwnd, uintptr(unsafe.Pointer(&el)))
	if err := hrErr("ElementFromHandle", hr); err != nil {
		return nil, err
	}
	if el == nil {
		return nil, hresult{code: hrElementNotAvailable, what: "ElementFromHandle returned nothing"}
	}
	return el, nil
}

func (a *iUIAutomation) createTrueCondition() (*iUIAutomationCondition, error) {
	var c *iUIAutomationCondition
	hr := a.vtbl.CreateTrueCondition.call(
		uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(&c)))
	if err := hrErr("CreateTrueCondition", hr); err != nil {
		return nil, err
	}
	return c, nil
}

type iUIAutomationCondition struct{ vtbl *iUnknownVtbl }

func (c *iUIAutomationCondition) release() {
	c.vtbl.Release.call(uintptr(unsafe.Pointer(c)))
}

// ---------------------------------------------------------------------------
// IUIAutomationElement
// ---------------------------------------------------------------------------

type iUIAutomationElementVtbl struct {
	iUnknownVtbl
	SetFocus                       comProc
	GetRuntimeId                   comProc
	FindFirst                      comProc
	FindAll                        comProc
	FindFirstBuildCache            comProc
	FindAllBuildCache              comProc
	BuildUpdatedCache              comProc
	GetCurrentPropertyValue        comProc
	GetCurrentPropertyValueEx      comProc
	GetCachedPropertyValue         comProc
	GetCachedPropertyValueEx       comProc
	GetCurrentPatternAs            comProc
	GetCachedPatternAs             comProc
	GetCurrentPattern              comProc
	GetCachedPattern               comProc
	GetCachedParent                comProc
	GetCachedChildren              comProc
	GetCurrentProcessId            comProc
	GetCurrentControlType          comProc
	GetCurrentLocalizedControlType comProc
	GetCurrentName                 comProc
	GetCurrentAcceleratorKey       comProc
	GetCurrentAccessKey            comProc
	GetCurrentHasKeyboardFocus     comProc
	GetCurrentIsKeyboardFocusable  comProc
	GetCurrentIsEnabled            comProc
	GetCurrentAutomationId         comProc
	GetCurrentClassName            comProc
	GetCurrentHelpText             comProc
	GetCurrentCulture              comProc
	GetCurrentIsControlElement     comProc
	GetCurrentIsContentElement     comProc
	GetCurrentIsPassword           comProc
	GetCurrentNativeWindowHandle   comProc
	GetCurrentItemType             comProc
	GetCurrentIsOffscreen          comProc
	GetCurrentOrientation          comProc
	GetCurrentFrameworkId          comProc
	GetCurrentIsRequiredForForm    comProc
	GetCurrentItemStatus           comProc
	GetCurrentBoundingRectangle    comProc
}

type iUIAutomationElement struct{ vtbl *iUIAutomationElementVtbl }

func (e *iUIAutomationElement) release() {
	if e != nil {
		e.vtbl.Release.call(uintptr(unsafe.Pointer(e)))
	}
}

func (e *iUIAutomationElement) this() uintptr { return uintptr(unsafe.Pointer(e)) }

// bstr reads one BSTR-returning getter and frees the string. Every text
// property on an element goes through here, so there is exactly one place that
// can leak one.
func (e *iUIAutomationElement) bstr(slot comProc, what string) (string, error) {
	var bs *uint16
	hr := slot.call(e.this(), uintptr(unsafe.Pointer(&bs)))
	if err := hrErr(what, hr); err != nil {
		return "", err
	}
	if bs == nil {
		return "", nil
	}
	defer procSysFreeString.Call(uintptr(unsafe.Pointer(bs)))
	return syscall.UTF16ToString(unsafe.Slice(bs, sysStringLen(bs))), nil
}

func (e *iUIAutomationElement) int32Prop(slot comProc, what string) (int32, error) {
	var v int32
	hr := slot.call(e.this(), uintptr(unsafe.Pointer(&v)))
	if err := hrErr(what, hr); err != nil {
		return 0, err
	}
	return v, nil
}

func (e *iUIAutomationElement) boolProp(slot comProc, what string) (bool, error) {
	// VARIANT_BOOL is 2 bytes, but the vtable writes through a BOOL* here —
	// UIA's IDL says BOOL for these, which is a 4-byte int.
	v, err := e.int32Prop(slot, what)
	return v != 0, err
}

func (e *iUIAutomationElement) name() (string, error) {
	return e.bstr(e.vtbl.GetCurrentName, "get_CurrentName")
}

func (e *iUIAutomationElement) className() (string, error) {
	return e.bstr(e.vtbl.GetCurrentClassName, "get_CurrentClassName")
}

func (e *iUIAutomationElement) localizedControlType() (string, error) {
	return e.bstr(e.vtbl.GetCurrentLocalizedControlType, "get_CurrentLocalizedControlType")
}

func (e *iUIAutomationElement) automationID() (string, error) {
	return e.bstr(e.vtbl.GetCurrentAutomationId, "get_CurrentAutomationId")
}

func (e *iUIAutomationElement) controlType() (int32, error) {
	return e.int32Prop(e.vtbl.GetCurrentControlType, "get_CurrentControlType")
}

func (e *iUIAutomationElement) processID() (int32, error) {
	return e.int32Prop(e.vtbl.GetCurrentProcessId, "get_CurrentProcessId")
}

func (e *iUIAutomationElement) isEnabled() (bool, error) {
	return e.boolProp(e.vtbl.GetCurrentIsEnabled, "get_CurrentIsEnabled")
}

func (e *iUIAutomationElement) isOffscreen() (bool, error) {
	return e.boolProp(e.vtbl.GetCurrentIsOffscreen, "get_CurrentIsOffscreen")
}

// isPassword is the one property this project must never get wrong: the line in
// the direction doc (§6.2) says never type a credential, and this is how that
// line is enforced rather than hoped for.
func (e *iUIAutomationElement) isPassword() (bool, error) {
	return e.boolProp(e.vtbl.GetCurrentIsPassword, "get_CurrentIsPassword")
}

func (e *iUIAutomationElement) nativeWindowHandle() (uintptr, error) {
	var h uintptr
	hr := e.vtbl.GetCurrentNativeWindowHandle.call(e.this(), uintptr(unsafe.Pointer(&h)))
	if err := hrErr("get_CurrentNativeWindowHandle", hr); err != nil {
		return 0, err
	}
	return h, nil
}

type uiaRect struct{ Left, Top, Right, Bottom float64 }

func (e *iUIAutomationElement) boundingRect() (uiaRect, error) {
	var r uiaRect
	hr := e.vtbl.GetCurrentBoundingRectangle.call(e.this(), uintptr(unsafe.Pointer(&r)))
	if err := hrErr("get_CurrentBoundingRectangle", hr); err != nil {
		return uiaRect{}, err
	}
	return r, nil
}

// runtimeID is how a ref survives the moment its pointer does not. UIA hands
// back a SAFEARRAY of ints that identifies the element inside its provider;
// computer_refs.go keeps it beside the pointer so a stale pointer can be looked
// up again instead of reported as "that is gone".
func (e *iUIAutomationElement) runtimeID() ([]int32, error) {
	var sa uintptr
	hr := e.vtbl.GetRuntimeId.call(e.this(), uintptr(unsafe.Pointer(&sa)))
	if err := hrErr("GetRuntimeId", hr); err != nil {
		return nil, err
	}
	if sa == 0 {
		return nil, nil
	}
	defer procSafeArrayDestry.Call(sa)
	var lo, hi int32
	if r, _, _ := procSafeArrayGetLBd.Call(sa, 1, uintptr(unsafe.Pointer(&lo))); !hrOK(r) {
		return nil, hrErr("SafeArrayGetLBound", r)
	}
	if r, _, _ := procSafeArrayGetUBd.Call(sa, 1, uintptr(unsafe.Pointer(&hi))); !hrOK(r) {
		return nil, hrErr("SafeArrayGetUBound", r)
	}
	out := make([]int32, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		var v int32
		idx := i
		if r, _, _ := procSafeArrayGetEl.Call(sa, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&v))); !hrOK(r) {
			return nil, hrErr("SafeArrayGetElement", r)
		}
		out = append(out, v)
	}
	return out, nil
}

func (e *iUIAutomationElement) findAll(scope int32, cond *iUIAutomationCondition) (*iUIAutomationElementArray, error) {
	var arr *iUIAutomationElementArray
	hr := e.vtbl.FindAll.call(e.this(), uintptr(scope),
		uintptr(unsafe.Pointer(cond)), uintptr(unsafe.Pointer(&arr)))
	if err := hrErr("FindAll", hr); err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, hresult{code: hrElementNotAvailable, what: "FindAll returned nothing"}
	}
	return arr, nil
}

func (e *iUIAutomationElement) setFocus() error {
	return hrErr("SetFocus", e.vtbl.SetFocus.call(e.this()))
}

// patternAs asks the element for one pattern interface. A nil result with a
// success code is UIA's way of saying "this control does not do that", which is
// a different sentence from an error and is reported as one.
// The result is an unsafe.Pointer rather than a uintptr on purpose: a COM
// interface pointer held as an integer is exactly the shape go vet flags, and
// it is right to — an integer is not a reference, so nothing stops the object
// moving out from under it between the call that made it and the call that uses
// it.
func (e *iUIAutomationElement) patternAs(id int32, iid *comGUID) (unsafe.Pointer, error) {
	var p unsafe.Pointer
	hr := e.vtbl.GetCurrentPatternAs.call(e.this(), uintptr(id),
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&p)))
	if err := hrErr("GetCurrentPatternAs", hr); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, hresult{code: hrNotSupported, what: "control does not support that pattern"}
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// IUIAutomationElementArray
// ---------------------------------------------------------------------------

type iUIAutomationElementArrayVtbl struct {
	iUnknownVtbl
	GetLength  comProc
	GetElement comProc
}

type iUIAutomationElementArray struct {
	vtbl *iUIAutomationElementArrayVtbl
}

func (a *iUIAutomationElementArray) release() {
	a.vtbl.Release.call(uintptr(unsafe.Pointer(a)))
}

func (a *iUIAutomationElementArray) length() (int32, error) {
	var n int32
	hr := a.vtbl.GetLength.call(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(&n)))
	if err := hrErr("IUIAutomationElementArray::get_Length", hr); err != nil {
		return 0, err
	}
	return n, nil
}

func (a *iUIAutomationElementArray) at(i int32) (*iUIAutomationElement, error) {
	var el *iUIAutomationElement
	hr := a.vtbl.GetElement.call(uintptr(unsafe.Pointer(a)), uintptr(i), uintptr(unsafe.Pointer(&el)))
	if err := hrErr("IUIAutomationElementArray::GetElement", hr); err != nil {
		return nil, err
	}
	return el, nil
}

// ---------------------------------------------------------------------------
// Patterns — the acting half
// ---------------------------------------------------------------------------

type invokePatternVtbl struct {
	iUnknownVtbl
	Invoke comProc
}

type valuePatternVtbl struct {
	iUnknownVtbl
	SetValue           comProc
	GetCurrentValue    comProc
	GetCurrentReadOnly comProc
}

type togglePatternVtbl struct {
	iUnknownVtbl
	Toggle comProc
}

type selItemPatternVtbl struct {
	iUnknownVtbl
	Select comProc
}

type legacyPatternVtbl struct {
	iUnknownVtbl
	Select          comProc
	DoDefaultAction comProc
	SetValue        comProc
}

type comObject struct{ vtbl *iUnknownVtbl }

func releaseCOM(p unsafe.Pointer) {
	if p == nil {
		return
	}
	(*comObject)(p).vtbl.Release.call(uintptr(p))
}

// invoke presses the element the way its own provider defines pressing, which
// is the whole reason this file exists: no cursor moves, no coordinate is
// guessed, and a button that has moved since the last read is still the button.
func (e *iUIAutomationElement) invoke() error {
	p, err := e.patternAs(patternInvoke, &iidInvokePattern)
	if err != nil {
		return err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *invokePatternVtbl })(p)
	return hrErr("InvokePattern::Invoke", v.vtbl.Invoke.call(uintptr(p)))
}

func (e *iUIAutomationElement) toggle() error {
	p, err := e.patternAs(patternToggle, &iidTogglePattern)
	if err != nil {
		return err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *togglePatternVtbl })(p)
	return hrErr("TogglePattern::Toggle", v.vtbl.Toggle.call(uintptr(p)))
}

func (e *iUIAutomationElement) selectItem() error {
	p, err := e.patternAs(patternSelectionItem, &iidSelItemPatt)
	if err != nil {
		return err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *selItemPatternVtbl })(p)
	return hrErr("SelectionItemPattern::Select", v.vtbl.Select.call(uintptr(p)))
}

// doDefaultAction is the fallback for controls too old to carry a modern
// pattern — Win32 dialogs, MFC panels, anything whose provider is the legacy
// IAccessible bridge. Tried last, never first: it is defined as "whatever
// double-clicking would do", which is exactly the vagueness the patterns above
// exist to avoid.
func (e *iUIAutomationElement) doDefaultAction() error {
	p, err := e.patternAs(patternLegacy, &iidLegacyPattern)
	if err != nil {
		return err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *legacyPatternVtbl })(p)
	return hrErr("LegacyIAccessible::DoDefaultAction", v.vtbl.DoDefaultAction.call(uintptr(p)))
}

// setValue writes text into a control without touching the keyboard. Thai and
// every other script arrive as themselves because this is a string handed to
// the provider, not a sequence of synthesized key events with a layout in the
// middle of them — which is the half of the old tool that was hardest to trust.
func (e *iUIAutomationElement) setValue(text string) error {
	p, err := e.patternAs(patternValue, &iidValuePattern)
	if err != nil {
		return err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *valuePatternVtbl })(p)

	var readOnly int32
	if hr := v.vtbl.GetCurrentReadOnly.call(uintptr(p), uintptr(unsafe.Pointer(&readOnly))); hrOK(hr) && readOnly != 0 {
		return hresult{code: hrElementNotEnabled, what: "that field is read-only"}
	}

	bs, err := sysAllocString(text)
	if err != nil {
		return err
	}
	defer procSysFreeString.Call(bs)
	return hrErr("ValuePattern::SetValue", v.vtbl.SetValue.call(uintptr(p), bs))
}

func (e *iUIAutomationElement) currentValue() (string, error) {
	p, err := e.patternAs(patternValue, &iidValuePattern)
	if err != nil {
		return "", err
	}
	defer releaseCOM(p)
	v := (*struct{ vtbl *valuePatternVtbl })(p)
	var bs *uint16
	hr := v.vtbl.GetCurrentValue.call(uintptr(p), uintptr(unsafe.Pointer(&bs)))
	if err := hrErr("ValuePattern::get_CurrentValue", hr); err != nil {
		return "", err
	}
	if bs == nil {
		return "", nil
	}
	defer procSysFreeString.Call(uintptr(unsafe.Pointer(bs)))
	return syscall.UTF16ToString(unsafe.Slice(bs, sysStringLen(bs))), nil
}

// ---------------------------------------------------------------------------
// BSTR helpers
// ---------------------------------------------------------------------------

var procSysStringLen = oleaut32.NewProc("SysStringLen")

func sysStringLen(bs *uint16) int {
	n, _, _ := procSysStringLen.Call(uintptr(unsafe.Pointer(bs)))
	return int(n)
}

func sysAllocString(s string) (uintptr, error) {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return 0, fmt.Errorf("text cannot be sent to Windows as-is: %w", err)
	}
	bs, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(p)))
	if bs == 0 {
		return 0, fmt.Errorf("out of memory allocating a string for Windows")
	}
	return bs, nil
}

// ---------------------------------------------------------------------------
// The thread
// ---------------------------------------------------------------------------

// uiaStartBudget bounds the wait for the automation thread, for the same reason
// browserStartBudget bounds the browser's: not a latency target, a promise that
// a tool call ENDS. A var so a test can shorten it.
var uiaStartBudget = 10 * time.Second

type uiaJob struct {
	fn   func(*iUIAutomation) error
	done chan error
}

// uiaHost owns the automation thread and the single IUIAutomation created on
// it. Every call into UIA in this process goes through do().
type uiaHost struct {
	mu       sync.Mutex
	started  bool
	ready    chan struct{}
	startErr error

	jobs chan uiaJob
	stop chan struct{}
}

var (
	uiaOnce sync.Once
	uiaSelf *uiaHost
)

// uia returns the process's automation host, starting its thread on first use.
// Lazy rather than started at boot: a user who never turns the feature on never
// pays for a COM apartment, and CoCreateInstance failing is then a message on
// the one tool call that wanted it rather than a line in a startup log.
func uia() *uiaHost {
	uiaOnce.Do(func() {
		uiaSelf = &uiaHost{
			ready: make(chan struct{}),
			jobs:  make(chan uiaJob),
			stop:  make(chan struct{}),
		}
	})
	return uiaSelf
}

func (h *uiaHost) start() error {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return h.await()
	}
	h.started = true
	h.mu.Unlock()

	go h.run()
	return h.await()
}

func (h *uiaHost) await() error {
	select {
	case <-h.ready:
		return h.startErr
	case <-time.After(uiaStartBudget):
		return fmt.Errorf("Windows UI Automation did not start within %s", uiaStartBudget)
	}
}

func (h *uiaHost) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	// S_FALSE (0x1) means this thread was already in an apartment, which is
	// fine and still needs its matching CoUninitialize.
	if !hrOK(hr) {
		h.startErr = hrErr("CoInitializeEx", hr)
		close(h.ready)
		return
	}
	defer procCoUninitialize.Call()

	// The app's DPI context, not the process default. See the file header.
	if parent := findOwnMainWindow(); parent != 0 {
		if ctx, _, _ := procGetWindowDpiAwarenessCtx.Call(parent); ctx != 0 {
			prev, _, _ := procSetThreadDpiAwarenessCtx.Call(ctx)
			debuglog.Msg("uia.run: thread DPI ctx set to app's (prev=%#x)", prev)
		}
	}

	var auto *iUIAutomation
	hr, _, _ = procCoCreateInst.Call(
		uintptr(unsafe.Pointer(&clsidCUIAutomation)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)), uintptr(unsafe.Pointer(&auto)))
	if !hrOK(hr) || auto == nil {
		h.startErr = hrErr("CoCreateInstance(CUIAutomation)", hr)
		close(h.ready)
		return
	}
	defer auto.release()
	debuglog.Msg("uia.run: automation ready")
	close(h.ready)

	for {
		select {
		case job := <-h.jobs:
			job.done <- job.fn(auto)
		case <-h.stop:
			return
		}
	}
}

// do runs fn on the automation thread and waits for it. Synchronous on purpose:
// every caller is a tool action that has to report what happened, and an async
// queue here would only move the wait somewhere with less context about it.
func (h *uiaHost) do(fn func(*iUIAutomation) error) error {
	if err := h.start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	select {
	case h.jobs <- uiaJob{fn: fn, done: done}:
	case <-time.After(uiaStartBudget):
		return fmt.Errorf("the automation thread is busy and did not take the work")
	}
	return <-done
}

// shutdown stops the thread. Called when the app closes; safe to call twice.
func (h *uiaHost) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.started {
		return
	}
	h.started = false
	close(h.stop)
}
