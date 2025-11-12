package cpu6502

// instructions_logical.go contains logical operation instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - AND - Logical AND
//   - ORA - Logical OR (inclusive)
//   - EOR - Exclusive OR
//   - BIT - Bit Test
//
// All logical instructions (except BIT) set the Z and N flags based on the result.

// AND - Logical AND
// Performs a bitwise AND between the accumulator and memory.
// Result is stored in the accumulator.
// Flags affected: Z, N
func (c *CPU) AND() uint8 {
	c.fetchDataIfNeeded()
	c.A &= c.fetchedData
	c.setZNFlags(c.A)
	return 0
}

// ORA - Logical OR (Inclusive)
// Performs a bitwise OR between the accumulator and memory.
// Result is stored in the accumulator.
// Flags affected: Z, N
func (c *CPU) ORA() uint8 {
	c.fetchDataIfNeeded()
	c.A |= c.fetchedData
	c.setZNFlags(c.A)
	return 0
}

// EOR - Exclusive OR
// Performs a bitwise XOR between the accumulator and memory.
// Result is stored in the accumulator.
// Flags affected: Z, N
func (c *CPU) EOR() uint8 {
	c.fetchDataIfNeeded()
	c.A ^= c.fetchedData
	c.setZNFlags(c.A)
	return 0
}

// BIT - Bit Test
// Tests bits in memory with the accumulator without storing the result.
// The Z flag is set based on the AND result (A & M).
// The N flag is set to bit 7 of the memory value.
// The V flag is set to bit 6 of the memory value.
// Flags affected: Z, N, V
func (c *CPU) BIT() uint8 {
	c.fetchDataIfNeeded()
	temp := c.A & c.fetchedData
	c.setFlag(Z, temp == 0)
	c.setFlag(N, (c.fetchedData&(1<<7)) > 0)
	c.setFlag(V, (c.fetchedData&(1<<6)) > 0)
	return 0
}
