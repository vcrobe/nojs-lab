//go:build js || wasm

package router

import (
	"github.com/vcrobe/nojs/runtime"
)

// Route defines a path and its component chain (layout hierarchy + page).
type Route struct {
	Path  string
	Chain []ComponentMetadata
}

// ComponentMetadata holds the factory and compile-time type ID for a component.
type ComponentMetadata struct {
	Factory runtime.ComponentFactory
	TypeID  uint32
}
