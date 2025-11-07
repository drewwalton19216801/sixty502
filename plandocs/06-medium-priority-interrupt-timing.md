# Medium Priority: Improve Interrupt Timing Accuracy

## Problem Statement

Current interrupt handling in [`InterruptRequest()`](../cpu.go:1088) and [`NonMaskableInterrupt()`](../cpu.go:1114) has timing issues:

1. **No Interrupt Polling**: Interrupts are handled immediately when called, not between instructions
2. **No Edge Detection**: NMI should be edge-triggered, not level-triggered
3. **No Interrupt Latency**: Real 6502 has specific timing for interrupt handling
4. **No IRQ Hijacking**: NMI can hijack an in-progress IRQ sequence

## Proposed Solution

Implement proper interrupt polling and edge detection with accurate timing.

### Implementation Steps

#### Step 1: Add Interrupt State Fields

```go
type CPU struct {
    // ... existing fields
    
    // Interrupt state
    irqLine     bool // Current state of IRQ line
    nmiLine     bool // Current state of NMI line
    nmiPrevious bool // Previous state of NMI line (for edge detection)
    nmiPending  bool // NMI edge detected and pending
    
    // Interrupt handling state
    inInterrupt     bool // Currently handling an interrupt
    interruptVector uint16 // Vector being used for current interrupt
}
```

#### Step 2: Add Interrupt Line Methods

```go
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
```

#### Step 3: Update Clock() for Interrupt Polling

```go
// Clock executes one clock cycle of the CPU
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
        c.PC++
        
        c.setFlag(U, true)
        c.currentInstruction = &c.lookup[c.opcode]
        
        // Check for illegal opcodes
        if c.currentInstruction.Illegal {
            err := &CPUError{
                Type:    ErrorIllegalOpcode,
                Opcode:  c.opcode,
                PC:      c.PC - 1,
                Message: fmt.Sprintf("illegal opcode $%02X", c.opcode),
            }
            c.lastError = err
            
            if handlerErr := c.errorHandler.HandleError(err); handlerErr != nil {
                return handlerErr
            }
        }
        
        baseCycles := c.currentInstruction.Cycles
        addrModeCycles := c.currentInstruction.AddrMode(c)
        opCycles := c.currentInstruction.Operate(c)
        c.cycles = baseCycles + addrModeCycles + opCycles
        c.setFlag(U, true)
    }
    
    c.cycles--
    c.totalCycles++
    return nil
}
```

#### Step 4: Implement Accurate Interrupt Handling

```go
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
```

#### Step 5: Update RTI to Clear Interrupt State

```go
// RTI returns from interrupt
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
```

#### Step 6: Update BRK to Set Interrupt State

```go
// BRK handles software interrupt
func (c *CPU) BRK() uint8 {
    c.PC++
    c.push16(c.PC)
    
    c.setFlag(B, true)
    c.setFlag(U, true)
    originalP := c.P
    c.push(uint8(originalP | B | U))
    
    c.setFlag(I, true)
    c.setFlag(B, false)
    
    lo := uint16(c.read(0xFFFE))
    hi := uint16(c.read(0xFFFF))
    c.PC = (hi << 8) | lo
    
    // Set interrupt state (BRK uses IRQ vector)
    c.inInterrupt = true
    c.interruptVector = 0xFFFE
    
    return 0
}
```

#### Step 7: Deprecate Old Interrupt Methods

```go
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
```

## Usage Examples

### Example 1: Level-Triggered IRQ

```go
// Assert IRQ line
cpu.SetIRQ(true)

// IRQ will be serviced at the next instruction boundary
// if I flag is clear
for i := 0; i < 100; i++ {
    cpu.Clock()
}

// Clear IRQ line when interrupt source is handled
cpu.SetIRQ(false)
```

### Example 2: Edge-Triggered NMI

```go
// Generate NMI by creating a falling edge
cpu.SetNMI(true)  // Assert NMI line
cpu.SetNMI(false) // Create falling edge

// NMI will be serviced at the next instruction boundary
for i := 0; i < 100; i++ {
    cpu.Clock()
}
```

### Example 3: NMI Hijacking IRQ

```go
// Assert IRQ
cpu.SetIRQ(true)

// Run a few cycles - IRQ starts being serviced
for i := 0; i < 3; i++ {
    cpu.Clock()
}

// Generate NMI during IRQ sequence
cpu.SetNMI(true)
cpu.SetNMI(false)

// NMI will hijack the IRQ - PC will jump to NMI vector
for i := 0; i < 10; i++ {
    cpu.Clock()
}
```

### Example 4: Checking Interrupt State

```go
if cpu.HasPendingInterrupt() {
    fmt.Println("Interrupt pending")
}

// Check if currently in interrupt handler
state := cpu.GetStateSnapshot()
if state.InInterrupt {
    fmt.Printf("In interrupt handler (vector: $%04X)\n", state.InterruptVector)
}
```

## Testing Strategy

1. **Edge Detection Tests**: Verify NMI edge triggering
2. **Level Detection Tests**: Verify IRQ level triggering
3. **Timing Tests**: Verify 7-cycle interrupt latency
4. **Hijacking Tests**: Verify NMI can hijack IRQ
5. **I Flag Tests**: Verify IRQ respects I flag
6. **RTI Tests**: Verify proper return from interrupt

### Test Cases

```go
func TestNMIEdgeDetection(t *testing.T) {
    cpu, bus := setupCPU()
    
    // Set NMI high - no interrupt
    cpu.SetNMI(true)
    runCycles(cpu, 10)
    // Verify no interrupt occurred
    
    // Create falling edge - interrupt should occur
    cpu.SetNMI(false)
    runCycles(cpu, 10)
    // Verify NMI was serviced
}

func TestIRQLevelTriggered(t *testing.T) {
    cpu, bus := setupCPU()
    cpu.setFlag(I, false) // Enable interrupts
    
    // Assert IRQ
    cpu.SetIRQ(true)
    runCycles(cpu, 10)
    // Verify IRQ was serviced
    
    // IRQ line still asserted - should service again after RTI
    // (if I flag is cleared by RTI)
}

func TestNMIHijacksIRQ(t *testing.T) {
    cpu, bus := setupCPU()
    cpu.setFlag(I, false)
    
    // Start IRQ
    cpu.SetIRQ(true)
    runCycles(cpu, 3) // Partial IRQ sequence
    
    // Generate NMI
    cpu.SetNMI(true)
    cpu.SetNMI(false)
    runCycles(cpu, 10)
    
    // Verify PC points to NMI vector, not IRQ vector
}
```

## Success Criteria

- [ ] Interrupt state fields added
- [ ] SetIRQ/SetNMI methods implemented
- [ ] Edge detection for NMI working
- [ ] Level detection for IRQ working
- [ ] Interrupt polling in Clock() implemented
- [ ] 7-cycle interrupt latency accurate
- [ ] NMI hijacking works correctly
- [ ] All interrupt tests pass
- [ ] Backward compatibility maintained
