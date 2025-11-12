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
)

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
