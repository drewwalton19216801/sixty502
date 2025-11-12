package cpu6502

// instructions_transfer.go contains load, store, and transfer instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Load instructions (LDA, LDX, LDY)
//   - Store instructions (STA, STX, STY)
//   - Register transfer instructions (TAX, TAY, TSX, TXA, TXS, TYA)
//
// Load and transfer instructions (except TXS) set the Z and N flags.
// Store instructions do not affect any flags.

// LDA - Load Accumulator
// Loads a value from memory into the accumulator.
// Flags affected: Z, N
func (c *CPU) LDA() uint8 {
	c.fetchDataIfNeeded()
	c.A = c.fetchedData
	c.setZNFlags(c.A)
	return 0
}

// LDX - Load X Register
// Loads a value from memory into the X register.
// Flags affected: Z, N
func (c *CPU) LDX() uint8 {
	c.fetchDataIfNeeded()
	c.X = c.fetchedData
	c.setZNFlags(c.X)
	return 0
}

// LDY - Load Y Register
// Loads a value from memory into the Y register.
// Flags affected: Z, N
func (c *CPU) LDY() uint8 {
	c.fetchDataIfNeeded()
	c.Y = c.fetchedData
	c.setZNFlags(c.Y)
	return 0
}

// STA - Store Accumulator
// Stores the accumulator value to memory.
// Flags affected: None
func (c *CPU) STA() uint8 {
	c.write(c.addrAbs, c.A)
	return 0
}

// STX - Store X Register
// Stores the X register value to memory.
// Flags affected: None
func (c *CPU) STX() uint8 {
	c.write(c.addrAbs, c.X)
	return 0
}

// STY - Store Y Register
// Stores the Y register value to memory.
// Flags affected: None
func (c *CPU) STY() uint8 {
	c.write(c.addrAbs, c.Y)
	return 0
}

// TAX - Transfer Accumulator to X
// Copies the accumulator value to the X register.
// Flags affected: Z, N
func (c *CPU) TAX() uint8 {
	c.X = c.A
	c.setZNFlags(c.X)
	return 0
}

// TAY - Transfer Accumulator to Y
// Copies the accumulator value to the Y register.
// Flags affected: Z, N
func (c *CPU) TAY() uint8 {
	c.Y = c.A
	c.setZNFlags(c.Y)
	return 0
}

// TSX - Transfer Stack Pointer to X
// Copies the stack pointer value to the X register.
// Flags affected: Z, N
func (c *CPU) TSX() uint8 {
	c.X = c.SP
	c.setZNFlags(c.X)
	return 0
}

// TXA - Transfer X to Accumulator
// Copies the X register value to the accumulator.
// Flags affected: Z, N
func (c *CPU) TXA() uint8 {
	c.A = c.X
	c.setZNFlags(c.A)
	return 0
}

// TXS - Transfer X to Stack Pointer
// Copies the X register value to the stack pointer.
// Flags affected: None
func (c *CPU) TXS() uint8 {
	c.SP = c.X
	return 0
}

// TYA - Transfer Y to Accumulator
// Copies the Y register value to the accumulator.
// Flags affected: Z, N
func (c *CPU) TYA() uint8 {
	c.A = c.Y
	c.setZNFlags(c.A)
	return 0
}
