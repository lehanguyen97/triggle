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
	// WindowAutoSizeY ignores rect.H and grows the window to fit the
	// vertical pen at EndWindow. Mirrors ImGui's AlwaysAutoResize on Y.
	WindowAutoSizeY
	// WindowAutoSizeW ignores rect.W and grows the window to fit the
	// widest reserved row at EndWindow.
	WindowAutoSizeW
)

type windowState struct {
	PosX        int32
	PosY        int32
	GrabOffX    int32
	GrabOffY    int32
	Dragging    bool
	Initialized bool
	// LastOuterH/W is the rect we ended with last frame (post-autosize).
	// Tools that stack windows can read it via Context.WindowSize(title).
	LastOuterH int32
	LastOuterW int32
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

// BeginWindow opens a draggable window with title bar and clips content to
// the client area.
//
// When opt has WindowAutoSizeY the rect.H is ignored: BeginWindow emits the
// bg quad and clip push with placeholder Y-extents, EndWindow patches them
// to the measured cursor + bottom padding. Width is always taken from rect.W
// (auto-W isn't supported).
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

	pad := c.theme.Padding
	autoH := opt&WindowAutoSizeY != 0

	// Provisional outer rect — H gets patched at End for autoH. Use a
	// minimal placeholder height (chrome + 1px) so a one-frame flash
	// doesn't show a giant box if patching is somehow skipped.
	outerH := rect.H
	if autoH {
		outerH = titleH + pad.Top + pad.Bottom
	}
	win := emath.Rect{X: ws.PosX, Y: ws.PosY, W: rect.W, H: outerH}
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
			win.X = ws.PosX
			win.Y = ws.PosY
			titleBar.X = win.X
			titleBar.Y = win.Y
		}
		if c.in.MouseReleased&MouseLeft != 0 {
			ws.Dragging = false
			c.ClearActive(id)
		}
	}

	bgIdx := -1
	if opt&WindowNoFrame == 0 {
		bgIdx = c.enc.CmdIndex()
		c.enc.QuadSolid(win, c.theme.Colors[theme.ColorWindowBG])
	}

	if opt&WindowNoTitle == 0 {
		c.enc.QuadSolid(titleBar, c.theme.Colors[theme.ColorTitleBG])
		c.emitWindowTitle(id, title, win, titleH)
	}

	contentW := win.W - pad.Left - pad.Right
	contentY := win.Y + titleH + pad.Top
	contentH := win.H - titleH - pad.Top - pad.Bottom
	if autoH {
		contentH = 1<<30 // pen extends; gets clipped by patched clipCmd at End.
	}
	content := emath.Rect{
		X: win.X + pad.Left,
		Y: contentY,
		W: contentW,
		H: contentH,
	}

	clipIdx := c.enc.CmdIndex()
	c.enc.PushClip(content)

	c.pushLayout(layoutFrame{
		rect:    content,
		spacing: c.theme.Spacing,
		autoH:   autoH,
		bgCmd:   bgIdx,
		clipCmd: clipIdx,
		winX:    win.X,
		winY:    win.Y,
		winW:    win.W,
		titleH:  titleH,
		padBot:  pad.Bottom,
	})
	return true
}

// EndWindow closes the innermost window from BeginWindow. For auto-sized
// windows, rewrites the bg quad and clip push to the measured outer rect
// and stashes the result on windowState so callers can stack the next
// window via Context.WindowSize(title).
func (c *Context) EndWindow() {
	if c == nil {
		return
	}
	c.enc.PopClip()
	frame, ok := c.popLayout()

	// Window id was hashed by BeginWindow against c.idStack; it's still
	// the top of the stack here, so re-hash to find the windowState.
	id := c.hashID()
	ws, _ := c.states[id].(*windowState)

	outerH := frame.rect.H
	if ok && frame.autoH {
		contentH := frame.cursorY - frame.startY
		if contentH < 0 {
			contentH = 0
		}
		topGap := frame.startY - (frame.winY + frame.titleH) // == padTop
		outerH = frame.titleH + topGap + contentH + frame.padBot
		outerRect := emath.Rect{X: frame.winX, Y: frame.winY, W: frame.winW, H: outerH}
		if frame.bgCmd >= 0 {
			c.enc.PatchRect(frame.bgCmd, outerRect)
		}
		clipRect := emath.Rect{
			X: frame.rect.X,
			Y: frame.rect.Y,
			W: frame.rect.W,
			H: contentH,
		}
		c.enc.PatchRect(frame.clipCmd, clipRect)
	}
	if ws != nil {
		ws.LastOuterH = outerH
		ws.LastOuterW = frame.winW
	}

	if len(c.idStack) > 0 {
		c.idStack = c.idStack[:len(c.idStack)-1]
	}
}

// WindowSize returns the outer (W, H) drawn for the named window on the
// previous frame, or zero if it hasn't been seen yet. Use this to stack
// auto-sized windows: layoutY += g.uiCtx.WindowSize("Log").H + gap.
func (c *Context) WindowSize(title string) emath.Rect {
	if c == nil {
		return emath.Rect{}
	}
	c.idStack = append(c.idStack, "win:"+title)
	id := c.hashID()
	c.idStack = c.idStack[:len(c.idStack)-1]
	if ws, ok := c.states[id].(*windowState); ok {
		return emath.Rect{W: ws.LastOuterW, H: ws.LastOuterH}
	}
	return emath.Rect{}
}
