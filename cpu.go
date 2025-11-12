package cpu6502

import (
	"fmt"
	"log"
	"reflect" // Import reflect package
)

// AddrModeType represents the addressing mode of an instruction
type AddrModeType uint8

const (
	AddrModeIMP AddrModeType = iota // Implied
	AddrModeIMM                     // Immediate
	AddrModeZP0                     // Zero Page
	AddrModeZPX                     // Zero Page, X
	AddrModeZPY                     // Zero Page, Y
	AddrModeREL                     // Relative
	AddrModeABS                     // Absolute
	AddrModeABX                     // Absolute, X
	AddrModeABY                     // Absolute, Y
	AddrModeIND                     // Indirect
	AddrModeIZX                     // Indexed Indirect
	AddrModeIZY                     // Indirect Indexed
)

// String returns the addressing mode name for debugging
func (a AddrModeType) String() string {
	names := []string{
		"IMP", "IMM", "ZP0", "ZPX", "ZPY", "REL",
		"ABS", "ABX", "ABY", "IND", "IZX", "IZY",
	}
	if int(a) < len(names) {
		return names[a]
	}
	return "UNKNOWN"
}

// Bus interface (remains the same)
type Bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, data uint8)
}

// Flags type (remains the same)
type Flags uint8

const (
	C Flags = 1 << 0 // Carry Bit
	Z Flags = 1 << 1 // Zero
	I Flags = 1 << 2 // Disable Interrupts
	D Flags = 1 << 3 // Decimal Mode (rarely used, often ignored in NES emu)
	B Flags = 1 << 4 // Break Command
	U Flags = 1 << 5 // Unused (always 1)
	V Flags = 1 << 6 // Overflow
	N Flags = 1 << 7 // Negative
)

// CPU struct (remains the same)
type CPU struct {
	// Registers
	A  uint8  // Accumulator
	X  uint8  // X Index Register
	Y  uint8  // Y Index Register
	SP uint8  // Stack Pointer (relative to $0100)
	PC uint16 // Program Counter
	P  Flags  // Processor Status Register (Flags)

	// Bus connection
	bus Bus

	// Internal state for instruction execution
	Cycles             uint8        // Cycles remaining for the current instruction
	opcode             uint8        // Current opcode being executed
	fetchedData        uint8        // Data fetched by addressing mode
	addrAbs            uint16       // Absolute address calculated by addressing mode
	addrRel            uint16       // Relative address (for branching) - stores absolute target address
	currentInstruction *Instruction // Pointer to the definition of the current instruction

	// Lookup table for instructions
	lookup [256]Instruction

	// Total cycles executed (for debugging/profiling)
	totalCycles uint64
}

// Instruction struct: Use method expressions for types
type Instruction struct {
	Name         string           // Mnemonic (e.g., "LDA")
	Operate      func(*CPU) uint8 // Function to execute the instruction's logic (accepts *CPU)
	AddrMode     func(*CPU) uint8 // Function to calculate the address and fetch data (accepts *CPU)
	AddrModeType AddrModeType     // Type of addressing mode
	Cycles       uint8            // Base cycles for this instruction/mode
	Length       uint8            // Length of the instruction in bytes
	Illegal      bool             // Whether this is an official or unofficial/illegal opcode
}

// NewCPU (remains the same, buildLookupTable called within)
func NewCPU(bus Bus) *CPU {
	c := &CPU{
		bus: bus,
		P:   U | I,
		SP:  0xFD,
	}
	c.buildLookupTable()
	return c
}

// Bus Interaction (remains the same)
func (c *CPU) read(addr uint16) uint8 {
	return c.bus.Read(addr)
}

func (c *CPU) write(addr uint16, data uint8) {
	c.bus.Write(addr, data)
}

// Stack Operations (remains the same)
const stackBase uint16 = 0x0100

func (c *CPU) push(data uint8) {
	c.write(stackBase+uint16(c.SP), data)
	c.SP--
}

func (c *CPU) pop() uint8 {
	c.SP++
	return c.read(stackBase + uint16(c.SP))
}

func (c *CPU) push16(data uint16) {
	c.push(uint8(data >> 8))
	c.push(uint8(data & 0xFF))
}

func (c *CPU) pop16() uint16 {
	lo := uint16(c.pop())
	hi := uint16(c.pop())
	return (hi << 8) | lo
}

// Flag Operations (remains the same)
func (c *CPU) setFlag(flag Flags, value bool) {
	if value {
		c.P |= flag
	} else {
		c.P &= ^flag
	}
}

func (c *CPU) getFlag(flag Flags) bool {
	return (c.P & flag) > 0
}

// --- Addressing Modes ---
// All addressing mode functions remain the same method signatures:
// func (c *CPU) ModeName() uint8 { ... }

func (c *CPU) IMP() uint8 {
	return 0
}

func (c *CPU) IMM() uint8 {
	c.addrAbs = c.PC
	c.PC++
	return 0
}

func (c *CPU) ZP0() uint8 {
	c.addrAbs = uint16(c.read(c.PC))
	c.PC++
	c.addrAbs &= 0x00FF
	return 0
}

func (c *CPU) ZPX() uint8 {
	baseAddr := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (baseAddr + uint16(c.X)) & 0x00FF
	return 0
}

func (c *CPU) ZPY() uint8 {
	baseAddr := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (baseAddr + uint16(c.Y)) & 0x00FF
	return 0
}

