package draw2d

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/render"
	"triggle/engine/shader"
)

const vertexStride = 8 * 4

// Renderer turns recorded 2D lists into low-level render commands.
type Renderer struct {
	backend backend.Backend

	whiteImage   render.ImageHandle
	whiteSampler render.SamplerHandle
	shader       int32
	family       render.PipelineFamilyID
	mesh         int32
	target       *render.Server
}

func NewRenderer(be backend.Backend) (*Renderer, error) {
	img := be.ImageCreateTexture(1, 1, shader.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("draw2d: white texture create failed")
	}
	smp := be.SamplerCreate(shader.FilterNearest, shader.FilterNearest, shader.WrapClampToEdge, shader.CmpNone)
	if smp < 0 {
		be.ImageDestroy(img)
		return nil, fmt.Errorf("draw2d: white sampler create failed")
	}
	pix := [4]byte{255, 255, 255, 255}
	be.ImageUpdateRGBA8(img, 1, 1, unsafe.Pointer(&pix[0]), 4)

	uiShader := shader.CreateShader(be, shader.UIShaderDesc())
	if uiShader < 0 {
		be.ImageDestroy(img)
		be.SamplerDestroy(smp)
		return nil, fmt.Errorf("draw2d: shader create failed")
	}

	return &Renderer{
		backend:      be,
		whiteImage:   render.ImageHandle(img),
		whiteSampler: render.SamplerHandle(smp),
		shader:       uiShader,
		mesh:         -1,
	}, nil
}

// RenderToFrame records draw commands into the renderer's open default pass.
// scale is the UI logical→physical multiplier; vertex coords and clip rects
// are emitted in lp and scaled to fb px at submit. scale<=0 is treated as 1.
func (dr *Renderer) RenderToFrame(rs *render.Server, list *List, scale float32) {
	if dr == nil || rs == nil || list == nil || dr.shader < 0 {
		return
	}
	if !dr.ensurePipeline(rs) {
		return
	}
	fbW, fbH := rs.FramebufferSize()
	if fbW <= 0 || fbH <= 0 {
		return
	}
	if scale <= 0 {
		scale = 1
	}
	lpW := float32(fbW) / scale
	lpH := float32(fbH) / scale
	if lpW < 1 {
		lpW = 1
	}
	if lpH < 1 {
		lpH = 1
	}
	cmds := list.Commands()
	if len(cmds) == 0 {
		dr.destroyMesh()
		return
	}
	bindings := dr.textureBindings(list)
	clipStack := []scissorRect{{x: 0, y: 0, w: lpW, h: lpH}}
	curClip := clipStack[len(clipStack)-1]

	var verts []float32
	var indices []uint16
	var batches []batchRange

	const noBindID = ^uint32(0)
	var batchBind uint32 = noBindID
	batchScissor := curClip
	batchStartIdx := int32(0)

	closeBatch := func() {
		nidx := int32(len(indices))
		if nidx <= batchStartIdx {
			return
		}
		batches = append(batches, batchRange{
			bindID:     batchBind,
			sc:         batchScissor,
			firstIndex: batchStartIdx,
			indexCount: nidx - batchStartIdx,
		})
		batchStartIdx = nidx
	}
	openBatch := func(bind uint32, sc scissorRect) {
		batchBind = bind
		batchScissor = sc
	}
	ensureBatch := func(bind uint32, sc scissorRect) {
		if batchBind == noBindID {
			openBatch(bind, sc)
			return
		}
		if bind != batchBind || sc != batchScissor {
			closeBatch()
			openBatch(bind, sc)
		}
	}

	for _, c := range cmds {
		switch c.Kind {
		case CmdClipPush:
			closeBatch()
			clipStack = append(clipStack, rectToScissor(c.Rect, lpW, lpH))
			curClip = clipStack[len(clipStack)-1]
			batchBind = noBindID
		case CmdClipPop:
			closeBatch()
			if len(clipStack) > 1 {
				clipStack = clipStack[:len(clipStack)-1]
			}
			curClip = clipStack[len(clipStack)-1]
			batchBind = noBindID
		case CmdQuad:
			col := c.Color.Floats()
			ensureBatch(c.BindID, curClip)
			x0, y0 := c.Rect.X, c.Rect.Y
			x1, y1 := c.Rect.X+c.Rect.W, c.Rect.Y+c.Rect.H
			appendQuad(&verts, &indices, x0, y0, x1, y1, c.UV.U0, c.UV.V0, c.UV.U1, c.UV.V1, col)
		}
	}
	closeBatch()
	if len(indices) == 0 {
		dr.destroyMesh()
		return
	}

	dr.destroyMesh()
	meshID := render.UploadMesh(dr.backend, verts, indices)
	if meshID < 0 {
		return
	}
	dr.mesh = meshID

	// Quads + scissors are in lp; ortho maps lp -> clip. Scissor still must be
	// expressed in fb px for the GPU.
	ortho := screenOrthoMat(lpW, lpH)
	pip := rs.Cache().Pipeline(dr.family, shader.IndexUint16)
	rs.EmitApplyScissor(0, 0, fbW, fbH)
	curEmit := scissorRect{0, 0, lpW, lpH}

	// Outward round on emit: floor x/y, ceil x+w/y+h. Float scissors never
	// under-clip painted content even at fractional UIScale.
	scaleScissor := func(s scissorRect) (x, y, w, h int32) {
		x0 := int32(emath.Floor32(s.x * scale))
		y0 := int32(emath.Floor32(s.y * scale))
		x1 := int32(emath.Ceil32((s.x + s.w) * scale))
		y1 := int32(emath.Ceil32((s.y + s.h) * scale))
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > fbW {
			x1 = fbW
		}
		if y1 > fbH {
			y1 = fbH
		}
		x = x0
		y = y0
		w = x1 - x0
		h = y1 - y0
		if w < 0 {
			w = 0
		}
		if h < 0 {
			h = 0
		}
		return
	}

	for _, b := range batches {
		if b.sc != curEmit {
			sx, sy, sw, sh := scaleScissor(b.sc)
			rs.EmitApplyScissor(sx, sy, sw, sh)
			curEmit = b.sc
		}
		img, smp, ok := lookupBinding(bindings, b.bindID)
		if !ok || img < 0 || smp < 0 {
			continue
		}
		rs.EmitApplyPipeline(pip)
		rs.EmitApplyUniforms(0, render.BytesFromFloat32Slice(ortho[:]))
		rs.EmitBindImage(0, int32(img), int32(smp))
		rs.EmitBindMesh(meshID)
		rs.EmitDrawElements(b.firstIndex, b.indexCount, 1)
	}
}

