//go:build js || wasm
// +build js wasm

package main

import (
	"strconv"

	"github.com/vcrobe/nojs/appcomponents"
	"github.com/vcrobe/nojs/appcomponents/admin"
	"github.com/vcrobe/nojs/appcomponents/admin/settings"
	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/router"
	"github.com/vcrobe/nojs/runtime"
)

func main() {
	// 1. Create the Router with path-based mode (clean URLs)
	// For hash-based routing (e.g., #/about), use:
	//   appRouter := router.New(&router.Config{Mode: router.HashMode})
	appRouter := router.New(&router.Config{Mode: router.PathMode})

	// 2. Define routes - map paths to component factories
	appRouter.Handle("/", func(params map[string]string) runtime.Component {
		return &appcomponents.HomePage{}
	})

	appRouter.Handle("/about", func(params map[string]string) runtime.Component {
		return &appcomponents.AboutPage{}
	})

	appRouter.Handle("/admin", func(params map[string]string) runtime.Component {
		return &admin.AdminPage{}
	})

	appRouter.Handle("/admin/settings", func(params map[string]string) runtime.Component {
		return &settings.Settings{}
	})

	appRouter.Handle("/blog/{year}", func(params map[string]string) runtime.Component {
		year, err := strconv.Atoi(params["year"])
		if err != nil {
			console.Warn("Error parsing {year} parameter in route `/blog/{year}`: ", err.Error())
		}

		return &appcomponents.BlogPage{Year: year}
	})

	// Optional: Handle 404 cases
	appRouter.HandleNotFound(func(params map[string]string) runtime.Component {
		return &appcomponents.PageNotFound{}
	})

	// 3. Create the Renderer, passing the router as the NavigationManager
	renderer := runtime.NewRenderer(appRouter, "#app")

	// 4. Define the callback that the router will call when navigation occurs
	onRouteChange := func(newComponent runtime.Component, key string) {
		// Tell the renderer which component to render with its key (path)
		renderer.SetCurrentComponent(newComponent, key)
		// Trigger the actual render
		renderer.ReRender()
	}

	// 5. Start the Router explicitly - this reads the initial URL and triggers first render
	if err := appRouter.Start(onRouteChange); err != nil {
		panic("Error starting router: " + err.Error())
	}

	// Keep the Go program running
	select {}
}
