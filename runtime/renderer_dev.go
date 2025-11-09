//go:build (js || wasm) && dev
// +build js wasm
// +build dev

package runtime

// callOnInit invokes the OnInit lifecycle method in development mode.
// In dev mode, panics propagate to aid debugging and fast failure.
func (r *RendererImpl) callOnInit(initializer Initializer, key string) {
	initializer.OnInit()
}

// callOnParametersSet invokes the OnParametersSet lifecycle method in development mode.
// In dev mode, panics propagate to aid debugging and fast failure.
func (r *RendererImpl) callOnParametersSet(receiver ParameterReceiver, key string) {
	receiver.OnPropertiesSet()
}

// callOnDestroy invokes the OnDestroy lifecycle method in development mode.
// In dev mode, panics propagate to aid debugging and fast failure.
func (r *RendererImpl) callOnDestroy(cleaner Cleaner, key string) {
	cleaner.OnDestroy()
}
