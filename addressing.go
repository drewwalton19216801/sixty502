package cpu6502

// Addressing Modes
//
// The 6502 supports 13 different addressing modes that determine how
// instructions access their operands. Each addressing mode method:
//   - Calculates the effective address (stored in c.addrAbs)
//   - Returns 0 or 1 to indicate if a page boundary was crossed
//   - Advances the program counter as needed
//
// Page boundary crossing occurs when the high byte of the effective
// address differs from the high byte of the base address. Some
// instructions add an extra cycle when this happens.

// IMP - Implied addressing mode.
//
// The operand is implied by the instruction itself. No additional
// bytes are read from memory. Used by instructions like:
//   - Register transfers (TAX, TXA, etc.)
//   - Stack operations (PHA, PLA, etc.)
//   - Flag operations (CLC, SEC, etc.)
//
// Example: TAX (Transfer A to X)
//   - No operand needed
//   - Operation is implied
//
// Returns 0 (no page cross possible).
func (c *CPU) IMP() uint8 {
	return 0
}

// IMM - Immediate addressing mode.
//
// The operand is the byte immediately following the opcode.
// The effective address is the current PC value.
//
// Example: LDA #$42
//   - Opcode at PC
//   - Operand $42 at PC+1
//   - Loads the literal value $42 into A
//
// Returns 0 (no page cross possible).
func (c *CPU) IMM() uint8 {
	c.addrAbs = c.PC
	c.PC++
	return 0
}

// ZP0 - Zero Page addressing mode.
//
// The operand is located in the zero page (addresses $0000-$00FF).
// Only one byte is needed to specify the address, making this mode
// faster and more compact than absolute addressing.
//
// Example: LDA $42
//   - Opcode at PC
//   - Zero page address $42 at PC+1
//   - Loads value from address $0042
//
// Returns 0 (no page cross possible - always in zero page).
func (c *CPU) ZP0() uint8 {
	c.addrAbs = uint16(c.read(c.PC))
	c.PC++
	c.addrAbs &= 0x00FF // Ensure we stay in zero page
	return 0
}

// ZPX - Zero Page,X addressing mode.
//
// Similar to ZP0, but the X register is added to the zero page address.
// The result wraps around within the zero page (no carry to high byte).
//
// Example: LDA $42,X (with X=$10)
//   - Opcode at PC
//   - Base address $42 at PC+1
//   - Effective address: ($42 + $10) & $FF = $52
//   - Loads value from address $0052
//
// Wrapping example: LDA $FF,X (with X=$02)
//   - Effective address: ($FF + $02) & $FF = $01
//   - Wraps to $0001, not $0101
//
// Returns 0 (no page cross possible - wraps within zero page).
func (c *CPU) ZPX() uint8 {
	baseAddr := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (baseAddr + uint16(c.X)) & 0x00FF
	return 0
}

// ZPY - Zero Page,Y addressing mode.
//
// Similar to ZPX, but uses the Y register instead of X.
// Only used by LDX and STX instructions.
//
// Example: LDX $42,Y (with Y=$10)
//   - Effective address: ($42 + $10) & $FF = $52
//   - Loads value from address $0052 into X
//
// Returns 0 (no page cross possible - wraps within zero page).
func (c *CPU) ZPY() uint8 {
	baseAddr := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (baseAddr + uint16(c.Y)) & 0x00FF
	return 0
}

// REL - Relative addressing mode.
//
// Used exclusively by branch instructions. The operand is a signed
// 8-bit offset (-128 to +127) relative to the address of the next
// instruction (PC after reading the offset).
//
// The effective address is calculated as: PC + offset
// If bit 7 of the offset is set, it's treated as negative.
//
// Example: BEQ $10 (at address $8000)
//   - Opcode at $8000
//   - Offset $10 at $8001
//   - PC after reading = $8002
//   - Branch target = $8002 + $10 = $8012
//
// Negative example: BEQ $FE (at address $8000)
//   - Offset $FE = -2 in two's complement
//   - PC after reading = $8002
//   - Branch target = $8002 + (-2) = $8000
//
// Returns 0 (page cross is handled by branch instructions).
func (c *CPU) REL() uint8 {
	addrRelOffset := uint16(c.read(c.PC))
	c.PC++
	// Sign extend if negative (bit 7 set)
	if addrRelOffset&0x80 > 0 {
		addrRelOffset |= 0xFF00
	}
	c.addrRel = c.PC + addrRelOffset
	return 0
}

// ABS - Absolute addressing mode.
//
// The full 16-bit address is specified in the next two bytes
// (low byte first, then high byte - little endian).
//
// Example: LDA $1234
//   - Opcode at PC
//   - Low byte $34 at PC+1
//   - High byte $12 at PC+2
//   - Loads value from address $1234
//
// Returns 0 (no page cross possible - address is explicit).
func (c *CPU) ABS() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	return 0
}

