package cpu6502

// instructions_arithmetic.go contains arithmetic instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Addition and subtraction (ADC, SBC)
//   - Increment and decrement (INC, INX, INY, DEC, DEX, DEY)
//   - Comparison operations (CMP, CPX, CPY)
//   - Binary Coded Decimal (BCD) support
//
// The 6502 supports both binary and BCD arithmetic modes, controlled by the D flag.

// ADC - Add with Carry
//
// Performs A = A + M + C, where:
//
//	A = Accumulator
//	M = Memory operand
//	C = Carry flag (0 or 1)
//
// In binary mode (D=0):
//   - Standard 8-bit addition with carry
//   - C flag set if result > 255 (unsigned overflow)
//   - V flag set if signed overflow occurs
//   - Z flag set if result is zero
//   - N flag set if bit 7 of result is 1
//
// In decimal mode (D=1):
//   - BCD (Binary Coded Decimal) addition
//   - Each nibble represents 0-9 (not 0-F)
//   - Adjustments made when nibble exceeds 9
//   - N/V flags based on binary intermediate result (NMOS behavior)
//   - C flag set if BCD result > 99
//   - Z flag set if BCD result is 00
func (c *CPU) ADC() uint8 {
	c.fetchDataIfNeeded()
	var carry uint16 = 0
	if c.getFlag(C) {
		carry = 1
	}

	// Check if decimal mode is supported and enabled
	if c.getFlag(D) && c.variant.SupportsDecimalMode() {
		return c.adcDecimal(carry)
	}

	// Binary mode (or decimal mode disabled)
	return c.adcBinary(carry)
}

// adcBinary performs binary mode addition
// Sets flags according to standard 8-bit arithmetic rules
func (c *CPU) adcBinary(carry uint16) uint8 {
	// Perform 16-bit addition to detect carry
	temp := uint16(c.A) + uint16(c.fetchedData) + carry

	// Set carry flag if result exceeds 8 bits (unsigned overflow)
	c.setFlag(C, temp > 0xFF)

	result := uint8(temp & 0x00FF)

	// Set overflow flag for signed arithmetic
	// V = (A^result) & (M^result) & 0x80
	// Overflow occurs when:
	// - Adding two positive numbers yields negative result
	// - Adding two negative numbers yields positive result
	c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^temp)&0x0080) > 0)

	c.A = result
	c.setZNFlags(c.A)
	return 0
}

// adcDecimal performs BCD (Binary Coded Decimal) addition
// Each nibble (4 bits) represents a decimal digit 0-9
func (c *CPU) adcDecimal(carry uint16) uint8 {
	// Step 1: Calculate binary result for N/V flags
	// NMOS 6502 sets N/V based on binary intermediate result, not BCD result
	binarySum := uint16(c.A) + uint16(c.fetchedData) + carry

	// Variant-specific N/V flag handling
	switch c.variant {
	case VariantNMOS6502:
		// NMOS: N/V based on binary intermediate result
		c.setFlag(N, (binarySum&0x80) > 0)
		c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
	case VariantCMOS65C02:
		// CMOS: N/V based on binary intermediate result (same as NMOS)
		c.setFlag(N, (binarySum&0x80) > 0)
		c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
	}

	// Step 2: BCD arithmetic - process lower nibble (ones digit)
	// Add lower 4 bits plus carry
	low := (c.A & 0x0F) + (c.fetchedData & 0x0F) + uint8(carry)
	lowCarry := uint8(0)

	// If lower nibble exceeds 9, adjust by adding 6 and carry to upper nibble
	// This converts invalid BCD (A-F) to valid BCD (0-9) with carry
	if low > 9 {
		low += 6     // Adjust to valid BCD
		lowCarry = 1 // Carry to upper nibble
	}

	// Step 3: BCD arithmetic - process upper nibble (tens digit)
	// Add upper 4 bits plus carry from lower nibble
	high := (c.A >> 4) + (c.fetchedData >> 4) + lowCarry

	// If upper nibble exceeds 9, adjust by adding 6 and set carry flag
	if high > 9 {
		high += 6          // Adjust to valid BCD
		c.setFlag(C, true) // Set carry for overflow beyond 99
	} else {
		c.setFlag(C, false)
	}

	// Step 4: Combine adjusted nibbles into final BCD result
	result := ((high & 0x0F) << 4) | (low & 0x0F)
	c.A = result
	c.setFlag(Z, c.A == 0)

	return 0
}

// SBC - Subtract with Carry (Borrow)
func (c *CPU) SBC() uint8 {
	c.fetchDataIfNeeded()

	// Check if decimal mode is supported and enabled
	if c.getFlag(D) && c.variant.SupportsDecimalMode() {
		return c.sbcDecimal()
	}

	// Binary mode (or decimal mode disabled)
	return c.sbcBinary()
}

func (c *CPU) sbcBinary() uint8 {
	value := uint16(c.fetchedData) ^ 0x00FF
	var carry uint16 = 0
	if c.getFlag(C) {
		carry = 1
	}

	temp := uint16(c.A) + value + carry
	c.setFlag(C, temp > 0xFF)
	result := uint8(temp & 0x00FF)
	c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^temp)&0x0080) > 0)
	c.A = result
	c.setZNFlags(c.A)

	return 0
}

