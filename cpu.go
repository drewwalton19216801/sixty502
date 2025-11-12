// Package cpu6502 provides a cycle-accurate emulator for the MOS Technology 6502
// microprocessor and its variants.
//
// The 6502 is an 8-bit microprocessor that was widely used in home computers
// and game consoles during the 1970s and 1980s, including the Apple II,
// Commodore 64, and Atari 2600.
//
// This implementation supports:
//   - All 151 official 6502 instructions
//   - Multiple CPU variants (NMOS 6502, CMOS 65C02, Ricoh 2A03)
//   - Cycle-accurate timing including page boundary crossing
//   - Decimal mode (BCD) arithmetic
//   - Interrupt handling (IRQ, NMI, BRK)
//   - Many unofficial/illegal opcodes
//
// Basic usage:
//
//	bus := &SimpleBus{}
//	cpu := cpu6502.NewCPU(bus)
//	cpu.Reset()
//	for cpu.RemainingCycles() > 0 {
//	    if err := cpu.Clock(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
package cpu6502

import (
	"fmt"
	"log"
)

// Bus defines the interface for memory access.
//
// Implementations of this interface provide the CPU with access to
// memory and memory-mapped I/O. The interface is intentionally simple
// to allow for flexible implementations.
//
// Example implementation:
//
//	type SimpleBus struct {
//	    ram [65536]uint8
//	}
//
//	func (b *SimpleBus) Read(addr uint16) uint8 {
//	    return b.ram[addr]
//	}
//
//	func (b *SimpleBus) Write(addr uint16, data uint8) {
//	    b.ram[addr] = data
//	}
type Bus interface {
	// Read returns the byte at the specified address.
	Read(addr uint16) uint8

	// Write stores a byte at the specified address.
	Write(addr uint16, data uint8)
}

