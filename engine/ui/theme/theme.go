// Package theme is data-only style tokens for the immediate-mode UI.
package theme

import (
	"triggle/engine/geom"
	"triggle/engine/ui/cmd"
)

// Theme holds default style tokens for immediate-mode UI.
type Theme struct {
	FontSize      float32
	Padding       Insets
	Spacing       int32
	Indent        int32
	TitleHeight   int32
	ScrollbarSize int32
	Colors        [ColorMax]cmd.Color
	StyleBoxes    [ClassMax][StateMax]StyleBox
}

// Insets are top, right, bottom, left padding in pixels.
type Insets struct {
	Top, Right, Bottom, Left int32
}

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
	ClassMax
)

type WidgetState int32

const (
	StateNormal WidgetState = iota
	StateHover
	StateActive
	StateDisabled
	StateFocused
	StateMax
)

// StyleBox draws a widget background into the UI command stream.
type StyleBox interface {
	Draw(enc *cmd.Encoder, rect geom.Rect)
	ContentMargin() Insets
}

// StyleBoxFlat is a solid fill with an optional border (border deferred; fill only in M1).
type StyleBoxFlat struct {
	Background cmd.Color
	Border     cmd.Color
	BorderPx   int32
	CornerPx   int32
	Padding    Insets
}

func (s StyleBoxFlat) ContentMargin() Insets { return s.Padding }

func (s StyleBoxFlat) Draw(enc *cmd.Encoder, rect geom.Rect) {
	if enc == nil {
		return
	}
	enc.QuadSolid(rect, s.Background)
}

// DefaultTheme builds a minimal baseline UI theme.
func DefaultTheme() *Theme {
	t := &Theme{
		FontSize:      16,
		Padding:       Insets{Top: 8, Right: 8, Bottom: 8, Left: 8},
		Spacing:       6,
		TitleHeight:   24,
		ScrollbarSize: 12,
	}
	t.Colors[ColorText] = cmd.Color{R: 235, G: 235, B: 240, A: 255}
	t.Colors[ColorTextDisabled] = cmd.Color{R: 140, G: 140, B: 150, A: 255}
	t.Colors[ColorBorder] = cmd.Color{R: 60, G: 60, B: 70, A: 255}
	t.Colors[ColorWindowBG] = cmd.Color{R: 28, G: 28, B: 34, A: 240}
	t.Colors[ColorTitleBG] = cmd.Color{R: 40, G: 42, B: 52, A: 255}
	t.Colors[ColorTitleText] = cmd.Color{R: 230, G: 230, B: 238, A: 255}
	t.Colors[ColorPanelBG] = cmd.Color{R: 32, G: 32, B: 40, A: 230}
	// TODO(M2): wire real per-(class,state) style; today every cell is the same flat box.
	for c := range ClassMax {
		for s := range StateMax {
			t.StyleBoxes[c][s] = StyleBoxFlat{
				Background: t.Colors[ColorPanelBG],
				Border:     t.Colors[ColorBorder],
				BorderPx:   1,
				Padding:    Insets{Top: 4, Right: 4, Bottom: 4, Left: 4},
			}
		}
	}
	return t
}
