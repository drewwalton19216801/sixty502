# Low Priority: Improve Documentation

## Problem Statement

Current documentation gaps:

1. **No Godoc Comments**: Most exported types lack documentation
2. **Complex Algorithms Unexplained**: Decimal mode, overflow detection lack inline comments
3. **No Architecture Decisions**: No ADR (Architecture Decision Records)
4. **Limited Examples**: README has basic example but lacks advanced usage
5. **No Performance Guide**: No guidance on optimization
6. **Missing Compatibility Notes**: Variant differences not documented

## Proposed Solution

Add comprehensive documentation at all levels.

### Implementation Steps

#### Step 1: Add Godoc Comments to All Exported Types

```go
// Package cpu6502 provides a cycle-accurate emulator for the MOS Technology 6502
// microprocessor and its variants.
//
// The 6502 is an 8-bit microprocessor that was widely used in home computers
// and game consoles during the 1970s and 1980s, including the Apple II,
// Commodore 64, Nintendo Entertainment System, and Atari 2600.
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
// bus := &SimpleBus{}
// cpu := cpu6502.NewCPU(bus)
// cpu.Reset()
// for cpu.RemainingCycles() > 0 {
//     if err := cpu.Clock(); err != nil {
//         log.Fatal(err)
//     }
// }
package cpu6502

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
// cpu := cpu6502.NewCPU(bus)
// cpu.Reset()
// for {
//     if err := cpu.Clock(); err != nil {
//         break
//     }
// }
type CPU struct {
    // ... fields ...
}

// Bus defines the interface for memory access.
//
// Implementations of this interface provide the CPU with access to
// memory and memory-mapped I/O. The interface is intentionally simple
// to allow for flexible implementations.
//
// Example implementation:
//
// type SimpleBus struct {
//     ram [65536]uint8
// }
// 
// func (b *SimpleBus) Read(addr uint16) uint8 {
//     return b.ram[addr]
// }
// 
// func (b *SimpleBus) Write(addr uint16, data uint8) {
//     b.ram[addr] = data
// }
type Bus interface {
    // Read returns the byte at the specified address.
    Read(addr uint16) uint8
    
    // Write stores a byte at the specified address.
    Write(addr uint16, data uint8)
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
// cpu := NewCPU(bus)
// cpu.Reset() // Loads PC from reset vector at $FFFC/FD
//
// For custom configuration, use NewCPUWithConfig instead.
func NewCPU(bus Bus) *CPU {
    // ... implementation ...
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
// for {
//     if err := cpu.Clock(); err != nil {
//         log.Printf("CPU error: %v", err)
//         break
//     }
// }
func (c *CPU) Clock() error {
    // ... implementation ...
}

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
// bus.Write(0xFFFC, 0x00) // Low byte
// bus.Write(0xFFFD, 0x80) // High byte -> PC = $8000
// cpu.Reset()
func (c *CPU) Reset() {
    // ... implementation ...
}
```

#### Step 2: Add Inline Comments for Complex Algorithms

```go
// ADC - Add with Carry
//
// Performs A = A + M + C, where:
//   A = Accumulator
//   M = Memory operand
//   C = Carry flag (0 or 1)
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
//
// Decimal mode algorithm:
//   1. Calculate binary sum for N/V flags: binSum = A + M + C
//   2. Add lower nibbles: low = (A & 0x0F) + (M & 0x0F) + C
//   3. If low > 9, adjust: low += 6, carry to high nibble
//   4. Add upper nibbles: high = (A >> 4) + (M >> 4) + lowCarry
//   5. If high > 9, adjust: high += 6, set C flag
//   6. Combine nibbles: result = (high << 4) | (low & 0x0F)
//
// Example:
//   A = $09, M = $01, C = 0 (decimal mode)
//   Binary: 09 + 01 = 0A (invalid BCD)
//   BCD: low = 9 + 1 = 10 (>9), adjust: 10 + 6 = 16, carry = 1
//        high = 0 + 0 + 1 = 1
//        result = $10 (correct BCD)
func (c *CPU) ADC() uint8 {
    // ... implementation with inline comments ...
}
```

