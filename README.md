# Sixty502 - 6502 CPU Emulator

A comprehensive and accurate 6502 microprocessor emulator written in Go. This library provides a cycle-accurate implementation of the MOS Technology 6502 CPU, including all official instructions, addressing modes, and many unofficial/illegal opcodes.

## Features

### Core CPU Implementation

- **Multiple CPU Variants**: Support for NMOS 6502 (Rev A and Rev B+), CMOS 65C02, and Ricoh 2A03/2A07 (NES) variants
- **Complete 6502 Instruction Set**: All 151 official instructions implemented
- **All Addressing Modes**: Immediate, Zero Page, Absolute, Indexed, Indirect, and Relative addressing
- **Unofficial Opcodes**: Support for many undocumented/illegal 6502 instructions
- **Cycle-Accurate Timing**: Precise cycle counting including page boundary crossing penalties
- **Status Flags**: Full implementation of all processor status flags (N, V, U, B, D, I, Z, C)
- **Variant-Specific Decimal Mode**: Accurate BCD arithmetic with variant-specific behavior
- **Interrupt Handling**: IRQ, NMI, and BRK interrupt support with proper vector handling
- **Hardware Bug Emulation**: Accurate emulation of historical hardware quirks:
  - Indirect JMP page boundary bug (NMOS variants)
  - ROR instruction quirk on Rev A (behaves like ASL)

### Architecture

- **Bus Interface**: Clean separation between CPU and memory through a Bus interface
- **Method-Based Design**: Uses Go method expressions for efficient instruction dispatch
- **Comprehensive Testing**: Extensive test suite covering all instructions and edge cases

### Debugging & Development Tools

- **Disassembler**: Built-in disassembly functionality for code analysis
- **State Inspection**: Methods to examine CPU registers, flags, and execution state
- **Cycle Counting**: Total cycle tracking for performance analysis
- **Instruction Lookup**: Access to the complete instruction table for tooling

## Installation

```bash
go get github.com/drewwalton19216801/sixty502
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/drewwalton19216801/sixty502"
)

// Simple memory implementation
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
    // Create bus and CPU (defaults to NMOS 6502)
    bus := &SimpleBus{}
    cpu := cpu6502.NewCPU(bus)
    
    // Load a simple program: LDA #$42, STA $0200, BRK
    bus.Write(0x8000, 0xA9) // LDA immediate
    bus.Write(0x8001, 0x42) // Value $42
    bus.Write(0x8002, 0x8D) // STA absolute
    bus.Write(0x8003, 0x00) // Low byte of address
    bus.Write(0x8004, 0x02) // High byte of address
    bus.Write(0x8005, 0x00) // BRK
    
    // Set reset vector
    bus.Write(0xFFFC, 0x00) // Reset vector low
    bus.Write(0xFFFD, 0x80) // Reset vector high
    
    // Reset and run
    cpu.Reset()
    
    // Execute until instruction completes
    for cpu.RemainingCycles() > 0 {
        cpu.Clock()
    }
    
    // Inspect CPU state
    state := cpu.GetStateSnapshot()
    fmt.Printf("CPU State: %s\n", state)
    fmt.Printf("Value at $0200: $%02X\n", bus.Read(0x0200))
}
```

## Architecture Overview

### Bus Interface

The CPU communicates with memory through a simple Bus interface:

```go
type Bus interface {
    Read(addr uint16) uint8
    Write(addr uint16, data uint8)
}
```

This design allows for flexible memory implementations, from simple RAM to complex memory-mapped I/O systems.

### Instruction System

Instructions are defined using method expressions for efficient dispatch:

```go
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
```

### Addressing Modes

All 13 addressing modes are implemented:

