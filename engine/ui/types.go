package ui

// WidgetID is a stable identity for a mounted node (text volatile, text-input host).
type WidgetID uint32

// WindowOpt are bit flags for Window (retained: mostly toggles title/frame).
type WindowOpt uint32

const (
	WindowNoTitle WindowOpt = 1 << iota
	WindowNoFrame
	WindowNoResize
	WindowNoClose
	WindowUseParentRect // parent supplies X/Y/W; height still shrinks to content
)
