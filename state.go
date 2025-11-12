package cpu6502

import "fmt"

// State Inspection and Snapshot
//
// This file provides methods for inspecting and capturing the CPU's
// internal state. These are useful for debugging, testing, and
// implementing save states in emulators.

// RemainingCycles returns the number of cycles remaining for the current instruction.
//
// Each instruction takes multiple cycles to complete. This method returns
// how many cycles are left before the next instruction will be fetched.
//
// Returns 0 if no instruction is currently executing (ready for next instruction).
//
// Example:
//
//	for cpu.RemainingCycles() > 0 {
//	    cpu.Clock()
//	}
func (c *CPU) RemainingCycles() uint8 {
	return c.cycles
}

// SetCycles sets the number of cycles remaining.
//
// This method is primarily for testing and debugging purposes.
// It allows manual control of the cycle counter, which can be
// useful for testing timing-sensitive code.
//
// Parameters:
//   - cycles: Number of cycles to set
//
// Warning: Modifying the cycle counter during normal execution
// can lead to incorrect timing and behavior.
func (c *CPU) SetCycles(cycles uint8) {
	c.cycles = cycles
}

// TotalCycles returns the total number of cycles executed since CPU creation.
//
// This counter increments with each clock cycle and is useful for:
//   - Performance profiling
//   - Timing synchronization with other components
//   - Debugging timing-sensitive issues
//
// The counter wraps around at 2^64 cycles (effectively never in practice).
//
// Example:
//
//	start := cpu.TotalCycles()
//	// ... execute some code ...
//	elapsed := cpu.TotalCycles() - start
//	fmt.Printf("Executed %d cycles\n", elapsed)
func (c *CPU) TotalCycles() uint64 {
	return c.totalCycles
}

// CurrentOpcode returns the opcode of the currently executing instruction.
//
// This is the opcode that was fetched at the start of the current
// instruction's execution. It remains valid until the next instruction
// is fetched.
//
// Returns the current opcode byte (0x00-0xFF).
//
// Example:
//
//	opcode := cpu.CurrentOpcode()
//	instr := cpu.LookupInstruction(opcode)
//	fmt.Printf("Executing: %s\n", instr.Name)
func (c *CPU) CurrentOpcode() uint8 {
	return c.opcode
}

// LookupInstruction returns the instruction definition for a given opcode.
//
// This provides access to the instruction's metadata including:
//   - Name (mnemonic)
//   - Cycle count
//   - Addressing mode
//   - Whether it's an illegal opcode
//
// Parameters:
//   - opcode: The opcode to look up (0x00-0xFF)
//
// Returns the Instruction struct for the given opcode.
//
// Example:
//
//	instr := cpu.LookupInstruction(0xA9) // LDA immediate
//	fmt.Printf("%s takes %d cycles\n", instr.Name, instr.Cycles)
func (c *CPU) LookupInstruction(opcode uint8) Instruction {
	return c.lookup[opcode]
}

// IsIllegalOpcode returns true if the given opcode is illegal/unofficial.
//
// Illegal opcodes are undocumented instructions that exist due to
// the 6502's internal logic but were not officially supported.
// Some programs use these for various purposes.
//
// Parameters:
//   - opcode: The opcode to check (0x00-0xFF)
//
// Returns true if the opcode is illegal, false if it's official.
//
// Example:
//
//	if cpu.IsIllegalOpcode(0x04) {
//	    fmt.Println("This is an illegal NOP")
//	}
func (c *CPU) IsIllegalOpcode(opcode uint8) bool {
	return c.lookup[opcode].Illegal
}

// GetStateSnapshot returns a snapshot of the current CPU state.
//
// This captures all relevant CPU state in a single struct, useful for:
//   - Debugging and logging
//   - Implementing save states
//   - Testing and verification
//   - Comparing states across executions
//
// The snapshot includes:
//   - All registers (A, X, Y, SP, PC, P)
//   - Cycle counters
//   - Current opcode and instruction
//   - Interrupt state
//
// Returns a State struct containing the complete CPU state.
//
// Example:
//
//	state := cpu.GetStateSnapshot()
//	fmt.Printf("PC: $%04X, A: $%02X\n", state.PC, state.A)
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

// LastError returns the last error that occurred during execution.
//
// This provides access to detailed error information including:
//   - Error type
//   - Opcode that caused the error
//   - Program counter at the time of error
//   - Descriptive message
//
// Returns nil if no error has occurred, or a pointer to the last CPUError.
//
// Example:
//
//	if err := cpu.Clock(); err != nil {
//	    if cpuErr := cpu.LastError(); cpuErr != nil {
//	        fmt.Printf("Error at $%04X: %s\n", cpuErr.PC, cpuErr.Message)
//	    }
//	}
func (c *CPU) LastError() *CPUError {
	return c.lastError
}

// GetState returns a formatted string representation of the CPU state.
//
// This provides a human-readable view of the CPU state, useful for
// debugging and logging. The format is similar to common 6502 debuggers.
//
// Format: PC:XXXX A:XX X:XX Y:XX P:XX[FLAGS] SP:XX CYC:NNNN (INSTR $XX)
//
// Where FLAGS is an 8-character string showing each flag:
//   - N: Negative
//   - V: Overflow
//   - U: Unused (always set)
//   - B: Break
//   - D: Decimal
//   - I: Interrupt Disable
//   - Z: Zero
//   - C: Carry
//
// Returns a formatted string describing the current state.
//
// Example output:
//
//	PC:8000 A:42 X:10 Y:20 P:24[..U.D...] SP:FD CYC:1234 (LDA $A9)
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
