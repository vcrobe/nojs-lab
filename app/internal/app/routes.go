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
	"github.com/vcrobe/nojs/router"
	"github.com/vcrobe/nojs/runtime"
)

func registerRoutes(routerEngine *router.Engine, mainLayout *sharedlayouts.MainLayout, mainLayoutCtx *context.MainLayoutCtx) {
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
}
