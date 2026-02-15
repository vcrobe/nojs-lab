//go:build js || wasm

package users

import (
	"github.com/vcrobe/nojs/runtime"
)

type Users struct {
	runtime.ComponentBase
}