#### Step 3: Create Architecture Decision Records

Create `plandocs/adr/` directory with ADRs:

**ADR-001: Use Method Expressions for Instruction Dispatch**

```markdown
# ADR-001: Use Method Expressions for Instruction Dispatch

## Status
Accepted

## Context
Need efficient way to dispatch instructions without reflection or type assertions.

## Decision
Use Go method expressions (e.g., `(*CPU).LDA`) stored in lookup table.

## Consequences
- Positive: Fast dispatch, type-safe
- Positive: No reflection overhead at runtime
- Negative: Requires reflection for addressing mode comparison (to be fixed)
- Negative: Slightly verbose initialization code
```

**ADR-002: Support Multiple CPU Variants**

```markdown
# ADR-002: Support Multiple CPU Variants

## Status
Proposed

## Context
Different systems use different 6502 variants with distinct behaviors.

## Decision
Add CPUVariant type with variant-specific behavior for:
- Decimal mode support
- Indirect JMP bug
- Flag behavior
- Additional instructions (65C02)

## Consequences
- Positive: Accurate emulation for different systems
- Positive: Single codebase for all variants
- Negative: Increased complexity
- Negative: More test cases needed
```

#### Step 4: Expand README with Advanced Examples

Add sections to [`README.md`](../README.md):

```markdown
## Advanced Usage

### Custom Memory Mapping

```go
type MemoryMappedBus struct {
    ram [0x8000]uint8
    rom [0x8000]uint8
}

func (b *MemoryMappedBus) Read(addr uint16) uint8 {
    if addr < 0x8000 {
        return b.ram[addr]
    }
    return b.rom[addr-0x8000]
}

func (b *MemoryMappedBus) Write(addr uint16, data uint8) {
    if addr < 0x8000 {
        b.ram[addr] = data
    }
    // ROM writes are ignored
}
```

### Interrupt Handling

```go
// Level-triggered IRQ
cpu.SetIRQ(true)
for i := 0; i < 100; i++ {
    cpu.Clock()
}
cpu.SetIRQ(false)

// Edge-triggered NMI
cpu.SetNMI(true)
cpu.SetNMI(false) // Falling edge triggers NMI
```

### CPU Variants

```go
// NES emulation
nesCPU := NewCPUWithVariant(bus, VariantRicoh2A03)

// Apple IIc emulation
appleCPU := NewCPUWithVariant(bus, VariantCMOS65C02)
```

### Error Handling

```go
// Strict mode - halt on illegal opcodes
cpu := NewCPUWithConfig(bus, CPUConfig{
    StrictMode: true,
})

for {
    if err := cpu.Clock(); err != nil {
        fmt.Printf("Execution halted: %v\n", err)
        break
    }
}
```

### Performance Monitoring

```go
// Track execution statistics
startCycles := cpu.TotalCycles()
runProgram(cpu)
endCycles := cpu.TotalCycles()
fmt.Printf("Executed %d cycles\n", endCycles - startCycles)

// Cache statistics
hits, misses, hitRate := cpu.InstructionCacheStats()
fmt.Printf("Cache: %.2f%% hit rate (%d hits, %d misses)\n",
    hitRate*100, hits, misses)
```

## Performance Guide

### Optimization Tips

1. **Enable Instruction Cache**: 5-15% performance improvement
2. **Use Appropriate Variant**: Ricoh 2A03 is faster (no decimal mode)
3. **Batch Clock Calls**: Call Clock() in tight loop
4. **Minimize Bus Overhead**: Optimize Bus implementation
5. **Profile First**: Use Go profiler to find actual bottlenecks

### Benchmarking

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkCPUClock -benchtime=10s

# Compare before/after
go test -bench=. > before.txt
# Make changes
go test -bench=. > after.txt
benchcmp before.txt after.txt
```

```

#### Step 5: Create Compatibility Matrix

```markdown
## Compatibility Matrix

