# Quick Start Guide

## Running

```bash
# Run the emulator
go run main.go
```

## First Steps

Once the emulator starts, try these commands:

### 1. View the Help

```text
> HELP
```

### 2. Examine the Pre-loaded Program

```text
> 8000.800C
```

This displays the example program loaded at address $8000.

### 3. Run the Example Program

```text
> 8000R
```

This runs the counter program that counts from 0 to 16.

### 4. Check the Result

```text
> 0200
```

You should see `0200: 10` (16 in decimal).

### 5. View CPU State

```text
> STATUS
```

Shows all CPU registers and flags.

## Writing a Simple Program

Let's write a program that loads the value 42 (hex) into the accumulator and stores it at address $0400:

```text
> 0300: A9     # LDA #$42 (opcode)
> 0301: 42     # Immediate value
> 0302: 8D     # STA $0400 (opcode)
> 0303: 00     # Low byte of address
> 0304: 04     # High byte of address
> 0305: 00     # BRK (halt)
```

Now run it:

```text
> 0300R
```

Check the result:

```text
> 0400
```

You should see `0400: 42`.

## Common 6502 Opcodes

Here are some useful opcodes for writing programs:

| Opcode | Instruction | Description |
|--------|-------------|-------------|
| `A9 nn` | LDA #$nn | Load accumulator with immediate value |
| `8D nn nn` | STA $nnnn | Store accumulator at address |
| `AD nn nn` | LDA $nnnn | Load accumulator from address |
| `69 nn` | ADC #$nn | Add with carry (immediate) |
| `E9 nn` | SBC #$nn | Subtract with carry (immediate) |
| `C9 nn` | CMP #$nn | Compare accumulator with immediate |
| `D0 nn` | BNE $nn | Branch if not equal (relative) |
| `F0 nn` | BEQ $nn | Branch if equal (relative) |
| `4C nn nn` | JMP $nnnn | Jump to address |
| `18` | CLC | Clear carry flag |
| `38` | SEC | Set carry flag |
| `EA` | NOP | No operation |
| `00` | BRK | Break/halt |

## Tips

1. **Addresses are in hexadecimal** - No need for `0x` prefix
2. **Use uppercase** - Commands are case-insensitive but uppercase is clearer
3. **BRK to halt** - Always end your programs with `00` (BRK) to stop execution
4. **Check STATUS** - Use `STATUS` to see what happened after running a program
5. **RESET to start over** - Use `RESET` to return the CPU to initial state

## Example: Fibonacci Sequence

Here's a more complex example that calculates Fibonacci numbers:

```text
> 0400: A9     # LDA #$00 (start with 0)
> 0401: 00
> 0402: 85     # STA $10 (store in zero page)
> 0403: 10
> 0404: A9     # LDA #$01 (next is 1)
> 0405: 01
> 0406: 85     # STA $11
> 0407: 11
> 0408: A5     # LDA $10 (load first number)
> 0409: 10
> 040A: 65     # ADC $11 (add second number)
> 040B: 11
> 040C: 85     # STA $12 (store result)
> 040D: 12
> 040E: A5     # LDA $11 (shift: second becomes first)
> 040F: 11
> 0410: 85     # STA $10
> 0411: 10
> 0412: A5     # LDA $12 (result becomes second)
> 0413: 12
> 0414: 85     # STA $11
> 0415: 11
> 0416: 00     # BRK
```

Run it:

```text
> 0400R
```

Check the results in zero page:

```text
> 0010.0012
```

## Troubleshooting

- **"Invalid hex value"** - Make sure you're using valid hexadecimal (0-9, A-F)
- **Program doesn't stop** - Make sure you included a BRK (`00`) instruction
- **Wrong results** - Use `STATUS` to check CPU state and verify your program logic
- **Need to start over** - Use `RESET` to return to initial state

## Exit

When you're done:

```text
> QUIT
```

Enjoy exploring 6502 assembly programming!
