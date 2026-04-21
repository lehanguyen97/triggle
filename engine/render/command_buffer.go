package render

import (
	"encoding/binary"
	"math"
)

const (
	cmdMagic   uint32 = 0x31424354 // TCB1
	cmdVersion uint16 = 1

	cmdPassBegin        uint8 = 1
	cmdPassBeginDefault uint8 = 2
	cmdPassEnd          uint8 = 3
	cmdApplyPipeline    uint8 = 4
	cmdBindMesh         uint8 = 5
	cmdBindImage        uint8 = 6
	cmdApplyUniforms    uint8 = 7
	cmdDrawElements     uint8 = 8
	cmdCommit           uint8 = 9
	cmdApplyScissor     uint8 = 10
)

type commandBuffer struct {
	buf      []byte
	cmdCount uint32
}

func (c *commandBuffer) beginFrame() {
	c.buf = c.buf[:0]
	c.cmdCount = 0
	c.appendU32(cmdMagic)
	c.appendU16(cmdVersion)
	c.appendU16(0) // flags
	c.appendU32(0) // command count, patched in finish()
}

func (c *commandBuffer) finish() []byte {
	binary.LittleEndian.PutUint32(c.buf[8:12], c.cmdCount)
	return c.buf
}

func (c *commandBuffer) emit(opcode uint8, payload func()) {
	start := len(c.buf)
	c.appendU8(opcode)
	c.appendU8(0) // flags
	c.appendU16(0)
	payload()
	payloadSize := len(c.buf) - start - 4
	binary.LittleEndian.PutUint16(c.buf[start+2:start+4], uint16(payloadSize))
	c.cmdCount++
}

func (c *commandBuffer) appendU8(v uint8) {
	c.buf = append(c.buf, v)
}

func (c *commandBuffer) appendU16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	c.buf = append(c.buf, b[:]...)
}

func (c *commandBuffer) appendU32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	c.buf = append(c.buf, b[:]...)
}

func (c *commandBuffer) appendI32(v int32) {
	c.appendU32(uint32(v))
}

func (c *commandBuffer) appendF32(v float32) {
	c.appendU32(math.Float32bits(v))
}

func (c *commandBuffer) appendBytes(v []byte) { c.buf = append(c.buf, v...) }
