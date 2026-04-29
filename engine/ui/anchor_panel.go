package ui

import "triggle/engine/emath"

// AnchorChild glues a Pivot point on the child to an Anchor point on the
// parent rect, then offsets by Offset (lp). Width/Height are optional Length
// sizes (frac+clamps); when unspecified the child measures itself.
type AnchorChild struct {
	Node   Node
	Anchor Alignment
	Pivot  Alignment
	Offset emath.Vec2
	Width  Length
	Height Length
}

// AnchorPanel positions each kid by (Anchor on parent) + (-Pivot on child) + Offset.
type AnchorPanel struct {
	BaseNode
	Kids     []AnchorChild
	measured []Size
}

func (p *AnchorPanel) Children() []Node {
	out := make([]Node, 0, len(p.Kids))
	for _, k := range p.Kids {
		if k.Node != nil {
			out = append(out, k.Node)
		}
	}
	return out
}

func (p *AnchorPanel) Measure(c Constraints) Size {
	cn := normConstraints(c)
	if cap(p.measured) < len(p.Kids) {
		p.measured = make([]Size, len(p.Kids))
	} else {
		p.measured = p.measured[:len(p.Kids)]
	}
	for i, k := range p.Kids {
		if k.Node == nil {
			p.measured[i] = Size{}
			continue
		}
		w, h := cn.MaxW, cn.MaxH
		if k.Width.specified() {
			w = k.Width.resolve(cn.MaxW, cn.MaxW)
		}
		if k.Height.specified() {
			h = k.Height.resolve(cn.MaxH, cn.MaxH)
		}
		sz := k.Node.Measure(Constraints{MaxW: w, MaxH: h})
		if k.Width.specified() {
			sz.W = w
		}
		if k.Height.specified() {
			sz.H = h
		}
		p.measured[i] = sz
	}
	return Size{W: cn.MaxW, H: cn.MaxH}
}

func (p *AnchorPanel) Place(outer emath.Rect) {
	p.rect = outer
	for i, k := range p.Kids {
		if k.Node == nil {
			continue
		}
		size := p.measured[i]
		ax := outer.X + outer.W*clamp01(k.Anchor.X)
		ay := outer.Y + outer.H*clamp01(k.Anchor.Y)
		x := ax + k.Offset[0] - size.W*clamp01(k.Pivot.X)
		y := ay + k.Offset[1] - size.H*clamp01(k.Pivot.Y)
		k.Node.Place(emath.Rect{X: x, Y: y, W: size.W, H: size.H})
	}
}

func (p *AnchorPanel) Paint(pc *PaintCtx) {
	for _, k := range p.Kids {
		if k.Node != nil {
			k.Node.Paint(pc)
		}
	}
}

func (p *AnchorPanel) Event(_ *Event, _ *EventCtx) bool { return false }