| System | Variant | Decimal Mode | Indirect JMP Bug | Notes |
|--------|---------|--------------|------------------|-------|
| Apple II | NMOS 6502 | ✓ | ✓ | Original 6502 |
| Apple IIc | CMOS 65C02 | ✓ | ✗ | Bug fixed |
| Commodore 64 | NMOS 6502 | ✓ | ✓ | Original 6502 |
| NES/Famicom | Ricoh 2A03 | ✗ | ✓ | No decimal mode |
| Atari 2600 | NMOS 6502 | ✓ | ✓ | Original 6502 |
| BBC Micro | NMOS 6502 | ✓ | ✓ | Original 6502 |

### Variant Differences

#### NMOS 6502 (Original)
- Decimal mode: Supported, N/V flags undefined
- Indirect JMP: Has page boundary bug
- Instructions: 151 official opcodes
- Power: Higher consumption

#### CMOS 65C02
- Decimal mode: Supported, N/V flags defined
- Indirect JMP: Bug fixed
- Instructions: Additional opcodes (BRA, PHX, PHY, etc.)
- Power: Lower consumption
- Timing: Some instructions have different cycle counts

#### Ricoh 2A03 (NES)
- Decimal mode: Disabled (SED/CLD are NOPs)
- Indirect JMP: Has page boundary bug
- Instructions: Same as NMOS 6502
- Additional: Integrated APU (not part of CPU emulation)
```

#### Step 6: Add Code Examples Directory

Create `examples/` directory with working examples:

**examples/basic/main.go**

```go
// Basic CPU usage example
package main

import (
    "fmt"
    "github.com/drewwalton19216801/sixty502"
)

type SimpleBus struct {
    ram [65536]uint8
}

func (b *SimpleBus) Read(addr uint16) uint8 {
    return b.ram[addr]
}

func (b *SimpleBus) Write(addr uint16, data uint8) {
    b.ram[addr] = data
}

func main() {
    bus := &SimpleBus{}
    cpu := cpu6502.NewCPU(bus)
    
    // Load program: Count from 0 to 10
    program := []uint8{
        0xA9, 0x00,       // LDA #$00
        0x69, 0x01,       // ADC #$01  ; Loop
        0xC9, 0x0A,       // CMP #$0A
        0xD0, 0xFA,       // BNE Loop
        0x00,             // BRK
    }
    
    // Load at $8000
    for i, b := range program {
        bus.Write(0x8000+uint16(i), b)
    }
    
    // Set reset vector
    bus.Write(0xFFFC, 0x00)
    bus.Write(0xFFFD, 0x80)
    
    cpu.Reset()
    
    // Execute
    for i := 0; i < 1000; i++ {
        if err := cpu.Clock(); err != nil {
            fmt.Printf("Error: %v\n", err)
            break
        }
        
        if cpu.CurrentOpcode() == 0x00 {
            break
        }
    }
    
    fmt.Printf("Final state: %s\n", cpu.GetState())
    fmt.Printf("Accumulator: $%02X\n", cpu.A)
}
```

**examples/nes/main.go**

```go
// NES CPU emulation example
package main

import (
    "fmt"
    "github.com/drewwalton19216801/sixty502"
)

func main() {
    bus := &NESBus{} // Your NES bus implementation
    cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
    
    // NES-specific initialization
    cpu.Reset()
    
    // Main emulation loop
    for {
        if err := cpu.Clock(); err != nil {
            fmt.Printf("CPU error: %v\n", err)
            break
        }
        
        // Handle NES-specific timing (PPU, APU, etc.)
    }
}
```

#### Step 7: Create Performance Guide

**docs/performance.md**

```markdown
# Performance Guide

## Benchmarking Results

Typical performance on modern hardware (AMD Ryzen 5):

| Operation | Cycles/sec | Instructions/sec |
|-----------|------------|------------------|
| NOP loop | ~500M | ~250M |
| Mixed code | ~300M | ~100M |
| With cache | ~350M | ~120M |

## Optimization Strategies

### 1. Enable Instruction Cache