// CPUConfig holds configuration options for CPU creation
type CPUConfig struct {
	// Variant specifies the CPU variant (NMOS, CMOS, Ricoh)
	Variant CPUVariant

	// ErrorHandler defines how errors are handled
	ErrorHandler ErrorHandler

	// StrictMode halts execution on illegal opcodes
	StrictMode bool

	// EnableDecimalMode allows disabling decimal mode even on variants that support it
	EnableDecimalMode bool

	// EnableInstructionCache enables instruction caching for performance
	EnableInstructionCache bool

	// InstructionCacheSize sets the cache size (default: 256 entries)
	InstructionCacheSize int
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() CPUConfig {
	return CPUConfig{
		Variant:                VariantNMOS6502,
		ErrorHandler:           &LoggingErrorHandler{Logger: log.Default()},
		StrictMode:             false,
		EnableDecimalMode:      true,
		EnableInstructionCache: true,
		InstructionCacheSize:   256,
	}
}

// InstructionCacheEntry represents a cached instruction
type InstructionCacheEntry struct {
	opcode      uint8
	instruction *Instruction
	valid       bool
}

// InstructionCache provides fast lookup for recently executed instructions
type InstructionCache struct {
	entries [256]InstructionCacheEntry // Direct-mapped cache
	hits    uint64
	misses  uint64
}

// NewInstructionCache creates a new instruction cache
func NewInstructionCache() *InstructionCache {
	return &InstructionCache{}
}

// Lookup attempts to find an instruction in the cache
func (ic *InstructionCache) Lookup(pc uint16, opcode uint8) (*Instruction, bool) {
	index := uint8(pc & 0xFF) // Use low byte of PC as cache index
	entry := &ic.entries[index]

	if entry.valid && entry.opcode == opcode {
		ic.hits++
		return entry.instruction, true
	}

	ic.misses++
	return nil, false
}

// Store adds an instruction to the cache
func (ic *InstructionCache) Store(pc uint16, opcode uint8, instruction *Instruction) {
	index := uint8(pc & 0xFF)
	ic.entries[index] = InstructionCacheEntry{
		opcode:      opcode,
		instruction: instruction,
		valid:       true,
	}
}

// Invalidate clears the cache (e.g., after self-modifying code)
func (ic *InstructionCache) Invalidate() {
	for i := range ic.entries {
		ic.entries[i].valid = false
	}
	// Reset statistics
	ic.hits = 0
	ic.misses = 0
}

// Stats returns cache statistics
func (ic *InstructionCache) Stats() (hits, misses uint64, hitRate float64) {
	total := ic.hits + ic.misses
	if total == 0 {
		return 0, 0, 0.0
	}
	return ic.hits, ic.misses, float64(ic.hits) / float64(total)
}

// CPU represents a MOS Technology 6502 microprocessor.
//
// The CPU executes instructions fetched from memory via the Bus interface.
// It maintains internal registers (A, X, Y, SP, PC, P) and provides
// cycle-accurate emulation of the 6502 instruction set.
//
// The CPU operates in a fetch-decode-execute cycle:
//  1. Fetch opcode from memory at PC
//  2. Decode opcode using lookup table
//  3. Execute addressing mode calculation
//  4. Execute instruction operation
//  5. Update cycle counter
//
// Example:
//
//	cpu := cpu6502.NewCPU(bus)
//	cpu.Reset()
//	for {
//	    if err := cpu.Clock(); err != nil {
//	        break
//	    }
//	}
type CPU struct {
	// Registers (public for direct access)
	A  uint8  // Accumulator
	X  uint8  // X Index Register
	Y  uint8  // Y Index Register
	SP uint8  // Stack Pointer (relative to $0100)
	PC uint16 // Program Counter
	P  Flags  // Processor Status Register (Flags)

	// Bus connection (public)
	bus Bus

	// PRIVATE: Internal state for instruction execution
	cycles             uint8        // Cycles remaining for the current instruction (was: Cycles)
	opcode             uint8        // Current opcode being executed
	fetchedData        uint8        // Data fetched by addressing mode
	addrAbs            uint16       // Absolute address calculated by addressing mode
	addrRel            uint16       // Relative address (for branching) - stores absolute target address
	currentInstruction *Instruction // Pointer to the definition of the current instruction

	// PRIVATE: Lookup table for instructions
	lookup [256]Instruction // (was: lookup - now private)

	// PRIVATE: Total cycles executed (for debugging/profiling)
	totalCycles uint64 // (was: totalCycles - now private)

	// Error handling
	errorHandler ErrorHandler
	lastError    *CPUError

	// Variant configuration
	variant CPUVariant

	// Interrupt state (for accurate interrupt timing)
	irqLine         bool   // Current state of IRQ line
	nmiLine         bool   // Current state of NMI line
	nmiPrevious     bool   // Previous state of NMI line (for edge detection)
	nmiPending      bool   // NMI edge detected and pending
	inInterrupt     bool   // Currently handling an interrupt
	interruptVector uint16 // Vector being used for current interrupt

	// Performance optimization
	instrCache *InstructionCache
}

// NewCPU creates a new 6502 CPU instance with default configuration.
//
// The CPU is initialized with:
//   - All registers cleared
//   - Stack pointer at $FD
//   - Status flags: U and I set
//   - NMOS 6502 variant
//   - Logging error handler
//
// The CPU must be reset before execution:
//
//	cpu := NewCPU(bus)
//	cpu.Reset() // Loads PC from reset vector at $FFFC/FD
//
// For custom configuration, use NewCPUWithConfig instead.
func NewCPU(bus Bus) *CPU {
	return NewCPUWithConfig(bus, DefaultConfig())
}

// NewCPUWithVariant creates a new CPU with specified variant
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU {
	config := DefaultConfig()
	config.Variant = variant
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithErrorHandler creates a new CPU with default NMOS 6502 variant and custom error handler
func NewCPUWithErrorHandler(bus Bus, handler ErrorHandler) *CPU {
	config := DefaultConfig()
	config.ErrorHandler = handler
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithVariantAndErrorHandler creates a new CPU with specified variant and error handler
func NewCPUWithVariantAndErrorHandler(bus Bus, variant CPUVariant, handler ErrorHandler) *CPU {
	config := DefaultConfig()
	config.Variant = variant
	config.ErrorHandler = handler
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithConfig creates a new CPU with full configuration
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU {
	// Apply strict mode to error handler if requested
	errorHandler := config.ErrorHandler
	if config.StrictMode {
		errorHandler = &StrictErrorHandler{}
	}

	c := &CPU{
		bus:          bus,
		P:            U | I,
		SP:           0xFD,
		variant:      config.Variant,
		errorHandler: errorHandler,
	}

	// Initialize instruction cache if enabled
	if config.EnableInstructionCache {
		c.instrCache = NewInstructionCache()
	}

	c.buildLookupTable()

	// Apply decimal mode configuration
	if !config.EnableDecimalMode {
		c.P &^= D // Clear decimal flag
	}

	return c
}

// CPUBuilder provides a fluent interface for CPU configuration
type CPUBuilder struct {
	bus    Bus
	config CPUConfig
}

// NewBuilder creates a new CPU builder
func NewBuilder(bus Bus) *CPUBuilder {
	return &CPUBuilder{
		bus:    bus,
		config: DefaultConfig(),
	}
}

// WithVariant sets the CPU variant
func (b *CPUBuilder) WithVariant(variant CPUVariant) *CPUBuilder {
	b.config.Variant = variant
	return b
}

// WithStrictMode enables strict mode
func (b *CPUBuilder) WithStrictMode() *CPUBuilder {
	b.config.StrictMode = true
	return b
}

// WithErrorHandler sets a custom error handler
func (b *CPUBuilder) WithErrorHandler(handler ErrorHandler) *CPUBuilder {
	b.config.ErrorHandler = handler
	return b
}

// DisableDecimalMode disables decimal mode
func (b *CPUBuilder) DisableDecimalMode() *CPUBuilder {
	b.config.EnableDecimalMode = false
	return b
}

// DisableInstructionCache disables the instruction cache
func (b *CPUBuilder) DisableInstructionCache() *CPUBuilder {
	b.config.EnableInstructionCache = false
	return b
}

// WithInstructionCacheSize sets the instruction cache size
func (b *CPUBuilder) WithInstructionCacheSize(size int) *CPUBuilder {
	b.config.InstructionCacheSize = size
	return b
}

// Build creates the configured CPU
func (b *CPUBuilder) Build() *CPU {
	return NewCPUWithConfig(b.bus, b.config)
}

// Variant returns the CPU variant
func (c *CPU) Variant() CPUVariant {
	return c.variant
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

	if c.variant.HasIndirectJMPBug() && ptrLo == 0x00FF {
		// NMOS bug: If low byte is $FF, high byte is fetched from $xx00
		c.addrAbs = uint16(c.read(ptr)) | (uint16(c.read(ptr&0xFF00)) << 8)
	} else {
		// CMOS fix: Normal behavior
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

func (c *CPU) AND() uint8 {
	c.fetchDataIfNeeded()
	c.A &= c.fetchedData
	c.setZNFlags(c.A)
	return 0
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

	// Set interrupt state (BRK uses IRQ vector)
	c.inInterrupt = true
	c.interruptVector = 0xFFFE

	// BRK takes 7 cycles total (base cycles handled by lookup table)
	// This function returns *extra* cycles, which should be 0 here.
	return 0
}

func (c *CPU) CMP() uint8 {
	c.fetchDataIfNeeded()
	temp := uint16(c.A) - uint16(c.fetchedData)
	c.setFlag(C, c.A >= c.fetchedData)
	c.setZNFlags(uint8(temp & 0x00FF))
	return 0
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
	return 0
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
	return 0
}

func (c *CPU) LDX() uint8 {
	c.fetchDataIfNeeded()
	c.X = c.fetchedData
	c.setZNFlags(c.X)
	return 0
}

func (c *CPU) LDY() uint8 {
	c.fetchDataIfNeeded()
	c.Y = c.fetchedData
	c.setZNFlags(c.Y)
	return 0
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
	// Page cross handling is now done via PageCrossPenalty field in lookup table
	return 0
}

func (c *CPU) ORA() uint8 {
	c.fetchDataIfNeeded()
	c.A |= c.fetchedData
	c.setZNFlags(c.A)
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

	// Clear interrupt state
	c.inInterrupt = false
	c.interruptVector = 0

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
	// Page Cross Penalty Reference
	//
	// The 6502 CPU has different cycle timing behavior when indexed addressing modes
	// (ABX, ABY, IZY) cross page boundaries. A page boundary is crossed when the
	// high byte of the effective address differs from the high byte of the base address.
	//
	// Instructions that ADD +1 cycle on page boundary cross:
	//   - Load: LDA, LDX, LDY (ABX, ABY, IZY modes)
	//   - Logic: AND, EOR, ORA (ABX, ABY, IZY modes)
	//   - Arithmetic: ADC, SBC (ABX, ABY, IZY modes)
	//   - Compare: CMP (ABX, ABY, IZY modes)
	//   - Unofficial: *NOP with ABX addressing (opcodes 0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC)
	//
	// Instructions that DO NOT add cycle on page cross (always use full cycles):
	//   - Store: STA, STX, STY (always take 5/6 cycles regardless of page cross)
	//   - Read-Modify-Write: ASL, LSR, ROL, ROR, INC, DEC (always take 7 cycles with ABX)
	//   - CPX, CPY (no indexed modes that can cross pages)
	//
	// Addressing modes that can cross pages:
	//   - ABX (Absolute,X): effective = base + X
	//   - ABY (Absolute,Y): effective = base + Y
	//   - IZY (Indirect,Y): effective = [zp] + Y
	//
	// Page cross detection: (effective & 0xFF00) != (base & 0xFF00)
	//
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
		c.lookup[i] = Instruction{Name: "XXX", Operate: XXX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Illegal: true, PageCrossPenalty: false}
	}

	// --- Official Opcodes --- Using the helper variables above
	// Load instructions - ADD page cross penalty for indexed modes
	c.lookup[0xA9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xB9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xA1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0xA2] = Instruction{Name: "LDX", Operate: LDX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}

	c.lookup[0xA0] = Instruction{Name: "LDY", Operate: LDY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}

	// Store instructions - NO page cross penalty (always use full cycles)
	c.lookup[0x85] = Instruction{Name: "STA", Operate: STA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x95] = Instruction{Name: "STA", Operate: STA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x9D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x99] = Instruction{Name: "STA", Operate: STA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x81] = Instruction{Name: "STA", Operate: STA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x91] = Instruction{Name: "STA", Operate: STA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 6, Length: 2, PageCrossPenalty: false}

	c.lookup[0x86] = Instruction{Name: "STX", Operate: STX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x96] = Instruction{Name: "STX", Operate: STX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8E] = Instruction{Name: "STX", Operate: STX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	c.lookup[0x84] = Instruction{Name: "STY", Operate: STY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x94] = Instruction{Name: "STY", Operate: STY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8C] = Instruction{Name: "STY", Operate: STY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Transfer and stack instructions - no page cross possible
	c.lookup[0xAA] = Instruction{Name: "TAX", Operate: TAX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xA8] = Instruction{Name: "TAY", Operate: TAY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xBA] = Instruction{Name: "TSX", Operate: TSX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x8A] = Instruction{Name: "TXA", Operate: TXA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x9A] = Instruction{Name: "TXS", Operate: TXS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x98] = Instruction{Name: "TYA", Operate: TYA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0x48] = Instruction{Name: "PHA", Operate: PHA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1, PageCrossPenalty: false}
	c.lookup[0x08] = Instruction{Name: "PHP", Operate: PHP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1, PageCrossPenalty: false}
	c.lookup[0x68] = Instruction{Name: "PLA", Operate: PLA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1, PageCrossPenalty: false}
	c.lookup[0x28] = Instruction{Name: "PLP", Operate: PLP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1, PageCrossPenalty: false}

	// Arithmetic instructions - ADD page cross penalty for indexed modes
	c.lookup[0x69] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x65] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x75] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x6D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x7D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x79] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x61] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x71] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0xE9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xE5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xED] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xFD] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xF9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xE1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// Read-modify-write instructions - NO page cross penalty (always use full cycles)
	c.lookup[0xE6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xEE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0xFE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}
	c.lookup[0xE8] = Instruction{Name: "INX", Operate: INX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xC8] = Instruction{Name: "INY", Operate: INY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0xC6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0xDE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}
	c.lookup[0xCA] = Instruction{Name: "DEX", Operate: DEX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x88] = Instruction{Name: "DEY", Operate: DEY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0x0A] = Instruction{Name: "ASL", Operate: ASL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x06] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x16] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x0E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x1E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x4A] = Instruction{Name: "LSR", Operate: LSR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x46] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x56] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x4E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x5E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x2A] = Instruction{Name: "ROL", Operate: ROL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x26] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x36] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x3E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x6A] = Instruction{Name: "ROR", Operate: ROR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x66] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x76] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x6E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x7E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	// Logical instructions - ADD page cross penalty for indexed modes
	c.lookup[0x29] = Instruction{Name: "AND", Operate: AND, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x25] = Instruction{Name: "AND", Operate: AND, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x35] = Instruction{Name: "AND", Operate: AND, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x3D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x39] = Instruction{Name: "AND", Operate: AND, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x21] = Instruction{Name: "AND", Operate: AND, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x31] = Instruction{Name: "AND", Operate: AND, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0x49] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x45] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x55] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x4D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x5D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x59] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x41] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x51] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0x09] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x05] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x15] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x0D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x1D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x19] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x01] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x11] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// BIT instruction - no page cross possible
	c.lookup[0x24] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2C] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Compare instructions - ADD page cross penalty for indexed modes
	c.lookup[0xC9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xC5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xDD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xD9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xC1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// CPX, CPY - no indexed modes that can cross pages
	c.lookup[0xE0] = Instruction{Name: "CPX", Operate: CPX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xE4] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xEC] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	c.lookup[0xC0] = Instruction{Name: "CPY", Operate: CPY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xC4] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCC] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Branch instructions - no page cross penalty (handled by branch logic)
	c.lookup[0x90] = Instruction{Name: "BCC", Operate: BCC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB0] = Instruction{Name: "BCS", Operate: BCS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF0] = Instruction{Name: "BEQ", Operate: BEQ, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x30] = Instruction{Name: "BMI", Operate: BMI, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD0] = Instruction{Name: "BNE", Operate: BNE, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x10] = Instruction{Name: "BPL", Operate: BPL, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x50] = Instruction{Name: "BVC", Operate: BVC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x70] = Instruction{Name: "BVS", Operate: BVS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}

	// Jump and control flow instructions - no page cross penalty
	c.lookup[0x4C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 3, Length: 3, PageCrossPenalty: false}
	c.lookup[0x6C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: IND, AddrModeType: AddrModeIND, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x20] = Instruction{Name: "JSR", Operate: JSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x60] = Instruction{Name: "RTS", Operate: RTS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1, PageCrossPenalty: false}

	c.lookup[0x00] = Instruction{Name: "BRK", Operate: BRK, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 7, Length: 1, PageCrossPenalty: false}
	c.lookup[0x40] = Instruction{Name: "RTI", Operate: RTI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1, PageCrossPenalty: false}

	// Flag instructions - no page cross penalty
	c.lookup[0x18] = Instruction{Name: "CLC", Operate: CLC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xD8] = Instruction{Name: "CLD", Operate: CLD, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x58] = Instruction{Name: "CLI", Operate: CLI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xB8] = Instruction{Name: "CLV", Operate: CLV, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x38] = Instruction{Name: "SEC", Operate: SEC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xF8] = Instruction{Name: "SED", Operate: SED, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x78] = Instruction{Name: "SEI", Operate: SEI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0xEA] = Instruction{Name: "NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	// Illegal NOPs
	// https://www.masswerk.at/6502/6502_instruction_set.html#NOPs
	c.lookup[0x1A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x3A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x5A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x7A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xDA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xFA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x80] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x82] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x89] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xC2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xE2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x04] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x44] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x64] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x14] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x34] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x54] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x74] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xD4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xF4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x0C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: false}

	// Unofficial NOPs with ABX addressing - ADD page cross penalty
	c.lookup[0x1C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x3C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x5C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x7C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0xDC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0xFC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}

}

