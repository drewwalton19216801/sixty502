# Troubleshooting Guide

This guide helps diagnose and resolve common issues when using the sixty502 emulator.

## Common Issues

### CPU Appears Stuck

**Symptoms**:

- `Clock()` returns but PC doesn't advance
- Program seems to hang
- No visible progress

**Possible Causes**:

1. **Infinite loop in program**
2. **Waiting for interrupt that never comes**
3. **Invalid opcode causing repeated errors**
4. **Incorrect reset vector**

**Solutions**:

```go
// Add timeout to detect stuck CPU
maxCycles := uint64(1000000)
startCycles := cpu.TotalCycles()

for cpu.TotalCycles() - startCycles < maxCycles {
    if err := cpu.Clock(); err != nil {
        fmt.Printf("Error at PC=$%04X: %v\n", cpu.PC, err)
        break
    }
}

if cpu.TotalCycles() - startCycles >= maxCycles {
    fmt.Printf("CPU appears stuck at PC=$%04X, opcode=$%02X\n",
        cpu.PC, cpu.CurrentOpcode())
    
    // Disassemble around current PC
    disasm := cpu.Disassemble(cpu.PC-5, cpu.PC+5)
    for addr, line := range disasm {
        marker := " "
        if addr == cpu.PC {
            marker = ">"
        }
        fmt.Printf("%s $%04X: %s\n", marker, addr, line)
    }
}
```

**Debug Steps**:

1. Check reset vector is set correctly
2. Verify program is loaded at correct address
3. Use disassembler to inspect code at PC
4. Check for infinite loops (BNE to same address)
5. Verify interrupt vectors if using interrupts

### Incorrect Arithmetic Results

**Symptoms**:

- ADC/SBC produce wrong results
- Unexpected flag values
- BCD arithmetic doesn't work

**Possible Causes**:

1. **Decimal mode enabled unexpectedly**
2. **Carry flag not set correctly**
3. **Variant doesn't support decimal mode**
4. **Overflow flag misunderstood**

**Solutions**:

```go
// Check decimal mode status
if cpu.P & cpu6502.D != 0 {
    fmt.Println("Decimal mode is enabled")
    if !cpu.Variant().SupportsDecimalMode() {
        fmt.Println("Warning: Variant doesn't support decimal mode")
    }
}

// Verify carry flag before ADC/SBC
fmt.Printf("Carry flag: %v\n", cpu.P & cpu6502.C != 0)

// Use correct variant for your system
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)

// Clear decimal mode if not needed
cpu.P &^= cpu6502.D
```

**Debug Steps**:

1. Print CPU state before and after operation
2. Verify carry flag is set/cleared as expected
3. Check if decimal mode should be enabled
4. Test with simple known values
5. Compare with reference implementation

### Interrupts Not Working

**Symptoms**:

- SetIRQ/SetNMI don't trigger interrupts
- IRQ handler never executes
- NMI doesn't fire

**Possible Causes**:

1. **I flag is set (IRQ only)**
2. **No falling edge for NMI**
3. **Interrupt vectors not set**
4. **CPU in middle of instruction**

**Solutions**:

```go
// Check I flag (IRQ only)
if cpu.P & cpu6502.I != 0 {
    fmt.Println("Interrupts disabled (I flag set)")
    cpu.P &^= cpu6502.I // Clear I flag
}

// Ensure NMI edge (must go high then low)
cpu.SetNMI(true)   // Set high
cpu.Clock()        // Let CPU see it
cpu.SetNMI(false)  // Create falling edge

// Verify interrupt vectors are set
nmiLow := bus.Read(0xFFFA)
nmiHigh := bus.Read(0xFFFB)
fmt.Printf("NMI vector: $%02X%02X\n", nmiHigh, nmiLow)

irqLow := bus.Read(0xFFFE)
irqHigh := bus.Read(0xFFFF)
fmt.Printf("IRQ vector: $%02X%02X\n", irqHigh, irqLow)

// Check for pending interrupts
if cpu.HasPendingInterrupt() {
    fmt.Println("Interrupt is pending")
}
```

**Debug Steps**:

1. Verify I flag is clear for IRQ
2. Check interrupt vectors are set
3. Ensure NMI has falling edge
4. Verify interrupt handler code exists
5. Check if CPU is stuck in instruction

### Memory Access Issues

**Symptoms**:

- Reading wrong values
- Writes don't persist
- Unexpected memory contents

**Possible Causes**:

1. **Bus implementation bug**
2. **Address calculation error**
3. **ROM/RAM boundaries wrong**
4. **Memory-mapped I/O conflict**

**Solutions**:

```go
// Add logging to Bus implementation
type DebugBus struct {
    ram [65536]uint8
}

func (b *DebugBus) Read(addr uint16) uint8 {
    value := b.ram[addr]
    fmt.Printf("Read $%02X from $%04X\n", value, addr)
    return value
}

func (b *DebugBus) Write(addr uint16, data uint8) {
    fmt.Printf("Write $%02X to $%04X\n", data, addr)
    b.ram[addr] = data
}

// Verify memory contents
func dumpMemory(bus *Bus, start, end uint16) {
    for addr := start; addr <= end; addr++ {
        if addr % 16 == 0 {
            fmt.Printf("\n$%04X: ", addr)
        }
        fmt.Printf("%02X ", bus.Read(addr))
    }
    fmt.Println()
}
```

