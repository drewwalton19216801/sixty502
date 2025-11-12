package cpu6502

// helpers.go contains shared helper functions used by instruction implementations.

// fetchDataIfNeeded fetches data from memory if not in implied addressing mode.
// For immediate mode, it reads the byte at addrAbs (which points to the operand).
// For other modes, it reads from the calculated addrAbs.
func (c *CPU) fetchDataIfNeeded() {
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType != AddrModeIMP {
		// Immediate mode reads the byte *at* addrAbs (which points to the operand)
		// Other modes read from the calculated addrAbs.
		c.fetchedData = c.read(c.addrAbs)
	}
	// No special handling needed here anymore for IMM vs others regarding fetching
	// as IMM sets addrAbs correctly.
}

// setZNFlags sets the Zero and Negative flags based on a value.
// The Zero flag is set if the value is 0.
// The Negative flag is set if bit 7 of the value is 1.
func (c *CPU) setZNFlags(value uint8) {
	c.setFlag(Z, value == 0)
	c.setFlag(N, (value&0x80) > 0)
}
