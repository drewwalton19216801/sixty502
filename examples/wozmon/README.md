# 6502 Emulator with WozMon Interface

A command-line 6502 microprocessor emulator featuring a WozMon-like interface for interactive memory examination and program execution. Built using the [sixty502](https://github.com/drewwalton19216801/sixty502) CPU core.

## Features

- Full 6502 CPU emulation with 64KB RAM
- Classic WozMon command interface
- Pre-loaded example program
- Interactive REPL for memory examination and manipulation
- CPU state inspection

## Usage

Run the emulator:

```bash
go run main.go
```

## WozMon Commands

The emulator supports the following commands (addresses and values in hexadecimal):

| Command | Description | Example |
|---------|-------------|---------|
| `AAAA` | Examine memory at address AAAA | `8000` |
| `AAAA: DD` | Deposit byte DD at address AAAA | `0200: FF` |
| `AAAA.BBBB` | Display memory range from AAAA to BBBB | `8000.8010` |
| `AAAAR` | Run/execute from address AAAA | `8000R` |
| `RESET` | Reset the CPU to initial state | `RESET` |
| `STATUS` | Display current CPU state | `STATUS` |
| `HELP` | Show available commands | `HELP` |
| `QUIT` or `EXIT` | Exit the emulator | `QUIT` |

## Example Session

```text
6502 Emulator with WozMon Interface
====================================
Type 'HELP' for available commands

> 8000
8000: A9

> 8000.800C
8000: A9 00 18 69 01 8D 00 02
8008: C9 10 D0 F7 00

> 8000R
Running from $8000...
Program halted at $800D (BRK)
PC:800D A:10 X:00 Y:00 SP:FF P:30 [nv-bdIZc]
Total Cycles: 1234

> 0200
0200: 10

> STATUS
PC:800D A:10 X:00 Y:00 SP:FF P:30 [nv-bdIZc]
Total Cycles: 1234

> QUIT
Goodbye!
```

## Pre-loaded Example Program

A simple counter program is pre-loaded at address `$8000`:

```assembly
      LDA #$00      ; Load 0 into accumulator
      CLC           ; Clear carry flag
LOOP: ADC #$01      ; Add 1 to accumulator
      STA $0200     ; Store result at $0200
      CMP #$10      ; Compare with 16
      BNE LOOP      ; Loop if not equal to 16
      BRK           ; Break/halt
```

This program counts from 0 to 16 and stores the final result at memory address `$0200`.

### Running the Example

```text
> 8000R           # Run the program
> 0200            # Check the result (should be 10 hex = 16 decimal)
```

## Memory Map

```text
0x0000-0x00FF: Zero Page
0x0100-0x01FF: Stack
0x0200-0x7FFF: General RAM
0x8000-0xFFFF: Program area
0xFFFC-0xFFFD: Reset vector (points to $8000)
0xFFFE-0xFFFF: IRQ/BRK vector
```

## CPU State Display

The `STATUS` command shows:

- **PC**: Program Counter (current instruction address)
- **A**: Accumulator register
- **X**: X index register
- **Y**: Y index register
- **SP**: Stack Pointer
- **P**: Processor status flags (hex value)
- **Flags**: Status flags in readable format
  - N = Negative
  - V = Overflow
  - B = Break
  - D = Decimal mode
  - I = Interrupt disable
  - Z = Zero
  - C = Carry

Uppercase letters indicate the flag is set, lowercase indicates cleared.

## Writing Your Own Programs

You can write programs directly into memory using the deposit command:

```text
> 0300: A9        # LDA #$42
> 0301: 42
> 0302: 8D        # STA $0400
> 0303: 00
> 0304: 04
> 0305: 00        # BRK
> 0300R           # Run from $0300
> 0400            # Check result
```

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed information about the emulator's design and implementation.

## Dependencies

- [sixty502](https://github.com/drewwalton19216801/sixty502) - Cycle-accurate 6502 CPU emulator

## License

This project uses the sixty502 CPU core. Please refer to the sixty502 project for its licensing terms.
