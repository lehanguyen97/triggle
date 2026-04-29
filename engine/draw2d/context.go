package draw2d

import (
	"triggle/engine/emath"
	"triggle/engine/render"
	"triggle/engine/text"
)

const bindWhite uint32 = 1

// CmdKind identifies a command in the recorded 2D list.
type CmdKind uint8

const (
	CmdQuad CmdKind = iota + 1
	CmdClipPush
	CmdClipPop
)

// Cmd is one draw or clip command.
type Cmd struct {
	Kind   CmdKind
	BindID uint32

	Rect  emath.Rect
	UV    emath.UVRect
	Color emath.Color
}

// Binding is one dynamic texture bind allocated by the context.
type Binding struct {
	BindID  uint32
	Image   render.ImageHandle
	Sampler render.SamplerHandle
}

// List is the recorded 2D command stream for one target.
type List struct {
	cmds     []Cmd
	bindings []Binding
}

func (l *List) Commands() []Cmd { return l.cmds }

func (l *List) Bindings() []Binding { return l.bindings }

// Context records high-level 2D drawing commands.
type Context struct {
	list      List
	clipStack []emath.Rect
	viewport  emath.Rect

	nextBind uint32
	texMap   map[texKey]uint32
}

type texKey struct {
	img render.ImageHandle
	smp render.SamplerHandle
}

func NewContext() *Context { return &Context{} }

// Reset clears the context for a new frame.
func (c *Context) Reset(viewport emath.Rect) {
	c.list.cmds = c.list.cmds[:0]
	c.list.bindings = c.list.bindings[:0]
	c.clipStack = c.clipStack[:0]
	c.viewport = viewport
	c.nextBind = 2
	c.texMap = nil
}

// Rect draws a solid color rectangle.
func (c *Context) Rect(r emath.Rect, color emath.Color) {
	c.Quad(bindWhite, r, emath.UVRect{U0: 0, V0: 0, U1: 1, V1: 1}, color)
}

// Quad records a textured quad by bind id.
func (c *Context) Quad(bindID uint32, dst emath.Rect, uv emath.UVRect, color emath.Color) {
	c.list.cmds = append(c.list.cmds, Cmd{
		Kind:   CmdQuad,
		BindID: bindID,
		Rect:   dst,
		UV:     uv,
		Color:  color,
	})
}

// Image records a textured quad. A bind id is allocated per image/sampler pair.
func (c *Context) Image(img render.ImageHandle, smp render.SamplerHandle, dst emath.Rect, uv emath.UVRect, color emath.Color) {
	if img < 0 || smp < 0 {
		return
	}
	k := texKey{img: img, smp: smp}
	if c.texMap == nil {
		c.texMap = make(map[texKey]uint32)
	}
	id, ok := c.texMap[k]
	if !ok {
		if c.nextBind < 2 {
			c.nextBind = 2
		}
		id = c.nextBind
		c.nextBind++
		c.texMap[k] = id
		c.list.bindings = append(c.list.bindings, Binding{BindID: id, Image: img, Sampler: smp})
	}
	c.Quad(id, dst, uv, color)
}

// TextLine draws a prepared text line at x,y. Segments are emitted in their
// native coordinate space (no scaling); use TextLineScaled when the segments
// were shaped at physical-px size but the destination grid is logical-px.
func (c *Context) TextLine(line *text.Line, x, y float32, color emath.Color) {
	if line == nil {
		return
	}
	for _, seg := range line.Segments {
		d := seg.Dst
		d.X += x
		d.Y += y
		c.Image(seg.Image, seg.Sampler, d, seg.UV, color)
	}
}

// TextLineScaled draws a prepared text line at lp x,y. Segments are assumed to
// be in physical px; offsets are multiplied by inv (= 1/UIScale) before being
// added to the lp anchor, then sized in lp.
func (c *Context) TextLineScaled(line *text.Line, x, y, inv float32, color emath.Color) {
	if line == nil {
		return
	}
	if inv <= 0 || inv == 1 {
		c.TextLine(line, x, y, color)
		return
	}
	for _, seg := range line.Segments {
		d := emath.Rect{
			X: x + seg.Dst.X*inv,
			Y: y + seg.Dst.Y*inv,
			W: seg.Dst.W * inv,
			H: seg.Dst.H * inv,
		}
		c.Image(seg.Image, seg.Sampler, d, seg.UV, color)
	}
}

// PushClip intersects with the current clip and records a push command.
func (c *Context) PushClip(r emath.Rect) {
	parent := c.viewport
	if len(c.clipStack) > 0 {
		parent = c.clipStack[len(c.clipStack)-1]
	}
	inter := parent.Intersect(r)
	c.clipStack = append(c.clipStack, inter)
	c.list.cmds = append(c.list.cmds, Cmd{Kind: CmdClipPush, Rect: inter})
}

// PopClip pops one clip level.
func (c *Context) PopClip() {
	if len(c.clipStack) == 0 {
		return
	}
	c.clipStack = c.clipStack[:len(c.clipStack)-1]
	c.list.cmds = append(c.list.cmds, Cmd{Kind: CmdClipPop})
}

func (c *Context) List() *List { return &c.list }

func (c *Context) CmdIndex() int { return len(c.list.cmds) }

func (c *Context) PatchRect(idx int, r emath.Rect) {
	if idx < 0 || idx >= len(c.list.cmds) {
		return
	}
	cmd := &c.list.cmds[idx]
	switch cmd.Kind {
	case CmdQuad, CmdClipPush:
		cmd.Rect = r
	}
}
