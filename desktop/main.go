//go:build windows

package main

import (
	"context"
	"embed"
	"io/fs"
	"log"

	"github.com/joseluisald/verman/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS
var Version = "dev"

type App struct {
	ctx     context.Context
	service *desktop.Service
}

func (a *App) startup(ctx context.Context)                   { a.ctx = ctx; a.service = desktop.New(ctx) }
func (a *App) Snapshot(dir string) (desktop.Snapshot, error) { return a.service.Snapshot(dir) }
func (a *App) Catalog(lang string, history, refresh, offline bool) (desktop.Catalog, error) {
	return a.service.Catalog(lang, history, refresh, offline)
}
func (a *App) Start(request desktop.Request) (string, error) { return a.service.Start(request) }
func (a *App) Operations() []desktop.Operation               { return a.service.Operations() }
func (a *App) Cancel(id string) error                        { return a.service.Cancel(id) }
func (a *App) PickProject() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Selecionar pasta do projeto"})
}
func main() {
	files, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	app := &App{}
	err = wails.Run(&options.App{Title: "Verman", Width: 1280, Height: 850, MinWidth: 940, MinHeight: 680,
		BackgroundColour: &options.RGBA{R: 248, G: 249, B: 251, A: 255},
		AssetServer:      &assetserver.Options{Assets: files}, OnStartup: app.startup,
		OnShutdown: func(context.Context) { app.service.Close() },
		OnBeforeClose: func(ctx context.Context) bool {
			if app.service.Busy() {
				wruntime.MessageDialog(ctx, wruntime.MessageDialogOptions{Type: wruntime.InfoDialog, Title: "Operação em andamento", Message: "Cancele ou aguarde a operação antes de fechar o Verman."})
				return true
			}
			return false
		},
		Windows: &windows.Options{WebviewIsTransparent: false, WindowIsTranslucent: false}, Bind: []interface{}{app}})
	if err != nil {
		log.Fatal(err)
	}
}
