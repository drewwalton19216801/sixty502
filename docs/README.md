# sixty502 Documentation

Complete documentation for the sixty502 MOS Technology 6502 CPU emulator.

## Overview

sixty502 is a cycle-accurate emulator for the MOS Technology 6502 microprocessor and its variants, written in Go. It provides accurate emulation of the 6502's instruction set, timing, and variant-specific behaviors.

## Quick Start

```go
package main

import (
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
    
    // Load program
    bus.Write(0x8000, 0xA9) // LDA #$42
    bus.Write(0x8001, 0x42)
    bus.Write(0x8002, 0x00) // BRK
    
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
    }
}
```

## Documentation Index

### Core Documentation

- **[API Reference](api-reference.md)** - Complete public API documentation
  - CPU creation and configuration
  - Execution methods
  - State inspection
  - Interrupt control
  - Debug utilities

- **[Configuration Guide](configuration.md)** - Configuration options and patterns
  - CPU variants
  - Error handlers
  - Performance options
  - Builder pattern
  - Best practices

- **[Addressing Modes](addressing-modes.md)** - 6502 addressing mode reference
  - All 13 addressing modes
  - Syntax and examples
  - Page boundary crossing
  - Performance characteristics

- **[Instruction Set](instruction-set.md)** - Complete instruction reference
  - All 151 official instructions
  - Illegal/unofficial opcodes
  - Cycle counts
  - Flag effects
  - Usage examples

- **[Error Handling](error-handling.md)** - Error handling strategies
  - Error types
  - Built-in handlers
  - Custom handlers
  - Error patterns
  - Debugging techniques

### System-Specific Documentation

- **[CPU Variants](compatibility.md)** - Variant compatibility matrix
  - NMOS 6502 (original)
  - NMOS 6502 Rev A (ROR quirk)
  - CMOS 65C02 (enhanced)
  - Ricoh 2A03 (NES NTSC)
  - Ricoh 2A07 (NES PAL)
  - Behavioral differences
  - System compatibility

- **[Performance Guide](performance.md)** - Optimization strategies
  - Benchmarking results
  - Instruction cache
  - Bus optimization
  - Profiling techniques
  - Performance comparison

- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions
  - CPU stuck/hanging
  - Arithmetic issues
  - Interrupt problems
  - Memory access issues
  - Flag behavior
  - Debugging techniques

## Features

### Accurate Emulation

- **Cycle-accurate timing** - Precise cycle counts for all instructions
- **Page boundary crossing** - Correct timing for page crosses
- **Variant support** - Multiple CPU variants with accurate behavior
- **Decimal mode** - Full BCD arithmetic support
- **Interrupts** - IRQ, NMI, and BRK handling

### Multiple Variants

- **NMOS 6502** - Original MOS Technology 6502
- **NMOS 6502 Rev A** - Early revision with ROR hardware quirk
- **CMOS 65C02** - Enhanced CMOS version with bug fixes
- **Ricoh 2A03** - Nintendo Entertainment System (NTSC)
- **Ricoh 2A07** - Nintendo Entertainment System (PAL)

### Performance Features

- **Instruction cache** - 5-15% performance improvement
- **Efficient bus interface** - Minimal overhead
- **Optimized instruction dispatch** - Fast execution

### Developer-Friendly

- **Comprehensive API** - Well-documented public interface
- **Flexible configuration** - Builder pattern and config structs
- **Error handling** - Customizable error handlers
- **Debug tools** - Disassembler, state inspection, breakpoints
- **Testing support** - Easy to test and verify behavior

## Architecture

### Core Components

```text
┌─────────────────────────────────────────────────────────┐
│                         CPU                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │ Registers: A, X, Y, SP, PC, P                    │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │ Instruction Decoder & Executor                   │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │ Instruction Cache (optional)                     │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │ Interrupt Controller                             │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Bus Interface
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Memory Bus                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │
│  │   RAM    │  │   ROM    │  │  Memory-Mapped I/O   │ │
│  └──────────┘  └──────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Execution Flow

```text
1. Clock() called
   ↓