func (c *CPU) REL() uint8 {
	addrRelOffset := uint16(c.read(c.PC))
	c.PC++
	if addrRelOffset&0x80 > 0 {
		addrRelOffset |= 0xFF00
	}
	c.addrRel = c.PC + addrRelOffset
	return 0
}

func (c *CPU) ABS() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	return 0
}

func (c *CPU) ABX() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	baseAddr := (hi << 8) | lo
	c.addrAbs = baseAddr + uint16(c.X)

	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}

func (c *CPU) ABY() uint8 {
	lo := uint16(c.read(c.PC))
	c.PC++
	hi := uint16(c.read(c.PC))
	c.PC++
	baseAddr := (hi << 8) | lo
	c.addrAbs = baseAddr + uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}

func (c *CPU) IND() uint8 {
	ptrLo := uint16(c.read(c.PC))
	c.PC++
	ptrHi := uint16(c.read(c.PC))
	c.PC++
	ptr := (ptrHi << 8) | ptrLo

	// Simulate the 6502 page boundary bug for indirect JMP:
	// If the low byte of the pointer is $FF, the high byte is fetched
	// from $xx00 instead of $xxFF + 1.
	if ptrLo == 0x00FF {
		// Read low byte from ptr, high byte from (ptr & 0xFF00)
		c.addrAbs = uint16(c.read(ptr)) | (uint16(c.read(ptr&0xFF00)) << 8)
	} else {
		// Normal case: read from ptr and ptr+1
		c.addrAbs = (uint16(c.read(ptr+1)) << 8) | uint16(c.read(ptr))
	}
	return 0
}

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

func (c *CPU) IZY() uint8 {
	zpAddr := uint16(c.read(c.PC))
	c.PC++

	baseAddrLo := uint16(c.read(zpAddr & 0x00FF))
	baseAddrHi := uint16(c.read((zpAddr + 1) & 0x00FF))
	baseAddr := (baseAddrHi << 8) | baseAddrLo

	c.addrAbs = baseAddr + uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (baseAddr & 0xFF00) {
		return 1
	}
	return 0
}

// --- Instruction Operations (Stubs) ---
// All operate functions remain the same method signatures:
// func (c *CPU) OpName() uint8 { ... }

// Helper function using reflect for comparison
func getFuncPtr(f any) uintptr {
	return reflect.ValueOf(f).Pointer()
}

// fetchDataIfNeeded - Fetch data if not in implied mode
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

// Helper for setting Zero and Negative flags based on a value.
func (c *CPU) setZNFlags(value uint8) {
	c.setFlag(Z, value == 0)
	c.setFlag(N, (value&0x80) > 0)
}

// ADC - Add with Carry
func (c *CPU) ADC() uint8 {
	c.fetchDataIfNeeded()
	var carry uint16 = 0
	if c.getFlag(C) {
		carry = 1
	}

	var temp uint16
	var result uint8

	if c.getFlag(D) {
		// --- Decimal Mode ---
		// N, V flags are officially undefined/unreliable in decimal mode.
		// Common emulation practice is to set them based on the intermediate *binary* addition result.
		// Z flag is set based on the final BCD result.
		// C flag is set based on whether the BCD result exceeds 99.

		// Calculate intermediate binary sum for N/V flags
		binarySum := uint16(c.A) + uint16(c.fetchedData) + carry
		// Set N based on bit 7 of binary sum
		c.setFlag(N, (binarySum&0x80) > 0)
		// Set V based on signed overflow of binary sum
		// Overflow = sign(A) == sign(M) && sign(A) != sign(Result)
		// Simplified: (~(A^M)) & (A^Result) & 0x80
		c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)

		// Perform BCD addition
		low := (c.A & 0x0F) + (c.fetchedData & 0x0F) + uint8(carry)
		if low > 9 {
			low += 6 // BCD adjustment for lower nibble
		}
		// Calculate high nibble sum including carry from low nibble BCD adjustment
		high := (c.A >> 4) + (c.fetchedData >> 4)
		if low > 0x0F { // Check if carry generated from lower nibble (original sum > 9 or adjusted sum >= 16)
			high++
		}

		if high > 9 {
			high += 6 // BCD adjustment for upper nibble
		}

		// Set C flag if the BCD result (represented by high nibble) exceeded 9 ($99)
		c.setFlag(C, high > 0x0F)

		// Combine nibbles for the final BCD result
		result = (high << 4) | (low & 0x0F)
		c.A = result

		// Set Z flag based on the final BCD result in A
		c.setFlag(Z, c.A == 0)

	} else {
		// --- Binary Mode ---
		temp = uint16(c.A) + uint16(c.fetchedData) + carry

		// Set C flag (unsigned overflow)
		c.setFlag(C, temp > 0xFF)

		result = uint8(temp & 0x00FF)

		// Set V flag (signed overflow)
		// Check if signs of operands are the same but sign of result is different
		// V = (~(A ^ M)) & (A ^ R) & 0x80 != 0
		c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^temp)&0x0080) > 0)

		// Update Accumulator
		c.A = result

		// Set Z and N flags based on the result
		c.setZNFlags(c.A)
	}

	// ADC potentially requires an extra cycle on page boundary crossing for certain indexed modes
	// The addressing mode function already returned 1 if a page cross occurred.
	return 1 // Return 1 indicating the operation itself takes at least 1 cycle, potentially more with page cross
}

