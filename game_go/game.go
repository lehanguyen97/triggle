package main

/*
#cgo CFLAGS: -I../engine/include -I../engine/vendor/cglm/include
#include <e/engine_api.h>
#include <stdlib.h>

// Forward declaration of the Go function
extern int8_t update_rotation_system(void* data);
*/
import "C"
import (
	"math"
	"unsafe"
)

var rotationSpeed float32 = 0.02

//export update_rotation_system
func update_rotation_system(data unsafe.Pointer) C.int8_t {
	// Work directly with C memory to avoid conversion issues
	cTransform := (*C.transform_t)(data)

	// Direct manipulation - safer approach
	rotPtr := (*[3]C.float)(unsafe.Pointer(&cTransform.rotation))
	posPtr := (*[3]C.float)(unsafe.Pointer(&cTransform.position))

	// Apply rotation system
	rotPtr[0] += C.float(rotationSpeed)
	rotPtr[1] += C.float(rotationSpeed * 1.5)

	// Apply oscillation system
	time := float64(rotPtr[0])
	posPtr[0] += C.float(math.Sin(time*2) * 0.1)
	posPtr[1] += C.float(math.Cos(time*3) * 0.05)

	return 0
}

//export e_game_init
func e_game_init(engine C.engine_t) C.int {
	// Simplified initialization - no registry for now
	// Register the C callback directly
	C.e_register_system(engine, C.system_fn(C.update_rotation_system))
	return 0
}

//export e_game_shutdown
func e_game_shutdown() {
}

func main() {
}
