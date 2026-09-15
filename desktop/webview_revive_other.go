//go:build !windows

package main

// WebView2 is Windows; WebKitGTK and WKWebView do not report a browser
// process the app could rebuild. Nothing to install.
func installWebviewRevival(*App) {}
