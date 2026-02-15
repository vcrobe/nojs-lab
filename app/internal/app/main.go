//go:build js || wasm

package main

import (
	"github.com/vcrobe/app/internal/app/components/pages"
	"github.com/vcrobe/app/internal/app/components/pages/admin"
	"github.com/vcrobe/app/internal/app/components/pages/admin/layouts"
	"github.com/vcrobe/app/internal/app/components/pages/admin/settings"
	"github.com/vcrobe/app/internal/app/components/pages/admin/users"
	sharedlayouts "github.com/vcrobe/app/internal/app/components/shared/layouts"
	"github.com/vcrobe/app/internal/app/context"
	"github.com/vcrobe/nojs/console"
	"github.com/vcrobe/nojs/core"
	"github.com/vcrobe/nojs/router"
	"github.com/vcrobe/nojs/runtime"
)

// TypeID constants for route components
const (
	MainLayout_TypeID   uint32 = 0x8F22A1BC
	AdminLayout_TypeID  uint32 = 0x7E11B2AD
	HomePage_TypeID     uint32 = 0x6C00C9FE
	AboutPage_TypeID    uint32 = 0x5D11D8CF
	AdminPage_TypeID    uint32 = 0x4E22E7B0
	SettingsPage_TypeID uint32 = 0x3F33F681
	BlogPage_TypeID     uint32 = 0x2E44F592
	UsersPage_TypeID    uint32 = 0x1D55E483
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

	// Define all routes with layout chains and TypeIDs
	routerEngine.RegisterRoutes([]router.Route{
		{
			Path: "/",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return &pages.HomePage{MainLayoutCtx: mainLayoutCtx} },
					TypeID:  HomePage_TypeID,
				},
			},
		},
		{
			Path: "/about",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return &pages.AboutPage{} },
					TypeID:  AboutPage_TypeID,
				},
			},
		},
		{
			Path: "/blog/{year}",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component {
						year := 2026 // Default, would be extracted from URL in real implementation
						return &pages.BlogPage{Year: year}
					},
					TypeID: BlogPage_TypeID,
				},
			},
		},
		{
			Path: "/admin",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return &admin.AdminPage{} },
					TypeID:  AdminPage_TypeID,
				},
			},
		},
		{
			Path: "/admin/settings",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return &settings.Settings{} },
					TypeID:  SettingsPage_TypeID,
				},
			},
		},
		{
			Path: "/admin/users",
			Chain: []router.ComponentMetadata{
				{
					Factory: func() runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func() runtime.Component { return &users.Users{} },
					TypeID:  UsersPage_TypeID,
				},
			},
		},
	})

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