// --- Core Execution ---

// Reset initializes the CPU to its power-on state.
//
// This method:
//   - Clears all registers (A, X, Y)
//   - Sets stack pointer to $FD
//   - Sets status flags to U | I
//   - Loads PC from reset vector at $FFFC/FD
//   - Takes 8 cycles to complete
//
// The reset vector should be set in memory before calling Reset:
//
//	bus.Write(0xFFFC, 0x00) // Low byte
//	bus.Write(0xFFFD, 0x80) // High byte -> PC = $8000
//	cpu.Reset()
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
	c.cycles = 8
}

// InterruptRequest is deprecated. Use SetIRQ(true) instead.
// This method is kept for backward compatibility.
func (c *CPU) InterruptRequest() {
	c.SetIRQ(true)
	// Force immediate handling for backward compatibility
	if !c.getFlag(I) && c.cycles == 0 {
		c.handleIRQ()
	}
}

// NonMaskableInterrupt is deprecated. Use SetNMI(false) after SetNMI(true) instead.
// This method is kept for backward compatibility.
func (c *CPU) NonMaskableInterrupt() {
	c.SetNMI(true)
	c.SetNMI(false) // Create falling edge
	// Force immediate handling for backward compatibility
	if c.cycles == 0 {
		c.handleNMI()
	}
}

