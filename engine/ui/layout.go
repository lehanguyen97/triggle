package ui

import "triggle/engine/emath"

// Primitive containers shared by every screen. Higher-level widgets
// (Align, AnchorPanel, Flex, ScrollView, Grid) live in their own files.

// ----------------------------------------------------------------------------
// Padding
// ----------------------------------------------------------------------------

// Padding insets a single child.
type Padding struct {
	BaseNode
	Insets Insets
	Child  Node
}

func (p *Padding) Children() []Node {
	if p.Child == nil {
		return nil
	}
	return []Node{p.Child}
}

func (p *Padding) Measure(con Constraints) Size {
	if p.Child == nil {
		return Size{}
	}
	cn := normConstraints(con)
	in := p.Insets
	innerW := max(cn.MaxW-in.Left-in.Right, 0)
	innerH := max(cn.MaxH-in.Top-in.Bottom, 0)
	ch := p.Child.Measure(Constraints{MaxW: innerW, MaxH: innerH})
	return Size{W: ch.W + in.Left + in.Right, H: ch.H + in.Top + in.Bottom}
}

func (p *Padding) Place(outer emath.Rect) {
	p.rect = outer
	if p.Child == nil {
		return
	}
	in := p.Insets
	inner := emath.Rect{
		X: outer.X + in.Left,
		Y: outer.Y + in.Top,
		W: max(outer.W-in.Left-in.Right, 0),
		H: max(outer.H-in.Top-in.Bottom, 0),
	}
	p.Child.Place(inner)
}

func (p *Padding) Paint(pc *PaintCtx) {
	if p.Child != nil {
		p.Child.Paint(pc)
	}
}

func (p *Padding) Event(e *Event, ec *EventCtx) bool {
	if p.Child == nil {
		return false
	}
	return p.Child.Event(e, ec)
}

// ----------------------------------------------------------------------------
// SizedBox
// ----------------------------------------------------------------------------

// SizedBox forces a size around an optional child. Sizes are lp.
type SizedBox struct {
	BaseNode
	W, H  float32
	Child Node
}

func (s *SizedBox) Children() []Node {
	if s.Child == nil {
		return nil
	}
	return []Node{s.Child}
}

func (s *SizedBox) Measure(con Constraints) Size {
	cn := normConstraints(con)
	w, h := s.W, s.H
	if w == 0 {
		w = cn.MaxW
	}
	if h == 0 {
		h = cn.MaxH
	}
	if s.Child != nil {
		s.Child.Measure(Constraints{MaxW: w, MaxH: h})
	}
	return Size{W: w, H: h}
}

func (s *SizedBox) Place(outer emath.Rect) {
	s.rect = outer
	if s.Child != nil {
		s.Child.Place(outer)
	}
}

func (s *SizedBox) Paint(pc *PaintCtx) {
	if s.Child != nil {
		s.Child.Paint(pc)
	}
}

func (s *SizedBox) Event(e *Event, ec *EventCtx) bool {
	if s.Child == nil {
		return false
	}
	return s.Child.Event(e, ec)
}

// ----------------------------------------------------------------------------
// Stack
// ----------------------------------------------------------------------------

// Stack overlays children: each kid gets the parent's full rect; later kids
// paint on top and receive hits first.
type Stack struct {
	BaseNode
	Kids []Node
}

func (s *Stack) Children() []Node { return s.Kids }

func (s *Stack) Measure(con Constraints) Size {
	cn := normConstraints(con)
	var maxW, maxH float32
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

func (s *Stack) Place(outer emath.Rect) {
	s.rect = outer
	for _, ch := range s.Kids {
		if ch != nil {
			ch.Place(outer)
		}
	}
}

func (s *Stack) Paint(pc *PaintCtx) {
	for _, ch := range s.Kids {
		if ch != nil {
			ch.Paint(pc)
		}
	}
}

func (s *Stack) Event(_ *Event, _ *EventCtx) bool { return false }
