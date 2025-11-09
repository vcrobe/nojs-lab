package multiline

import "github.com/vcrobe/nojs/runtime"

// MultilineText is a test component for verifying multi-line HTML tag support.
type MultilineText struct {
	runtime.ComponentBase
	Title   string
	Message string
	Count   int
}
