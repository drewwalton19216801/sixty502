# Sixty502 - 6502 CPU Emulator

A comprehensive and accurate 6502 microprocessor emulator written in Go. This library provides a cycle-accurate implementation of the MOS Technology 6502 CPU, including all official instructions, addressing modes, and many unofficial/illegal opcodes.

## Features

### Core CPU Implementation
- **Complete 6502 Instruction Set**: All 151 official instructions implemented
- **All Addressing Modes**: Immediate, Zero Page, Absolute, Indexed, Indirect, and Relative addressing
- **Unofficial Opcodes**: Support for many undocumented/illegal 6502 instructions
- **Cycle-Accurate Timing**: Precise cycle counting including page boundary crossing penalties
- **Status Flags**: Full implementation of all processor status flags (N, V, U, B, D, I, Z, C)
- **Decimal Mode**: BCD arithmetic support for ADC and SBC instructions
- **Interrupt Handling**: IRQ, NMI, and BRK interrupt support with proper vector handling

### Architecture
- **Bus Interface**: Clean separation between CPU and memory through a Bus interface
- **Method-Based Design**: Uses Go method expressions for efficient instruction dispatch
- **Reflection-Based Comparisons**: Advanced function pointer comparison for addressing mode detection
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
    // Create bus and CPU
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
    
    // Execute until BRK
    for cpu.Cycles > 0 {
        cpu.Clock()
    }
    
    fmt.Printf("CPU State: %s\n", cpu.GetState())
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

### Decimal Mode
The emulator supports BCD (Binary Coded Decimal) arithmetic:

```go
cpu.setFlag(cpu6502.D, true) // Enable decimal mode
// ADC and SBC will now perform BCD arithmetic
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
Comprehensive state inspection methods:

```go
fmt.Println(cpu.GetState())           // Human-readable state
fmt.Printf("Flags: %s\n", cpu6502.FormatFlags(cpu.P))
fmt.Printf("Total cycles: %d\n", cpu.TotalCycles())
```

## Performance

The emulator is designed for accuracy over raw speed, but still provides good performance:
- Cycle-accurate timing
- Efficient instruction dispatch using method expressions
- Minimal memory allocations during execution
- Suitable for real-time emulation of 6502-based systems

## Compatibility

This emulator is designed to be compatible with:
- Original MOS Technology 6502
- NES/Famicom CPU (Ricoh 2A03/2A07)
- Apple II series
- Commodore 64
- Atari 8-bit computers
- And other 6502-based systems

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