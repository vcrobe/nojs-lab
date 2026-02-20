module github.com/vcrobe/app

go 1.25.1

replace github.com/vcrobe/nojs => ../nojs

replace github.com/vcrobe/nojs-router => ../router

require (
	github.com/vcrobe/nojs v0.0.0-00010101000000-000000000000
	github.com/vcrobe/nojs-router v0.0.0-00010101000000-000000000000
)
