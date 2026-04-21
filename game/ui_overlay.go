package main

import (
	"triggle/engine/geom"
	"triggle/engine/text"
	"triggle/engine/ui"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

func (g *Game) initUI() error {
	b := g.rnd.GPU()
	font, err := text.OpenFontFile(b, defaultUIFontPath())
	if err != nil {
		return err
	}
	g.uiFontRes = font
	face, err := font.NewFace(text.FaceOptions{PtSize: 16, DPIScale: g.dpiScale})
	if err != nil {
		g.uiFontRes = nil
		return err
	}
	g.uiFace = face
	uiFont := ui.NewUIFont(face)
	g.uiFont = uiFont
	ctx, err := ui.NewContext(ui.ContextOptions{
		Backend:  b,
		Theme:    theme.DefaultTheme(),
		UIFont:   uiFont,
		DPIScale: g.dpiScale,
	})
	if err != nil {
		uiFont.Close()
		g.uiFont = nil
		g.uiFace = nil
		g.uiFontRes = nil
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
		geom.Rect{X: 10, Y: 10, W: 360, H: 220},
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
	if g.uiFace != nil {
		g.uiFace.Close()
		g.uiFace = nil
	}
	if g.uiFontRes != nil {
		g.uiFontRes.Close()
		g.uiFontRes = nil
	}
}
