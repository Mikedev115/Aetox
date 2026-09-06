package main

// engineClosed separates "this webview is closed" from every other refusal.
// Getting it wrong in either direction is expensive: a false negative is the
// dead-tab-forever failure of 6 ก.ย., a false positive revives a live tab
// over an ordinary bad argument.

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestEngineClosedRecognisesTheClosedWebviewCodes(t *testing.T) {
	closed := []syscall.Errno{
		0x139F,     // ERROR_INVALID_STATE as Win32
		0x8007139F, // the same as an HRESULT — what the vtbl calls return
		0x80010108, // RPC_E_DISCONNECTED
		0x800706BA, // RPC_S_SERVER_UNAVAILABLE
		0x800706BE, // RPC_S_CALL_FAILED
	}
	for _, code := range closed {
		if !engineClosed(code) {
			t.Errorf("%#x (%v) should read as the engine being closed", uint32(code), code)
		}
		if !engineClosed(fmt.Errorf("navigate: %w", code)) {
			t.Errorf("%#x wrapped should still read as the engine being closed", uint32(code))
		}
	}
}

func TestEngineClosedLeavesOtherRefusalsAlone(t *testing.T) {
	alive := []error{
		syscall.Errno(0x80070057), // E_INVALIDARG
		syscall.Errno(0x8000FFFF), // E_UNEXPECTED
		syscall.Errno(0x80004005), // E_FAIL
		errors.New("browser.navigate was called from thread 7, not the webview's thread 9"),
		nil,
	}
	for _, err := range alive {
		if engineClosed(err) {
			t.Errorf("%v should not read as the engine being closed", err)
		}
	}
}
