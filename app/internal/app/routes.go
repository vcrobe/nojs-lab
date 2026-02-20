//go:build js || wasm

package main

import (
	"strconv"

	"github.com/vcrobe/app/internal/app/components/pages"
	"github.com/vcrobe/app/internal/app/components/pages/admin"
	"github.com/vcrobe/app/internal/app/components/pages/admin/layouts"
	"github.com/vcrobe/app/internal/app/components/pages/admin/settings"
	"github.com/vcrobe/app/internal/app/components/pages/admin/users"
	sharedlayouts "github.com/vcrobe/app/internal/app/components/shared/layouts"
	"github.com/vcrobe/app/internal/app/context"
	router "github.com/vcrobe/nojs-router"
	"github.com/vcrobe/nojs/runtime"
)

func registerRoutes(routerEngine *router.Engine, mainLayout *sharedlayouts.MainLayout, mainLayoutCtx *context.MainLayoutCtx) {
	// Define all routes with layout chains and TypeIDs
	routerEngine.RegisterRoutes([]router.Route{
		{
			Path: "/",
			Chain: []router.ComponentMetadata{
				{
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return &pages.HomePage{MainLayoutCtx: mainLayoutCtx} },
					TypeID:  HomePage_TypeID,
				},
			},
		},
		{
			Path: "/about",
			Chain: []router.ComponentMetadata{
				{
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return &pages.AboutPage{} },
					TypeID:  AboutPage_TypeID,
				},
			},
		},
		{
			Path: "/blog/{year}",
			Chain: []router.ComponentMetadata{
				{
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component {
						year := 2026 // Default value
						if yearStr, ok := params["year"]; ok {
							if parsed, err := strconv.Atoi(yearStr); err == nil {
								year = parsed
							}
						}
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
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return &admin.AdminPage{} },
					TypeID:  AdminPage_TypeID,
				},
			},
		},
		{
			Path: "/admin/settings",
			Chain: []router.ComponentMetadata{
				{
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return &settings.Settings{} },
					TypeID:  SettingsPage_TypeID,
				},
			},
		},
		{
			Path: "/admin/users",
			Chain: []router.ComponentMetadata{
				{
					Factory: func(params map[string]string) runtime.Component { return mainLayout },
					TypeID:  MainLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return layouts.NewAdminLayout() },
					TypeID:  AdminLayout_TypeID,
				},
				{
					Factory: func(params map[string]string) runtime.Component { return &users.Users{} },
					TypeID:  UsersPage_TypeID,
				},
			},
		},
	})
}
