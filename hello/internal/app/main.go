//go:build js || wasm

package main

import (
	"github.com/ForgeLogic/nojs/runtime"

	"github.com/ForgeLogic/hello/internal/app/components"
)

func main() {
	// Create the Counter component instance
	counterInstance := &components.Counter{Count: 0}

	// Create the renderer
	renderer := runtime.NewRenderer(nil, "#app")

	// Set the component and render
	renderer.SetCurrentComponent(counterInstance, "counter")
	renderer.ReRender()

	// Keep the Go program running
	select {}
}