// ABX - Absolute,X addressing mode.
//
// The X register is added to the 16-bit base address to form
// the effective address. If the addition crosses a page boundary,
// returns 1 to indicate an extra cycle may be needed.
//
// Example: LDA $1234,X (with X=$10)
//   - Base address $1234
//   - Effective address: $1234 + $10 = $1244
//   - No page cross (both in page $12)
//
// Page cross example: LDA $12FF,X (with X=$02)
//   - Base address $12FF
//   - Effective address: $12FF + $02 = $1301
//   - Page cross! ($12 -> $13)
//   - Returns 1 for potential extra cycle
//
// Returns 1 if page boundary crossed, 0 otherwise.
func (c *CPU) ABX() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	baseAddr := (hi << 8) | lo
	c.addrAbs = baseAddr + uint16(c.X)

	// Check if page boundary was crossed
	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}

// ABY - Absolute,Y addressing mode.
//
// Similar to ABX, but uses the Y register instead of X.
//
// Example: LDA $1234,Y (with Y=$10)
//   - Effective address: $1234 + $10 = $1244
//
// Returns 1 if page boundary crossed, 0 otherwise.
func (c *CPU) ABY() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	baseAddr := (hi << 8) | lo
	c.addrAbs = baseAddr + uint16(c.Y)

	// Check if page boundary was crossed
	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}

// IND - Indirect addressing mode.
//
// Used only by JMP instruction. The operand is a 16-bit address
// that points to the actual target address (pointer to pointer).
//
// NMOS 6502 Bug: If the low byte of the pointer is $FF, the high
// byte is fetched from $xx00 instead of $xx00+1, due to a hardware
// bug. The CMOS 65C02 fixes this bug.
//
// Example: JMP ($1234)
//   - Pointer address $1234 specified in instruction
//   - Low byte of target read from $1234
//   - High byte of target read from $1235
//   - Jump to the constructed 16-bit address
//
// Bug example (NMOS only): JMP ($12FF)
//   - Low byte read from $12FF
//   - High byte read from $1200 (not $1300!)
//   - This is the famous indirect JMP bug
//
// Returns 0 (no page cross possible).
func (c *CPU) IND() uint8 {
	ptrLo := uint16(c.read(c.PC))
	c.PC++
	ptrHi := uint16(c.read(c.PC))
	c.PC++
	ptr := (ptrHi << 8) | ptrLo

	if c.variant.HasIndirectJMPBug() && ptrLo == 0x00FF {
		// NMOS bug: If low byte is $FF, high byte is fetched from $xx00
		c.addrAbs = uint16(c.read(ptr)) | (uint16(c.read(ptr&0xFF00)) << 8)
	} else {
		// CMOS fix: Normal behavior
		c.addrAbs = (uint16(c.read(ptr+1)) << 8) | uint16(c.read(ptr))
	}
	return 0
}

// IZX - Indexed Indirect addressing mode (Indirect,X).
//
// The X register is added to the zero page address to get a pointer
// address. The actual operand address is then read from this pointer.
// All arithmetic wraps within the zero page.
//
// Example: LDA ($40,X) with X=$05
//   - Base zero page address $40
//   - Add X: $40 + $05 = $45
//   - Read pointer from $0045 (low) and $0046 (high)
//   - If pointer = $1234, load from $1234
//
// Wrapping example: LDA ($FF,X) with X=$02
//   - Pointer address: ($FF + $02) & $FF = $01
//   - Read pointer from $0001 and $0002
//   - Note: wraps within zero page
//
// Returns 0 (no page cross possible in pointer calculation).
func (c *CPU) IZX() uint8 {
	zpAddrBase := uint16(c.read(c.PC))
	c.PC++
	// Add X and wrap around zero page
	zpAddr := (zpAddrBase + uint16(c.X)) & 0x00FF

	// Read the effective address from zero page, wrapping around if needed
	effAddrLo := uint16(c.read(zpAddr))
	effAddrHi := uint16(c.read((zpAddr + 1) & 0x00FF))
	c.addrAbs = (effAddrHi << 8) | effAddrLo
	return 0
}

// IZY - Indirect Indexed addressing mode (Indirect),Y.
//
// A zero page address points to a base address. The Y register is
// then added to this base address to get the effective address.
// If the addition crosses a page boundary, returns 1.
//
// Example: LDA ($40),Y with Y=$10
//   - Read pointer from $0040 (low) and $0041 (high)
//   - If pointer = $1234, base address = $1234
//   - Add Y: $1234 + $10 = $1244
//   - Load from $1244
//
// Page cross example: LDA ($40),Y with Y=$10
//   - Pointer at $40 = $12FF
//   - Add Y: $12FF + $10 = $130F
//   - Page cross! ($12 -> $13)
//   - Returns 1 for potential extra cycle
//
// Returns 1 if page boundary crossed, 0 otherwise.
func (c *CPU) IZY() uint8 {
	zpAddr := uint16(c.read(c.PC))
	c.PC++

	// Read base address from zero page (with wrapping)
	baseAddrLo := uint16(c.read(zpAddr & 0x00FF))
	baseAddrHi := uint16(c.read((zpAddr + 1) & 0x00FF))
	baseAddr := (baseAddrHi << 8) | baseAddrLo

	// Add Y to base address
	c.addrAbs = baseAddr + uint16(c.Y)

	// Check if page boundary was crossed
	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}
