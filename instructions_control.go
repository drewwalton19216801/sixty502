package cpu6502

import "log"

// instructions_control.go contains control flow instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Jump instructions (JMP, JSR, RTS, RTI)
//   - Break instruction (BRK)
//   - No operation (NOP)
//   - Illegal opcode handler (XXX)
//
// These instructions control program flow, handle interrupts, and manage
// the call stack.

// JMP - Jump
// Sets the program counter to the specified address.
// This is an unconditional jump.
// Flags affected: None
func (c *CPU) JMP() uint8 {
	c.PC = c.addrAbs
	return 0
}

// JSR - Jump to Subroutine
// Pushes the return address (PC-1) onto the stack and jumps to the subroutine.
// The return address points to the last byte of the JSR instruction,
// so RTS will increment it to return to the next instruction.
// Flags affected: None
func (c *CPU) JSR() uint8 {
	c.PC--
	c.push16(c.PC)
	c.PC = c.addrAbs
	return 0
}

// RTS - Return from Subroutine
// Pulls the return address from the stack and increments it.
// This returns control to the instruction after the JSR.
// Flags affected: None
func (c *CPU) RTS() uint8 {
	c.PC = c.pop16()
	c.PC++
	return 0
}

// RTI - Return from Interrupt
// Pulls the processor status and program counter from the stack.
// This returns control from an interrupt handler.
// The B flag is cleared and U flag is set in the restored status.
// Flags affected: All (restored from stack)
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

// BRK - Break
// Triggers a software interrupt (IRQ).
// Pushes PC+2 and processor status (with B and U flags set) onto the stack,
// sets the I flag, and jumps to the IRQ vector at $FFFE/F.
//
// Note: The 6502 increments PC twice after fetching the BRK opcode,
// so the pushed PC points two bytes after the BRK instruction.
// Flags affected: B (set in pushed copy only), I (set in actual register)
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

// NOP - No Operation
// Does nothing. Takes 2 cycles.
// Some unofficial opcodes are also NOPs with different addressing modes
// and cycle counts, but they all use this same implementation.
// Flags affected: None
func (c *CPU) NOP() uint8 {
	// Page cross handling is now done via PageCrossPenalty field in lookup table
	return 0
}

// XXX - Illegal Opcode Handler
// Called when an illegal/unofficial opcode is encountered.
// Logs an error message and returns 1 to prevent infinite loops.
// The actual error handling behavior depends on the configured error handler.
// Flags affected: None
func (c *CPU) XXX() uint8 {
	log.Printf("ERROR: Illegal opcode $%02X encountered at $%04X", c.opcode, c.PC-1)
	return 1 // Return 1 to prevent potential infinite loops if Clock checks >=0
}
