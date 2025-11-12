# Sixty502 - 6502 CPU Emulator

A comprehensive and accurate 6502 microprocessor emulator written in Go. This library provides a cycle-accurate implementation of the MOS Technology 6502 CPU, including all official instructions, addressing modes, and many unofficial/illegal opcodes.

## Features

### Core CPU Implementation

- **Multiple CPU Variants**: Support for NMOS 6502, CMOS 65C02, and Ricoh 2A03/2A07 (NES) variants
- **Complete 6502 Instruction Set**: All 151 official instructions implemented
- **All Addressing Modes**: Immediate, Zero Page, Absolute, Indexed, Indirect, and Relative addressing
- **Unofficial Opcodes**: Support for many undocumented/illegal 6502 instructions
- **Cycle-Accurate Timing**: Precise cycle counting including page boundary crossing penalties
- **Status Flags**: Full implementation of all processor status flags (N, V, U, B, D, I, Z, C)
- **Variant-Specific Decimal Mode**: Accurate BCD arithmetic with variant-specific behavior
- **Interrupt Handling**: IRQ, NMI, and BRK interrupt support with proper vector handling
- **Hardware Bug Emulation**: Accurate emulation of the indirect JMP page boundary bug on NMOS variants

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
    Name     string           // Mnemonic (e.g., "LDA")
    Operate  func(*CPU) uint8 // Instruction logic
    AddrMode func(*CPU) uint8 // Addressing mode calculation
    Cycles   uint8            // Base cycle count
    Length   uint8            // Instruction length in bytes
    Illegal  bool             // Unofficial opcode flag
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
// NMOS 6502 - Original chip with all documented bugs
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)

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
```

### Decimal Mode

Variant-specific BCD (Binary Coded Decimal) arithmetic:

```go
// Only works on variants that support decimal mode (NMOS, CMOS)
cpu.setFlag(cpu6502.D, true) // Enable decimal mode
// ADC and SBC will now perform BCD arithmetic
// Ricoh variants ignore the D flag and always use binary mode
```

### Interrupt Handling

Full interrupt support with proper vector handling:

```go
cpu.InterruptRequest()     // Handle IRQ
cpu.NonMaskableInterrupt() // Handle NMI
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

## Performance

The emulator is designed for accuracy over raw speed, but still provides good performance:

- Cycle-accurate timing
- Efficient instruction dispatch using method expressions
- Minimal memory allocations during execution
- Suitable for real-time emulation of 6502-based systems

## Compatibility

This emulator accurately emulates multiple 6502 variants:

- **NMOS 6502** - Original MOS Technology chip (Apple II, Commodore 64, Atari 8-bit)
  - Supports decimal mode with NMOS-specific N/V flag behavior
  - Includes the indirect JMP page boundary bug
- **CMOS 65C02** - Enhanced Western Design Center version (Apple IIc, IIe enhanced)
  - Supports decimal mode with improved N/V flag behavior
  - Fixes the indirect JMP page boundary bug
- **Ricoh 2A03** - NES/Famicom CPU (NTSC)
  - Decimal mode disabled (D flag ignored)
  - Includes the indirect JMP page boundary bug
- **Ricoh 2A07** - PAL NES CPU
  - Decimal mode disabled (D flag ignored)
  - Includes the indirect JMP page boundary bug

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
