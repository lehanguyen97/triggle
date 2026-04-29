package runtime

import (
	"triggle/engine/render"
)

// SceneSource prepares one retained scene for the pipeline to submit.
type SceneSource interface {
	PrepareFrame()
}

// OverlaySource renders UI into the renderer's open default pass.
type OverlaySource interface {
	Render(r *render.Server)
}

// RenderView describes one scene plus an optional overlay for a frame.
type RenderView struct {
	Scene   SceneSource
	Overlay OverlaySource
}

// RenderPipeline orchestrates frame submission for one or more views over a
// single renderer.
type RenderPipeline struct {
	renderer *render.Server
}

func NewRenderPipeline(renderer *render.Server) *RenderPipeline {
	return &RenderPipeline{renderer: renderer}
}

// Render draws one view. Nil Scene is a no-op; nil Overlay means scene only.
func (p *RenderPipeline) Render(view RenderView) {
	if p == nil || view.Scene == nil || p.renderer == nil {
		return
	}
	view.Scene.PrepareFrame()
	p.renderer.RenderScene()
	if view.Overlay != nil {
		view.Overlay.Render(p.renderer)
	}
	p.renderer.EndFrame()
}

// RenderViews draws multiple views in order.
func (p *RenderPipeline) RenderViews(views ...RenderView) {
	for _, view := range views {
		p.Render(view)
	}
}

// Release is a no-op: the renderer is owned by Host.
func (p *RenderPipeline) Release() {}