**Debug Steps**:

1. Add logging to Bus Read/Write
2. Verify ROM/RAM boundaries
3. Check address calculations
4. Dump memory to verify contents
5. Test Bus implementation separately

### Flag Behavior Issues

**Symptoms**:

- Flags set/cleared unexpectedly
- Branch instructions don't work
- Comparison results wrong

**Possible Causes**:

1. **Misunderstanding flag behavior**
2. **Overflow vs Carry confusion**
3. **Decimal mode affecting flags**
4. **Variant-specific differences**

**Solutions**:

```go
// Print flag state
func printFlags(cpu *cpu6502.CPU) {
    fmt.Printf("Flags: %s\n", cpu6502.FormatFlags(cpu.P))
    fmt.Printf("  N (Negative):  %v\n", cpu.P & cpu6502.N != 0)
    fmt.Printf("  V (Overflow):  %v\n", cpu.P & cpu6502.V != 0)
    fmt.Printf("  D (Decimal):   %v\n", cpu.P & cpu6502.D != 0)
    fmt.Printf("  I (Interrupt): %v\n", cpu.P & cpu6502.I != 0)
    fmt.Printf("  Z (Zero):      %v\n", cpu.P & cpu6502.Z != 0)
    fmt.Printf("  C (Carry):     %v\n", cpu.P & cpu6502.C != 0)
}

// Test flag behavior
func testFlags() {
    bus := &SimpleBus{}
    cpu := cpu6502.NewCPU(bus)
    
    // Test CMP
    cpu.A = 0x50
    bus.Write(0x8000, 0xC9) // CMP #$30
    bus.Write(0x8001, 0x30)
    
    cpu.PC = 0x8000
    cpu.Clock()
    cpu.Clock()
    
    // A >= M, so C should be set
    fmt.Printf("After CMP: C=%v Z=%v N=%v\n",
        cpu.P & cpu6502.C != 0,
        cpu.P & cpu6502.Z != 0,
        cpu.P & cpu6502.N != 0)
}
```

**Flag Reference**:

| Flag | Set When | Cleared When |
|------|----------|--------------|
| N | Result bit 7 = 1 | Result bit 7 = 0 |
| V | Signed overflow | No signed overflow |
| D | SED executed | CLD executed |
| I | SEI or interrupt | CLI |
| Z | Result = 0 | Result ≠ 0 |
| C | Unsigned overflow/no borrow | No overflow/borrow |

### Page Boundary Crossing Issues

**Symptoms**:

- Incorrect cycle counts
- Timing doesn't match expected
- Performance issues

**Possible Causes**:

1. **Not accounting for page cross penalty**
2. **Incorrect addressing mode**
3. **Misunderstanding which instructions add cycles**

**Solutions**:

```go
// Monitor page crossings
type PageCrossMonitor struct {
    bus *SimpleBus
    crosses int
}

func (m *PageCrossMonitor) Read(addr uint16) uint8 {
    return m.bus.Read(addr)
}

func (m *PageCrossMonitor) Write(addr uint16, data uint8) {
    m.bus.Write(addr, data)
}

// Check if instruction will cross page
func willCrossPage(base uint16, index uint8) bool {
    result := base + uint16(index)
    return (base & 0xFF00) != (result & 0xFF00)
}

// Test page crossing
baseAddr := uint16(0x10FF)
index := uint8(0x01)
if willCrossPage(baseAddr, index) {
    fmt.Println("This access will cross page boundary")
}
```

**Instructions that add cycle on page cross**:

- LDA, LDX, LDY (ABX, ABY, IZY modes)
- ADC, SBC (ABX, ABY, IZY modes)
- AND, EOR, ORA (ABX, ABY, IZY modes)
- CMP (ABX, ABY, IZY modes)

**Instructions that DON'T add cycle**:

- STA, STX, STY (always use full cycles)
- ASL, LSR, ROL, ROR, INC, DEC (always use full cycles)

### Illegal Opcode Handling

**Symptoms**:

- Execution halts unexpectedly
- Illegal opcode errors
- Program crashes

**Possible Causes**:

1. **Strict mode enabled**
2. **Program jumped to data**
3. **Incorrect program loading**
4. **Stack corruption**

**Solutions**:

```go
// Use lenient error handling
type LenientErrorHandler struct{}

func (h *LenientErrorHandler) HandleError(err *cpu6502.CPUError) error {
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        fmt.Printf("Warning: Illegal opcode $%02X at $%04X\n",
            err.Opcode, err.PC)
        return nil // Continue execution
    default:
        return err // Halt on other errors
    }
}

cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&LenientErrorHandler{}).
    Build()

// Check if opcode is illegal
if cpu.IsIllegalOpcode(0x02) {
    fmt.Println("Opcode $02 is illegal")
}

// Verify program loaded correctly
func verifyProgram(bus *Bus, addr uint16, expected []byte) bool {
    for i, b := range expected {
        actual := bus.Read(addr + uint16(i))
        if actual != b {
            fmt.Printf("Mismatch at $%04X: expected $%02X, got $%02X\n",
                addr + uint16(i), b, actual)
            return false
        }
    }
    return true
}
```