```go
config := cpu6502.DefaultConfig()
config.EnableInstructionCache = true
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

**Impact**: 5-15% improvement for code with loops

### 2. Optimize Bus Implementation

The Bus interface is called for every memory access. Optimize it:

```go
// Slow
type SlowBus struct {
    ram map[uint16]uint8
}

func (b *SlowBus) Read(addr uint16) uint8 {
    return b.ram[addr] // Map lookup is slow
}

// Fast
type FastBus struct {
    ram [65536]uint8
}

func (b *FastBus) Read(addr uint16) uint8 {
    return b.ram[addr] // Array access is fast
}
```

**Impact**: 20-30% improvement

### 3. Batch Clock Calls

```go
// Slow
for i := 0; i < 1000000; i++ {
    cpu.Clock()
    // Do something every cycle
}

// Fast
for i := 0; i < 1000; i++ {
    for j := 0; j < 1000; j++ {
        cpu.Clock()
    }
    // Do something every 1000 cycles
}
```

### 4. Use Appropriate Variant

```go
// Slower (has decimal mode)
cpu := NewCPUWithVariant(bus, VariantNMOS6502)

// Faster (no decimal mode)
cpu := NewCPUWithVariant(bus, VariantRicoh2A03)
```

**Impact**: 2-5% improvement for Ricoh variant

## Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

```

#### Step 8: Add Troubleshooting Guide

**docs/troubleshooting.md**
```markdown
# Troubleshooting Guide

## Common Issues

### Issue: CPU Appears Stuck

**Symptoms**: Clock() returns but PC doesn't advance

**Causes**:
1. Infinite loop in program
2. Waiting for interrupt that never comes
3. Invalid opcode causing repeated errors

**Solutions**:
```go
// Add timeout
maxCycles := uint64(1000000)
for i := uint64(0); i < maxCycles; i++ {
    if err := cpu.Clock(); err != nil {
        break
    }
}

// Check if stuck
if cpu.TotalCycles() >= maxCycles {
    fmt.Printf("CPU stuck at PC=$%04X, opcode=$%02X\n",
        cpu.PC, cpu.CurrentOpcode())
}
```

### Issue: Incorrect Arithmetic Results

**Symptoms**: ADC/SBC produce wrong results

**Causes**:

1. Decimal mode enabled unexpectedly
2. Carry flag not set correctly
3. Variant doesn't support decimal mode

**Solutions**:

```go
// Check decimal mode
if cpu.P & cpu6502.D != 0 {
    fmt.Println("Decimal mode is enabled")
}

// Use correct variant
cpu := NewCPUWithVariant(bus, VariantRicoh2A03) // No decimal mode
```

### Issue: Interrupts Not Working

**Symptoms**: SetIRQ/SetNMI don't trigger interrupts

**Causes**:

1. I flag is set (IRQ only)
2. No falling edge for NMI
3. Interrupt vectors not set

**Solutions**:

```go
// Check I flag
if cpu.P & cpu6502.I != 0 {
    fmt.Println("Interrupts disabled")
    cpu.P &^= cpu6502.I // Clear I flag
}

// Ensure NMI edge
cpu.SetNMI(true)
cpu.SetNMI(false) // Must create falling edge

// Set vectors
bus.Write(0xFFFA, 0x00) // NMI vector
bus.Write(0xFFFB, 0xF0)
bus.Write(0xFFFE, 0x00) // IRQ vector
bus.Write(0xFFFF, 0xF2)
```

```

## Documentation Checklist

- [ ] Godoc comments on all exported types
- [ ] Godoc comments on all exported functions
- [ ] Inline comments for complex algorithms
- [ ] Architecture Decision Records created
- [ ] README expanded with advanced examples
- [ ] Performance guide created
- [ ] Troubleshooting guide created
- [ ] Compatibility matrix documented
- [ ] Code examples directory created
- [ ] API reference generated

## Success Criteria

- [ ] `go doc` shows comprehensive documentation
- [ ] All exported symbols documented
- [ ] Complex algorithms explained
- [ ] Examples compile and run
- [ ] Performance guide accurate
- [ ] Troubleshooting guide helpful
- [ ] Documentation reviewed and approved
