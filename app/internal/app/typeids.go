//go:build js || wasm

package main

// Component TypeIDs uniquely identify each component (page) type for the router pivot algorithm.
// Each value must be unique within the application. Use any positive integer.
const (
	// Layouts
	MainLayout_TypeID  uint32 = 100
	AdminLayout_TypeID uint32 = 200

	// Pages
	HomePage_TypeID     uint32 = 300
	AboutPage_TypeID    uint32 = 400
	AdminPage_TypeID    uint32 = 500
	SettingsPage_TypeID uint32 = 600
	UsersPage_TypeID    uint32 = 700
	BlogPage_TypeID     uint32 = 800
	PageNotFound_TypeID uint32 = 900

	// Shared components
	RouterLink_TypeID uint32 = 10000
)
