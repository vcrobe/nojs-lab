//go:build js && wasm

package appcomponents

import (
	"github.com/vcrobe/nojs/runtime"
)

// InlineConditionals demonstrates the inline conditional expressions feature.
type InlineConditionals struct {
	runtime.ComponentBase

	HasError   bool
	IsSaving   bool
	IsReady    bool
	IsActive   bool
	IsLarge    bool
	IsLocked   bool
	IsRequired bool
}