func (c *CPU) AND() uint8 {
	c.fetchDataIfNeeded()
	c.A &= c.fetchedData
	c.setZNFlags(c.A)
	return 1
}

// ASL - Arithmetic Shift Left
func (c *CPU) ASL() uint8 {
	var temp uint16
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		temp = uint16(c.A) << 1
		c.setFlag(C, (temp&0xFF00) > 0)
		c.A = uint8(temp & 0x00FF)
		c.setZNFlags(c.A)
	} else {
		c.fetchDataIfNeeded() // Need data before modifying
		temp = uint16(c.fetchedData) << 1
		c.setFlag(C, (temp&0xFF00) > 0)
		result := uint8(temp & 0x00FF)
		c.write(c.addrAbs, result)
		c.setZNFlags(result)
	}
	return 0
}

// --- Branch Instructions ---
func (c *CPU) branchIf(condition bool) uint8 {
	cycles := uint8(0)
	if condition {
		cycles++
		if (c.addrRel & 0xFF00) != (c.PC & 0xFF00) {
			cycles++
		}
		c.PC = c.addrRel
	}
	return cycles
}

func (c *CPU) BCC() uint8 { return c.branchIf(!c.getFlag(C)) }
func (c *CPU) BCS() uint8 { return c.branchIf(c.getFlag(C)) }
func (c *CPU) BEQ() uint8 { return c.branchIf(c.getFlag(Z)) }
func (c *CPU) BMI() uint8 { return c.branchIf(c.getFlag(N)) }
func (c *CPU) BNE() uint8 { return c.branchIf(!c.getFlag(Z)) }
func (c *CPU) BPL() uint8 { return c.branchIf(!c.getFlag(N)) }
func (c *CPU) BVC() uint8 { return c.branchIf(!c.getFlag(V)) }
func (c *CPU) BVS() uint8 { return c.branchIf(c.getFlag(V)) }

func (c *CPU) BIT() uint8 {
	c.fetchDataIfNeeded()
	temp := c.A & c.fetchedData
	c.setFlag(Z, temp == 0)
	c.setFlag(N, (c.fetchedData&(1<<7)) > 0)
	c.setFlag(V, (c.fetchedData&(1<<6)) > 0)
	return 0
}

func (c *CPU) BRK() uint8 {
	// Note: PC is already incremented once by Clock() to point after the $00 opcode.
	// The 6502 pushes PC+2 relative to the opcode address.
	// The extra PC++ here achieves that. If removed, PC+1 would be pushed.
	c.PC++ // Make PC point to PC+2 relative to opcode fetch address

	// Push PC onto stack (now PC+2)
	c.push16(c.PC)

	// Push status register onto stack
	// Note: B flag is SET, U flag is SET in the pushed copy
	c.setFlag(B, true)
	c.setFlag(U, true)
	// --- Push P BEFORE setting I flag ---
	originalP := c.P                 // Capture P before setting I
	c.push(uint8(originalP | B | U)) // Push original flags + B + U
	// Push P with B and U set
	// c.push(uint8(c.P)) // Original incorrect line

	// Set Interrupt Disable flag AFTER push
	c.setFlag(I, true)

	// Clear B flag in processor's actual register (it was only set for the push)
	c.setFlag(B, false)

	// Load interrupt vector ($FFFE/F)
	lo := uint16(c.read(0xFFFE))
	hi := uint16(c.read(0xFFFF))
	c.PC = (hi << 8) | lo

	// BRK takes 7 cycles total (base cycles handled by lookup table)
	// This function returns *extra* cycles, which should be 0 here.
	return 0
}

func (c *CPU) CLC() uint8 { c.setFlag(C, false); return 0 }
func (c *CPU) CLD() uint8 { c.setFlag(D, false); return 0 }
func (c *CPU) CLI() uint8 { c.setFlag(I, false); return 0 }
func (c *CPU) CLV() uint8 { c.setFlag(V, false); return 0 }

func (c *CPU) CMP() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.A) - uint16(c.fetchedData)
	c.setFlag(C, c.A >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 1
}

func (c *CPU) CPX() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.X) - uint16(c.fetchedData)
	c.setFlag(C, c.X >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
}

func (c *CPU) CPY() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.Y) - uint16(c.fetchedData)
	c.setFlag(C, c.Y >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
}

func (c *CPU) DEC() uint8 {
	c.fetchDataIfNeeded()
	temp := c.fetchedData - 1
	c.write(c.addrAbs, temp)
	c.setZNFlags(temp)
	return 0
}

func (c *CPU) DEX() uint8 { c.X--; c.setZNFlags(c.X); return 0 }
func (c *CPU) DEY() uint8 { c.Y--; c.setZNFlags(c.Y); return 0 }

func (c *CPU) EOR() uint8 {
	c.fetchDataIfNeeded()
	c.A ^= c.fetchedData
	c.setZNFlags(c.A)
	return 1
}

func (c *CPU) INC() uint8 {
	c.fetchDataIfNeeded()
	temp := c.fetchedData + 1
	c.write(c.addrAbs, temp)
	c.setZNFlags(temp)
	return 0
}

func (c *CPU) INX() uint8 { c.X++; c.setZNFlags(c.X); return 0 }
func (c *CPU) INY() uint8 { c.Y++; c.setZNFlags(c.Y); return 0 }

func (c *CPU) JMP() uint8 {
	c.PC = c.addrAbs
	return 0
}

func (c *CPU) JSR() uint8 {
	c.PC--
	c.push16(c.PC)
	c.PC = c.addrAbs
	return 0
}

