package cmd

import (
	"triggle/engine/geom"
	"triggle/engine/text"
)

// UICmdKind identifies a command in the flat stream.
type UICmdKind uint8

const (
	CmdQuad UICmdKind = iota + 1
	CmdClipPush
	CmdClipPop
)

// Color is straight RGBA8; multiplies atlas sample in the UI shader.
type Color struct {
	R, G, B, A uint8
}

// Floats returns linear 0..1 components for GPU attributes.
func (c Color) Floats() [4]float32 {
	return [4]float32{
		float32(c.R) / 255,
		float32(c.G) / 255,
		float32(c.B) / 255,
		float32(c.A) / 255,
	}
}

// UICmd is one draw or clip command.
type UICmd struct {
	Kind   UICmdKind
	BindID uint32

	Rect  geom.Rect
	UV    geom.UVRect
	Color Color
}

// Binding is one dynamic texture bind allocated by the encoder (ids start at 2).
type Binding struct {
	BindID  uint32
	Image   int32
	Sampler int32
}

// Encoder accumulates UI commands and dynamic texture bind ids (bind 1 = white, reserved elsewhere).
type Encoder struct {
	cmds      []UICmd
	clipStack []geom.Rect
	viewport  geom.Rect

	nextBind uint32
	texMap   map[texKey]uint32
	bindings []Binding
}

type texKey struct {
	img int32
	smp int32
}

// Reset clears the encoder for a new frame (dynamic binds start at id 2).
func (e *Encoder) Reset(viewport geom.Rect) {
	if e == nil {
		return
	}
	e.cmds = e.cmds[:0]
	e.clipStack = e.clipStack[:0]
	e.viewport = viewport
	e.nextBind = 2
	e.texMap = nil
	e.bindings = e.bindings[:0]
}

// Quad draws a textured quad.
func (e *Encoder) Quad(bindID uint32, dst geom.Rect, uv geom.UVRect, color Color) {
	if e == nil {
		return
	}
	e.cmds = append(e.cmds, UICmd{
		Kind:   CmdQuad,
		BindID: bindID,
		Rect:   dst,
		UV:     uv,
		Color:  color,
	})
}

// QuadSolid draws a solid color quad using bind id 1 (1×1 white texture, UV 0..1).
func (e *Encoder) QuadSolid(dst geom.Rect, color Color) {
	e.Quad(1, dst, geom.UVRect{U0: 0, V0: 0, U1: 1, V1: 1}, color)
}

// AddTexturedQuad implements text.QuadSink: allocates a bind id per (image, sampler) each frame.
func (e *Encoder) AddTexturedQuad(image, sampler int32, dst geom.Rect, uv geom.UVRect, color text.Color) {
	if e == nil || image < 0 || sampler < 0 {
		return
	}
	k := texKey{img: image, smp: sampler}
	if e.texMap == nil {
		e.texMap = make(map[texKey]uint32)
	}
	id, ok := e.texMap[k]
	if !ok {
		if e.nextBind < 2 {
			e.nextBind = 2
		}
		id = e.nextBind
		e.nextBind++
		e.texMap[k] = id
		e.bindings = append(e.bindings, Binding{BindID: id, Image: image, Sampler: sampler})
	}
	c := Color{R: color.R, G: color.G, B: color.B, A: color.A}
	e.Quad(id, dst, uv, c)
}

// Bindings returns dynamic texture bindings recorded this frame (excludes bind 1 white).
func (e *Encoder) Bindings() []Binding {
	if e == nil {
		return nil
	}
	return e.bindings
}

// PushClip intersects with the current clip and records a push command.
func (e *Encoder) PushClip(r geom.Rect) {
	if e == nil {
		return
	}
	parent := e.viewport
	if len(e.clipStack) > 0 {
		parent = e.clipStack[len(e.clipStack)-1]
	}
	inter := parent.Intersect(r)
	e.clipStack = append(e.clipStack, inter)
	e.cmds = append(e.cmds, UICmd{Kind: CmdClipPush, Rect: inter})
}

// PopClip pops one clip level.
func (e *Encoder) PopClip() {
	if e == nil || len(e.clipStack) == 0 {
		return
	}
	e.clipStack = e.clipStack[:len(e.clipStack)-1]
	e.cmds = append(e.cmds, UICmd{Kind: CmdClipPop})
}

// Commands returns the encoded stream for this frame.
func (e *Encoder) Commands() []UICmd {
	if e == nil {
		return nil
	}
	return e.cmds
}
