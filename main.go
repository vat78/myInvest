package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := newApp()
	if err := wails.Run(&options.App{
		Title: "myInvest", Width: 1024, Height: 720, MinWidth: 640, MinHeight: 480,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   app.startup, OnDomReady: app.domReady, OnShutdown: app.shutdown,
		Bind: []interface{}{app, app.currencies, app.portfolios, app.cashFlows, app.overview, app.quotes, app.allocation},
	}); err != nil {
		log.Printf("Не удалось запустить myInvest: %v", err)
	}
}
