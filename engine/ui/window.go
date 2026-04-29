package ui

import (
	"triggle/engine/emath"
)

// Window is a titled, draggable container with a padded content area.
type Window struct {
	BaseNode
	Title string
	Pos   emath.Vec2
	Width float32 // 0 = shrink to child
	Child Node
	Flags WindowOpt

	Dragging     bool
	grabX, grabY float32

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

func (w *Window) titleHeight() float32 {
	if w.app == nil {
		return 0
	}
	if w.Flags&WindowNoTitle != 0 {
		return 0
	}
	return w.app.theme.TitleHeight
}

// titleBarRect returns the title bar in absolute coords, if any.
func (w *Window) titleBarRect() (x, y, th float32, ok bool) {
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
	if w.Flags&WindowUseParentRect != 0 {
		outerW = cn.MaxW
	}
	if outerW == 0 {
		outerW = chSize.W + pad.Left + pad.Right
	}
	outerH := th + pad.Top + chSize.H + pad.Bottom
	w.outerSize = Size{W: outerW, H: outerH}
	return w.outerSize
}

// Place implements Node. Floating windows use Pos for placement; anchored
// windows can opt into parent-provided X/Y/W via WindowUseParentRect.
func (w *Window) Place(outer emath.Rect) {
	x, y := w.Pos[0], w.Pos[1]
	width := w.outerSize.W
	if w.Flags&WindowUseParentRect != 0 {
		x, y = outer.X, outer.Y
		if outer.W > 0 {
			width = outer.W
		}
	}
	w.rect = emath.Rect{
		X: x, Y: y,
		W: width, H: w.outerSize.H,
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
		pc.Rect(w.rect, w.app.theme.Colors[ColorWindowBG])
	}
	th := w.titleHeight()
	if th > 0 {
		tb := emath.Rect{X: w.rect.X, Y: w.rect.Y, W: w.rect.W, H: th}
		pc.Rect(tb, w.app.theme.Colors[ColorTitleBG])
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
	pc.PushClip(clipR)
	if w.Child != nil {
		w.Child.Paint(pc)
	}
	pc.PopClip()
}

func (w *Window) drawTitle(pc *PaintCtx, win emath.Rect, titleH float32) {
	if w.Title == "" {
		return
	}
	th := w.app.theme
	style := resolveRootTextStyle(w.app, TextStyle{SizeLp: th.TitleLp, Color: th.Colors[ColorTitleText]})
	if style.Font == nil {
		return
	}
	tx := win.X + 6
	ty := win.Y + titleH/2
	sz := w.app.MeasureText(w.Title, TextStyle{SizeLp: th.TitleLp, Color: th.Colors[ColorTitleText]})
	if sz.Height > 0 {
		ty -= sz.Height / 2
	} else {
		// Metrics returns physical-px ascent; convert to lp.
		asc := style.Font.Metrics(style.SizePx).Ascent
		if scale := w.app.vpState.UIScale; scale > 0 {
			asc = asc / scale
		}
		ty -= asc / 2
	}
	pc.Text(w.Title, TextStyle{SizeLp: th.TitleLp, Color: th.Colors[ColorTitleText]}, tx, ty)
}

// Event implements Node.
func (w *Window) Event(_ *Event, _ *EventCtx) bool { return false }
