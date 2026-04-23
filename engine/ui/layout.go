package ui

import (
	"triggle/engine/emath"
	"triggle/engine/ui/theme"
)

// Column stacks children vertically (full width of constraint).
type Column struct {
	BaseNode
	parent  Node
	Kids    []Node
	Spacing int32 // 0 => theme Spacing
}

// Children reports child nodes.
func (c *Column) Children() []Node { return c.Kids }

// Measure implements Node.
func (c *Column) Measure(con Constraints) Size {
	cn := normConstraints(con)
	if c.app == nil {
		return Size{W: cn.MinW, H: cn.MinH}
	}
	gap := c.Spacing
	if gap == 0 {
		gap = c.app.theme.Spacing
	}
	var maxW, totalH int32
	for i, ch := range c.Kids {
		if ch == nil {
			continue
		}
		if i > 0 {
			totalH += gap
		}
		m := ch.Measure(Constraints{MaxW: cn.MaxW, MinW: cn.MinW, MaxH: cn.MaxH, MinH: 0})
		if m.W > maxW {
			maxW = m.W
		}
		totalH += m.H
	}
	if maxW < cn.MinW {
		maxW = cn.MinW
	}
	if maxW > cn.MaxW {
		maxW = cn.MaxW
	}
	return Size{W: maxW, H: totalH}
}

// Place implements Node.
func (c *Column) Place(outer emath.Rect) {
	c.rect = outer
	if c.app == nil {
		return
	}
	gap := c.Spacing
	if gap == 0 {
		gap = c.app.theme.Spacing
	}
	y := outer.Y
	for i, ch := range c.Kids {
		if ch == nil {
			continue
		}
		if i > 0 {
			y += gap
		}
		m := ch.Measure(Constraints{MaxW: outer.W, MinW: 0, MaxH: outer.H, MinH: 0})
		ch.Place(emath.Rect{X: outer.X, Y: y, W: outer.W, H: m.H})
		y += m.H
	}
}

// Paint implements Node.
func (c *Column) Paint(pc *PaintCtx) {
	for _, ch := range c.Kids {
		if ch != nil {
			ch.Paint(pc)
		}
	}
}

// Event implements Node.
func (c *Column) Event(_ *Event, _ *EventCtx) bool { return false }

// Padding insets a single child.
type Padding struct {
	BaseNode
	parent Node
	Insets theme.Insets
	Child  Node
}

// Children implements Node.
func (p *Padding) Children() []Node {
	if p.Child == nil {
		return nil
	}
	return []Node{p.Child}
}

// Measure implements Node.
func (p *Padding) Measure(con Constraints) Size {
	if p.app == nil || p.Child == nil {
		return Size{}
	}
	in := p.Insets
	innerW := con.MaxW - in.Left - in.Right
	innerH := con.MaxH - in.Top - in.Bottom
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	ch := p.Child.Measure(Constraints{MaxW: innerW, MaxH: innerH, MinW: 0, MinH: 0})
	return Size{W: ch.W + in.Left + in.Right, H: ch.H + in.Top + in.Bottom}
}

// Place implements Node.
func (p *Padding) Place(outer emath.Rect) {
	p.rect = outer
	if p.Child == nil {
		return
	}
	in := p.Insets
	inner := emath.Rect{
		X: outer.X + in.Left,
		Y: outer.Y + in.Top,
		W: outer.W - in.Left - in.Right,
		H: outer.H - in.Top - in.Bottom,
	}
	if inner.W < 0 {
		inner.W = 0
	}
	if inner.H < 0 {
		inner.H = 0
	}
	p.Child.Place(inner)
}

// Paint implements Node.
func (p *Padding) Paint(pc *PaintCtx) {
	if p.Child != nil {
		p.Child.Paint(pc)
	}
}

// Event implements Node.
func (p *Padding) Event(e *Event, ec *EventCtx) bool {
	if p.Child == nil {
		return false
	}
	return p.Child.Event(e, ec)
}

// SizedBox forces a size around an optional child.
type SizedBox struct {
	BaseNode
	parent Node
	W, H  int32
	Child Node
}

// Children implements Node.
func (s *SizedBox) Children() []Node {
	if s.Child == nil {
		return nil
	}
	return []Node{s.Child}
}

// Measure implements Node.
func (s *SizedBox) Measure(con Constraints) Size {
	if s.Child == nil {
		w, h := s.W, s.H
		if w == 0 {
			w = con.MaxW
		}
		if h == 0 {
			h = con.MaxH
		}
		return Size{W: w, H: h}
	}
	inner := Constraints{MaxW: s.W, MaxH: s.H}
	if s.W == 0 {
		inner.MaxW = con.MaxW
	}
	if s.H == 0 {
		inner.MaxH = con.MaxH
	}
	inner = normConstraints(inner)
	_ = s.Child.Measure(inner)
	return Size{W: s.W, H: s.H}
}

// Place implements Node.
func (s *SizedBox) Place(outer emath.Rect) {
	s.rect = outer
	if s.Child == nil {
		return
	}
	s.Child.Place(outer)
}

// Paint implements Node.
func (s *SizedBox) Paint(pc *PaintCtx) {
	if s.Child != nil {
		s.Child.Paint(pc)
	}
}

// Event implements Node.
func (s *SizedBox) Event(e *Event, ec *EventCtx) bool {
	if s.Child == nil {
		return false
	}
	return s.Child.Event(e, ec)
}

// Stack positions children in z-order: later children paint on top and receive hits first.
type Stack struct {
	BaseNode
	parent Node
	Kids   []Node
}

// Children implements Node.
func (s *Stack) Children() []Node { return s.Kids }

// Measure implements Node.
func (s *Stack) Measure(con Constraints) Size {
	cn := normConstraints(con)
	var maxW, maxH int32
	for _, ch := range s.Kids {
		if ch == nil {
			continue
		}
		z := ch.Measure(cn)
		if z.W > maxW {
			maxW = z.W
		}
		if z.H > maxH {
			maxH = z.H
		}
	}
	return Size{W: maxW, H: maxH}
}

// Place implements Node: each child gets the same area (typically full viewport for floating windows).
func (s *Stack) Place(outer emath.Rect) {
	s.rect = outer
	for _, ch := range s.Kids {
		if ch != nil {
			ch.Place(outer)
		}
	}
}

// Paint implements Node.
func (s *Stack) Paint(pc *PaintCtx) {
	for _, ch := range s.Kids {
		if ch != nil {
			ch.Paint(pc)
		}
	}
}

// Event implements Node.
func (s *Stack) Event(_ *Event, _ *EventCtx) bool { return false }
