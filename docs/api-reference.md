# API Reference

This document provides a comprehensive reference for the sixty502 public API.

## Package Overview

```go
import "github.com/yourusername/sixty502"
```

The `cpu6502` package provides a cycle-accurate emulator for the MOS Technology 6502 microprocessor and its variants.

## Core Types

### CPU

The main CPU emulator type that executes 6502 instructions.

```go
type CPU struct {
    // Public registers (direct access allowed)
    A  uint8  // Accumulator
    X  uint8  // X Index Register
    Y  uint8  // Y Index Register
    SP uint8  // Stack Pointer (relative to $0100)
    PC uint16 // Program Counter
    P  Flags  // Processor Status Register
}
```

**Note**: While registers are public for direct access, most operations should use the provided methods for proper flag handling and cycle-accurate behavior.

### Bus Interface

The Bus interface defines memory access for the CPU.

```go
type Bus interface {
    Read(addr uint16) uint8
    Write(addr uint16, data uint8)
}
```

Implementations must handle all 64KB addresses (0x0000-0xFFFF).

**Example Implementation**:

```go
type SimpleBus struct {
    ram [65536]uint8
}

func (b *SimpleBus) Read(addr uint16) uint8 {
    return b.ram[addr]
}

func (b *SimpleBus) Write(addr uint16, data uint8) {
    b.ram[addr] = data
}
```

### Flags

Processor status flags.

```go
type Flags uint8

const (
    C Flags = 1 << 0 // Carry
    Z Flags = 1 << 1 // Zero
    I Flags = 1 << 2 // Interrupt Disable
    D Flags = 1 << 3 // Decimal Mode
    B Flags = 1 << 4 // Break Command
    U Flags = 1 << 5 // Unused (always 1)
    V Flags = 1 << 6 // Overflow
    N Flags = 1 << 7 // Negative
)
```

### CPUVariant

Represents different 6502 processor variants.

```go
type CPUVariant int

const (
    VariantNMOS6502      // Original NMOS 6502 (Rev B+)
    VariantNMOS6502RevA  // Early NMOS 6502 Rev A (ROR quirk)
    VariantCMOS65C02     // CMOS 65C02 (enhanced)
    VariantRicoh2A03     // NES/Famicom CPU (NTSC)
    VariantRicoh2A07     // PAL NES CPU
)
```

**Methods**:

```go
func (v CPUVariant) String() string
func (v CPUVariant) SupportsDecimalMode() bool
func (v CPUVariant) HasIndirectJMPBug() bool
func (v CPUVariant) HasRORQuirk() bool
```

## CPU Creation

### NewCPU

Creates a new CPU with default configuration.

```go
func NewCPU(bus Bus) *CPU
```

**Default Configuration**:

- Variant: NMOS 6502
- Error Handler: Logging (continues on errors)
- Decimal Mode: Enabled
- Instruction Cache: Enabled

**Example**:

```go
bus := &SimpleBus{}
cpu := cpu6502.NewCPU(bus)
cpu.Reset()
```

### NewCPUWithVariant

Creates a CPU with a specific variant.

```go
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU
```

**Example**:

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
```

### NewCPUWithConfig

Creates a CPU with full configuration control.

```go
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU
```

**Example**:

```go
config := cpu6502.DefaultConfig()
config.Variant = cpu6502.VariantRicoh2A03
config.StrictMode = true
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

### Builder Pattern

Fluent interface for CPU configuration.

```go
func NewBuilder(bus Bus) *CPUBuilder
```

**Methods**:

- `WithVariant(variant CPUVariant) *CPUBuilder`
- `WithStrictMode() *CPUBuilder`
- `WithErrorHandler(handler ErrorHandler) *CPUBuilder`
- `DisableDecimalMode() *CPUBuilder`
- `DisableInstructionCache() *CPUBuilder`
- `Build() *CPU`

**Example**:

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantCMOS65C02).
    WithStrictMode().
    DisableDecimalMode().
    Build()
