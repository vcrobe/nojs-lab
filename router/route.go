//go:build js || wasm

package router

import (
	"github.com/ForgeLogic/nojs/runtime"
)

// Route defines a path and its component chain (layout hierarchy + page).
type Route struct {
	Path  string
	Chain []ComponentMetadata
}

// ComponentMetadata holds the factory and compile-time type ID for a component.
type ComponentMetadata struct {
	Factory ComponentFactory
	TypeID  uint32
}

// ComponentFactory creates a new instance of a component.
// Used by the router to instantiate components for routes.
// The params map contains URL path parameters extracted from route patterns (e.g., {year} -> "2026").
type ComponentFactory func(params map[string]string) runtime.Component