func (c *CPU) LDA() uint8 {
	c.fetchDataIfNeeded()
	c.A = c.fetchedData
	c.setZNFlags(c.A)
	return 1
}

func (c *CPU) LDX() uint8 {
	c.fetchDataIfNeeded()
	c.X = c.fetchedData
	c.setZNFlags(c.X)
	return 1
}

func (c *CPU) LDY() uint8 {
	c.fetchDataIfNeeded()
	c.Y = c.fetchedData
	c.setZNFlags(c.Y)
	return 1
}

// LSR - Logical Shift Right
func (c *CPU) LSR() uint8 {
	var temp uint8
	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		c.setFlag(C, (c.A&0x01) > 0)
		c.A >>= 1
		c.setZNFlags(c.A)
	} else {
		c.fetchDataIfNeeded() // Need data before modifying
		c.setFlag(C, (c.fetchedData&0x01) > 0)
		temp = c.fetchedData >> 1
		c.write(c.addrAbs, temp)
		c.setZNFlags(temp)
	}
	return 0
}

// NOP - No Operation
func (c *CPU) NOP() uint8 {
	// Some unofficial NOPs using indexed addressing might technically read
	// from memory, potentially causing a page cross and taking an extra cycle.
	// We handle this by having the addressing mode function return 1 if a cross occurs,
	// and returning 1 from here allows that cycle to be added.
	// Official NOP (EA) and simple unofficial NOPs (IMP, IMM, ZP0) don't have this.
	switch c.opcode {
	// NOPs with ABX addressing mode that can cross pages:
	case 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC:
		// These modes *do* calculate an address, even if not used.
		// Fetch might not be strictly necessary for NOP, but the address calc is.
		// The AddrMode function already returns 1 on page cross.
		return 1 // Signal that page cross cycle might apply
	// NOPs with IZY addressing mode that can cross pages:
	// (None documented, but if added, would need similar handling)
	default:
		return 0 // No potential for extra cycle from page cross
	}
}

func (c *CPU) ORA() uint8 {
	c.fetchDataIfNeeded()
	c.A |= c.fetchedData
	c.setZNFlags(c.A)
	return 1
}

func (c *CPU) PHA() uint8 { c.push(c.A); return 0 }
func (c *CPU) PHP() uint8 {
	c.push(uint8(c.P | B | U))
	return 0
}
func (c *CPU) PLA() uint8 { c.A = c.pop(); c.setZNFlags(c.A); return 0 }
func (c *CPU) PLP() uint8 {
	poppedP := Flags(c.pop())
	c.P = (poppedP & ^(B | U)) | (c.P & (B | U))
	c.setFlag(U, true)
	return 0
}

// ROL - Rotate Left
func (c *CPU) ROL() uint8 {
	var temp uint16
	var carryBit uint16 = 0
	if c.getFlag(C) {
		carryBit = 1
	}

	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		temp = (uint16(c.A) << 1) | carryBit
		c.setFlag(C, (temp&0xFF00) > 0)
		c.A = uint8(temp & 0x00FF)
		c.setZNFlags(c.A)
	} else {
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
func (c *CPU) ROR() uint8 {
	var temp uint8
	var carryBit uint8 = 0
	if c.getFlag(C) {
		carryBit = 0x80
	}

	// Use enum comparison instead of reflection
	if c.currentInstruction.AddrModeType == AddrModeIMP {
		newCarry := (c.A & 0x01) > 0
		temp = (c.A >> 1) | carryBit
		c.setFlag(C, newCarry)
		c.A = temp
		c.setZNFlags(c.A)
	} else {
		c.fetchDataIfNeeded() // Need data before modifying
		newCarry := (c.fetchedData & 0x01) > 0
		temp = (c.fetchedData >> 1) | carryBit
		c.write(c.addrAbs, temp)
		c.setFlag(C, newCarry)
		c.setZNFlags(temp)
	}
	return 0
}

func (c *CPU) RTI() uint8 {
	c.P = Flags(c.pop())
	c.P &^= B
	c.P |= U
	c.PC = c.pop16()
	return 0
}

func (c *CPU) RTS() uint8 {
	c.PC = c.pop16()
	c.PC++
	return 0
}

// SBC - Subtract with Carry (Borrow)
func (c *CPU) SBC() uint8 {
	c.fetchDataIfNeeded()
	// SBC is effectively A - M - (1 - C)
	// which is equivalent to ADC with A + (~M) + C
	value := uint16(c.fetchedData) ^ 0x00FF // ~M (one's complement)
	var carry uint16 = 0                    // Carry In for the A + (~M) + C operation
	if c.getFlag(C) {
		carry = 1
	}

	var temp uint16
	var result uint8

	if c.getFlag(D) {
		// --- Decimal Mode ---
		// Similar to ADC, N/V flags based on intermediate binary result, C/Z on final BCD result.

		// Calculate intermediate binary result A + (~M) + C for N/V flags
		binarySum := uint16(c.A) + value + carry
		// Set N based on bit 7 of binary sum
		c.setFlag(N, (binarySum&0x80) > 0)
		// Set V based on signed overflow of binary sum (A + ~M + C)
		// Overflow = sign(A) == sign(~M) && sign(A) != sign(Result)
		// Simplified: (~(A ^ ~M)) & (A ^ Result) & 0x80
		c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)

		// Perform BCD subtraction (A - M - borrow)
		borrow_in := 1 - uint8(carry) // Convert C flag to borrow (0 or 1)

		// Use int16 for intermediate subtraction to handle potential negative results easily
		sub_res := int16(c.A) - int16(c.fetchedData) - int16(borrow_in)

		low := (int16(c.A&0x0F) - int16(c.fetchedData&0x0F) - int16(borrow_in))
		borrow_low := uint8(0)
		if low < 0 {
			low -= 6 // BCD adjust low nibble
			borrow_low = 1
		}

		high := (int16(c.A>>4) - int16(c.fetchedData>>4) - int16(borrow_low))
		if high < 0 {
			high -= 6 // BCD adjust high nibble
		}

		// Combine nibbles
		result = (uint8(high&0x0F) << 4) | uint8(low&0x0F)
		c.A = result

		// Set C flag: Set if result >= 0 (no borrow needed overall)
		c.setFlag(C, sub_res >= 0)
		// Set Z flag based on the final BCD result
		c.setFlag(Z, c.A == 0)

	} else {
		// --- Binary Mode ---
		temp = uint16(c.A) + value + carry

		// Set C flag: Based on the carry out of the A + ~M + C addition.
		// This is equivalent to "no borrow needed" for A - M - (1-C).
		c.setFlag(C, temp > 0xFF)

		result = uint8(temp & 0x00FF)

		// Set V flag (signed overflow)
		// V = (A ^ M) & (A ^ R) & 0x80 != 0  (for A - M - B)
		// Equivalently use A + ~M + C: V = (~(A ^ ~M)) & (A ^ R) & 0x80
		c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^temp)&0x0080) > 0)

		// Update Accumulator
		c.A = result

		// Set Z and N flags based on the result
		c.setZNFlags(c.A)
	}

	// SBC potentially requires an extra cycle on page boundary crossing
	return 1
}

