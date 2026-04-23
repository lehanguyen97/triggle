package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/theme"
)

// Window is a titled, draggable container with a padded content area.
type Window struct {
	BaseNode
	parent Node
	Title  string
	Pos    emath.Vec2
	Width  int32   // 0 = shrink to child
	Child  Node
	Flags  WindowOpt

	Dragging   bool
	grabX, grabY int32

	outerSize Size
	lastChild Size
}

// Children implements Node.
func (w *Window) Children() []Node {
	if w.Child == nil {
		return nil
	}
	return []Node{w.Child}
}

func (w *Window) titleHeight() int32 {
	if w.app == nil {
		return 0
	}
	if w.Flags&WindowNoTitle != 0 {
		return 0
	}
	return w.app.theme.TitleHeight
}

// titleBarRect returns the title bar in absolute coords, if any.
func (w *Window) titleBarRect() (x, y, th int32, ok bool) {
	th = w.titleHeight()
	if th == 0 {
		return 0, 0, 0, false
	}
	return w.rect.X, w.rect.Y, th, true
}

// Measure implements Node.
func (w *Window) Measure(c Constraints) Size {
	if w.app == nil {
		return Size{}
	}
	cn := normConstraints(c)
	pad := w.app.theme.Padding
	th := w.titleHeight()
	maxInnerW := cn.MaxW - pad.Left - pad.Right
	if maxInnerW < 0 {
		maxInnerW = 0
	}
	if w.Width > 0 {
		maxInnerW = w.Width - pad.Left - pad.Right
		if maxInnerW < 0 {
			maxInnerW = 0
		}
	}
	innerH := cn.MaxH - th - pad.Top - pad.Bottom
	if innerH < 0 {
		innerH = 0
	}
	var chSize Size
	if w.Child != nil {
		chSize = w.Child.Measure(Constraints{MaxW: maxInnerW, MinW: 0, MaxH: innerH, MinH: 0})
	}
	w.lastChild = chSize
	outerW := w.Width
	if outerW == 0 {
		outerW = chSize.W + pad.Left + pad.Right
	}
	outerH := th + pad.Top + chSize.H + pad.Bottom
	w.outerSize = Size{W: outerW, H: outerH}
	return w.outerSize
}

// Place implements Node. Root passes viewport; window uses Pos for (X,Y).
func (w *Window) Place(outer emath.Rect) {
	w.rect = emath.Rect{
		X: int32(w.Pos[0]), Y: int32(w.Pos[1]),
		W: w.outerSize.W, H: w.outerSize.H,
	}
	if w.app == nil {
		return
	}
	pad := w.app.theme.Padding
	th := w.titleHeight()
	if w.Child != nil {
		content := emath.Rect{
			X: w.rect.X + pad.Left,
			Y: w.rect.Y + th + pad.Top,
			W: w.rect.W - pad.Left - pad.Right,
			H: w.lastChild.H,
		}
		if content.W < 0 {
			content.W = 0
		}
		if content.H < 0 {
			content.H = 0
		}
		w.Child.Place(content)
	}
}

// Paint implements Node.
func (w *Window) Paint(pc *PaintCtx) {
	if w.app == nil {
		return
	}
	if w.Flags&WindowNoFrame == 0 {
		pc.Enc.QuadSolid(w.rect, w.app.theme.Colors[theme.ColorWindowBG])
	}
	th := w.titleHeight()
	if th > 0 {
		tb := emath.Rect{X: w.rect.X, Y: w.rect.Y, W: w.rect.W, H: th}
		pc.Enc.QuadSolid(tb, w.app.theme.Colors[theme.ColorTitleBG])
		w.drawTitle(pc, w.rect, th)
	}
	pad := w.app.theme.Padding
	clipH := w.rect.H - th - pad.Top - pad.Bottom
	if clipH < 0 {
		clipH = 0
	}
	clipR := emath.Rect{
		X: w.rect.X + pad.Left,
		Y: w.rect.Y + th + pad.Top,
		W: w.rect.W - pad.Left - pad.Right,
		H: clipH,
	}
	pc.Enc.PushClip(clipR)
	if w.Child != nil {
		w.Child.Paint(pc)
	}
	pc.Enc.PopClip()
}

func (w *Window) drawTitle(pc *PaintCtx, win emath.Rect, titleH int32) {
	if w.Title == "" || pc.Font == nil {
		return
	}
	th := w.app.theme
	px := th.TitlePx
	tx := win.X + 6
	ty := win.Y + titleH/2
	sz := pc.Font.Measure(w.Title, px)
	if sz[1] > 0 {
		ty -= int32(sz[1] * 0.5)
	} else {
		ty -= pc.Font.Metrics(px).Ascent / 2
	}
	col := th.Colors[theme.ColorTitleText]
	pc.Font.Draw(pc.Enc, w.Title, tx, ty, px,
		text.Color{R: col.R, G: col.G, B: col.B, A: col.A})
}

// Event implements Node.
func (w *Window) Event(_ *Event, _ *EventCtx) bool { return false }