// SetIRQ sets the IRQ line state
// IRQ is level-triggered: it will be serviced as long as the line is asserted
func (c *CPU) SetIRQ(asserted bool) {
	c.irqLine = asserted
}

// SetNMI sets the NMI line state
// NMI is edge-triggered: it will be serviced on a falling edge (high to low)
func (c *CPU) SetNMI(asserted bool) {
	// Detect falling edge (high to low transition)
	if c.nmiPrevious && !asserted {
		c.nmiPending = true
	}
	c.nmiPrevious = asserted
	c.nmiLine = asserted
}

// ClearNMI clears the pending NMI
// This is called after NMI is serviced
func (c *CPU) ClearNMI() {
	c.nmiPending = false
}

// HasPendingInterrupt returns true if any interrupt is pending
func (c *CPU) HasPendingInterrupt() bool {
	return c.nmiPending || (c.irqLine && !c.getFlag(I))
}

// handleNMI handles a Non-Maskable Interrupt
// NMI takes 7 cycles and cannot be disabled
func (c *CPU) handleNMI() error {
	// NMI can hijack an IRQ sequence
	// If we're in the middle of an IRQ, the vector changes to NMI

	// Push PC onto stack (current PC, not PC+1)
	c.push16(c.PC)

	// Push status register with B clear, U set
	c.setFlag(B, false)
	c.setFlag(U, true)
	c.push(uint8(c.P))

	// Set Interrupt Disable flag
	c.setFlag(I, true)

	// Read NMI vector ($FFFA/B)
	lo := uint16(c.read(0xFFFA))
	hi := uint16(c.read(0xFFFB))
	c.PC = (hi << 8) | lo

	// Clear the pending NMI
	c.nmiPending = false

	// Set interrupt state
	c.inInterrupt = true
	c.interruptVector = 0xFFFA

	// NMI takes 7 cycles
	c.cycles = 7

	return nil
}

