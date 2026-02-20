//go:build js || wasm

package main

// Component TypeIDs uniquely identify each component type for the router pivot algorithm.
// Each value must be unique within the application. Use any positive integer.
const (
	// Layouts
	MainLayout_TypeID uint32 = 100

	// Pages
	LandingPage_TypeID      uint32 = 200
	CounterPage_TypeID      uint32 = 300
	LifecyclePage_TypeID    uint32 = 400
	FormsPage_TypeID        uint32 = 500
	ConditionalsPage_TypeID uint32 = 600
	ListsPage_TypeID        uint32 = 700
	SlotsPage_TypeID        uint32 = 800
	RouterParamsPage_TypeID uint32 = 900

	// Shared
	PageNotFound_TypeID uint32 = 1000
	RouterLink_TypeID   uint32 = 10000
)
