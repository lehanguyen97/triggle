package ui

import (
	"math"

	"triggle/engine/emath"
)

// ScrollView is a vertical viewport over a single child. Wheel events drive
// Offset (in lp) directly via Root.dispatchInput; Place bakes the offset into
// the child's rect so hit-test and paint share one geometry. Hit-test is
// naturally clipped because the panel's rect rejects out-of-viewport points
// before the walk descends into Children().
type ScrollView struct {
	BaseNode
	Child  Node
	Offset float32

	contentH float32
	contentW float32
}

func (s *ScrollView) Children() []Node {
	if s.Child == nil {
		return nil
	}
	return []Node{s.Child}
}

func (s *ScrollView) Measure(c Constraints) Size {
	cn := normConstraints(c)
	if s.Child == nil {
		return Size{W: cn.MaxW, H: cn.MaxH}
	}
	cs := s.Child.Measure(Constraints{MaxW: cn.MaxW, MaxH: math.MaxFloat32})
	s.contentH = cs.H
	s.contentW = cs.W
	return Size{W: cn.MaxW, H: cn.MaxH}
}

func (s *ScrollView) Place(outer emath.Rect) {
	s.rect = outer
	if s.Child == nil {
		return
	}
	s.Offset = min(max(s.Offset, 0), s.MaxOffset())
	s.Child.Place(emath.Rect{X: outer.X, Y: outer.Y - s.Offset, W: outer.W, H: s.contentH})
}

func (s *ScrollView) Paint(pc *PaintCtx) {
	if s.Child == nil {
		return
	}
	pc.PushClip(s.rect)
	s.Child.Paint(pc)
	pc.PopClip()
}

func (s *ScrollView) Event(_ *Event, _ *EventCtx) bool { return false }

// MaxOffset is the largest offset that still keeps content visible.
func (s *ScrollView) MaxOffset() float32 { return max(s.contentH-s.rect.H, 0) }
