package runtime

import (
	"math"

	"triggle/engine/camera"
	"triggle/engine/emath"
	"triggle/engine/event"
	"triggle/engine/render"
	"triggle/engine/ui"
)

// GameplayEvent is the gameplay-facing event payload for one frame.
// Mouse events include a world-space ray generated from the active camera.
type GameplayEvent struct {
	Event  event.Event
	Ray    camera.Ray
	HasRay bool
}

// FrameHooks customize one runtime frame.
type FrameHooks struct {
	BeforeUITick func()
	OnGameplay   func(GameplayEvent)
}

// View owns per-frame orchestration for one scene/view under the current
// single-camera assumption.
type View struct {
	scene    SceneSource
	renderer *render.Server
	pipeline *RenderPipeline
	overlay  *ui.Root

	cam camera.Orbit

	fbW int32
	fbH int32
	dpi float32

	wantsMouse bool

	middleDown bool
	dragLastX  float32
	dragLastY  float32
	dragDist   float32
}

func NewView(scene SceneSource, renderer *render.Server, pipeline *RenderPipeline) *View {
	return &View{
		scene:    scene,
		renderer: renderer,
		pipeline: pipeline,
		fbW:      800,
		fbH:      600,
		dpi:      1.0,
	}
}

func (v *View) SetOverlay(overlay *ui.Root) {
	if v == nil {
		return
	}
	v.overlay = overlay
}

func (v *View) SetOrbit(cam camera.Orbit) {
	if v == nil {
		return
	}
	v.cam = cam
	v.syncCameraState()
}

func (v *View) WantsMouse() bool {
	if v == nil {
		return false
	}
	return v.wantsMouse
}

func (v *View) Frame(dt float32, hooks FrameHooks) {
	if v == nil {
		return
	}
	frameEvents := collectFrameEvents()
	v.applyWindowEvents(frameEvents)
	v.forwardFrameEventsToUI(frameEvents)
	if hooks.BeforeUITick != nil {
		hooks.BeforeUITick()
	}
	v.tickUI(dt)
	v.dispatchGameplayEvents(frameEvents, hooks.OnGameplay)
}

// Render delegates to the shared pipeline so multi-view composition (when it
// arrives) routes through one frame submitter.
func (v *View) Render() {
	if v == nil || v.scene == nil || v.pipeline == nil {
		return
	}
	var overlay OverlaySource
	if v.overlay != nil {
		overlay = v.overlay
	}
	v.pipeline.Render(RenderView{Scene: v.scene, Overlay: overlay})
}

func (v *View) applyWindowEvents(events []event.Event) {
	for i := range events {
		ev := &events[i]
		switch ev.Kind {
		case event.KindResize, event.KindDPIChanged:
			v.onResize(int32(ev.Size.W), int32(ev.Size.H), ev.DPIScale)
		}
	}
}

func (v *View) forwardFrameEventsToUI(events []event.Event) {
	if v == nil || v.overlay == nil {
		return
	}
	frame := v.overlay.Input()
	for i := range events {
		applyEventToUIFrame(frame, &events[i])
	}
}

func (v *View) tickUI(dt float32) {
	v.wantsMouse = false
	if v.overlay == nil {
		return
	}
	v.overlay.Tick(dt)
	v.wantsMouse = v.overlay.WantsMouse()
}

func (v *View) dispatchGameplayEvents(events []event.Event, onGameplay func(GameplayEvent)) {
	if onGameplay == nil {
		return
	}
	for i := range events {
		ev := &events[i]
		switch ev.Kind {
		case event.KindKey, event.KindText:
			// UI already consumed text/key input for this frame.
			continue
		case event.KindResize, event.KindDPIChanged, event.KindFocus:
			// Runtime owns window/dpi handling; focus is currently UI-only.
			continue
		case event.KindMouseScroll:
			v.handleMouseScroll(ev)
		case event.KindMouseButton:
			v.handleMouseButton(ev)
		case event.KindMouseMove:
			v.handleMouseMove(ev)
		}
		if v.wantsMouse && isPointerEvent(ev.Kind) {
			continue
		}
		ge := GameplayEvent{Event: *ev}
		if isPointerEvent(ev.Kind) {
			ge.Ray = v.screenRay(ev.Pos[0], ev.Pos[1])
			ge.HasRay = true
		}
		onGameplay(ge)
	}
}

func isPointerEvent(kind event.Kind) bool {
	return kind == event.KindMouseButton || kind == event.KindMouseMove || kind == event.KindMouseScroll
}

func (v *View) handleMouseButton(ev *event.Event) {
	if ev == nil {
		return
	}
	if ev.Down {
		v.handleMouseDown(ev)
		return
	}
	v.handleMouseUp(ev)
}

func (v *View) handleMouseDown(ev *event.Event) {
	if ev == nil || v.wantsMouse {
		return
	}
	mx, my := ev.Pos[0], ev.Pos[1]
	if ev.Button == event.MouseMiddle ||
		(ev.Button == event.MouseLeft && ev.Mods&event.ModShift != 0) {
		v.middleDown = true
		v.dragLastX = mx
		v.dragLastY = my
		v.dragDist = 0
	}
}

func (v *View) handleMouseUp(ev *event.Event) {
	if ev == nil {
		return
	}
	if ev.Button == event.MouseMiddle || (ev.Button == event.MouseLeft && v.middleDown) {
		v.middleDown = false
	}
}

func (v *View) handleMouseMove(ev *event.Event) {
	if ev == nil || v.wantsMouse || !v.middleDown {
		return
	}
	mx, my := ev.Pos[0], ev.Pos[1]
	dx := mx - v.dragLastX
	dy := my - v.dragLastY
	v.dragDist += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
	if v.dragDist >= 5.0 {
		v.cam.Rotate(dx, dy, 0.005, 0.005)
		v.syncCameraState()
	}
	v.dragLastX = mx
	v.dragLastY = my
}

func (v *View) handleMouseScroll(ev *event.Event) {
	if ev == nil || v.wantsMouse {
		return
	}
	if v.overlay != nil && v.overlay.WantsScroll() {
		return
	}
	v.cam.Zoom(ev.Delta[1], 0.5)
	v.syncCameraState()
}

func (v *View) onResize(fbW, fbH int32, dpi float32) {
	if fbW <= 0 || fbH <= 0 {
		return
	}
	v.fbW, v.fbH = fbW, fbH
	if dpi > 0 {
		v.dpi = dpi
	}
	if v.renderer != nil {
		v.renderer.SetFramebuffer(fbW, fbH)
	}
	if v.overlay != nil {
		v.overlay.SetViewport(emath.Rect{W: float32(fbW), H: float32(fbH)}, v.dpi)
	}
	v.syncCameraState()
}

func (v *View) syncCameraState() {
	if v == nil || v.renderer == nil {
		return
	}
	v.cam.Sync(v.renderer.Projection())
	v.renderer.SetCamera(render.CameraState{ViewProj: v.cam.ViewProj, CameraPos: v.cam.Pos})
}

func (v *View) screenRay(sx, sy float32) camera.Ray {
	return v.cam.ScreenRay(sx, sy, v.fbW, v.fbH)
}

func collectFrameEvents() []event.Event {
	events := make([]event.Event, 0, 32)
	event.Drain(func(ev *event.Event) {
		if ev == nil {
			return
		}
		events = append(events, *ev)
	})
	return events
}
