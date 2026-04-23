package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// Size is a measured width/height in pixels.
type Size struct {
	W, H int32
}

// Constraints are passed top-down from parents during layout.
type Constraints struct {
	MinW, MaxW, MinH, MaxH int32
}

// PaintCtx is passed during painting.
type PaintCtx struct {
	Enc   *cmd.Encoder
	Theme *theme.Theme
	Font  *text.Font
}

// Event is passed to focused widgets; Frame points at the current InputFrame.
type Event struct {
	Frame *InputFrame
}

// EventCtx is passed to Event handlers; carries focus/mouse state for widgets.
type EventCtx struct {
	App     *App
	Frame   InputFrame
	DT      float32
	MouseX  int32
	MouseY  int32
	LocalX  int32 // in receiver's local rect
	LocalY  int32
}

// Node is a retained tree node. Measure returns intrinsic size; Place assigns
// absolute bounds and lays out children.
type Node interface {
	Base() *BaseNode
	Measure(Constraints) Size
	Place(outer emath.Rect)
	Paint(*PaintCtx)
	// Event returns true if fully handled (bubbles if false, depending on type).
	Event(*Event, *EventCtx) bool
	Children() []Node
}

// BaseNode is embedded by every Node; holds app link, id, and bounds.
type BaseNode struct {
	id   uint32
	app  *App
	rect emath.Rect
}

// Base implements Node.Base for *BaseNode receiver used via embedding.
func (b *BaseNode) Base() *BaseNode { return b }

// Rect returns the last placed bounds (absolute framebuffer coords).
func (b *BaseNode) Rect() emath.Rect { return b.rect }

// WidgetID returns the node id (stable after mount) for text volatile and host text-input.
func (b *BaseNode) WidgetID() WidgetID {
	if b == nil {
		return 0
	}
	return WidgetID(b.id)
}

// Invalidate requests a relayout for the next Tick.
func (b *BaseNode) Invalidate() {
	if b.app == nil {
		return
	}
	b.app.layoutDirty = true
}

func (b *BaseNode) mount(app *App) {
	b.app = app
	if b.app != nil && b.id == 0 {
		b.id = b.app.nextNodeID()
	}
}

// App returns the owning app (set after SetRoot / mount).
func (b *BaseNode) App() *App { return b.app }

// normConstraints clamps invalid Max values (0 Max => huge).
func normConstraints(c Constraints) Constraints {
	const big int32 = 1<<30
	if c.MaxW <= 0 {
		c.MaxW = big
	}
	if c.MaxH <= 0 {
		c.MaxH = big
	}
	if c.MinW < 0 {
		c.MinW = 0
	}
	if c.MinH < 0 {
		c.MinH = 0
	}
	return c
}
