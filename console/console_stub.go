//go:build !wasm
// +build !wasm

package console

// Stub file for non-WASM builds to allow generated code to compile.
// The actual implementation is in console.go with js/wasm build tags.

// Log is a no-op in non-WASM builds.
func Log(args ...any) {
	// No-op for tests
}

// Warn is a no-op in non-WASM builds.
func Warn(args ...any) {
	// No-op for tests
}

// Error is a no-op in non-WASM builds.
func Error(args ...any) {
	// No-op for tests
}