```

## Core Execution Methods

### Reset

Initializes the CPU to power-on state.

```go
func (c *CPU) Reset()
```

- Clears all registers
- Sets SP to $FD
- Sets P to U | I
- Loads PC from reset vector ($FFFC/FD)
- Takes 8 cycles

**Example**:

```go
bus.Write(0xFFFC, 0x00) // Reset vector low byte
bus.Write(0xFFFD, 0x80) // Reset vector high byte
cpu.Reset()             // PC now = $8000
```

### Clock

Executes one clock cycle.

```go
func (c *CPU) Clock() error
```

Call repeatedly to execute instructions. Each instruction takes multiple cycles.

**Returns**: Error if unrecoverable error occurs (e.g., illegal opcode in strict mode).

**Example**:

```go
for {
    if err := cpu.Clock(); err != nil {
        log.Printf("CPU error: %v", err)
        break
    }
}
```

## State Inspection

### RemainingCycles

Returns cycles remaining for current instruction.

```go
func (c *CPU) RemainingCycles() uint8
```

Returns 0 if ready for next instruction.

**Example**:

```go
// Execute one complete instruction
for cpu.RemainingCycles() > 0 {
    cpu.Clock()
}
```

### TotalCycles

Returns total cycles executed since creation.

```go
func (c *CPU) TotalCycles() uint64
```

**Example**:

```go
start := cpu.TotalCycles()
// ... execute code ...
elapsed := cpu.TotalCycles() - start
fmt.Printf("Executed %d cycles\n", elapsed)
```

### CurrentOpcode

Returns the current opcode being executed.

```go
func (c *CPU) CurrentOpcode() uint8
```

### GetStateSnapshot

Captures complete CPU state.

```go
func (c *CPU) GetStateSnapshot() State
```

Returns a [`State`](types.go:179) struct containing all registers, cycle counters, and current instruction.

**Example**:

```go
state := cpu.GetStateSnapshot()
fmt.Printf("PC: $%04X, A: $%02X\n", state.PC, state.A)
```

### GetState

Returns formatted string representation of CPU state.

```go
func (c *CPU) GetState() string
```

Format: `PC:XXXX A:XX X:XX Y:XX P:XX[FLAGS] SP:XX CYC:NNNN (INSTR $XX)`

**Example Output**:

```text
PC:8000 A:42 X:10 Y:20 P:24[..U.D...] SP:FD CYC:1234 (LDA $A9)
```

### Variant

Returns the CPU variant.

```go
func (c *CPU) Variant() CPUVariant
```

## Interrupt Control

### SetIRQ

Sets the IRQ (Interrupt Request) line state.

```go
func (c *CPU) SetIRQ(asserted bool)
```

IRQ is level-triggered and maskable by the I flag.

**Example**:

```go
// Assert IRQ
cpu.SetIRQ(true)

// Later, clear IRQ
cpu.SetIRQ(false)
```

### SetNMI

Sets the NMI (Non-Maskable Interrupt) line state.

```go
func (c *CPU) SetNMI(asserted bool)
```

NMI is edge-triggered (falling edge: high→low).

**Example**:

```go
// Trigger NMI with falling edge
cpu.SetNMI(true)  // Set high
cpu.SetNMI(false) // Set low - triggers NMI
```

### HasPendingInterrupt

Checks if any interrupt is pending.

```go
func (c *CPU) HasPendingInterrupt() bool
```

Returns true if NMI is pending or IRQ is asserted with I flag clear.

## Instruction Cache Control

### InvalidateInstructionCache

Clears the instruction cache.

```go
func (c *CPU) InvalidateInstructionCache()
```

Call after self-modifying code or loading new programs.

**Example**:

```go
// Load program
for i, b := range program {
    bus.Write(0x8000+uint16(i), b)
}
cpu.InvalidateInstructionCache()
```

### InstructionCacheStats

Returns cache performance statistics.

```go
func (c *CPU) InstructionCacheStats() (hits, misses uint64, hitRate float64)
```

**Example**:

```go
hits, misses, rate := cpu.InstructionCacheStats()
fmt.Printf("Cache: %d hits, %d misses, %.1f%% hit rate\n",
    hits, misses, rate*100)
