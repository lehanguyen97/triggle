package runtime

import (
	"unicode/utf8"

	"triggle/engine/event"
	"triggle/engine/ui"
)

// applyEventToUIFrame updates a ui.InputFrame from one host event.
func applyEventToUIFrame(frame *ui.InputFrame, ev *event.Event) {
	if frame == nil || ev == nil {
		return
	}
	switch ev.Kind {
	case event.KindKey:
		key := ui.KeyCode(ev.Key)
		if key != ui.KeyUnknown {
			frame.KeyEvents = append(frame.KeyEvents, ui.KeyEvent{
				Key:    key,
				Down:   ev.Down,
				Repeat: ev.Repeat,
				Mods:   uint8(ev.Mods),
			})
		}
	case event.KindText:
		if ev.Rune > 0 && utf8.ValidRune(ev.Rune) {
			var buf [4]byte
			n := utf8.EncodeRune(buf[:], ev.Rune)
			frame.Text += string(buf[:n])
		}
	case event.KindMouseButton:
		frame.MousePos = ev.Pos
		if ev.Button > event.MouseMiddle {
			return
		}
		bit := uint8(1) << ev.Button
		if ev.Down {
			frame.MouseDown |= bit
			frame.MousePressed |= bit
			return
		}
		frame.MouseReleased |= bit
		frame.MouseDown &^= bit
	case event.KindMouseMove:
		frame.MousePos = ev.Pos
		frame.MouseDelta = ev.Delta
	case event.KindMouseScroll:
		frame.MousePos = ev.Pos
		frame.ScrollDelta = frame.ScrollDelta.Add(ev.Delta)
	}
}
