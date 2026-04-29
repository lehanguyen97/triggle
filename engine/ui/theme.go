package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
)

// Theme holds default style tokens for retained UI. All "Lp" values are in
// logical pixels; the renderer multiplies by Root.UIScale at submit time.
type Theme struct {
	Text          TextStyle
	BodyLp        float32
	TitleLp       float32
	Padding       Insets
	Spacing       float32
	Indent        float32
	TitleHeight   float32
	ScrollbarSize float32
	Colors        [ColorMax]emath.Color
}

type TextStyle struct {
	Font   text.FontID
	SizeLp float32
	Color  emath.Color
}

func ResolveTextStyle(base, override TextStyle) TextStyle {
	out := base
	if override.Font != "" {
		out.Font = override.Font
	}
	if override.SizeLp != 0 {
		out.SizeLp = override.SizeLp
	}
	if !isZeroColor(override.Color) {
		out.Color = override.Color
	}
	return out
}

type resolvedTextStyle struct {
	Font   *text.Font
	SizeLp float32 // logical
	SizePx int32   // physical (SizeLp * UIScale, rounded)
	Color  emath.Color
}

func resolveRootTextStyle(root *Root, override TextStyle) resolvedTextStyle {
	if root == nil || root.theme == nil {
		px := emath.RoundToI32(override.SizeLp)
		if px < 1 {
			px = 1
		}
		return resolvedTextStyle{SizeLp: override.SizeLp, SizePx: px, Color: override.Color}
	}
	style := ResolveTextStyle(root.theme.Text, override)
	fontID := style.Font
	if fontID == "" {
		fontID = text.FontDefault
	}
	var font *text.Font
	if root.fonts != nil {
		font = root.fonts.Font(fontID)
	}
	if style.SizeLp == 0 {
		style.SizeLp = root.theme.BodyLp
	}
	if style.SizeLp < 1 {
		style.SizeLp = 1
	}
	scale := root.vpState.UIScale
	if scale <= 0 {
		scale = 1
	}
	px := emath.RoundToI32(style.SizeLp * scale)
	if px < 1 {
		px = 1
	}
	return resolvedTextStyle{
		Font:   font,
		SizeLp: style.SizeLp,
		SizePx: px,
		Color:  style.Color,
	}
}

func isZeroColor(c emath.Color) bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// Insets are top, right, bottom, left padding in logical pixels.
type Insets struct {
	Top, Right, Bottom, Left float32
}

// InsetsAll returns Insets{v,v,v,v}.
func InsetsAll(v float32) Insets { return Insets{Top: v, Right: v, Bottom: v, Left: v} }

// InsetsXY returns Insets with the same horizontal and vertical insets.
func InsetsXY(x, y float32) Insets { return Insets{Top: y, Right: x, Bottom: y, Left: x} }

// InsetsTRBL returns Insets with explicit values; identical to the struct literal,
// but unambiguous at call sites that prefer named arguments.
func InsetsTRBL(t, r, b, l float32) Insets { return Insets{Top: t, Right: r, Bottom: b, Left: l} }

type ColorID int32

const (
	ColorText ColorID = iota
	ColorTextDisabled
	ColorBorder
	ColorWindowBG
	ColorTitleBG
	ColorTitleText
	ColorPanelBG
	ColorButton
	ColorButtonHover
	ColorButtonActive
	ColorBase
	ColorBaseHover
	ColorBaseActive
	ColorScrollBase
	ColorScrollThumb
	ColorMax
)

type WidgetClass int32

const (
	ClassWindow WidgetClass = iota
	ClassTitleBar
	ClassPanel
	ClassButton
	ClassCheckbox
	ClassSlider
	ClassTextInput
)

type WidgetState int32

const (
	StateNormal WidgetState = iota
	StateHover
	StateActive
	StateDisabled
	StateFocused
)

// StyleBoxFlat is a solid fill with an optional border. All sizes in lp.
type StyleBoxFlat struct {
	Background emath.Color
	Border     emath.Color
	BorderLp   float32
	CornerLp   float32
	Padding    Insets
}

func (s StyleBoxFlat) ContentMargin() Insets { return s.Padding }

// DefaultTheme builds a minimal baseline UI theme.
func DefaultTheme() *Theme {
	t := &Theme{
		BodyLp:        16,
		TitleLp:       16,
		Padding:       Insets{Top: 8, Right: 8, Bottom: 8, Left: 8},
		Spacing:       6,
		TitleHeight:   24,
		ScrollbarSize: 12,
	}
	t.Colors[ColorText] = emath.Color{R: 235, G: 235, B: 240, A: 255}
	t.Colors[ColorTextDisabled] = emath.Color{R: 140, G: 140, B: 150, A: 255}
	t.Colors[ColorBorder] = emath.Color{R: 60, G: 60, B: 70, A: 255}
	t.Colors[ColorWindowBG] = emath.Color{R: 28, G: 28, B: 34, A: 240}
	t.Colors[ColorTitleBG] = emath.Color{R: 40, G: 42, B: 52, A: 255}
	t.Colors[ColorTitleText] = emath.Color{R: 230, G: 230, B: 238, A: 255}
	t.Colors[ColorPanelBG] = emath.Color{R: 32, G: 32, B: 40, A: 230}
	t.Text = TextStyle{Font: text.FontDefault, SizeLp: t.BodyLp, Color: t.Colors[ColorText]}
	return t
}
