package main

import (
	"os"
	"strings"
	"testing"
)

// One list, in Go, and the menu reads it.
//
// The eight devices lived in Workbench.svelte until the agent needed to name one
// too. A second copy in TypeScript would have drifted the first time either side
// gained a phone, and then the menu and the agent would be offering different
// machines under the same name — which is the failure nobody notices, because
// each side is self-consistent.
func TestTheDeviceListIsTheOnlyDeviceList(t *testing.T) {
	src, err := os.ReadFile("frontend/src/lib/workbench/Workbench.svelte")
	if err != nil {
		t.Fatalf("reading the workbench: %v", err)
	}
	for _, d := range browserDevices {
		// The name may appear nowhere in the frontend: the rows are rendered
		// from what Go hands over, so a literal here is a second list starting.
		if strings.Contains(string(src), d.Name) {
			t.Errorf("Workbench.svelte still names %q; the device list belongs to browser_device.go alone", d.Name)
		}
	}
}

// Every device says what it is, and the mobile ones say it consistently. A
// phone that reports one device pixel per CSS pixel is a phone that gets served
// the images meant for a 2012 desktop.
func TestEveryDeviceIsFullyDescribed(t *testing.T) {
	for _, d := range browserDevices {
		if d.W <= 0 || d.H <= 0 || d.DPR <= 0 || d.platform == "" {
			t.Errorf("%+v is missing part of what a device is", d)
		}
		if d.Mobile && d.DPR < 2 {
			t.Errorf("%s is mobile with dpr %g; no phone has shipped at 1 in over a decade", d.Name, d.DPR)
		}
		// Only the mobile ones need an OS version, and the reason is the shape
		// of the change rather than tidiness: client hints are sent for those
		// alone, and Desktop's whole job is to clear the override rather than
		// describe a machine.
		if d.Mobile && d.platformVersion == "" {
			t.Errorf("%s is emulated but names no OS version; the metadata block would go out incomplete", d.Name)
		}
		if d.platform == "iOS" && d.ua == "" {
			t.Errorf("%s is an Apple device with no user agent; nothing about Safari can be derived from this engine", d.Name)
		}
	}
}