func (c *CPU) SEC() uint8 { c.setFlag(C, true); return 0 }
func (c *CPU) SED() uint8 { c.setFlag(D, true); return 0 }
func (c *CPU) SEI() uint8 { c.setFlag(I, true); return 0 }

func (c *CPU) STA() uint8 {
	c.write(c.addrAbs, c.A)
	return 0
}

func (c *CPU) STX() uint8 { c.write(c.addrAbs, c.X); return 0 }
func (c *CPU) STY() uint8 { c.write(c.addrAbs, c.Y); return 0 }

func (c *CPU) TAX() uint8 { c.X = c.A; c.setZNFlags(c.X); return 0 }
func (c *CPU) TAY() uint8 { c.Y = c.A; c.setZNFlags(c.Y); return 0 }
func (c *CPU) TSX() uint8 { c.X = c.SP; c.setZNFlags(c.X); return 0 }
func (c *CPU) TXA() uint8 { c.A = c.X; c.setZNFlags(c.A); return 0 }
func (c *CPU) TXS() uint8 { c.SP = c.X; return 0 }
func (c *CPU) TYA() uint8 { c.A = c.Y; c.setZNFlags(c.A); return 0 }

func (c *CPU) XXX() uint8 {
	log.Printf("ERROR: Illegal opcode $%02X encountered at $%04X", c.opcode, c.PC-1)
	return 1 // Return 1 to prevent potential infinite loops if Clock checks >=0
}

