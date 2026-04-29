package ui

import "triggle/engine/emath"

// Alignment is normalized [0,1]. 0=start, 0.5=center, 1=end. Components clamp
// to [0,1]. Used by Align (point-on-self) and AnchorPanel (anchor + pivot).
type Alignment struct{ X, Y float32 }

var (
	AlignTopLeft      = Alignment{0, 0}
	AlignTopCenter    = Alignment{0.5, 0}
	AlignTopRight     = Alignment{1, 0}
	AlignCenterLeft   = Alignment{0, 0.5}
	AlignCenter       = Alignment{0.5, 0.5}
	AlignCenterRight  = Alignment{1, 0.5}
	AlignBottomLeft   = Alignment{0, 1}
	AlignBottomCenter = Alignment{0.5, 1}
	AlignBottomRight  = Alignment{1, 1}
)

func alignRect(outer emath.Rect, child Size, a Alignment) emath.Rect {
	dx := outer.W - child.W
	dy := outer.H - child.H
	if dx < 0 {
		dx = 0
	}
	if dy < 0 {
		dy = 0
	}
	return emath.Rect{
		X: outer.X + dx*clamp01(a.X),
		Y: outer.Y + dy*clamp01(a.Y),
		W: child.W,
		H: child.H,
	}
}

// Align measures Child loose and places it at a normalized point inside outer.
type Align struct {
	BaseNode
	Child Node
	A     Alignment
}

func (a *Align) Children() []Node {
	if a.Child == nil {
		return nil
	}
	return []Node{a.Child}
}

func (a *Align) Measure(c Constraints) Size {
	cn := normConstraints(c)
	if a.Child == nil {
		return Size{W: cn.MinW, H: cn.MinH}
	}
	cs := a.Child.Measure(looseConstraints(cn))
	return clampSize(cn, cs)
}

func (a *Align) Place(outer emath.Rect) {
	a.rect = outer
	if a.Child == nil {
		return
	}
	cs := a.Child.Measure(Constraints{MaxW: outer.W, MaxH: outer.H})
	a.Child.Place(alignRect(outer, cs, a.A))
}

func (a *Align) Paint(pc *PaintCtx) {
	if a.Child != nil {
		a.Child.Paint(pc)
	}
}

func (a *Align) Event(e *Event, ec *EventCtx) bool {
	if a.Child == nil {
		return false
	}
	return a.Child.Event(e, ec)
}
