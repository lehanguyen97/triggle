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

	ctx, err := ui.NewContext(ui.ContextOptions{
		Backend: b,
		Theme:   th,
		Font:    font,
	})
	if err != nil {
		font.Close()
		g.uiFont = nil
		return err
	}
	g.uiCtx = ctx
	return nil
}

func (g *Game) buildUI() {
	if g.uiCtx == nil {
		return
	}
	if g.uiCtx.BeginWindow("Log",
		emath.Rect{X: 10, Y: 10, W: 360, H: 220},
		ui.WindowNoResize|ui.WindowNoClose) {
		g.uiCtx.LogView(g.logBuf, ui.LogViewOpt{
			MaxVisible: 12,
			AutoScroll: true,
			Color:      cmd.Color{R: 235, G: 235, B: 240, A: 255},
		})
		g.uiCtx.EndWindow()
	}
}

func (g *Game) closeUI() {
	if g.uiCtx != nil {
		g.uiCtx.Close()
		g.uiCtx = nil
	}
	if g.uiFont != nil {
		g.uiFont.Close()
		g.uiFont = nil
	}
}