### Stack Issues

**Symptoms**:

- Stack overflow/underflow
- RTS returns to wrong address
- JSR/RTS don't work correctly

**Possible Causes**:

1. **Stack pointer corruption**
2. **Unbalanced push/pop**
3. **Stack wrapping around**
4. **Incorrect JSR/RTS usage**

**Solutions**:

```go
// Monitor stack pointer
func monitorStack(cpu *cpu6502.CPU) {
    fmt.Printf("Stack pointer: $%02X (stack at $01%02X)\n",
        cpu.SP, cpu.SP)
    
    if cpu.SP < 0x80 {
        fmt.Println("Warning: Stack pointer getting low")
    }
    if cpu.SP == 0xFF {
        fmt.Println("Warning: Stack may have wrapped")
    }
}

// Dump stack contents
func dumpStack(bus *Bus, sp uint8) {
    fmt.Println("Stack contents:")
    for i := uint8(0xFF); i > sp; i-- {
        addr := uint16(0x0100) | uint16(i)
        fmt.Printf("  $01%02X: $%02X\n", i, bus.Read(addr))
    }
}

// Verify stack operations
func testStack() {
    bus := &SimpleBus{}
    cpu := cpu6502.NewCPU(bus)
    
    initialSP := cpu.SP
    
    // PHA
    cpu.A = 0x42
    bus.Write(0x8000, 0x48) // PHA
    cpu.PC = 0x8000
    cpu.Clock()
    cpu.Clock()
    
    fmt.Printf("After PHA: SP=$%02X (was $%02X)\n", cpu.SP, initialSP)
    fmt.Printf("Stack top: $%02X\n", bus.Read(0x0100 | uint16(cpu.SP+1)))
}
```

## Debugging Techniques

### Single-Step Execution

```go
func singleStep(cpu *cpu6502.CPU) {
    fmt.Printf("Before: %s\n", cpu.GetState())
    
    // Execute one instruction
    cycles := cpu.RemainingCycles()
    if cycles == 0 {
        // Fetch next instruction
        cpu.Clock()
        cycles = cpu.RemainingCycles()
    }
    
    // Complete current instruction
    for cpu.RemainingCycles() > 0 {
        cpu.Clock()
    }
    
    fmt.Printf("After:  %s\n", cpu.GetState())
}
```

### Breakpoints

```go
type BreakpointBus struct {
    bus *SimpleBus
    breakpoints map[uint16]bool
}

func (b *BreakpointBus) Read(addr uint16) uint8 {
    if b.breakpoints[addr] {
        fmt.Printf("Breakpoint hit at $%04X\n", addr)
        // Could pause execution here
    }
    return b.bus.Read(addr)
}

func (b *BreakpointBus) SetBreakpoint(addr uint16) {
    b.breakpoints[addr] = true
}
```

### Execution Trace

```go
func traceExecution(cpu *cpu6502.CPU, count int) {
    for i := 0; i < count; i++ {
        state := cpu.GetStateSnapshot()
        instr := cpu.GetCurrentInstruction()
        
        fmt.Printf("$%04X: %-4s A:%02X X:%02X Y:%02X P:%02X SP:%02X\n",
            state.PC, instr.Name,
            state.A, state.X, state.Y,
            uint8(state.P), state.SP)
        
        // Execute one instruction
        for cpu.RemainingCycles() > 0 {
            cpu.Clock()
        }
        cpu.Clock() // Fetch next
    }
}
```

### Memory Watch

```go
type WatchedBus struct {
    bus *SimpleBus
    watches map[uint16]string
}

func (b *WatchedBus) Write(addr uint16, data uint8) {
    if name, ok := b.watches[addr]; ok {
        fmt.Printf("Write to %s ($%04X): $%02X\n", name, addr, data)
    }
    b.bus.Write(addr, data)
}

func (b *WatchedBus) AddWatch(addr uint16, name string) {
    b.watches[addr] = name
}
```

## Getting Help

If you're still experiencing issues:

1. **Check the examples** - See `examples/` directory
2. **Review documentation** - Read `docs/` files
3. **Run tests** - `go test -v` to verify emulator works
4. **Enable logging** - Add debug output to your Bus
5. **Compare with reference** - Test against known-good 6502 code
6. **File an issue** - Report bugs on GitHub with:
   - Minimal reproduction case
   - Expected vs actual behavior
   - CPU state and memory dump
   - Go version and OS

## Reference Resources

- [6502.org](http://www.6502.org/) - Official 6502 documentation
- [Visual 6502](http://visual6502.org/) - Visual transistor simulation
- [6502 Instruction Reference](http://www.6502.org/tutorials/6502opcodes.html) - Opcode reference
- [Nesdev Wiki](https://wiki.nesdev.com/) - Detailed 6502 information
