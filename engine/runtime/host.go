package runtime

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/camera"
	"triggle/engine/event"
	"triggle/engine/render"
	"triggle/engine/scene"
	"triggle/engine/text"
	"triggle/engine/ui"
)

// ViewOptions bundles per-view wiring so game code can construct a ready-to-run
// View in a single call.
type ViewOptions struct {
	Overlay *ui.Root
	Orbit   camera.Orbit
}

type Host struct {
	backend  backend.Backend
	renderer *render.Server
	scene    *scene.Scene
	pipeline *RenderPipeline
	events   *event.Queue
	fonts    *text.FontSet
}

func NewHost() (*Host, error) {
	be := backend.NewBackend()
	if be.Handle() != 0 {
		return nil, fmt.Errorf("triggle: backend init failed")
	}

	q := event.NewQueue()
	event.SetActive(q)

	r := render.NewServer(be)
	s := scene.New(be, r)
	p := NewRenderPipeline(r)
	h := &Host{
		backend:  be,
		renderer: r,
		scene:    s,
		pipeline: p,
		events:   q,
		fonts:    text.NewFontSet(be),
	}
	return h, nil
}

func (h *Host) Scene() *scene.Scene {
	if h == nil {
		return nil
	}
	return h.scene
}

func (h *Host) Events() *event.Queue {
	if h == nil {
		return nil
	}
	return h.events
}

func (h *Host) Fonts() *text.FontSet {
	if h == nil {
		return nil
	}
	return h.fonts
}

func (h *Host) NewRoot(opts ui.RootOptions) (*ui.Root, error) {
	if h == nil {
		return nil, fmt.Errorf("triggle: nil runtime host")
	}
	return ui.NewRoot(h.backend, h.fonts, opts)
}

func (h *Host) NewView(sc SceneSource, opts ViewOptions) *View {
	if h == nil {
		return nil
	}
	v := NewView(sc, h.renderer, h.pipeline)
	if opts.Overlay != nil {
		v.SetOverlay(opts.Overlay)
	}
	v.SetOrbit(opts.Orbit)
	return v
}

func (h *Host) Close() {
	if h == nil {
		return
	}
	if h.pipeline != nil {
		h.pipeline.Release()
		h.pipeline = nil
	}
	if h.fonts != nil {
		h.fonts.Close()
		h.fonts = nil
	}
	if h.scene != nil {
		h.scene.Release()
		h.scene = nil
	}
	if h.renderer != nil {
		h.renderer.Release()
		h.renderer = nil
	}
	if h.backend.Handle() != 0 {
		_ = h.backend.Cleanup()
	}
	h.backend = backend.Backend{}
	event.SetActive(nil)
	h.events = nil
}
