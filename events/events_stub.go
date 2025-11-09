//go:build !wasm
// +build !wasm

package events

// Stub file for non-WASM builds to allow generated code to compile.
// The actual implementation is in events.go with js/wasm build tags.

// AdaptNoArgEvent is a stub for non-WASM builds.
func AdaptNoArgEvent(handler func()) func() {
	return handler
}
