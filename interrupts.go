package cpu6502

// Interrupt Handling
//
// The 6502 supports three types of interrupts:
//   - IRQ (Interrupt Request): Maskable, level-triggered
//   - NMI (Non-Maskable Interrupt): Edge-triggered (falling edge)
//   - BRK (Break): Software interrupt
//
// Each interrupt has a specific vector address:
//   - NMI: $FFFA/FB
//   - Reset: $FFFC/FD
//   - IRQ/BRK: $FFFE/FF

// InterruptRequest is deprecated. Use SetIRQ(true) instead.
//
// This method is kept for backward compatibility with existing code.
// It immediately asserts the IRQ line and attempts to handle the
// interrupt if conditions allow.
//
// Deprecated: Use SetIRQ(true) for proper interrupt handling.
func (c *CPU) InterruptRequest() {
	c.SetIRQ(true)
	// Force immediate handling for backward compatibility
	if !c.getFlag(I) && c.cycles == 0 {
		c.handleIRQ()
	}
}

// NonMaskableInterrupt is deprecated. Use SetNMI(false) after SetNMI(true) instead.
//
// This method is kept for backward compatibility with existing code.
// It creates a falling edge on the NMI line by setting it high then low,
// which triggers the NMI interrupt.
//
// Deprecated: Use SetNMI(true) followed by SetNMI(false) for proper NMI handling.
func (c *CPU) NonMaskableInterrupt() {
	c.SetNMI(true)
	c.SetNMI(false) // Create falling edge
	// Force immediate handling for backward compatibility
	if c.cycles == 0 {
		c.handleNMI()
	}
}

// SetIRQ sets the IRQ line state.
//
// IRQ is level-triggered: it will be serviced as long as the line is
// asserted and the I (Interrupt Disable) flag is clear. The interrupt
// is checked between instructions, not during instruction execution.
//
// Parameters:
//   - asserted: true to assert IRQ, false to clear it
//
// Example:
//
//	// Assert IRQ (e.g., from a timer)
//	cpu.SetIRQ(true)
//
//	// Later, clear IRQ after handling
//	cpu.SetIRQ(false)
//
// Note: The interrupt will only be serviced if:
//   - The I flag is clear (interrupts enabled)
//   - No instruction is currently executing (cycles == 0)
//   - Not already handling an interrupt
func (c *CPU) SetIRQ(asserted bool) {
	c.irqLine = asserted
}

// SetNMI sets the NMI line state.
//
// NMI is edge-triggered: it will be serviced on a falling edge
// (high to low transition). Once triggered, the NMI is pending
// until serviced, even if the line goes high again.
//
// Parameters:
//   - asserted: true to set line high, false to set line low
//
// Example:
//
//	// Trigger NMI with falling edge
//	cpu.SetNMI(true)  // Set high
//	cpu.SetNMI(false) // Set low - triggers NMI
//
// Note: The interrupt will be serviced at the next instruction
// boundary, regardless of the I flag state.
func (c *CPU) SetNMI(asserted bool) {
	// Detect falling edge (high to low transition)
	if c.nmiPrevious && !asserted {
		c.nmiPending = true
	}
	c.nmiPrevious = asserted
	c.nmiLine = asserted
}

// ClearNMI clears the pending NMI.
//
// This is called internally after NMI is serviced. It should not
// normally be called by user code, as the CPU handles this automatically.
func (c *CPU) ClearNMI() {
	c.nmiPending = false
}

// HasPendingInterrupt returns true if any interrupt is pending.
//
// This checks for:
//   - Pending NMI (always serviceable)
//   - Asserted IRQ with I flag clear (maskable)
//
// Returns true if an interrupt will be serviced at the next
// instruction boundary.
//
// Example:
//
//	if cpu.HasPendingInterrupt() {
//	    fmt.Println("Interrupt will be serviced soon")
//	}
func (c *CPU) HasPendingInterrupt() bool {
	return c.nmiPending || (c.irqLine && !c.getFlag(I))
}

// handleNMI handles a Non-Maskable Interrupt.
//
// NMI processing:
//  1. Push PC onto stack (current PC, not PC+1)
//  2. Push status register with B clear, U set
//  3. Set I flag (disable further IRQs)
//  4. Load PC from NMI vector ($FFFA/FB)
//  5. Takes 7 cycles
//
// NMI can hijack an IRQ sequence. If an IRQ is being processed
// and NMI occurs, the vector changes to NMI.
//
// Returns nil on success, error if stack operations fail.
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

// handleIRQ handles an Interrupt Request.
//
// IRQ processing:
//  1. Check if I flag is set (if so, ignore interrupt)
//  2. Push PC onto stack (current PC, not PC+1)
//  3. Push status register with B clear, U set
//  4. Set I flag (disable further IRQs)
//  5. Load PC from IRQ vector ($FFFE/FF)
//  6. Takes 7 cycles
//
// IRQ is ignored if the I (Interrupt Disable) flag is set.
// This allows critical sections of code to run without interruption.
//
// Returns nil on success, error if stack operations fail.
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