// RenderToTexture is reserved for callers that need 2D output into offscreen targets.
func (dr *Renderer) RenderToTexture(rs *render.Server, target render.TextureHandle, list *List) {
	_ = dr
	_ = rs
	_ = target
	_ = list
}

func (dr *Renderer) Release() {
	if dr == nil {
		return
	}
	dr.destroyMesh()
	if dr.shader >= 0 {
		dr.backend.ShaderDestroy(dr.shader)
		dr.shader = -1
	}
	if dr.whiteImage >= 0 {
		dr.backend.ImageDestroy(int32(dr.whiteImage))
		dr.whiteImage = -1
	}
	if dr.whiteSampler >= 0 {
		dr.backend.SamplerDestroy(int32(dr.whiteSampler))
		dr.whiteSampler = -1
	}
	dr.family = 0
	dr.target = nil
}

func (dr *Renderer) ensurePipeline(rs *render.Server) bool {
	if dr == nil || rs == nil || dr.shader < 0 {
		return false
	}
	if dr.target == rs && dr.family != 0 {
		return true
	}
	dr.target = rs
	dr.family = rs.Cache().RegisterPipelineFamily(render.PipelineFamilyDesc{
		Shader: dr.shader,
		Buffers: []render.VertexBufferLayout{
			{Stride: vertexStride, Step: render.StepPerVertex},
		},
		Attrs: []render.VertexAttr{
			{Slot: 0, BufferIndex: 0, Format: shader.AttrFloat2},
			{Slot: 1, BufferIndex: 0, Format: shader.AttrFloat2},
			{Slot: 2, BufferIndex: 0, Format: shader.AttrFloat4},
		},
		DepthCmp:   shader.CmpAlways,
		DepthWrite: false,
		Cull:       shader.CullNone,
		ColorCount: 1,
		Blend:      true,
	})
	return dr.family != 0
}

func (dr *Renderer) textureBindings(list *List) []Binding {
	dyn := list.Bindings()
	out := make([]Binding, 0, 1+len(dyn))
	out = append(out, Binding{BindID: bindWhite, Image: dr.whiteImage, Sampler: dr.whiteSampler})
	out = append(out, dyn...)
	return out
}

func (dr *Renderer) destroyMesh() {
	if dr == nil || dr.mesh < 0 {
		return
	}
	dr.backend.MeshDestroy(dr.mesh)
	dr.mesh = -1
}

// scissorRect is a clip rect in lp; converted to fb-px (outward-rounded) at
// emit so fractional clips never under-clip painted content.
type scissorRect struct {
	x, y, w, h float32
}

func rectToScissor(r emath.Rect, vpW, vpH float32) scissorRect {
	x := r.X
	y := r.Y
	w := r.W
	h := r.H
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > vpW {
		w = vpW - x
	}
	if y+h > vpH {
		h = vpH - y
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return scissorRect{x: x, y: y, w: w, h: h}
}

func lookupBinding(bindings []Binding, id uint32) (img render.ImageHandle, smp render.SamplerHandle, ok bool) {
	for _, b := range bindings {
		if b.BindID == id {
			return b.Image, b.Sampler, true
		}
	}
	return -1, -1, false
}

type batchRange struct {
	bindID     uint32
	sc         scissorRect
	firstIndex int32
	indexCount int32
}

func appendQuad(verts *[]float32, indices *[]uint16,
	x0, y0, x1, y1, u0, v0, u1, v1 float32, col [4]float32,
) {
	base := uint16(len(*verts) / 8)
	*verts = append(*verts,
		x0, y0, u0, v0, col[0], col[1], col[2], col[3],
		x1, y0, u1, v0, col[0], col[1], col[2], col[3],
		x1, y1, u1, v1, col[0], col[1], col[2], col[3],
		x0, y1, u0, v1, col[0], col[1], col[2], col[3],
	)
	*indices = append(*indices,
		base+0, base+1, base+2,
		base+0, base+2, base+3,
	)
}

func screenOrthoMat(w, h float32) [16]float32 {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return [16]float32{
		2 / w, 0, 0, 0,
		0, -2 / h, 0, 0,
		0, 0, 1, 0,
		-1, 1, 0, 1,
	}
}