// sbcDecimal performs BCD (Binary Coded Decimal) subtraction
// Each nibble (4 bits) represents a decimal digit 0-9
func (c *CPU) sbcDecimal() uint8 {
	// Step 1: Calculate binary result for N/V flags
	// Invert operand for binary calculation
	value := uint16(c.fetchedData) ^ 0x00FF
	var carry uint16 = 0
	if c.getFlag(C) {
		carry = 1
	}

	// NMOS 6502 sets N/V based on binary intermediate result, not BCD result
	binarySum := uint16(c.A) + value + carry

	// Variant-specific N/V flag handling
	switch c.variant {
	case VariantNMOS6502:
		c.setFlag(N, (binarySum&0x80) > 0)
		c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)
	case VariantCMOS65C02:
		c.setFlag(N, (binarySum&0x80) > 0)
		c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)
	}

	// Step 2: BCD subtraction - process lower nibble (ones digit)
	// Convert carry flag to borrow: borrow = 1 - carry
	borrow_in := 1 - uint8(carry)

	// Subtract lower nibbles with borrow
	low := int16(c.A&0x0F) - int16(c.fetchedData&0x0F) - int16(borrow_in)
	borrow_low := uint8(0)

	// If lower nibble goes negative, adjust by adding 10 and borrow from upper nibble
	// This converts negative BCD to valid BCD (0-9) with borrow
	if low < 0 {
		low += 10      // Adjust to valid BCD
		borrow_low = 1 // Borrow from upper nibble
	}

	// Step 3: BCD subtraction - process upper nibble (tens digit)
	// Subtract upper nibbles with borrow from lower nibble
	high := int16(c.A>>4) - int16(c.fetchedData>>4) - int16(borrow_low)

	// If upper nibble goes negative, adjust by adding 10 and clear carry flag
	if high < 0 {
		high += 10          // Adjust to valid BCD
		c.setFlag(C, false) // Clear carry (borrow occurred)
	} else {
		c.setFlag(C, true) // Set carry (no borrow)
	}

	// Step 4: Combine adjusted nibbles into final BCD result
	result := (uint8(high&0x0F) << 4) | uint8(low&0x0F)
	c.A = result
	c.setFlag(Z, c.A == 0)

	return 0
}

// INC - Increment Memory
// Adds 1 to the value at the memory location.
// Flags affected: Z, N
func (c *CPU) INC() uint8 {
	c.fetchDataIfNeeded()
	temp := c.fetchedData + 1
	c.write(c.addrAbs, temp)
	c.setZNFlags(temp)
	return 0
}

// INX - Increment X Register
// Adds 1 to the X register.
// Flags affected: Z, N
func (c *CPU) INX() uint8 {
	c.X++
	c.setZNFlags(c.X)
	return 0
}

// INY - Increment Y Register
// Adds 1 to the Y register.
// Flags affected: Z, N
func (c *CPU) INY() uint8 {
	c.Y++
	c.setZNFlags(c.Y)
	return 0
}

// DEC - Decrement Memory
// Subtracts 1 from the value at the memory location.
// Flags affected: Z, N
func (c *CPU) DEC() uint8 {
	c.fetchDataIfNeeded()
	temp := c.fetchedData - 1
	c.write(c.addrAbs, temp)
	c.setZNFlags(temp)
	return 0
}

// DEX - Decrement X Register
// Subtracts 1 from the X register.
// Flags affected: Z, N
func (c *CPU) DEX() uint8 {
	c.X--
	c.setZNFlags(c.X)
	return 0
}

// DEY - Decrement Y Register
// Subtracts 1 from the Y register.
// Flags affected: Z, N
func (c *CPU) DEY() uint8 {
	c.Y--
	c.setZNFlags(c.Y)
	return 0
}

// CMP - Compare Accumulator
// Compares the accumulator with a memory value by subtracting (A - M).
// The result is not stored, only flags are affected.
// C flag is set if A >= M (no borrow needed).
// Z flag is set if A == M.
// N flag is set based on bit 7 of the result.
// Flags affected: C, Z, N
func (c *CPU) CMP() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.A) - uint16(c.fetchedData)
	c.setFlag(C, c.A >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
}

// CPX - Compare X Register
// Compares the X register with a memory value by subtracting (X - M).
// The result is not stored, only flags are affected.
// C flag is set if X >= M (no borrow needed).
// Z flag is set if X == M.
// N flag is set based on bit 7 of the result.
// Flags affected: C, Z, N
func (c *CPU) CPX() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.X) - uint16(c.fetchedData)
	c.setFlag(C, c.X >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
}

// CPY - Compare Y Register
// Compares the Y register with a memory value by subtracting (Y - M).
// The result is not stored, only flags are affected.
// C flag is set if Y >= M (no borrow needed).
// Z flag is set if Y == M.
// N flag is set based on bit 7 of the result.
// Flags affected: C, Z, N
func (c *CPU) CPY() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.Y) - uint16(c.fetchedData)
	c.setFlag(C, c.Y >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
}