// handleIRQ handles an Interrupt Request
// IRQ takes 7 cycles and can be disabled by the I flag
func (c *CPU) handleIRQ() error {
	// IRQ is ignored if I flag is set
	if c.getFlag(I) {
		return nil
	}

	// Push PC onto stack (current PC, not PC+1)
	c.push16(c.PC)

	// Push status register with B clear, U set
	c.setFlag(B, false)
	c.setFlag(U, true)
	c.push(uint8(c.P))

	// Set Interrupt Disable flag
	c.setFlag(I, true)

	// Read IRQ vector ($FFFE/F)
	lo := uint16(c.read(0xFFFE))
	hi := uint16(c.read(0xFFFF))
	c.PC = (hi << 8) | lo

	// Set interrupt state
	c.inInterrupt = true
	c.interruptVector = 0xFFFE

	// IRQ takes 7 cycles
	c.cycles = 7

	return nil
}

// Clock executes one clock cycle of the CPU.
//
// This method should be called repeatedly to execute instructions.
// Each instruction takes multiple cycles to complete. The CPU tracks
// remaining cycles internally and fetches the next instruction when
// the current one completes.
//
// Returns an error if an unrecoverable error occurs (e.g., illegal
// opcode in strict mode). The error can be handled or ignored based
// on the configured error handler.
//
// Example:
//
//	for {
//	    if err := cpu.Clock(); err != nil {
//	        log.Printf("CPU error: %v", err)
//	        break
//	    }
//	}
func (c *CPU) Clock() error {
	if c.cycles == 0 {
		// Check for interrupts BEFORE fetching next instruction
		// This ensures interrupts are serviced between instructions
		if c.nmiPending {
			return c.handleNMI()
		}

		if c.irqLine && !c.getFlag(I) && !c.inInterrupt {
			return c.handleIRQ()
		}

		// Normal instruction fetch
		c.opcode = c.read(c.PC)
		fetchPC := c.PC // Save PC for cache lookup
		c.PC++

		c.setFlag(U, true) // Ensure U is always set before execution

		// Try cache lookup first
		if c.instrCache != nil {
			if instr, hit := c.instrCache.Lookup(fetchPC, c.opcode); hit {
				c.currentInstruction = instr
			} else {
				// Cache miss - use lookup table
				c.currentInstruction = &c.lookup[c.opcode]
				// Store in cache for next time
				c.instrCache.Store(fetchPC, c.opcode, c.currentInstruction)
			}
		} else {
			// Cache disabled
			c.currentInstruction = &c.lookup[c.opcode]
		}

		// Check for illegal opcodes
		if c.currentInstruction.Illegal {
			err := &CPUError{
				Type:    ErrorIllegalOpcode,
				Opcode:  c.opcode,
				PC:      c.PC - 1,
				Message: fmt.Sprintf("illegal opcode $%02X", c.opcode),
			}
			c.lastError = err

			if c.errorHandler != nil {
				if handlerErr := c.errorHandler.HandleError(err); handlerErr != nil {
					return handlerErr
				}
			}
		}

		baseCycles := c.currentInstruction.Cycles
		// Call AddrMode and Operate methods via the function pointers in the struct
		// These methods implicitly receive 'c' as their receiver.
		addrModeCycles := c.currentInstruction.AddrMode(c)
		opCycles := c.currentInstruction.Operate(c)

		// Only add addressing mode cycles if instruction supports page cross penalty
		if c.currentInstruction.PageCrossPenalty {
			c.cycles = baseCycles + addrModeCycles + opCycles
		} else {
			// Ignore addressing mode cycles (page cross doesn't add cycle)
			c.cycles = baseCycles + opCycles
		}

		// Ensure U is set after execution as well (might be cleared by PLP?)
		c.setFlag(U, true)

	}

	c.cycles--
	c.totalCycles++
	return nil
}

