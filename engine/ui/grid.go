package ui

import "triggle/engine/emath"

// Grid lays out fixed-size cells in a wrapping row-major flow. Cell width
// expands proportionally so the row exactly fills the cross extent.
type Grid struct {
	BaseNode
	MinCellW, MinCellH float32
	GapX, GapY         float32
	Kids               []Node

	cellW float32
	cols  int32
}

func (g *Grid) Children() []Node { return g.Kids }

func (g *Grid) Measure(c Constraints) Size {
	cn := normConstraints(c)
	n := int32(len(g.Kids))
	if g.MinCellW <= 0 || g.MinCellH <= 0 || n == 0 {
		return Size{W: cn.MaxW}
	}
	cols := int32((cn.MaxW + g.GapX) / (g.MinCellW + g.GapX))
	if cols < 1 {
		cols = 1
	}
	cellW := (cn.MaxW - g.GapX*float32(cols-1)) / float32(cols)
	if cellW < g.MinCellW {
		cellW = g.MinCellW
	}
	g.cellW = cellW
	g.cols = cols
	rows := (n + cols - 1) / cols
	cc := tightConstraints(Size{W: cellW, H: g.MinCellH})
	for _, k := range g.Kids {
		if k != nil {
			k.Measure(cc)
		}
	}
	gapTotal := float32(rows-1) * g.GapY
	if rows-1 < 0 {
		gapTotal = 0
	}
	return Size{W: cn.MaxW, H: float32(rows)*g.MinCellH + gapTotal}
}

func (g *Grid) Place(outer emath.Rect) {
	g.rect = outer
	cols := g.cols
	if cols < 1 {
		cols = 1
	}
	for i, k := range g.Kids {
		if k == nil {
			continue
		}
		col := int32(i) % cols
		row := int32(i) / cols
		x := outer.X + float32(col)*(g.cellW+g.GapX)
		y := outer.Y + float32(row)*(g.MinCellH+g.GapY)
		k.Place(emath.Rect{X: x, Y: y, W: g.cellW, H: g.MinCellH})
	}
}

func (g *Grid) Paint(pc *PaintCtx) {
	for _, k := range g.Kids {
		if k != nil {
			k.Paint(pc)
		}
	}
}

func (g *Grid) Event(_ *Event, _ *EventCtx) bool { return false }
