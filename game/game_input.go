package main

import (
	"math"

	"triggle/engine/event"
	engineruntime "triggle/engine/runtime"
)

func (g *Game) handleGameplayEvent(ge engineruntime.GameplayEvent) {
	ev := &ge.Event
	switch ev.Kind {
	case event.KindMouseButton:
		if ev.Down {
			g.handleMouseDown(ev, ge)
		} else {
			g.handleMouseUp(ev, ge)
		}
	case event.KindMouseMove:
		g.handleMouseMove(ev, ge)
	}
}

func (g *Game) handleMouseDown(ev *event.Event, ge engineruntime.GameplayEvent) {
	if ev == nil || ev.Button != event.MouseLeft {
		return
	}
	mx, my := ev.Pos[0], ev.Pos[1]
	g.leftDown = true
	g.dragLastX = mx
	g.dragLastY = my
	g.dragDist = 0
	if !ge.HasRay {
		g.dragStartPeg = -1
		return
	}
	g.dragStartPeg = pickPeg(g.board.Pegs, ge.Ray.Origin, ge.Ray.Dir)
}

func (g *Game) handleMouseUp(ev *event.Event, ge engineruntime.GameplayEvent) {
	if ev == nil || ev.Button != event.MouseLeft {
		return
	}
	if g.leftDown && g.dragDist < 5.0 {
		g.doClick(ge)
	} else if g.leftDown && g.dragStartPeg >= 0 && g.hoveredPeg >= 0 && g.dragStartPeg != g.hoveredPeg {
		g.tryPlaceBand(g.dragStartPeg, g.hoveredPeg)
	}
	g.leftDown = false
	g.dragStartPeg = -1
	g.hoveredPeg = -1
	g.setPreviewLine(nil)
}

func (g *Game) handleMouseMove(ev *event.Event, ge engineruntime.GameplayEvent) {
	if ev == nil {
		return
	}
	mx, my := ev.Pos[0], ev.Pos[1]
	if g.leftDown {
		dx := mx - g.dragLastX
		dy := my - g.dragLastY
		g.dragDist += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
		g.dragLastX = mx
		g.dragLastY = my
	}
	// Hover detection for drag or click-click
	if g.leftDown && g.dragStartPeg >= 0 || g.selectedPeg >= 0 {
		if !ge.HasRay {
			return
		}
		g.hoveredPeg = pickPeg(g.board.Pegs, ge.Ray.Origin, ge.Ray.Dir)
		startPeg := g.dragStartPeg
		if startPeg < 0 {
			startPeg = g.selectedPeg
		}
		if startPeg >= 0 && g.hoveredPeg >= 0 && startPeg != g.hoveredPeg {
			line := g.board.FindLine(startPeg, g.hoveredPeg)
			if line != nil && g.board.CanPlace(line) {
				g.setPreviewLine(line)
			} else {
				g.setPreviewLine(nil)
			}
		} else {
			g.setPreviewLine(nil)
		}
	}
}

func (g *Game) doClick(ge engineruntime.GameplayEvent) {
	if !ge.HasRay {
		g.selectedPeg = -1
		g.setPreviewLine(nil)
		return
	}
	hit := pickPeg(g.board.Pegs, ge.Ray.Origin, ge.Ray.Dir)
	if hit >= 0 {
		if g.selectedPeg >= 0 && g.selectedPeg != hit {
			// Second peg click — try to place band
			g.tryPlaceBand(g.selectedPeg, hit)
			g.selectedPeg = -1
		} else if g.selectedPeg == hit {
			g.selectedPeg = -1 // deselect
		} else {
			g.selectedPeg = hit
		}
	} else {
		g.selectedPeg = -1
	}
	g.setPreviewLine(nil)
}

func (g *Game) tryPlaceBand(a, b int) {
	line := g.board.FindLine(a, b)
	if line == nil {
		return
	}
	if !g.board.CanPlace(line) {
		return
	}
	claimed := g.board.PlaceBand(*line)
	g.bandsDirty = true
	if claimed > 0 {
		g.trisDirty = true
	}
}

func (g *Game) setPreviewLine(line *Line) {
	if sameLine(g.previewLine, line) {
		return
	}
	g.previewLine = line
	g.previewDirty = true
}

func sameLine(a, b *Line) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Pegs == b.Pegs
}