// --- Accessor Methods ---

// RemainingCycles returns the number of cycles remaining for the current instruction
func (c *CPU) RemainingCycles() uint8 {
	return c.cycles
}

// SetCycles sets the number of cycles remaining (for testing/debugging)
func (c *CPU) SetCycles(cycles uint8) {
	c.cycles = cycles
}

// TotalCycles returns the total number of cycles executed by the CPU since its
// creation. This is useful for profiling and debugging purposes.
func (c *CPU) TotalCycles() uint64 {
	return c.totalCycles
}

// CurrentOpcode returns the opcode of the currently executing instruction
func (c *CPU) CurrentOpcode() uint8 {
	return c.opcode
}

// LookupInstruction returns the instruction definition for a given opcode
func (c *CPU) LookupInstruction(opcode uint8) Instruction {
	return c.lookup[opcode]
}

// IsIllegalOpcode returns true if the given opcode is illegal/unofficial
func (c *CPU) IsIllegalOpcode(opcode uint8) bool {
	return c.lookup[opcode].Illegal
}

// --- Instruction Cache Control ---

// InvalidateInstructionCache clears the instruction cache
// Call this after self-modifying code or when loading new programs
func (c *CPU) InvalidateInstructionCache() {
	if c.instrCache != nil {
		c.instrCache.Invalidate()
	}
}

