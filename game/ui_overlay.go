package main

import (
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

func (g *Game) initUI() error {
	b := g.backend
	font, err := text.OpenFont(b, defaultUIFontPath())
	if err != nil {
		return err
	}
	g.uiFont = font

	th := theme.DefaultTheme()
	th.BodyPx = int32(float32(th.BodyPx) * g.dpiScale)
	th.TitlePx = int32(float32(th.TitlePx) * g.dpiScale)

	g.uiLog = &ui.LogView{
		MaxVisible: 12,
		AutoScroll: true,
		Color:      cmd.Color{R: 235, G: 235, B: 240, A: 255},
	}
	g.uiName = &ui.TextInput{Value: "hello", MaxBytes: 128}
	g.uiMsg = &ui.TextInput{Value: "message", MaxBytes: 128}
	hud := &ui.Window{
		Title: "HUD",
		Pos:   emath.Vec2{10, 10},
		Width: 360,
		Flags: ui.WindowNoResize | ui.WindowNoClose,
		Child: &ui.Padding{
			Insets: th.Padding,
			Child: &ui.Column{
				Kids: []ui.Node{g.uiLog, g.uiName, g.uiMsg},
			},
		},
	}

	app, err := ui.NewApp(ui.AppOptions{
		Backend: b,
		Theme:   th,
		Font:    font,
	})
	if err != nil {
		font.Close()
		g.uiFont = nil
		return err
	}
	g.uiApp = app
	g.uiApp.SetRoot(hud)
	return nil
}

func (g *Game) buildUI() {
	if g.uiApp == nil {
		return
	}
	g.uiLog.Lines = g.logBuf
}

func (g *Game) closeUI() {
	if g.uiApp != nil {
		g.uiApp.Close()
		g.uiApp = nil
	}
	if g.uiFont != nil {
		g.uiFont.Close()
		g.uiFont = nil
	}
}
