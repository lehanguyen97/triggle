package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
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
	PosX        int32
	PosY        int32
	GrabOffX    int32
	GrabOffY    int32
	Dragging    bool
	Initialized bool
}

func (c *Context) emitWindowTitle(_ WidgetID, title string, win emath.Rect, titleH int32) {
	if c == nil || c.font == nil || title == "" {
		return
	}
	px := c.theme.TitlePx
	tx := win.X + 6
	ty := win.Y + titleH/2
	sz := c.font.Measure(title, px)
	if sz[1] > 0 {
		ty -= int32(sz[1] * 0.5)
	} else {
		ty -= c.font.Metrics(px).Ascent / 2
	}
	col := c.theme.Colors[theme.ColorTitleText]
	c.font.Draw(&c.enc, title, tx, ty, px,
		text.Color{R: col.R, G: col.G, B: col.B, A: col.A})
}

// BeginWindow opens a draggable window with title bar and clips content to the client area.
func (c *Context) BeginWindow(title string, rect emath.Rect, opt WindowOpt) bool {
	if c == nil {
		return false
	}
	c.idStack = append(c.idStack, "win:"+title)
	id := c.hashID()
	ws := StateOf[windowState](c, id)
	if !ws.Initialized {
		ws.PosX = rect.X
		ws.PosY = rect.Y
		ws.Initialized = true
	}

	titleH := int32(0)
	if opt&WindowNoTitle == 0 {
		titleH = c.theme.TitleHeight
	}

	win := emath.Rect{X: ws.PosX, Y: ws.PosY, W: rect.W, H: rect.H}
	titleBar := emath.Rect{X: win.X, Y: win.Y, W: win.W, H: titleH}

	mx, my := int32(c.in.MousePos[0]), int32(c.in.MousePos[1])
	if win.Contains(mx, my) || ws.Dragging {
		c.MarkHover()
	}

	if opt&WindowNoTitle == 0 {
		if titleBar.Contains(mx, my) && c.in.MousePressed&MouseLeft != 0 {
			ws.Dragging = true
			ws.GrabOffX = mx - ws.PosX
			ws.GrabOffY = my - ws.PosY
			c.SetActive(id)
		}
		if ws.Dragging && c.in.MouseDown&MouseLeft != 0 {
			ws.PosX = mx - ws.GrabOffX
			ws.PosY = my - ws.GrabOffY
			win = emath.Rect{X: ws.PosX, Y: ws.PosY, W: rect.W, H: rect.H}
			titleBar = emath.Rect{X: win.X, Y: win.Y, W: win.W, H: titleH}
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
	content := emath.Rect{
		X: win.X + pad.Left,
		Y: win.Y + titleH + pad.Top,
		W: win.W - pad.Left - pad.Right,
		H: win.H - titleH - pad.Top - pad.Bottom,
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