// InstructionCacheStats returns cache performance statistics
func (c *CPU) InstructionCacheStats() (hits, misses uint64, hitRate float64) {
	if c.instrCache != nil {
		return c.instrCache.Stats()
	}
	return 0, 0, 0.0
}

// DisableInstructionCache disables the instruction cache
func (c *CPU) DisableInstructionCache() {
	c.instrCache = nil
}

// EnableInstructionCache enables the instruction cache
func (c *CPU) EnableInstructionCache() {
	if c.instrCache == nil {
		c.instrCache = NewInstructionCache()
	}
}

// --- State Inspection ---

// GetStateSnapshot returns a snapshot of the current CPU state
func (c *CPU) GetStateSnapshot() State {
	instrName := "???"
	if c.currentInstruction != nil {
		instrName = c.currentInstruction.Name
	}

	return State{
		A:               c.A,
		X:               c.X,
		Y:               c.Y,
		SP:              c.SP,
		PC:              c.PC,
		P:               c.P,
		Cycles:          c.cycles,
		TotalCycles:     c.totalCycles,
		Opcode:          c.opcode,
		Instruction:     instrName,
		InInterrupt:     c.inInterrupt,
		InterruptVector: c.interruptVector,
	}
}

// --- Debug Helpers ---

// GetCurrentInstruction returns a pointer to the current instruction definition.
// Returns nil if no instruction has been fetched yet.
func (c *CPU) GetCurrentInstruction() *Instruction {
	return c.currentInstruction
}

