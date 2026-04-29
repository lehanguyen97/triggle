package ui

import "triggle/engine/emath"

type FlexDirection uint8

const (
	FlexRow FlexDirection = iota
	FlexColumn
)

type MainAlign uint8

const (
	MainStart MainAlign = iota
	MainCenter
	MainEnd
	MainSpaceBetween
)

type CrossAlign uint8

const (
	CrossStart CrossAlign = iota
	CrossCenter
	CrossEnd
	CrossStretch
)

// FlexItem is one child within a Flex.
//
//	Grow > 0 means the item shares remaining main-axis space.
//	Basis sets the preferred main extent (lp / Frac); grow children measure with
//	their assigned share, non-grow children measure at basis.
type FlexItem struct {
	Node  Node
	Grow  float32
	Basis Length
}

// Flex stacks children along Direction. Two-pass: P1 measure non-grow at basis,
// P2 distribute remaining main with float share.
type Flex struct {
	BaseNode
	Direction  FlexDirection
	Gap        float32
	MainAlign  MainAlign
	CrossAlign CrossAlign
	Kids       []FlexItem

	measured []Size
}

func (f *Flex) Children() []Node {
	out := make([]Node, 0, len(f.Kids))
	for _, k := range f.Kids {
		if k.Node != nil {
			out = append(out, k.Node)
		}
	}
	return out
}

func (f *Flex) mainOf(s Size) float32 {
	if f.Direction == FlexColumn {
		return s.H
	}
	return s.W
}

func (f *Flex) crossOf(s Size) float32 {
	if f.Direction == FlexColumn {
		return s.W
	}
	return s.H
}

func (f *Flex) mainCap(c Constraints) float32 {
	if f.Direction == FlexColumn {
		return c.MaxH
	}
	return c.MaxW
}

func (f *Flex) crossCap(c Constraints) float32 {
	if f.Direction == FlexColumn {
		return c.MaxW
	}
	return c.MaxH
}

func (f *Flex) sizeFromAxes(main, cross float32) Size {
	if f.Direction == FlexColumn {
		return Size{W: cross, H: main}
	}
	return Size{W: main, H: cross}
}

func (f *Flex) childConstraints(parent Constraints, basis Length, mainSize float32, tightMain bool) Constraints {
	crossMax := f.crossCap(parent)
	mainMax := f.mainCap(parent)
	if mainSize > 0 {
		mainMax = mainSize
	} else if basis.specified() {
		mainMax = basis.resolve(f.mainCap(parent), f.mainCap(parent))
	}
	cc := Constraints{}
	if f.Direction == FlexColumn {
		cc.MaxW = crossMax
		cc.MaxH = mainMax
		if tightMain && mainSize > 0 {
			cc.MinH = mainMax
		}
	} else {
		cc.MaxH = crossMax
		cc.MaxW = mainMax
		if tightMain && mainSize > 0 {
			cc.MinW = mainMax
		}
	}
	return cc
}

func (f *Flex) Measure(c Constraints) Size {
	cn := normConstraints(c)
	n := len(f.Kids)
	if cap(f.measured) < n {
		f.measured = make([]Size, n)
	} else {
		f.measured = f.measured[:n]
	}
	if n == 0 {
		return Size{}
	}

	var (
		usedMain  float32
		maxCross  float32
		totalGrow float32
	)
	if n > 1 {
		usedMain += f.Gap * float32(n-1)
	}

	for i, it := range f.Kids {
		if it.Node == nil {
			f.measured[i] = Size{}
			continue
		}
		if it.Grow > 0 {
			totalGrow += it.Grow
			continue
		}
		cc := f.childConstraints(cn, it.Basis, 0, false)
		sz := it.Node.Measure(cc)
		f.measured[i] = sz
		usedMain += f.mainOf(sz)
		if cv := f.crossOf(sz); cv > maxCross {
			maxCross = cv
		}
	}

	remaining := f.mainCap(cn) - usedMain
	if remaining < 0 {
		remaining = 0
	}
	for i, it := range f.Kids {
		if it.Node == nil || it.Grow <= 0 {
			continue
		}
		share := remaining * it.Grow / totalGrow
		cc := f.childConstraints(cn, it.Basis, share, true)
		sz := it.Node.Measure(cc)
		if f.Direction == FlexColumn {
			sz.H = share
		} else {
			sz.W = share
		}
		f.measured[i] = sz
		usedMain += f.mainOf(sz)
		if cv := f.crossOf(sz); cv > maxCross {
			maxCross = cv
		}
	}

	return clampSize(cn, f.sizeFromAxes(usedMain, maxCross))
}

func (f *Flex) Place(outer emath.Rect) {
	f.rect = outer
	n := len(f.Kids)
	if n == 0 {
		return
	}
	var totalMain float32
	if n > 1 {
		totalMain = f.Gap * float32(n-1)
	}
	for i, it := range f.Kids {
		if it.Node == nil {
			continue
		}
		totalMain += f.mainOf(f.measured[i])
	}
	mainExtent := f.mainCapFromRect(outer)
	leftover := mainExtent - totalMain
	if leftover < 0 {
		leftover = 0
	}

	curGap := f.Gap
	startMain := f.mainStart(outer)
	switch f.MainAlign {
	case MainCenter:
		startMain += leftover / 2
	case MainEnd:
		startMain += leftover
	case MainSpaceBetween:
		if n > 1 {
			curGap = f.Gap + leftover/float32(n-1)
		}
	}

	cursor := startMain
	for i, it := range f.Kids {
		if it.Node == nil {
			continue
		}
		sz := f.measured[i]
		mainSize := f.mainOf(sz)
		crossSize := f.crossOf(sz)
		if f.CrossAlign == CrossStretch {
			crossSize = f.crossCapFromRect(outer)
		}
		var cx, cy, cw, ch float32
		if f.Direction == FlexColumn {
			cw = crossSize
			ch = mainSize
			cy = cursor
			cx = outer.X + f.crossOffset(outer.W, cw)
		} else {
			cw = mainSize
			ch = crossSize
			cx = cursor
			cy = outer.Y + f.crossOffset(outer.H, ch)
		}
		it.Node.Place(emath.Rect{X: cx, Y: cy, W: cw, H: ch})
		cursor += mainSize + curGap
	}
}

func (f *Flex) mainStart(outer emath.Rect) float32 {
	if f.Direction == FlexColumn {
		return outer.Y
	}
	return outer.X
}

func (f *Flex) mainCapFromRect(outer emath.Rect) float32 {
	if f.Direction == FlexColumn {
		return outer.H
	}
	return outer.W
}

func (f *Flex) crossCapFromRect(outer emath.Rect) float32 {
	if f.Direction == FlexColumn {
		return outer.W
	}
	return outer.H
}

func (f *Flex) crossOffset(outerCross, childCross float32) float32 {
	free := outerCross - childCross
	if free <= 0 {
		return 0
	}
	switch f.CrossAlign {
	case CrossCenter:
		return free / 2
	case CrossEnd:
		return free
	}
	return 0
}

func (f *Flex) Paint(pc *PaintCtx) {
	for _, it := range f.Kids {
		if it.Node != nil {
			it.Node.Paint(pc)
		}
	}
}

func (f *Flex) Event(_ *Event, _ *EventCtx) bool { return false }