// --- Lookup Table Initialization ---
// *Use method expressions for assignment*
func (c *CPU) buildLookupTable() {
	// Helper to get method expression pointer
	IMP := (*CPU).IMP
	IMM := (*CPU).IMM
	ZP0 := (*CPU).ZP0
	ZPX := (*CPU).ZPX
	ZPY := (*CPU).ZPY
	REL := (*CPU).REL
	ABS := (*CPU).ABS
	ABX := (*CPU).ABX
	ABY := (*CPU).ABY
	IND := (*CPU).IND
	IZX := (*CPU).IZX
	IZY := (*CPU).IZY

	ADC := (*CPU).ADC
	AND := (*CPU).AND
	ASL := (*CPU).ASL
	BCC := (*CPU).BCC
	BCS := (*CPU).BCS
	BEQ := (*CPU).BEQ
	BIT := (*CPU).BIT
	BMI := (*CPU).BMI
	BNE := (*CPU).BNE
	BPL := (*CPU).BPL
	BRK := (*CPU).BRK
	BVC := (*CPU).BVC
	BVS := (*CPU).BVS
	CLC := (*CPU).CLC
	CLD := (*CPU).CLD
	CLI := (*CPU).CLI
	CLV := (*CPU).CLV
	CMP := (*CPU).CMP
	CPX := (*CPU).CPX
	CPY := (*CPU).CPY
	DEC := (*CPU).DEC
	DEX := (*CPU).DEX
	DEY := (*CPU).DEY
	EOR := (*CPU).EOR
	INC := (*CPU).INC
	INX := (*CPU).INX
	INY := (*CPU).INY
	JMP := (*CPU).JMP
	JSR := (*CPU).JSR
	LDA := (*CPU).LDA
	LDX := (*CPU).LDX
	LDY := (*CPU).LDY
	LSR := (*CPU).LSR
	NOP := (*CPU).NOP
	ORA := (*CPU).ORA
	PHA := (*CPU).PHA
	PHP := (*CPU).PHP
	PLA := (*CPU).PLA
	PLP := (*CPU).PLP
	ROL := (*CPU).ROL
	ROR := (*CPU).ROR
	RTI := (*CPU).RTI
	RTS := (*CPU).RTS
	SBC := (*CPU).SBC
	SEC := (*CPU).SEC
	SED := (*CPU).SED
	SEI := (*CPU).SEI
	STA := (*CPU).STA
	STX := (*CPU).STX
	STY := (*CPU).STY
	TAX := (*CPU).TAX
	TAY := (*CPU).TAY
	TSX := (*CPU).TSX
	TXA := (*CPU).TXA
	TXS := (*CPU).TXS
	TYA := (*CPU).TYA
	XXX := (*CPU).XXX

	// Fill with illegal opcodes first
	for i := range c.lookup {
		c.lookup[i] = Instruction{Name: "XXX", Operate: XXX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Illegal: true}
	}

	// --- Official Opcodes --- Using the helper variables above
	c.lookup[0xA9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xA5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xB5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0xAD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0xBD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0xB9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0xA1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0xB1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0xA2] = Instruction{Name: "LDX", Operate: LDX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xA6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xB6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2}
	c.lookup[0xAE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0xBE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}

	c.lookup[0xA0] = Instruction{Name: "LDY", Operate: LDY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xA4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xB4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0xAC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0xBC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}

	c.lookup[0x85] = Instruction{Name: "STA", Operate: STA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x95] = Instruction{Name: "STA", Operate: STA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x8D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0x9D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 5, Length: 3}
	c.lookup[0x99] = Instruction{Name: "STA", Operate: STA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 5, Length: 3}
	c.lookup[0x81] = Instruction{Name: "STA", Operate: STA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0x91] = Instruction{Name: "STA", Operate: STA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 6, Length: 2}

	c.lookup[0x86] = Instruction{Name: "STX", Operate: STX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x96] = Instruction{Name: "STX", Operate: STX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2}
	c.lookup[0x8E] = Instruction{Name: "STX", Operate: STX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}

	c.lookup[0x84] = Instruction{Name: "STY", Operate: STY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x94] = Instruction{Name: "STY", Operate: STY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x8C] = Instruction{Name: "STY", Operate: STY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}

	c.lookup[0xAA] = Instruction{Name: "TAX", Operate: TAX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xA8] = Instruction{Name: "TAY", Operate: TAY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xBA] = Instruction{Name: "TSX", Operate: TSX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x8A] = Instruction{Name: "TXA", Operate: TXA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x9A] = Instruction{Name: "TXS", Operate: TXS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x98] = Instruction{Name: "TYA", Operate: TYA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}

	c.lookup[0x48] = Instruction{Name: "PHA", Operate: PHA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1}
	c.lookup[0x08] = Instruction{Name: "PHP", Operate: PHP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1}
	c.lookup[0x68] = Instruction{Name: "PLA", Operate: PLA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1}
	c.lookup[0x28] = Instruction{Name: "PLP", Operate: PLP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1}

	c.lookup[0x69] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0x65] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x75] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x6D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0x7D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0x79] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0x61] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0x71] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0xE9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xE5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xF5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0xED] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0xFD] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0xF9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0xE1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0xF1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0xE6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0xF6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0xEE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0xFE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}
	c.lookup[0xE8] = Instruction{Name: "INX", Operate: INX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xC8] = Instruction{Name: "INY", Operate: INY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}

	c.lookup[0xC6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0xD6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0xCE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0xDE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}
	c.lookup[0xCA] = Instruction{Name: "DEX", Operate: DEX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x88] = Instruction{Name: "DEY", Operate: DEY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}

	c.lookup[0x0A] = Instruction{Name: "ASL", Operate: ASL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x06] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0x16] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0x0E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0x1E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}

	c.lookup[0x4A] = Instruction{Name: "LSR", Operate: LSR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x46] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0x56] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0x4E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0x5E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}

	c.lookup[0x2A] = Instruction{Name: "ROL", Operate: ROL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x26] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0x36] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0x2E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0x3E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}

	c.lookup[0x6A] = Instruction{Name: "ROR", Operate: ROR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x66] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2}
	c.lookup[0x76] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2}
	c.lookup[0x6E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0x7E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3}

	c.lookup[0x29] = Instruction{Name: "AND", Operate: AND, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0x25] = Instruction{Name: "AND", Operate: AND, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x35] = Instruction{Name: "AND", Operate: AND, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x2D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0x3D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0x39] = Instruction{Name: "AND", Operate: AND, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0x21] = Instruction{Name: "AND", Operate: AND, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0x31] = Instruction{Name: "AND", Operate: AND, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0x49] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0x45] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x55] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x4D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0x5D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0x59] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0x41] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0x51] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0x09] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0x05] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x15] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0x0D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0x1D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0x19] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0x01] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0x11] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0x24] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0x2C] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}

	c.lookup[0xC9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xC5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xD5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2}
	c.lookup[0xCD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}
	c.lookup[0xDD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3}
	c.lookup[0xD9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3}
	c.lookup[0xC1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2}
	c.lookup[0xD1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2}

	c.lookup[0xE0] = Instruction{Name: "CPX", Operate: CPX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xE4] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xEC] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}

	c.lookup[0xC0] = Instruction{Name: "CPY", Operate: CPY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2}
	c.lookup[0xC4] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2}
	c.lookup[0xCC] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3}

	c.lookup[0x90] = Instruction{Name: "BCC", Operate: BCC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0xB0] = Instruction{Name: "BCS", Operate: BCS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0xF0] = Instruction{Name: "BEQ", Operate: BEQ, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0x30] = Instruction{Name: "BMI", Operate: BMI, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0xD0] = Instruction{Name: "BNE", Operate: BNE, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0x10] = Instruction{Name: "BPL", Operate: BPL, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0x50] = Instruction{Name: "BVC", Operate: BVC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}
	c.lookup[0x70] = Instruction{Name: "BVS", Operate: BVS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2}

	c.lookup[0x4C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 3, Length: 3}
	c.lookup[0x6C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: IND, AddrModeType: AddrModeIND, Cycles: 5, Length: 3}
	c.lookup[0x20] = Instruction{Name: "JSR", Operate: JSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3}
	c.lookup[0x60] = Instruction{Name: "RTS", Operate: RTS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1}

	c.lookup[0x00] = Instruction{Name: "BRK", Operate: BRK, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 7, Length: 1}
	c.lookup[0x40] = Instruction{Name: "RTI", Operate: RTI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1}

	c.lookup[0x18] = Instruction{Name: "CLC", Operate: CLC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xD8] = Instruction{Name: "CLD", Operate: CLD, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x58] = Instruction{Name: "CLI", Operate: CLI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xB8] = Instruction{Name: "CLV", Operate: CLV, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x38] = Instruction{Name: "SEC", Operate: SEC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0xF8] = Instruction{Name: "SED", Operate: SED, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}
	c.lookup[0x78] = Instruction{Name: "SEI", Operate: SEI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}

	c.lookup[0xEA] = Instruction{Name: "NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1}

	// Illegal NOPs
	// https://www.masswerk.at/6502/6502_instruction_set.html#NOPs
	c.lookup[0x1A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}
	c.lookup[0x3A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}
	c.lookup[0x5A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}
	c.lookup[0x7A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}
	c.lookup[0xDA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}
	c.lookup[0xFA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true}

	c.lookup[0x80] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true}
	c.lookup[0x82] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true}
	c.lookup[0x89] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true}
	c.lookup[0xC2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true}
	c.lookup[0xE2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true}

	c.lookup[0x04] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true}
	c.lookup[0x44] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true}
	c.lookup[0x64] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true}

	c.lookup[0x14] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}
	c.lookup[0x34] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}
	c.lookup[0x54] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}
	c.lookup[0x74] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}
	c.lookup[0xD4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}
	c.lookup[0xF4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true}

	c.lookup[0x0C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, Illegal: true}

	c.lookup[0x1C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed
	c.lookup[0x3C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed
	c.lookup[0x5C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed
	c.lookup[0x7C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed
	c.lookup[0xDC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed
	c.lookup[0xFC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true} // +1 cycle if page crossed

}

// --- Core Execution ---

// Reset
func (c *CPU) Reset() {
	lo := uint16(c.read(0xFFFC))
	hi := uint16(c.read(0xFFFD))
	c.PC = (hi << 8) | lo
	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = U | I
	c.addrAbs = 0x0000
	c.addrRel = 0x0000
	c.fetchedData = 0x00
	c.Cycles = 8
}

// InterruptRequest
func (c *CPU) InterruptRequest() {
	if !c.getFlag(I) {
		// Push PC onto stack (current PC)
		c.push16(c.PC)

		// Push status register onto stack
		// Note: B flag is CLEARed, U flag is set in the pushed copy
		c.setFlag(B, false)
		c.setFlag(U, true)
		// --- Push P BEFORE setting I flag ---
		c.push(uint8(c.P))

		// Set Interrupt Disable flag AFTER push
		c.setFlag(I, true)

		// Read interrupt vector ($FFFE/F)
		lo := uint16(c.read(0xFFFE))
		hi := uint16(c.read(0xFFFF))
		c.PC = (hi << 8) | lo

		// Interrupts take time
		c.Cycles = 7
	}
}

// NonMaskableInterrupt
func (c *CPU) NonMaskableInterrupt() {
	// Push PC onto stack (current PC)
	c.push16(c.PC)

	// Push status register onto stack
	// Note: B flag is CLEARed, U flag is set in the pushed copy
	c.setFlag(B, false)
	c.setFlag(U, true)
	// --- Push P BEFORE setting I flag ---
	c.push(uint8(c.P))

	// Set Interrupt Disable flag AFTER push
	c.setFlag(I, true)

	// Read NMI vector ($FFFA/B)
	lo := uint16(c.read(0xFFFA))
	hi := uint16(c.read(0xFFFB))
	c.PC = (hi << 8) | lo

	// NMIs take time
	c.Cycles = 8
}

// Clock
func (c *CPU) Clock() {
	if c.Cycles == 0 {
		c.opcode = c.read(c.PC)
		c.PC++

		c.setFlag(U, true) // Ensure U is always set before execution

		c.currentInstruction = &c.lookup[c.opcode]

		baseCycles := c.currentInstruction.Cycles
		// Call AddrMode and Operate methods via the function pointers in the struct
		// These methods implicitly receive 'c' as their receiver.
		addrModeCycles := c.currentInstruction.AddrMode(c)
		opCycles := c.currentInstruction.Operate(c)

		c.Cycles = baseCycles + addrModeCycles + opCycles

		// Ensure U is set after execution as well (might be cleared by PLP?)
		c.setFlag(U, true)

	}

	c.Cycles--
	c.totalCycles++
}

// --- Debug Helpers ---

// GetCurrentInstruction returns a pointer to the current instruction definition.
// Returns nil if no instruction has been fetched yet.
func (c *CPU) GetCurrentInstruction() *Instruction {
	return c.currentInstruction
}

// TotalCycles returns the total number of cycles executed by the CPU since its
// creation. This is useful for profiling and debugging purposes.
func (c *CPU) TotalCycles() uint64 {
	return c.totalCycles
}

// Opcode returns the last fetched opcode. Useful for debugging/halt conditions.
func (c *CPU) Opcode() uint8 {
	return c.opcode
}

// GetState
func (c *CPU) GetState() string {
	flagsStr := ""
	if c.getFlag(N) {
		flagsStr += "N"
	} else {
		flagsStr += "."
	}
	if c.getFlag(V) {
		flagsStr += "V"
	} else {
		flagsStr += "."
	}
	if c.getFlag(U) {
		flagsStr += "U"
	} else {
		flagsStr += "."
	}
	if c.getFlag(B) {
		flagsStr += "B"
	} else {
		flagsStr += "."
	}
	if c.getFlag(D) {
		flagsStr += "D"
	} else {
		flagsStr += "."
	}
	if c.getFlag(I) {
		flagsStr += "I"
	} else {
		flagsStr += "."
	}
	if c.getFlag(Z) {
		flagsStr += "Z"
	} else {
		flagsStr += "."
	}
	if c.getFlag(C) {
		flagsStr += "C"
	} else {
		flagsStr += "."
	}

	instrName := "???"
	if c.currentInstruction != nil {
		instrName = c.currentInstruction.Name
	}

	return fmt.Sprintf("PC:%04X A:%02X X:%02X Y:%02X P:%02X[%s] SP:%02X CYC:%d (%s $%02X)",
		c.PC, c.A, c.X, c.Y, uint8(c.P), flagsStr, c.SP, c.totalCycles, instrName, c.opcode)
}

// FormatFlags - Helper to create the NVUBDIZC string
func FormatFlags(p Flags) string {
	flags := []struct {
		flag Flags
		char byte
	}{
		{N, 'N'}, {V, 'V'}, {U, 'U'}, {B, 'B'},
		{D, 'D'}, {I, 'I'}, {Z, 'Z'}, {C, 'C'},
	}
	s := make([]byte, 8)
	for i, f := range flags {
		if (p & f.flag) != 0 {
			s[i] = f.char
		} else {
			s[i] = '.'
		}
	}
	return string(s)
}

// Disassemble - *Use reflect for AdrMode comparison*
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string {
	disassembly := make(map[uint16]string)
	addr := startAddr

	// Get function pointers for comparison only once
	impPtr := getFuncPtr((*CPU).IMP)
	immPtr := getFuncPtr((*CPU).IMM)
	zp0Ptr := getFuncPtr((*CPU).ZP0)
	zpxPtr := getFuncPtr((*CPU).ZPX)
	zpyPtr := getFuncPtr((*CPU).ZPY)
	relPtr := getFuncPtr((*CPU).REL)
	absPtr := getFuncPtr((*CPU).ABS)
	abxPtr := getFuncPtr((*CPU).ABX)
	abyPtr := getFuncPtr((*CPU).ABY)
	indPtr := getFuncPtr((*CPU).IND)
	izxPtr := getFuncPtr((*CPU).IZX)
	izyPtr := getFuncPtr((*CPU).IZY)

	for addr <= endAddr && addr >= startAddr { // Check for wrap around too
		lineAddr := addr
		opcode := c.read(addr)
		addr++ // Consume opcode byte
		instr := c.lookup[opcode]
		operandStr := ""
		addrModePtr := getFuncPtr(instr.AddrMode) // Get ptr of the mode for this instruction

		switch addrModePtr { // Compare pointers
		case impPtr:
			// No operand bytes
		case immPtr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("#$%02X", val)
		case zp0Ptr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X", val)
		case zpxPtr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,X", val)
		case zpyPtr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,Y", val)
		case relPtr:
			val := c.read(addr)
			addr++            // Consume operand byte
			relTarget := addr // Relative jump target is relative to the *next* instruction
			if val&0x80 > 0 {
				relTarget += uint16(int16(val)) // Handle negative offset
			} else {
				relTarget += uint16(val) // Positive offset
			}
			operandStr = fmt.Sprintf("$%04X", relTarget)
		case absPtr:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X", (hi<<8)|lo)
		case abxPtr:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,X", (hi<<8)|lo)
		case abyPtr:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,Y", (hi<<8)|lo)
		case indPtr:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("($%04X)", (hi<<8)|lo)
		case izxPtr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("($%02X,X)", val)
		case izyPtr:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("($%02X),Y", val)
		default:
			operandStr = "???"
		}

		// Add safety checks in case addr went beyond bounds during operand read
		if addr > endAddr+4 {
			disassembly[lineAddr] = fmt.Sprintf("%s ??? ; Address bounds exceeded", instr.Name)
			break
		}

		disassembly[lineAddr] = fmt.Sprintf("%s %s", instr.Name, operandStr)

		// Safety break for very large ranges or infinite loops from bad data
		if addr < lineAddr && lineAddr != 0 {
			break
		} // Address wrapped around negatively

	}
	return disassembly
}

// LookupTable exposes the instruction lookup table (use with caution).
// Primarily intended for tools like disassemblers or UI displays.
func (c *CPU) LookupTable() [256]Instruction {
	return c.lookup
}