- **IMP**: Implied (no operand)
- **IMM**: Immediate (#value)
- **ZP0**: Zero Page ($00-FF)
- **ZPX**: Zero Page, X indexed
- **ZPY**: Zero Page, Y indexed
- **REL**: Relative (for branches)
- **ABS**: Absolute ($0000-FFFF)
- **ABX**: Absolute, X indexed
- **ABY**: Absolute, Y indexed
- **IND**: Indirect (for JMP)
- **IZX**: Indirect, X indexed
- **IZY**: Indirect, Y indexed

## Testing

The library includes comprehensive tests covering:

- All official instructions and addressing modes
- Flag behavior and edge cases
- Cycle accuracy including page boundary crossing
- Decimal mode arithmetic
- Interrupt handling
- Unofficial opcodes
- Stack operations
- Branch instructions

Run the test suite:

```bash
go test -v
```

## Examples

The repository includes working examples demonstrating various features:

### Running the Examples

```bash
# Basic example - simple program execution
cd examples/basic
go run main.go

# Memory-mapped I/O example
cd examples/memory-mapped
go run main.go

# WozMon-like example
cd examples/wozmon
go run main.go
```

### Available Examples

- **`examples/basic/`** - Demonstrates basic CPU usage with a simple counting program
  - Shows how to load a program into memory
  - Demonstrates setting up reset vectors
  - Shows state inspection and cycle counting

- **`examples/memory-mapped/`** - Demonstrates memory-mapped I/O
  - Shows ROM/RAM/IO memory separation
  - Demonstrates writing to memory-mapped I/O ports
  - Example outputs "HELLO" via I/O port

- **`examples/wozmon/`** - Demonstrates CPU and memory interaction
  - Implements WozMon-like interface
  - Demonstrates machine level programming
  - Built-in demo counts to 16 and stops

### Creating Your Own Examples

Use the examples as templates for your own programs:

```go
// 1. Implement the Bus interface
type MyBus struct {
    ram [65536]uint8
}

func (b *MyBus) Read(addr uint16) uint8 { return b.ram[addr] }
func (b *MyBus) Write(addr uint16, data uint8) { b.ram[addr] = data }

// 2. Create CPU and load program
bus := &MyBus{}
cpu := cpu6502.NewCPU(bus)

// 3. Set reset vector and reset CPU
bus.Write(0xFFFC, 0x00)
bus.Write(0xFFFD, 0x80)
cpu.Reset()

// 4. Execute
for cpu.RemainingCycles() > 0 {
    cpu.Clock()
}
```

## Advanced Features

### CPU Configuration

Multiple ways to create and configure a CPU instance:

```go
// Simple creation with defaults (NMOS 6502)
cpu := cpu6502.NewCPU(bus)

// Specify variant
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)

// Full configuration
config := cpu6502.DefaultConfig()
config.Variant = cpu6502.VariantCMOS65C02
config.StrictMode = true  // Halt on illegal opcodes
config.EnableDecimalMode = false
cpu := cpu6502.NewCPUWithConfig(bus, config)

// Builder pattern for fluent configuration
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantCMOS65C02).
    WithStrictMode().
    DisableDecimalMode().
    Build()
```

### CPU Variants

The emulator supports multiple 6502 variants with accurate behavior differences:

```go
// NMOS 6502 (Rev B+) - Original chip with documented bugs (ROR works correctly)
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)

// NMOS 6502 Rev A - Early revision with ROR hardware quirk
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502RevA)

// CMOS 65C02 - Enhanced version with bug fixes
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)

// Ricoh 2A03 - NES/Famicom CPU (no decimal mode)
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)

// Ricoh 2A07 - PAL NES CPU (no decimal mode)
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A07)

// Check variant capabilities
if cpu.Variant().SupportsDecimalMode() {
    // Decimal mode available
}
if cpu.Variant().HasIndirectJMPBug() {
    // Has page boundary bug in JMP ($xxFF)
}
if cpu.Variant().HasRORQuirk() {
    // ROR behaves like ASL (Rev A only)
}
```

### Decimal Mode

Variant-specific BCD (Binary Coded Decimal) arithmetic:

```go
// Only works on variants that support decimal mode (NMOS, CMOS)
cpu.setFlag(cpu6502.D, true) // Enable decimal mode
// ADC and SBC will now perform BCD arithmetic
// Ricoh variants ignore the D flag and always use binary mode
```

### Disassembly

Built-in disassembler for code analysis:

```go
disassembly := cpu.Disassemble(0x8000, 0x8010)
for addr, line := range disassembly {
    fmt.Printf("$%04X: %s\n", addr, line)
}
```

### State Inspection

Comprehensive state inspection with accessor methods:

```go
// Get complete state snapshot
state := cpu.GetStateSnapshot()
fmt.Println(state)  // Human-readable output

// Access individual state components
fmt.Printf("PC: $%04X\n", state.PC)
fmt.Printf("A: $%02X X: $%02X Y: $%02X\n", state.A, state.X, state.Y)
fmt.Printf("Flags: %s\n", cpu6502.FormatFlags(state.P))

// Use accessor methods
fmt.Printf("Total cycles: %d\n", cpu.TotalCycles())
fmt.Printf("Remaining cycles: %d\n", cpu.RemainingCycles())
fmt.Printf("Current opcode: $%02X\n", cpu.CurrentOpcode())

// Check instruction legality
if cpu.IsIllegalOpcode(0x02) {
    fmt.Println("Opcode $02 is illegal")
}
```

## Advanced Usage

### Custom Memory Mapping

Implement complex memory systems with memory-mapped I/O:

```go
type MemoryMappedBus struct {
    ram [0x8000]uint8  // RAM: $0000-$7FFF
    rom [0x8000]uint8  // ROM: $8000-$FFFF
    ioPort uint8       // Memory-mapped I/O at $6000
}

func (b *MemoryMappedBus) Read(addr uint16) uint8 {
    switch {
    case addr < 0x6000:
        return b.ram[addr]
    case addr == 0x6000:
        // Memory-mapped I/O read
        return b.ioPort
    case addr < 0x8000:
        return b.ram[addr]
    default:
        // ROM area
        return b.rom[addr-0x8000]
    }
}

func (b *MemoryMappedBus) Write(addr uint16, data uint8) {
    switch {
    case addr < 0x6000:
        b.ram[addr] = data
    case addr == 0x6000:
        // Memory-mapped I/O write
        b.ioPort = data
        fmt.Printf("I/O Port write: $%02X\n", data)
    case addr < 0x8000:
        b.ram[addr] = data
    default:
        // ROM writes are ignored
    }
}
```

### Interrupt Handling

Handle IRQ and NMI interrupts with proper timing:

```go
// Level-triggered IRQ
cpu.SetIRQ(true)  // Assert IRQ line
for i := 0; i < 100; i++ {
    cpu.Clock()
}
cpu.SetIRQ(false) // Clear IRQ line

// Edge-triggered NMI (falling edge)
cpu.SetNMI(true)   // Set NMI line high
cpu.SetNMI(false)  // Falling edge triggers NMI

// Check for pending interrupts
if cpu.HasPendingInterrupt() {
    fmt.Println("Interrupt pending")
}

// Set interrupt vectors in memory
bus.Write(0xFFFA, 0x00) // NMI vector low
bus.Write(0xFFFB, 0xF0) // NMI vector high -> $F000
bus.Write(0xFFFE, 0x00) // IRQ vector low
bus.Write(0xFFFF, 0xF2) // IRQ vector high -> $F200
```

### Error Handling

Configure how the CPU handles errors:

```go
// Strict mode - halt on illegal opcodes
cpu := cpu6502.NewBuilder(bus).
    WithStrictMode().
    Build()

for {
    if err := cpu.Clock(); err != nil {
        fmt.Printf("Execution halted: %v\n", err)
        break
    }
}

// Custom error handler
type CustomErrorHandler struct{}

func (h *CustomErrorHandler) HandleError(err *cpu6502.CPUError) error {
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        fmt.Printf("Illegal opcode $%02X at $%04X\n", err.Opcode, err.PC)
        return nil // Continue execution
    default:
        return err // Halt on other errors
    }
}

cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&CustomErrorHandler{}).
    Build()
```

### Performance Monitoring

Track execution statistics and optimize performance:

```go
// Track execution time
startCycles := cpu.TotalCycles()
runProgram(cpu)
endCycles := cpu.TotalCycles()
fmt.Printf("Executed %d cycles\n", endCycles - startCycles)

// Monitor instruction cache performance
hits, misses, hitRate := cpu.InstructionCacheStats()
fmt.Printf("Cache: %.2f%% hit rate (%d hits, %d misses)\n",
    hitRate*100, hits, misses)

// Invalidate cache after self-modifying code
bus.Write(0x8000, 0xEA) // Modify code
cpu.InvalidateInstructionCache()

// Profile instruction execution
type InstructionProfiler struct {
    counts map[string]int
}

func (p *InstructionProfiler) Profile(cpu *cpu6502.CPU) {
    instr := cpu.GetCurrentInstruction()
    if instr != nil {
        p.counts[instr.Name]++
    }
}
```

## Performance

The emulator is designed for accuracy over raw speed, but still provides good performance:

- Cycle-accurate timing
- Efficient instruction dispatch using method expressions
- Minimal memory allocations during execution
- Suitable for real-time emulation of 6502-based systems

### Instruction Cache

An optional instruction cache optimizes repeated instruction fetches in tight loops:

```go
// Cache is enabled by default
cpu := cpu6502.NewCPU(bus)

// Disable cache via configuration
config := cpu6502.DefaultConfig()
config.EnableInstructionCache = false
cpu := cpu6502.NewCPUWithConfig(bus, config)

// Or via builder
cpu := cpu6502.NewBuilder(bus).
    DisableInstructionCache().
    Build()

// Runtime control
cpu.DisableInstructionCache()
cpu.EnableInstructionCache()
cpu.InvalidateInstructionCache() // Clear cache after self-modifying code

// Get cache statistics
hits, misses, hitRate := cpu.InstructionCacheStats()
fmt.Printf("Cache hit rate: %.2f%%\n", hitRate*100)
```

The cache provides:

- **High hit rates** on tight loops (90%+ typical)
- **Direct-mapped design** with 256 entries
- **Transparent operation** - no behavioral changes
- **Statistics tracking** for performance analysis
- **Invalidation support** for self-modifying code

## Compatibility

This emulator accurately emulates multiple 6502 variants:

- **NMOS 6502 (Rev B+)** - Original MOS Technology chip (Apple II, Commodore 64, Atari 8-bit)
  - Supports decimal mode with NMOS-specific N/V flag behavior
  - Includes the indirect JMP page boundary bug
  - ROR instruction works correctly
- **NMOS 6502 Rev A** - Early revision (rare in production systems)
  - Supports decimal mode with NMOS-specific N/V flag behavior
  - Includes the indirect JMP page boundary bug
  - **ROR instruction hardware quirk**: ROR behaves like ASL (shifts left, doesn't update carry)
- **CMOS 65C02** - Enhanced Western Design Center version (Apple IIc, IIe enhanced)
  - Supports decimal mode with improved N/V flag behavior
  - Fixes the indirect JMP page boundary bug
  - All instructions work correctly
- **Ricoh 2A03** - NES/Famicom CPU (NTSC)
  - Decimal mode disabled (D flag ignored)
  - Includes the indirect JMP page boundary bug
  - ROR instruction works correctly
- **Ricoh 2A07** - PAL NES CPU
  - Decimal mode disabled (D flag ignored)
  - Includes the indirect JMP page boundary bug
  - ROR instruction works correctly

## Contributing

Contributions are welcome! Areas for improvement include:

- Additional unofficial opcode implementations
- Performance optimizations
- More comprehensive test cases
- Documentation improvements
- Example programs and demos

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This emulator is based on extensive research of the 6502 architecture and behavior. Special thanks to the 6502 community for documentation and testing resources.
