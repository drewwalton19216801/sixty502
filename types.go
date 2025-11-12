// Package cpu6502 provides type definitions for the 6502 CPU emulator.
//
// This file contains all core type definitions, constants, and enums used
// throughout the emulator. Having types in a dedicated file provides:
//   - A single source of truth for all data structures
//   - Easy reference for developers
//   - Better documentation organization
//   - Reduced clutter in the main CPU implementation
package cpu6502

import "fmt"

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

// CPUVariant represents different 6502 processor variants
type CPUVariant int

const (
	// VariantNMOS6502 is the original NMOS 6502 (1975)
	// Used in: Apple II, Commodore 64, Atari 2600/800, BBC Micro
	// Features: All documented bugs, decimal mode supported
	// Note: This represents Rev B and later which have working ROR
	VariantNMOS6502 CPUVariant = iota

	// VariantNMOS6502RevA is the original Rev A NMOS 6502
	// This early revision had a hardware bug where ROR was not implemented
	// and performed a modified ROL operation instead
	// Features: ROR quirk, all other NMOS bugs, decimal mode supported
	VariantNMOS6502RevA

	// VariantCMOS65C02 is the CMOS 65C02 (1982)
	// Used in: Apple IIc, Apple IIe (enhanced), later systems
	// Features: Bug fixes, additional instructions, lower power
	VariantCMOS65C02

	// VariantRicoh2A03 is the NES/Famicom CPU (1983)
	// Used in: Nintendo Entertainment System, Famicom
	// Features: No decimal mode, integrated APU, different timing
	VariantRicoh2A03

	// VariantRicoh2A07 is the PAL NES CPU
	// Same as 2A03 but with PAL timing
	VariantRicoh2A07
)

// String returns the variant name
func (v CPUVariant) String() string {
	names := []string{
		"NMOS 6502",
		"NMOS 6502 Rev A",
		"CMOS 65C02",
		"Ricoh 2A03 (NTSC)",
		"Ricoh 2A07 (PAL)",
	}
	if int(v) < len(names) {
		return names[v]
	}
	return "Unknown"
}

// SupportsDecimalMode returns true if the variant supports decimal mode
func (v CPUVariant) SupportsDecimalMode() bool {
	switch v {
	case VariantRicoh2A03, VariantRicoh2A07:
		return false
	default:
		return true
	}
}

// HasIndirectJMPBug returns true if the variant has the indirect JMP page boundary bug
func (v CPUVariant) HasIndirectJMPBug() bool {
	switch v {
	case VariantNMOS6502, VariantNMOS6502RevA, VariantRicoh2A03, VariantRicoh2A07:
		return true
	case VariantCMOS65C02:
		return false
	default:
		return true
	}
}

// HasRORQuirk returns true if the variant has the ROR hardware bug.
// Only the original Rev A NMOS 6502 had this quirk where ROR didn't have
// proper circuitry and performed a modified ROL operation instead.
// This was fixed in Rev B and was never present in Ricoh or CMOS variants.
func (v CPUVariant) HasRORQuirk() bool {
	switch v {
	case VariantNMOS6502RevA:
		return true
	case VariantNMOS6502, VariantCMOS65C02, VariantRicoh2A03, VariantRicoh2A07:
		return false
	default:
		return false // Conservative: assume no quirk for unknown variants
	}
}

// Flags represents the processor status register.
//
// The 6502 has 8 status flags that indicate the result of operations:
//   - N (Negative): Set if result is negative (bit 7 = 1)
//   - V (Overflow): Set if signed overflow occurred
//   - U (Unused): Always set to 1
//   - B (Break): Set when BRK instruction executed
//   - D (Decimal): Enables BCD arithmetic mode
//   - I (Interrupt Disable): When set, IRQ interrupts are ignored
//   - Z (Zero): Set if result is zero
//   - C (Carry): Set if unsigned overflow/borrow occurred
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

// Instruction represents a single 6502 instruction with its associated metadata.
//
// Each instruction consists of:
//   - Name: The mnemonic (e.g., "LDA", "STA")
//   - Operate: Function that executes the instruction's logic
//   - AddrMode: Function that calculates the effective address
//   - AddrModeType: Enum identifying the addressing mode
//   - Cycles: Base number of cycles the instruction takes
//   - Length: Size of the instruction in bytes (including opcode)
//   - Illegal: Whether this is an unofficial/illegal opcode
//   - PageCrossPenalty: Whether to add +1 cycle on page boundary cross
type Instruction struct {
	Name             string           // Mnemonic (e.g., "LDA")
	Operate          func(*CPU) uint8 // Function to execute the instruction's logic (accepts *CPU)
	AddrMode         func(*CPU) uint8 // Function to calculate the address and fetch data (accepts *CPU)
	AddrModeType     AddrModeType     // Type of addressing mode
	Cycles           uint8            // Base cycles for this instruction/mode
	Length           uint8            // Length of the instruction in bytes
	Illegal          bool             // Whether this is an official or unofficial/illegal opcode
	PageCrossPenalty bool             // Whether to add +1 cycle on page boundary cross
}

// State represents a snapshot of CPU state at a point in time.
//
// This structure is useful for:
//   - Debugging and inspection
//   - Save states in emulators
//   - Testing and verification
//   - Logging execution traces
type State struct {
	A               uint8  // Accumulator
	X               uint8  // X Index Register
	Y               uint8  // Y Index Register
	SP              uint8  // Stack Pointer
	PC              uint16 // Program Counter
	P               Flags  // Processor Status Register
	Cycles          uint8  // Cycles remaining for current instruction
	TotalCycles     uint64 // Total cycles executed since creation
	Opcode          uint8  // Current opcode being executed
	Instruction     string // Current instruction mnemonic
	InInterrupt     bool   // Whether currently handling an interrupt
	InterruptVector uint16 // Vector being used for current interrupt
}

// String returns a human-readable representation of the state
func (s State) String() string {
	return fmt.Sprintf(
		"PC:%04X A:%02X X:%02X Y:%02X P:%02X[%s] SP:%02X CYC:%d (%s $%02X)",
		s.PC, s.A, s.X, s.Y, uint8(s.P), FormatFlags(s.P),
		s.SP, s.TotalCycles, s.Instruction, s.Opcode,
	)
}

// FormatFlags returns a human-readable string representation of processor flags.
//
// The format is "NVUBDIZC" where each letter represents a flag:
//   - N: Negative
//   - V: Overflow
//   - U: Unused (always 1)
//   - B: Break
//   - D: Decimal
//   - I: Interrupt Disable
//   - Z: Zero
//   - C: Carry
//
// Set flags are shown as their letter, cleared flags as '.'
//
// Example: "N.U.D.Z." means N, U, D, and Z flags are set
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
