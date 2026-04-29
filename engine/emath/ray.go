package emath

// RaySphere tests a ray against a sphere. Returns distance t or -1 if no hit.
func RaySphere(origin, dir, center [3]float32, radius float32) float32 {
	ox := origin[0] - center[0]
	oy := origin[1] - center[1]
	oz := origin[2] - center[2]
	a := dir[0]*dir[0] + dir[1]*dir[1] + dir[2]*dir[2]
	b := 2.0 * (ox*dir[0] + oy*dir[1] + oz*dir[2])
	c := ox*ox + oy*oy + oz*oz - radius*radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return -1
	}
	sqrtDisc := Sqrt32(disc)
	t := (-b - sqrtDisc) / (2 * a)
	if t < 0 {
		t = (-b + sqrtDisc) / (2 * a)
	}
	if t < 0 {
		return -1
	}
	return t
}
