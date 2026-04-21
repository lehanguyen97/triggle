package ui

import (
	"fmt"

	"triggle/engine/geom"
	"triggle/engine/ui/theme"
)

// WindowOpt are bit flags for BeginWindow.
type WindowOpt uint32

const (
	WindowNoTitle WindowOpt = 1 << iota
	WindowNoFrame
	WindowNoResize
	WindowNoClose
)

type windowState struct {
	Pos        geom.Vec2
	GrabOffset geom.Vec2
	Dragging   bool
	Inited     bool
}

func (c *Context) emitWindowTitle(winID WidgetID, title string, win geom.Rect, titleH float32) {
	if c == nil || c.uiFont == nil || title == "" {
		return
	}
	tx := win.X + 6
	ty := win.Y + titleH*0.5
	if sz, err := c.uiFont.MeasureLine(title); err == nil {
		ty -= sz[1] * 0.5
	} else {
		ty -= c.uiFont.Metrics().Ascent * 0.5
	}
	cacheKey := fmt.Sprintf("wintitle:%d", winID)
	c.DrawText(cacheKey, title, tx, ty, c.theme.Colors[theme.ColorTitleText])
}

// BeginWindow opens a draggable window with title bar and clips content to the client area.
func (c *Context) BeginWindow(title string, rect geom.Rect, opt WindowOpt) bool {
	if c == nil {
		return false
	}
	c.idStack = append(c.idStack, "win:"+title)
	id := c.hashID()
	ws := StateOf[windowState](c, id)
	if !ws.Inited {
		ws.Pos = geom.Vec2{rect.X, rect.Y}
		ws.Inited = true
	}

	titleH := float32(0)
	if opt&WindowNoTitle == 0 {
		titleH = float32(c.theme.TitleHeight)
	}

	win := geom.Rect{X: ws.Pos[0], Y: ws.Pos[1], W: rect.W, H: rect.H}
	titleBar := geom.Rect{X: win.X, Y: win.Y, W: win.W, H: titleH}

	mx, my := c.in.MousePos[0], c.in.MousePos[1]
	if win.Contains(mx, my) || ws.Dragging {
		c.MarkHover()
	}

	if opt&WindowNoTitle == 0 {
		if titleBar.Contains(mx, my) && c.in.MousePressed&MouseLeft != 0 {
			ws.Dragging = true
			ws.GrabOffset = geom.Vec2{mx - ws.Pos[0], my - ws.Pos[1]}
			c.SetActive(id)
		}
		if ws.Dragging && c.in.MouseDown&MouseLeft != 0 {
			ws.Pos = geom.Vec2{mx - ws.GrabOffset[0], my - ws.GrabOffset[1]}
			win = geom.Rect{X: ws.Pos[0], Y: ws.Pos[1], W: rect.W, H: rect.H}
			titleBar = geom.Rect{X: win.X, Y: win.Y, W: win.W, H: titleH}
		}
		if c.in.MouseReleased&MouseLeft != 0 {
			ws.Dragging = false
			c.ClearActive(id)
		}
	}

	if opt&WindowNoFrame == 0 {
		c.enc.QuadSolid(win, c.theme.Colors[theme.ColorWindowBG])
	}

	if opt&WindowNoTitle == 0 {
		c.enc.QuadSolid(titleBar, c.theme.Colors[theme.ColorTitleBG])
		c.emitWindowTitle(id, title, win, titleH)
	}

	pad := c.theme.Padding
	content := geom.Rect{
		X: win.X + float32(pad.Left),
		Y: win.Y + titleH + float32(pad.Top),
		W: win.W - float32(pad.Left+pad.Right),
		H: win.H - titleH - float32(pad.Top+pad.Bottom),
	}

	c.pushLayout(content)
	c.enc.PushClip(content)
	return true
}

// EndWindow closes the innermost window from BeginWindow.
func (c *Context) EndWindow() {
	if c == nil {
		return
	}
	c.enc.PopClip()
	c.popLayout()
	if len(c.idStack) > 0 {
		c.idStack = c.idStack[:len(c.idStack)-1]
	}
}
