package render

import (
	"triggle/engine/emath"
	"triggle/engine/shader"
	"triggle/engine/ui"
	uicmd "triggle/engine/ui/cmd"
)

type scissorRect struct {
	x, y, w, h int32
}

func rectToScissor(r emath.Rect, vpW, vpH int32) scissorRect {
	x := int32(r.X)
	y := int32(r.Y)
	w := int32(r.W)
	h := int32(r.H)
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

func lookupTextureBinding(bindings []ui.TextureBinding, id uint32) (img, smp int32, ok bool) {
	for _, b := range bindings {
		if b.BindID == id {
			return b.Image, b.Sampler, true
		}
	}
	return -1, -1, false
}

type uiBatchRange struct {
	bindID     uint32
	sc         scissorRect
	firstIndex int32
	indexCount int32
}

func appendUIQuad(verts *[]float32, indices *[]uint16,
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

// EmitUI encodes UI draws into the forward command buffer (single mesh, batched by texture binding + scissor).
func (p *UIProgram) EmitUI(r *ForwardRenderer, cmds []uicmd.UICmd, bindings []ui.TextureBinding, vpW, vpH int32) {
	if vpW <= 0 || vpH <= 0 {
		return
	}
	if len(cmds) == 0 {
		r.destroyUIMesh()
		return
	}

	clipStack := []scissorRect{{x: 0, y: 0, w: vpW, h: vpH}}
	curClip := clipStack[len(clipStack)-1]

	var verts []float32
	var indices []uint16
	var batches []uiBatchRange

	const noBindID = ^uint32(0)
	var batchBind uint32 = noBindID
	batchScissor := curClip
	batchStartIdx := int32(0)

	closeBatch := func() {
		nidx := int32(len(indices))
		if nidx <= batchStartIdx {
			return
		}
		batches = append(batches, uiBatchRange{
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
		case uicmd.CmdClipPush:
			closeBatch()
			clipStack = append(clipStack, rectToScissor(c.Rect, vpW, vpH))
			curClip = clipStack[len(clipStack)-1]
			batchBind = noBindID
		case uicmd.CmdClipPop:
			closeBatch()
			if len(clipStack) > 1 {
				clipStack = clipStack[:len(clipStack)-1]
			}
			curClip = clipStack[len(clipStack)-1]
			batchBind = noBindID
		case uicmd.CmdQuad:
			col := c.Color.Floats()
			ensureBatch(c.BindID, curClip)
			x0, y0 := float32(c.Rect.X), float32(c.Rect.Y)
			x1, y1 := float32(c.Rect.X+c.Rect.W), float32(c.Rect.Y+c.Rect.H)
			appendUIQuad(&verts, &indices, x0, y0, x1, y1, c.UV.U0, c.UV.V0, c.UV.U1, c.UV.V1, col)
		}
	}
	closeBatch()

	if len(indices) == 0 {
		r.destroyUIMesh()
		return
	}

	// TODO(M3+): reuse one dynamic UI mesh + in-place updates instead of destroy/recreate each frame.
	r.destroyUIMesh()
	mesh := r.UploadMesh(verts, indices)
	if mesh < 0 {
		return
	}
	r.uiMesh = mesh

	if _, ok := r.meshInfo(mesh); !ok {
		r.DestroyMesh(mesh)
		r.uiMesh = -1
		return
	}

	ortho := screenOrthoMat(vpW, vpH)
	pip := r.cache.Pipeline(p.mainFamily, shader.IndexUint16)

	r.emitApplyScissor(0, 0, vpW, vpH)
	curEmit := scissorRect{0, 0, vpW, vpH}

	for _, b := range batches {
		if b.sc != curEmit {
			r.emitApplyScissor(b.sc.x, b.sc.y, b.sc.w, b.sc.h)
			curEmit = b.sc
		}
		img, smp, ok := lookupTextureBinding(bindings, b.bindID)
		if !ok || img < 0 || smp < 0 {
			continue
		}
		r.emitApplyPipeline(pip)
		r.emitApplyUniforms(0, bytesFromFloat32Slice(ortho[:]))
		r.emitBindImage(0, img, smp)
		r.emitBindMesh(mesh)
		r.emitDrawElements(b.firstIndex, b.indexCount, 1)
	}
}
