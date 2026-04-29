package emath

import "math"

// Float32 wrappers over math.Floor/Ceil/Round/Sqrt. Stdlib has no float32
// versions; the float64 round-trip compiles to a single SSE/NEON op plus two
// single-cycle conversions. All functions auto-inline.

func Floor32(v float32) float32 { return float32(math.Floor(float64(v))) }
func Ceil32(v float32) float32  { return float32(math.Ceil(float64(v))) }
func Round32(v float32) float32 { return float32(math.Round(float64(v))) }
func Sqrt32(v float32) float32  { return float32(math.Sqrt(float64(v))) }

// RoundToI32 rounds to nearest int32 (half-away-from-zero, matches math.Round).
// Use this instead of int32(v + 0.5) — that pattern is wrong for v < 0.
func RoundToI32(v float32) int32 { return int32(math.Round(float64(v))) }
