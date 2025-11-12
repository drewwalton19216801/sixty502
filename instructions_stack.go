package cpu6502

// instructions_stack.go contains stack operation instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Push instructions (PHA, PHP)
//   - Pull instructions (PLA, PLP)
//
// The 6502 stack is located at $0100-$01FF and grows downward.
// The stack pointer (SP) points to the next available location.

// PHA - Push Accumulator
// Pushes the accumulator value onto the stack.
// Stack pointer is decremented after the push.
// Flags affected: None
func (c *CPU) PHA() uint8 {
	c.push(c.A)
	return 0
}

// PHP - Push Processor Status
// Pushes the processor status register (flags) onto the stack.
// The B and U flags are set in the pushed copy but not in the actual register.
// Stack pointer is decremented after the push.
// Flags affected: None
func (c *CPU) PHP() uint8 {
	c.push(uint8(c.P | B | U))
	return 0
}

// PLA - Pull Accumulator
// Pulls a value from the stack into the accumulator.
// Stack pointer is incremented before the pull.
// Flags affected: Z, N
func (c *CPU) PLA() uint8 {
	c.A = c.pop()
	c.setZNFlags(c.A)
	return 0
}

// PLP - Pull Processor Status
// Pulls a value from the stack into the processor status register (flags).
// The B and U flags are preserved from the current register, not the pulled value.
// Stack pointer is incremented before the pull.
// Flags affected: All (except B and U which are preserved)
func (c *CPU) PLP() uint8 {
	poppedP := Flags(c.pop())
	// Preserve B and U flags from current P, take all others from popped value
	c.P = (poppedP & ^(B | U)) | (c.P & (B | U))
	// Ensure U is always set
	c.setFlag(U, true)
	return 0
}
