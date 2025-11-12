package cpu6502

// instructions_shift.go contains shift and rotate instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - ASL - Arithmetic Shift Left
//   - LSR - Logical Shift Right
//   - ROL - Rotate Left
//   - ROR - Rotate Right
//
// All shift/rotate instructions set the C, Z, and N flags.
// They can operate on the accumulator (implied mode) or memory.

// ASL - Arithmetic Shift Left
// Shifts all bits left one position. Bit 0 is set to 0.
// The original bit 7 is shifted into the Carry flag.
// Flags affected: C, Z, N
func (c *CPU) ASL() uint8 {
	var temp uint16
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		// Operate on accumulator
		temp = uint16(c.A) << 1
		c.setFlag(C, (temp&0xFF00) > 0)
		c.A = uint8(temp & 0x00FF)
		c.setZNFlags(c.A)
	} else {
		// Operate on memory
		c.fetchDataIfNeeded() // Need data before modifying
		temp = uint16(c.fetchedData) << 1
		c.setFlag(C, (temp&0xFF00) > 0)
		result := uint8(temp & 0x00FF)
		c.write(c.addrAbs, result)
		c.setZNFlags(result)
	}
	return 0
}

// LSR - Logical Shift Right
// Shifts all bits right one position. Bit 7 is set to 0.
// The original bit 0 is shifted into the Carry flag.
// Flags affected: C, Z, N
func (c *CPU) LSR() uint8 {
	var temp uint8
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		// Operate on accumulator
		c.setFlag(C, (c.A&0x01) > 0)
		c.A >>= 1
		c.setZNFlags(c.A)
	} else {
		// Operate on memory
		c.fetchDataIfNeeded() // Need data before modifying
		c.setFlag(C, (c.fetchedData&0x01) > 0)
		temp = c.fetchedData >> 1
		c.write(c.addrAbs, temp)
		c.setZNFlags(temp)
	}
	return 0
}

// ROL - Rotate Left
// Shifts all bits left one position. The Carry flag is shifted into bit 0.
// The original bit 7 is shifted into the Carry flag.
// Flags affected: C, Z, N
func (c *CPU) ROL() uint8 {
	var temp uint16
	var carryBit uint16 = 0
	if c.getFlag(C) {
		carryBit = 1
	}

	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		// Operate on accumulator
		temp = (uint16(c.A) << 1) | carryBit
		c.setFlag(C, (temp&0xFF00) > 0)
		c.A = uint8(temp & 0x00FF)
		c.setZNFlags(c.A)
	} else {
		// Operate on memory
		c.fetchDataIfNeeded() // Need data before modifying
		temp = (uint16(c.fetchedData) << 1) | carryBit
		c.setFlag(C, (temp&0xFF00) > 0)
		result := uint8(temp & 0x00FF)
		c.write(c.addrAbs, result)
		c.setZNFlags(result)
	}
	return 0
}

// ROR - Rotate Right
// Shifts all bits right one position. The Carry flag is shifted into bit 7.
// The original bit 0 is shifted into the Carry flag.
// Flags affected: C, Z, N
func (c *CPU) ROR() uint8 {
	var temp uint8
	var carryBit uint8 = 0
	if c.getFlag(C) {
		carryBit = 0x80
	}

	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		// Operate on accumulator
		newCarry := (c.A & 0x01) > 0
		temp = (c.A >> 1) | carryBit
		c.setFlag(C, newCarry)
		c.A = temp
		c.setZNFlags(c.A)
	} else {
		// Operate on memory
		c.fetchDataIfNeeded() // Need data before modifying
		newCarry := (c.fetchedData & 0x01) > 0
		temp = (c.fetchedData >> 1) | carryBit
		c.write(c.addrAbs, temp)
		c.setFlag(C, newCarry)
		c.setZNFlags(temp)
	}
	return 0
}

// ROR_RevA - Rotate Right (Rev A Hardware Quirk Version)
// The original Rev A NMOS 6502 didn't have proper ROR circuitry.
// Instead of rotating right, it behaves like ASL (Arithmetic Shift Left):
//   - Shifts left instead of right (like ASL)
//   - Shifts a zero in instead of C (like ASL)
//   - Doesn't update C (unlike ASL, the carry flag is not modified)
//
// This quirk was fixed in Rev B and later revisions.
// Flags affected: Z, N (C is NOT affected, unlike normal ASL)
func (c *CPU) ROR_RevA() uint8 {
	var temp uint16
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		// Operate on accumulator - behaves like ASL but doesn't set C
		temp = uint16(c.A) << 1
		// Quirk: Carry flag is NOT updated (unlike ASL)
		c.A = uint8(temp & 0x00FF)
		c.setZNFlags(c.A)
	} else {
		// Operate on memory - behaves like ASL but doesn't set C
		c.fetchDataIfNeeded() // Need data before modifying
		temp = uint16(c.fetchedData) << 1
		// Quirk: Carry flag is NOT updated (unlike ASL)
		result := uint8(temp & 0x00FF)
		c.write(c.addrAbs, result)
		c.setZNFlags(result)
	}
	return 0
}
