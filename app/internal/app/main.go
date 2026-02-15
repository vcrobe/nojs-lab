//go:build js || wasm

package main

import (
	sharedlayouts "github.com/vcrobe/app/internal/app/components/shared/layouts"
	"github.com/vcrobe/app/internal/app/context"
	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/core"
	"github.com/vcrobe/nojs/router"
	"github.com/vcrobe/nojs/runtime"
)

func main() {
	// Create shared layout context
	mainLayoutCtx := &context.MainLayoutCtx{
		Title: "My App",
	}

	// Create persistent main layout instance (app shell)
	mainLayout := &sharedlayouts.MainLayout{
		MainLayoutCtx: mainLayoutCtx,
	}

	// Create the router engine first (it will be passed as navigation manager to renderer)
	routerEngine := router.NewEngine(nil)

	// Create the renderer with the engine as the navigation manager
	renderer := runtime.NewRenderer(routerEngine, "#app")

	// Set the renderer on the engine so it can render components
	routerEngine.SetRenderer(renderer)

	// Register routes with their components and layouts
	registerRoutes(routerEngine, mainLayout, mainLayoutCtx)

	// Create AppShell to wrap the router's page rendering
	appShell := core.NewAppShell(mainLayout)
	renderer.SetCurrentComponent(appShell, "app-shell")
	renderer.ReRender()

	// Initialize the router with a callback to update AppShell when navigation occurs
	err := routerEngine.Start(func(chain []runtime.Component, key string) {
		appShell.SetPage(chain, key)
	})
	if err != nil {
		console.Error("Failed to start router:", err.Error())
		panic(err)
	}

	// Keep the Go program running
	select {}
}
