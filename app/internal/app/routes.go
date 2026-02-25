//go:build js || wasm

package main

import (
	"github.com/ForgeLogic/app/internal/app/components/pages"
	sharedlayouts "github.com/ForgeLogic/app/internal/app/components/shared/layouts"
	"github.com/ForgeLogic/app/internal/app/context"
	router "github.com/ForgeLogic/nojs-router"
	"github.com/ForgeLogic/nojs/runtime"
)

123

func registerRoutes(routerEngine *router.Engine, mainLayout *sharedlayouts.MainLayout, mainLayoutCtx *context.MainLayoutCtx) {
	_ = mainLayoutCtx // reserved for future use

	ml := func(p map[string]string) runtime.Component { return mainLayout }

	routerEngine.RegisterRoutes([]router.Route{
		{
			Path: "/",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.LandingPage{} }, TypeID: LandingPage_TypeID},
			},
		},
		{
			Path: "/demo/counter",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.CounterPage{} }, TypeID: CounterPage_TypeID},
			},
		},
		{
			Path: "/demo/lifecycle",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.LifecyclePage{} }, TypeID: LifecyclePage_TypeID},
			},
		},
		{
			Path: "/demo/forms",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.FormsPage{} }, TypeID: FormsPage_TypeID},
			},
		},
		{
			Path: "/demo/conditionals",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.ConditionalsPage{} }, TypeID: ConditionalsPage_TypeID},
			},
		},
		{
			Path: "/demo/lists",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.ListsPage{} }, TypeID: ListsPage_TypeID},
			},
		},
		{
			Path: "/demo/slots",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.SlotsPage{} }, TypeID: SlotsPage_TypeID},
			},
		},
		{
			Path: "/demo/router/{id}",
			Chain: []router.ComponentMetadata{
				{Factory: ml, TypeID: MainLayout_TypeID},
				{Factory: func(p map[string]string) runtime.Component { return &pages.RouterParamsPage{ID: p["id"]} }, TypeID: RouterParamsPage_TypeID},
			},
		},
	})
}