// The Android user agent is built from the engine's own, so the Chrome version
// in it is never written down twice. The literal here plays the part of what
// Browser.getVersion hands back.
func TestTheAndroidUserAgentIsDerivedFromTheEngines(t *testing.T) {
	const engine = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36 Edg/152.0.4191.53"

	got := androidUA(engine, "Pixel 7")

	for _, want := range []string{"Linux; Android 15; Pixel 7", "Chrome/152.0.0.0", " Mobile Safari/"} {
		if !strings.Contains(got, want) {
			t.Errorf("androidUA = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, "Windows") {
		t.Errorf("androidUA = %q, still says Windows", got)
	}
	// Edge names itself at the end, and on a phone that names a browser the
	// page is not being shown by.
	if strings.Contains(got, "Edg/") {
		t.Errorf("androidUA = %q, still claims to be Edge", got)
	}
}

// Handed something it cannot read, it says so by staying silent, and applyDevice
// reads that as "leave the user agent alone". A wrong user agent is worse than
// the true one: the true one at least matches the engine actually rendering.
func TestAnUnreadableUserAgentIsLeftAlone(t *testing.T) {
	if got := androidUA("something with no platform token", "Pixel 7"); got != "" {
		t.Errorf("androidUA on unrecognised input = %q, want empty", got)
	}
}

// The agent types the name, so it is matched the way a person would forgive.
func TestADeviceIsFoundByTheNameEitherSideWouldType(t *testing.T) {
	for _, in := range []string{"iPhone SE", "iphone se", "  IPHONE SE  "} {
		if d, ok := deviceNamed(in); !ok || d.W != 375 {
			t.Errorf("deviceNamed(%q) = %+v, %v; want the iPhone SE", in, d, ok)
		}
	}
	if _, ok := deviceNamed("Nokia 3310"); ok {
		t.Error("deviceNamed invented a device that is not on the list")
	}
}

// The engine is not allowed to set the size, and this is the test that says so.
//
// The pane already shrinks the native window to the device and zooms the page to
// match, which is what makes the CSS viewport 390 for real. Sending width and
// height to the engine as well set it a second time and the two disagreed: the
// engine painted its 390-wide viewport inside a window that was still the pane's
// width, leaving a white band down the right of the page. 0 is CDP's own word
// for "do not override this".
func TestTheEngineIsNotTheOneThatSetsTheSize(t *testing.T) {
	dev, _ := deviceNamed("iPhone 12 Pro")
	method, params := deviceMetrics(dev)

	if method != "Emulation.setDeviceMetricsOverride" {
		t.Fatalf("method = %q", method)
	}
	if !strings.Contains(params, `"width":0`) || !strings.Contains(params, `"height":0`) {
		t.Errorf("params = %s, want width and height left to the window", params)
	}
	// What the engine DOES own: what the size means.
	for _, want := range []string{`"deviceScaleFactor":3`, `"mobile":true`} {
		if !strings.Contains(params, want) {
			t.Errorf("params = %s, want %s", params, want)
		}
	}
	if m, p := deviceMetrics(deviceProfile{}); m != "Emulation.clearDeviceMetricsOverride" || p != "{}" {
		t.Errorf("เต็มแผง = %s %s, want the override cleared", m, p)
	}
}

// userAgentMetadata is all or nothing.
//
// Aetox shipped it with two fields — platform and mobile, which is what every
// example shows — and WebView2 answered hr=0x80070057 on every device, naming
// no parameter. Verified against a real Chromium on 31 ส.ค.: a partial metadata
// block is "Invalid parameters", a complete one is accepted. So the test is the
// field list itself.
func TestTheClientHintsAreSentWholeOrNotAtAll(t *testing.T) {
	const engine = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36 Edg/152.0.4191.53"
	required := []string{`"brands"`, `"fullVersion"`, `"platform"`, `"platformVersion"`, `"architecture"`, `"model"`, `"mobile"`}

	for _, name := range []string{"Pixel 7", "iPhone 12 Pro", "iPad Mini"} {
		dev, _ := deviceNamed(name)
		got := deviceUA(dev, engine)
		for _, field := range required {
			if !strings.Contains(got, field) {
				t.Errorf("%s: %s is missing from %s — a partial metadata block is refused outright", name, field, got)
			}
		}
	}

	// Android's brands carry the engine's own version, so the number is never
	// written down in this repository.
	if got := deviceUA(mustDevice(t, "Pixel 7"), engine); !strings.Contains(got, `"version":"152"`) {
		t.Errorf("Pixel 7 metadata = %s, want the engine's own major version", got)
	}
	// Safari sends no client hints at all, so claiming Chromium brands on an
	// iPhone is a contradiction a site can catch.
	if got := deviceUA(mustDevice(t, "iPhone SE"), engine); !strings.Contains(got, `"brands":[]`) {
		t.Errorf("iPhone metadata = %s, want no Chromium brands", got)
	}
}

// Going back is one empty string. There is no clearUserAgentOverride, and
// re-sending the engine's real user agent looks like it works while leaving the
// client hints still saying phone. Verified against a real Chromium.
func TestGoingBackToTheDesktopClearsTheOverride(t *testing.T) {
	for _, dev := range []deviceProfile{{}, mustDevice(t, "Desktop")} {
		if got := deviceUA(dev, ""); got != `{"userAgent":""}` {
			t.Errorf("deviceUA(%q) = %s, want the override cleared", dev.Name, got)
		}
	}
}

// No engine version, no derived user agent, and no call — the true one is left
// in place rather than a half-built one being sent.
func TestAnAndroidWithNothingToDeriveFromMakesNoCall(t *testing.T) {
	if got := deviceUA(mustDevice(t, "Pixel 7"), "not a user agent"); got != "" {
		t.Errorf("deviceUA = %s, want no call at all", got)
	}
}

func mustDevice(t *testing.T, name string) deviceProfile {
	t.Helper()
	d, ok := deviceNamed(name)
	if !ok {
		t.Fatalf("no device called %q", name)
	}
	return d
}

// Turning the touch screen OFF is where this broke, twice.
//
// maxTouchPoints has a minimum of 1, so the obvious spelling of switching touch
// off — enabled false, zero points — is refused as Invalid parameters, and
// WebView2 reports that as hr=0x80070057 naming no field. Going back to Desktop
// failed on it every time.
//
// The reason it survived a round of testing is worth more than the fix: the ON
// path was replayed against a real Chromium and the OFF path was not, because
// turning something off does not look like it can be wrong. Both paths of every
// device are replayed now, and the strings they produce are pinned here.
func TestTurningTouchOffSendsNoPointCount(t *testing.T) {
	if got := deviceTouch(deviceProfile{}); got != `{"enabled":false}` {
		t.Errorf("เต็มแผง touch = %s; maxTouchPoints has a minimum of 1, so switching off must not send one", got)
	}
	if got := deviceTouch(mustDevice(t, "Desktop")); got != `{"enabled":false}` {
		t.Errorf("Desktop touch = %s, want no point count", got)
	}
	if got := deviceTouch(mustDevice(t, "Pixel 7")); got != `{"enabled":true,"maxTouchPoints":5}` {
		t.Errorf("Pixel 7 touch = %s", got)
	}
	// The shape of the bug, stated directly: no call this file can produce may
	// carry a zero point count.
	for _, d := range append([]deviceProfile{{}}, browserDevices...) {
		if strings.Contains(deviceTouch(d), `"maxTouchPoints":0`) {
			t.Errorf("%q would send a zero point count, which is refused outright", d.Name)
		}
	}
}

// Every phone in the list has to know its own shape, and every non-phone has to
// have none.
//
// A device emulator that gets the size right and the shape wrong is a rectangle
// claiming to be a phone: the two things anybody actually checks on a mobile
// layout — is something hidden under the notch, is something lost in the
// corners — are exactly the two a rectangle cannot show. Radius and notch
// therefore travel with the size, in the same CSS pixels, and the native window
// is really cut to them (browser_windows.go, applyShape).
func TestEveryPhoneKnowsTheShapeOfItsScreen(t *testing.T) {
	for _, d := range browserDevices {
		if !d.Mobile {
			if d.Radius != 0 || d.Notch != "" {
				t.Errorf("%s is not a phone and should not be drawn as one", d.Name)
			}
			continue
		}
		// A radius of 0 is allowed and is not laziness: the iPhone SE really is
		// a rectangle with a home button, and rounding it would be drawing a
		// phone nobody makes. What is not allowed is a cut-out on a screen with
		// square corners — no such device exists, and it would look like a bug.
		if d.Notch != "" && d.Radius <= 0 {
			t.Errorf("%s has a %s and square corners, which is no phone", d.Name, d.Notch)
		}
		// A notch is a hole in the screen: it is cut out of the window and the
		// panel paints black through it, so a name without a size would be a
		// hole of nothing.
		if d.Notch != "" && (d.NotchW <= 0 || d.NotchH <= 0) {
			t.Errorf("%s says %q and gives it no size", d.Name, d.Notch)
		}
		if d.Notch == "" && (d.NotchW != 0 || d.NotchH != 0) {
			t.Errorf("%s is sized for a notch it does not have", d.Name)
		}
		if d.Notch != "" && d.Notch != "notch" && d.Notch != "island" {
			t.Errorf("%s has an unknown cut-out %q — the pane draws two shapes", d.Name, d.Notch)
		}
		// The cut has to fit in the screen it is cut from, with room either
		// side, or it is not a notch but a slice off the top.
		if d.NotchW > 0 && d.NotchW >= d.W {
			t.Errorf("%s's notch (%d) is as wide as its screen (%d)", d.Name, d.NotchW, d.W)
		}
	}
}

// The shape reaches the window, scaled by the same factor the phone is drawn
// at — a 932-tall iPhone in a 700-tall panel has corners three quarters the
// size, because they are three quarters of a corner.
func TestTheScreenShapeReachesTheWindow(t *testing.T) {
	app, b, view := detachedHost(t)

	app.BrowserSetScreenShape("web-agent-1", 41, 94, 28)
	b.drain()

	if view.shape != [3]int{41, 94, 28} {
		t.Errorf("shape = %v, want the numbers the pane measured", view.shape)
	}
}

// Going back to เต็มแผง has to say so. A window keeps the region nobody
// replaced, so a desktop page would otherwise keep the rounded corners and the
// bite out of the top of whatever phone was chosen before it.
func TestLeavingAPhoneClearsTheShape(t *testing.T) {
	app, b, view := detachedHost(t)

	app.BrowserSetScreenShape("web-agent-1", 41, 94, 28)
	b.drain()
	app.BrowserSetScreenShape("web-agent-1", 0, 0, 0)
	b.drain()

	if view.shape != [3]int{0, 0, 0} {
		t.Errorf("shape = %v, want it cleared", view.shape)
	}
}
