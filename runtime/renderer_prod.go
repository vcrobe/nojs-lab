//go:build (js || wasm) && !dev
// +build js wasm
// +build !dev

package runtime

import "fmt"

// callOnInit invokes the OnInit lifecycle method in production mode.
// In production mode, panics are recovered and logged to prevent application crashes.
func (r *RendererImpl) callOnInit(initializer Initializer, key string) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("ERROR: OnInit panic in component %s: %v\n", key, rec)
			// In a real production environment, this could be sent to an error tracking service
		}
	}()
	initializer.OnInit()
}

// callOnParametersSet invokes the OnParametersSet lifecycle method in production mode.
// In production mode, panics are recovered and logged to prevent application crashes.
func (r *RendererImpl) callOnParametersSet(receiver ParameterReceiver, key string) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("ERROR: OnParametersSet panic in component %s: %v\n", key, rec)
			// In a real production environment, this could be sent to an error tracking service
		}
	}()
	receiver.OnPropertiesSet()
}

// callOnDestroy invokes the OnDestroy lifecycle method in production mode.
// In production mode, panics are recovered and logged to prevent application crashes.
func (r *RendererImpl) callOnDestroy(cleaner Cleaner, key string) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("ERROR: OnDestroy panic in component %s: %v\n", key, rec)
			// In a real production environment, this could be sent to an error tracking service
		}
	}()
	cleaner.OnDestroy()
}