2. Check for interrupts (NMI, IRQ)
   ↓
3. Fetch opcode from PC
   ↓
4. Lookup instruction (cache or table)
   ↓
5. Execute addressing mode
   ↓
6. Execute instruction operation
   ↓
7. Update cycle counter
   ↓
8. Repeat
```

## Usage Examples

### Basic Execution

```go
cpu := cpu6502.NewCPU(bus)
cpu.Reset()

for {
    if err := cpu.Clock(); err != nil {
        break
    }
}
```

### With Variant Selection

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
cpu.Reset()
```

### Using Builder Pattern

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantRicoh2A03).
    WithStrictMode().
    DisableDecimalMode().
    Build()
```

### With Custom Error Handler

```go
type MyHandler struct{}

func (h *MyHandler) HandleError(err *cpu6502.CPUError) error {
    log.Printf("Error: %v", err)
    return nil // Continue
}

cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&MyHandler{}).
    Build()
```

### Interrupt Handling

```go
// Trigger NMI
cpu.SetNMI(true)
cpu.SetNMI(false) // Falling edge

// Assert IRQ
cpu.SetIRQ(true)

// Check for pending interrupts
if cpu.HasPendingInterrupt() {
    fmt.Println("Interrupt pending")
}
```

### State Inspection

```go
// Get current state
state := cpu.GetStateSnapshot()
fmt.Printf("PC: $%04X, A: $%02X\n", state.PC, state.A)

// Get formatted state
fmt.Println(cpu.GetState())

// Check cycles
fmt.Printf("Total cycles: %d\n", cpu.TotalCycles())
```

### Disassembly

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

## System Emulation Examples

### Apple II

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)
// Apple II runs at ~1.023 MHz
```

### Commodore 64

```go
cpu := cpu6502.NewCPU(bus) // Default NMOS 6502
// C64 runs at ~1.023 MHz (NTSC) or ~0.985 MHz (PAL)
```

### Nintendo Entertainment System

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
// NES runs at ~1.789773 MHz (NTSC)
```

### Apple IIc

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
// Apple IIc runs at ~1.023 MHz
```

## Testing

The emulator includes comprehensive test coverage:

```bash
# Run all tests
go test -v

# Run with coverage
go test -cover

# Run benchmarks
go test -bench=.

# Run specific test
go test -run TestCPUBasic
```

## Performance

Typical performance on modern hardware:

- **~300-500M cycles/second** - Mixed code
- **~350M cycles/second** - With instruction cache
- **5-15% improvement** - Cache hit rate dependent

See [Performance Guide](performance.md) for optimization strategies.

## Contributing

When contributing to the documentation:

1. Follow the existing structure and style
2. Include code examples for all features
3. Link to related documentation
4. Test all code examples
5. Update the index when adding new docs

## Documentation Style Guide

### Code Examples

- Use complete, runnable examples
- Include necessary imports
- Show expected output where relevant
- Use realistic variable names

### Links

- Use relative links between docs
- Link to source code with line numbers: `[Type](file.go:line)`
- Link to related documentation

### Formatting

- Use tables for structured data
- Use code blocks with language hints
- Use bold for emphasis, italic for terms
- Use bullet points for lists

## Getting Help

1. **Check the documentation** - Most questions are answered here
2. **Review examples** - See `examples/` directory
3. **Run tests** - Verify emulator behavior
4. **File an issue** - Report bugs or request features

## Additional Resources

### External References

- [6502.org](http://www.6502.org/) - Official 6502 documentation
- [Visual 6502](http://visual6502.org/) - Visual transistor simulation
- [6502 Instruction Reference](http://www.6502.org/tutorials/6502opcodes.html)
- [Nesdev Wiki](https://wiki.nesdev.com/) - NES/Ricoh documentation
- [WDC 65C02 Datasheet](https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf)

### Books

- "Programming the 6502" by Rodnay Zaks
- "6502 Assembly Language Programming" by Lance Leventhal
- "The Art of Assembly Language" (6502 sections)

## License

See LICENSE file for details.

## Version History

See CHANGELOG.md for version history and release notes.
