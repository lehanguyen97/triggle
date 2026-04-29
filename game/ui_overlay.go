package main

import (
	"errors"
	"strings"

	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui"
)

// initUI builds the HUD as plain ui primitives. The "chat box" is just a
// vertical Flex of (host log + chat log + name input + message input) wrapped
// in the HUD window. Game owns the widget refs and acts as the controller
// (see reconcileChatLog and handleChatSubmit), mirroring a Godot script
// attached to a VBoxContainer.
func (g *Game) initUI() error {
	fonts := g.host.Fonts()
	if fonts == nil {
		return errors.New("triggle: nil font set")
	}
	if _, err := fonts.Open(text.FontDefault, defaultUIFontPath()); err != nil {
		return err
	}

	th := ui.DefaultTheme()
	logColor := emath.Color{R: 235, G: 235, B: 240, A: 255}

	g.uiFPS = &ui.FPSCounter{}
	g.uiHostLog = &ui.Flex{Direction: ui.FlexColumn, Gap: th.Spacing}
	g.uiChatLog = &ui.Flex{Direction: ui.FlexColumn, Gap: th.Spacing}

	g.chat.UserName = "hello"
	g.uiNameInput = &ui.TextInput{
		MaxBytes: 128,
		Binding: ui.StringBinding{
			Get: func() string { return g.chat.UserName },
			Set: func(v string) { g.chat.UserName = v },
		},
	}
	g.uiMsgInput = &ui.TextInput{
		MaxBytes: 128,
		Binding: ui.StringBinding{
			Get: func() string { return g.chat.Draft },
			Set: func(v string) { g.chat.Draft = v },
		},
		OnSubmit: g.handleChatSubmit,
	}
	g.uiLogColor = logColor

	hud := &ui.Window{
		Title: "HUD",
		Flags: ui.WindowNoResize | ui.WindowNoClose | ui.WindowUseParentRect,
		Child: &ui.Padding{
			Insets: th.Padding,
			Child: &ui.Flex{
				Direction: ui.FlexColumn,
				Gap:       th.Spacing,
				Kids: []ui.FlexItem{
					{Node: g.uiFPS},
					{Node: g.uiHostLog},
					{Node: g.uiChatLog},
					{Node: g.uiNameInput},
					{Node: g.uiMsgInput},
				},
			},
		},
	}

	overlay, err := g.host.NewRoot(ui.RootOptions{
		Theme: th,
	})
	if err != nil {
		return err
	}
	g.overlay = overlay
	overlay.SetRoot(&ui.AnchorPanel{
		Kids: []ui.AnchorChild{{
			Node:   hud,
			Anchor: ui.AlignTopLeft,
			Pivot:  ui.AlignTopLeft,
			Offset: emath.Vec2{10, 10},
			Width:  ui.ClampFrac(0.30, 320, 480),
		}},
	})
	return nil
}

// buildUI runs once per frame before the UI ticks. It reconciles the
// game-owned log slices into the widget tree (append-only).
func (g *Game) buildUI() {
	if g.overlay == nil {
		return
	}
	g.reconcileLogColumn(g.uiHostLog, &g.lastHostLogN, g.logBuf)
	g.reconcileChatLog()
}

// reconcileLogColumn appends a Label child for each new line in lines. lastN
// tracks how many lines have already been mirrored. Replaces the whole column
// when the source slice shrinks (e.g. logs were rotated).
func (g *Game) reconcileLogColumn(col *ui.Flex, lastN *int, lines []string) {
	if col == nil {
		return
	}
	if len(lines) < *lastN {
		col.Kids = col.Kids[:0]
		*lastN = 0
	}
	for i := *lastN; i < len(lines); i++ {
		col.Kids = append(col.Kids, ui.FlexItem{Node: &ui.Label{
			Text:  lines[i],
			Style: ui.TextStyle{Color: g.uiLogColor},
		}})
	}
	if len(lines) != *lastN {
		*lastN = len(lines)
		col.Invalidate()
	}
}

// reconcileChatLog mirrors chat.Messages into the chat log column as
// formatted Labels. Append-only fast path; full rebuild on shrink.
func (g *Game) reconcileChatLog() {
	col := g.uiChatLog
	if col == nil {
		return
	}
	msgs := g.chat.Messages
	if len(msgs) < g.lastChatN {
		col.Kids = col.Kids[:0]
		g.lastChatN = 0
	}
	for i := g.lastChatN; i < len(msgs); i++ {
		col.Kids = append(col.Kids, ui.FlexItem{Node: &ui.Label{
			Text:  msgs[i].User + ": " + msgs[i].Text,
			Style: ui.TextStyle{Color: g.uiLogColor},
		}})
	}
	if len(msgs) != g.lastChatN {
		g.lastChatN = len(msgs)
		col.Invalidate()
	}
}

// handleChatSubmit is wired as MsgInput.OnSubmit. text is the just-typed
// message buffer (TextInput passes this in before applying its own write-back
// to Draft.Set, so clearing Draft below is not stomped).
func (g *Game) handleChatSubmit(text string) {
	msg := strings.TrimSpace(text)
	if msg == "" {
		return
	}
	user := strings.TrimSpace(g.chat.UserName)
	if user == "" {
		user = "anon"
	}
	g.chat.Messages = append(g.chat.Messages, ChatMessage{User: user, Text: msg})
	g.chat.Draft = ""
	g.uiMsgInput.SetValue("")
	g.uiMsgInput.RequestFocus()
}

func (g *Game) closeUI() {
	if overlay := g.overlay; overlay != nil {
		overlay.Close()
		g.overlay = nil
	}
}
