package ui

import "triggle/engine/emath"

// ViewportState is the resolved viewport for one Root, computed from the
// host-supplied framebuffer rect and DPI scale. UI is DPI-only responsive:
// UIScale = DPIScale, lp viewport = framebuffer / DPIScale, layout reflows
// naturally via Flex / ScrollView / ClampFrac. There is no design-resolution
// stretch mode for UI — scaled-cinematic content belongs to the 3D viewport,
// not here.
type ViewportState struct {
	Framebuffer emath.Rect // physical px (X,Y unused; W,H are size)
	Design      emath.Rect // logical-px viewport = Framebuffer / DPIScale
	UIScale     float32    // logical→physical multiplier (= DPIScale)
	DPIScale    float32    // physical px per CSS px (informational)
}