// Opcode returns the last fetched opcode. Useful for debugging/halt conditions.
// Deprecated: Use CurrentOpcode() instead
func (c *CPU) Opcode() uint8 {
	return c.opcode
}

// LastError returns the last error that occurred during execution
func (c *CPU) LastError() *CPUError {
	return c.lastError
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

// Disassemble - Disassemble instructions in memory range
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string {
	disassembly := make(map[uint16]string)
	addr := startAddr

	for addr <= endAddr && addr >= startAddr { // Check for wrap around too
		lineAddr := addr
		opcode := c.read(addr)
		addr++ // Consume opcode byte
		instr := c.lookup[opcode]
		operandStr := ""

		switch instr.AddrModeType { // Use enum comparison
		case AddrModeIMP:
			// No operand bytes
		case AddrModeIMM:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("#$%02X", val)
		case AddrModeZP0:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X", val)
		case AddrModeZPX:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,X", val)
		case AddrModeZPY:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,Y", val)
		case AddrModeREL:
			val := c.read(addr)
			addr++            // Consume operand byte
			relTarget := addr // Relative jump target is relative to the *next* instruction
			if val&0x80 > 0 {
				relTarget += uint16(int16(val)) // Handle negative offset
			} else {
				relTarget += uint16(val) // Positive offset
			}
			operandStr = fmt.Sprintf("$%04X", relTarget)
		case AddrModeABS:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X", (hi<<8)|lo)
		case AddrModeABX:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,X", (hi<<8)|lo)
		case AddrModeABY:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,Y", (hi<<8)|lo)
		case AddrModeIND:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("($%04X)", (hi<<8)|lo)
		case AddrModeIZX:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("($%02X,X)", val)
		case AddrModeIZY:
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
