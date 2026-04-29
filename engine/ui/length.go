package ui

// Length resolves a size from a fixed lp value or a fraction of the parent,
// with optional min/max clamps. Zero means "use the child's measured size".
type Length struct {
	Lp   float32
	Frac float32
	Min  float32
	Max  float32
}

// Lp returns a fixed logical-pixel length.
func Lp(v float32) Length { return Length{Lp: v} }

// Frac returns a parent-relative length.
func Frac(v float32) Length { return Length{Frac: v} }

// ClampFrac returns a parent-relative length clamped to [min,max] lp.
func ClampFrac(frac, min, max float32) Length {
	return Length{Frac: frac, Min: min, Max: max}
}

func (l Length) specified() bool { return l.Lp > 0 || l.Frac > 0 || l.Min > 0 || l.Max > 0 }

func (l Length) resolve(parent, fallback float32) float32 {
	out := fallback
	if l.Lp > 0 {
		out = l.Lp
	} else if l.Frac > 0 {
		out = parent * l.Frac
	}
	if l.Min > 0 && out < l.Min {
		out = l.Min
	}
	if l.Max > 0 && out > l.Max {
		out = l.Max
	}
	if out < 0 {
		out = 0
	}
	return out
}

// ----------------------------------------------------------------------------
// Constraint / size helpers shared across widgets.
// ----------------------------------------------------------------------------

func looseConstraints(c Constraints) Constraints {
	return Constraints{MaxW: c.MaxW, MaxH: c.MaxH}
}

func tightConstraints(s Size) Constraints {
	return Constraints{MinW: s.W, MaxW: s.W, MinH: s.H, MaxH: s.H}
}

func clampSize(c Constraints, s Size) Size {
	if s.W < c.MinW {
		s.W = c.MinW
	}
	if c.MaxW > 0 && s.W > c.MaxW {
		s.W = c.MaxW
	}
	if s.H < c.MinH {
		s.H = c.MinH
	}
	if c.MaxH > 0 && s.H > c.MaxH {
		s.H = c.MaxH
	}
	return s
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
