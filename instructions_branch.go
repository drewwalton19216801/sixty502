package cpu6502

// instructions_branch.go contains branch instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Branch helper function
//   - All conditional branch instructions (BCC, BCS, BEQ, BMI, BNE, BPL, BVC, BVS)
//
// Branch instructions take 2 cycles normally, +1 if branch is taken,
// +1 more if the branch crosses a page boundary.

// branchIf is a helper function that handles conditional branching.
// It takes a condition and branches to addrRel if the condition is true.
// Returns extra cycles: +1 if branch taken, +1 more if page boundary crossed.
func (c *CPU) branchIf(condition bool) uint8 {
	cycles := uint8(0)
	if condition {
		cycles++
		// Check if branch crosses page boundary
		if (c.addrRel & 0xFF00) != (c.PC & 0xFF00) {
			cycles++
		}
		c.PC = c.addrRel
	}
	return cycles
}

// BCC - Branch if Carry Clear
// Branches if the Carry flag is 0.
// Flags affected: None
func (c *CPU) BCC() uint8 {
	return c.branchIf(!c.getFlag(C))
}

// BCS - Branch if Carry Set
// Branches if the Carry flag is 1.
// Flags affected: None
func (c *CPU) BCS() uint8 {
	return c.branchIf(c.getFlag(C))
}

// BEQ - Branch if Equal (Zero Set)
// Branches if the Zero flag is 1.
// Flags affected: None
func (c *CPU) BEQ() uint8 {
	return c.branchIf(c.getFlag(Z))
}

// BMI - Branch if Minus (Negative Set)
// Branches if the Negative flag is 1.
// Flags affected: None
func (c *CPU) BMI() uint8 {
	return c.branchIf(c.getFlag(N))
}

// BNE - Branch if Not Equal (Zero Clear)
// Branches if the Zero flag is 0.
// Flags affected: None
func (c *CPU) BNE() uint8 {
	return c.branchIf(!c.getFlag(Z))
}

// BPL - Branch if Plus (Negative Clear)
// Branches if the Negative flag is 0.
// Flags affected: None
func (c *CPU) BPL() uint8 {
	return c.branchIf(!c.getFlag(N))
}

// BVC - Branch if Overflow Clear
// Branches if the Overflow flag is 0.
// Flags affected: None
func (c *CPU) BVC() uint8 {
	return c.branchIf(!c.getFlag(V))
}

// BVS - Branch if Overflow Set
// Branches if the Overflow flag is 1.
// Flags affected: None
func (c *CPU) BVS() uint8 {
	return c.branchIf(c.getFlag(V))
}