```

### EnableInstructionCache / DisableInstructionCache

Controls instruction cache.

```go
func (c *CPU) EnableInstructionCache()
func (c *CPU) DisableInstructionCache()
```

## Debug and Disassembly

### Disassemble

Disassembles instructions in a memory range.

```go
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string
```

Returns map of addresses to disassembled instruction strings.

**Example**:

```go
disasm := cpu.Disassemble(0x8000, 0x8010)
for addr := uint16(0x8000); addr <= 0x8010; {
    if instr, ok := disasm[addr]; ok {
        fmt.Printf("$%04X: %s\n", addr, instr)
        opcode := bus.Read(addr)
        addr += uint16(cpu.LookupInstruction(opcode).Length)
    }
}
```

### LookupInstruction

Returns instruction definition for an opcode.

```go
func (c *CPU) LookupInstruction(opcode uint8) Instruction
```

**Example**:

```go
instr := cpu.LookupInstruction(0xA9) // LDA immediate
fmt.Printf("%s takes %d cycles\n", instr.Name, instr.Cycles)
```

### IsIllegalOpcode

Checks if an opcode is illegal/unofficial.

```go
func (c *CPU) IsIllegalOpcode(opcode uint8) bool
```

### LookupTable

Returns the complete instruction lookup table.

```go
func (c *CPU) LookupTable() [256]Instruction
```

**Warning**: Returns a copy; modifications don't affect the CPU.

## Utility Functions

### FormatFlags

Returns human-readable flag representation.

```go
func FormatFlags(p Flags) string
```

Format: "NVUBDIZC" where each letter represents a flag (set) or '.' (clear).

**Example**:

```go
fmt.Println(cpu6502.FormatFlags(cpu.P)) // "N.U.D.Z."
```

## State Types

### State

Snapshot of CPU state.

```go
type State struct {
    A               uint8
    X               uint8
    Y               uint8
    SP              uint8
    PC              uint16
    P               Flags
    Cycles          uint8
    TotalCycles     uint64
    Opcode          uint8
    Instruction     string
    InInterrupt     bool
    InterruptVector uint16
}
```

**Methods**:

```go
func (s State) String() string
```

### Instruction

Instruction definition.

```go
type Instruction struct {
    Name             string
    Operate          func(*CPU) uint8
    AddrMode         func(*CPU) uint8
    AddrModeType     AddrModeType
    Cycles           uint8
    Length           uint8
    Illegal          bool
    PageCrossPenalty bool
}
```

## Error Handling

### CPUError

Error during CPU execution.

```go
type CPUError struct {
    Type    ErrorType
    Opcode  uint8
    PC      uint16
    Message string
}
```

**Methods**:

```go
func (e *CPUError) Error() string
```

### ErrorHandler Interface

```go
type ErrorHandler interface {
    HandleError(err *CPUError) error
}
```

**Built-in Handlers**:

- `StrictErrorHandler`: Halts on any error
- `LoggingErrorHandler`: Logs errors but continues

### LastError

Returns the last error that occurred.

```go
func (c *CPU) LastError() *CPUError
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/sixty502"
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
    // Create bus and CPU
    bus := &SimpleBus{}
    cpu := cpu6502.NewCPU(bus)
    
    // Load program: LDA #$42, STA $0200, BRK
    bus.Write(0x8000, 0xA9) // LDA #$42
    bus.Write(0x8001, 0x42)
    bus.Write(0x8002, 0x8D) // STA $0200
    bus.Write(0x8003, 0x00)
    bus.Write(0x8004, 0x02)
    bus.Write(0x8005, 0x00) // BRK
    
    // Set reset vector
    bus.Write(0xFFFC, 0x00)
    bus.Write(0xFFFD, 0x80)
    
    // Reset and run
    cpu.Reset()
    
    for i := 0; i < 100; i++ {
        if err := cpu.Clock(); err != nil {
            log.Printf("Error: %v", err)
            break
        }
        
        if cpu.RemainingCycles() == 0 {
            fmt.Println(cpu.GetState())
        }
    }
    
    fmt.Printf("Value at $0200: $%02X\n", bus.Read(0x0200))
}
```

## See Also

- [Configuration Guide](configuration.md) - Detailed configuration options
- [Addressing Modes](addressing-modes.md) - 6502 addressing mode reference
- [Instruction Set](instruction-set.md) - Complete instruction reference
- [CPU Variants](compatibility.md) - Variant-specific behavior
