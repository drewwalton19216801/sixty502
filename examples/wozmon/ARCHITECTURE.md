# 6502 Emulator with WozMon Interface - Architecture

## Overview

A command-line 6502 emulator using the sixty502 CPU core with a WozMon-like interface for interactive memory examination and program execution.

## Components

### 1. Memory Bus (`SimpleBus`)

- 64KB RAM array (0x0000 - 0xFFFF)
- Implements `Bus` interface with `Read(addr uint16) uint8` and `Write(addr uint16, data uint8)`
- Standard 6502 memory map

### 2. CPU Core

- Uses `github.com/drewwalton19216801/sixty502` package
- Default NMOS 6502 variant
- Reset vector at 0xFFFC/0xFFFD
- BRK/IRQ vector at 0xFFFE/0xFFFF

### 3. WozMon Command Parser

Implements classic WozMon commands:

- `AAAA` - Examine memory at address AAAA (hex)
- `AAAA: DD` - Deposit byte DD at address AAAA
- `AAAA.BBBB` - Display memory range from AAAA to BBBB
- `AAAAR` - Run/execute from address AAAA
- `RESET` - Reset the CPU
- `STATUS` - Display CPU state
- `HELP` - Show available commands
- `QUIT` - Exit the emulator

### 4. REPL Loop

- Read user input
- Parse commands
- Execute commands
- Display results
- Handle errors gracefully

## Memory Layout

```text
0x0000-0x00FF: Zero Page
0x0100-0x01FF: Stack
0x0200-0x7FFF: General RAM
0x8000-0xFFFF: Program area
0xFFFC-0xFFFD: Reset vector
0xFFFE-0xFFFF: IRQ/BRK vector
```

## Example Program

A simple counter program will be pre-loaded at 0x8000:

```assembly
LDA #$00      ; Load 0 into accumulator
LOOP:
  CLC         ; Clear carry
  ADC #$01    ; Add 1
  STA $0200   ; Store at 0x0200
  CMP #$10    ; Compare with 16
  BNE LOOP    ; Loop if not equal
  BRK         ; Break/halt
```

## Error Handling

- Invalid hex addresses
- Invalid hex values
- Out of range addresses
- CPU execution errors
- Invalid commands

## User Experience

```text
6502 Emulator with WozMon Interface
Type 'HELP' for available commands

> 8000
8000: A9

> 8000.8010
8000: A9 00 18 69 01 8D 00 02
8008: C9 10 D0 F7 00 00 00 00

> 8000R
Running from $8000...
Program halted at $800D
A:10 X:00 Y:00 SP:FF P:NV-BDIZC

> 0200
0200: 10

> QUIT
