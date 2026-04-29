package emath

// Color is straight RGBA8.
type Color struct {
	R, G, B, A uint8
}

// Floats returns 0..1 components for shader attributes.
func (c Color) Floats() [4]float32 {
	return [4]float32{
		float32(c.R) / 255,
		float32(c.G) / 255,
		float32(c.B) / 255,
		float32(c.A) / 255,
	}
}
