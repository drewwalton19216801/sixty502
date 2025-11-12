package cpu6502

// instructions_flags.go contains flag manipulation instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Clear flag instructions (CLC, CLD, CLI, CLV)
//   - Set flag instructions (SEC, SED, SEI)
//
// All flag instructions execute in 2 cycles and use implied addressing mode.

// CLC - Clear Carry Flag
// Sets the Carry flag to 0.
// Flags affected: C
func (c *CPU) CLC() uint8 {
	c.setFlag(C, false)
	return 0
}

// CLD - Clear Decimal Mode Flag
// Sets the Decimal mode flag to 0, disabling BCD arithmetic.
// Flags affected: D
func (c *CPU) CLD() uint8 {
	c.setFlag(D, false)
	return 0
}

// CLI - Clear Interrupt Disable Flag
// Sets the Interrupt disable flag to 0, enabling IRQ interrupts.
// Flags affected: I
func (c *CPU) CLI() uint8 {
	c.setFlag(I, false)
	return 0
}

// CLV - Clear Overflow Flag
// Sets the Overflow flag to 0.
// Flags affected: V
func (c *CPU) CLV() uint8 {
	c.setFlag(V, false)
	return 0
}

// SEC - Set Carry Flag
// Sets the Carry flag to 1.
// Flags affected: C
func (c *CPU) SEC() uint8 {
	c.setFlag(C, true)
	return 0
}

// SED - Set Decimal Mode Flag
// Sets the Decimal mode flag to 1, enabling BCD arithmetic.
// Flags affected: D
func (c *CPU) SED() uint8 {
	c.setFlag(D, true)
	return 0
}

// SEI - Set Interrupt Disable Flag
// Sets the Interrupt disable flag to 1, disabling IRQ interrupts.
// Note: NMI interrupts cannot be disabled.
// Flags affected: I
func (c *CPU) SEI() uint8 {
	c.setFlag(I, true)
	return 0
}
