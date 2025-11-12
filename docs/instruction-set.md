# Instruction Set Reference

Complete reference for all 6502 instructions supported by the sixty502 emulator.

## Instruction Categories

- [Load/Store/Transfer](#loadstore-instructions)
- [Arithmetic](#arithmetic-instructions)
- [Logical](#logical-instructions)
- [Shift/Rotate](#shiftrotate-instructions)
- [Increment/Decrement](#incrementdecrement-instructions)
- [Compare](#compare-instructions)
- [Branch](#branch-instructions)
- [Jump/Call](#jumpcall-instructions)
- [Stack](#stack-instructions)
- [Flag](#flag-instructions)
- [System](#system-instructions)

## Instruction Format

Each instruction entry includes:

- **Mnemonic**: Instruction name
- **Description**: What the instruction does
- **Flags Affected**: Which status flags are modified
- **Addressing Modes**: Available modes with opcodes and cycle counts
- **Examples**: Usage examples

### Flag Legend

- **N**: Negative (bit 7 of result)
- **V**: Overflow (signed overflow)
- **B**: Break (BRK instruction)
- **D**: Decimal mode
- **I**: Interrupt disable
- **Z**: Zero (result is zero)
- **C**: Carry (unsigned overflow/borrow)

## Load/Store Instructions

### LDA - Load Accumulator

Loads a value into the accumulator.

**Flags**: N, Z

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles | Page Cross |
|------|--------|--------|--------|------------|
| IMM | `LDA #$nn` | $A9 | 2 | - |
| ZP0 | `LDA $nn` | $A5 | 3 | - |
| ZPX | `LDA $nn,X` | $B5 | 4 | - |
| ABS | `LDA $nnnn` | $AD | 4 | - |
| ABX | `LDA $nnnn,X` | $BD | 4 | +1 |
| ABY | `LDA $nnnn,Y` | $B9 | 4 | +1 |
| IZX | `LDA ($nn,X)` | $A1 | 6 | - |
| IZY | `LDA ($nn),Y` | $B1 | 5 | +1 |

**Example**:

```assembly
LDA #$42    ; Load literal $42
LDA $80     ; Load from zero page $80
LDA $1234,X ; Load from $1234 + X
```

### LDX - Load X Register

Loads a value into the X register.

**Flags**: N, Z

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles | Page Cross |
|------|--------|--------|--------|------------|
| IMM | `LDX #$nn` | $A2 | 2 | - |
| ZP0 | `LDX $nn` | $A6 | 3 | - |
| ZPY | `LDX $nn,Y` | $B6 | 4 | - |
| ABS | `LDX $nnnn` | $AE | 4 | - |
| ABY | `LDX $nnnn,Y` | $BE | 4 | +1 |

### LDY - Load Y Register

Loads a value into the Y register.

**Flags**: N, Z

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles | Page Cross |
|------|--------|--------|--------|------------|
| IMM | `LDY #$nn` | $A0 | 2 | - |
| ZP0 | `LDY $nn` | $A4 | 3 | - |
| ZPX | `LDY $nn,X` | $B4 | 4 | - |
| ABS | `LDY $nnnn` | $AC | 4 | - |
| ABX | `LDY $nnnn,X` | $BC | 4 | +1 |

### STA - Store Accumulator

Stores the accumulator value to memory.

**Flags**: None

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ZP0 | `STA $nn` | $85 | 3 |
| ZPX | `STA $nn,X` | $95 | 4 |
| ABS | `STA $nnnn` | $8D | 4 |
| ABX | `STA $nnnn,X` | $9D | 5 |
| ABY | `STA $nnnn,Y` | $99 | 5 |
| IZX | `STA ($nn,X)` | $81 | 6 |
| IZY | `STA ($nn),Y` | $91 | 6 |

**Note**: Store instructions always use full cycles (no page cross penalty).

### STX - Store X Register

Stores the X register value to memory.

**Flags**: None

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ZP0 | `STX $nn` | $86 | 3 |
| ZPY | `STX $nn,Y` | $96 | 4 |
| ABS | `STX $nnnn` | $8E | 4 |

### STY - Store Y Register

Stores the Y register value to memory.

**Flags**: None

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ZP0 | `STY $nn` | $84 | 3 |
| ZPX | `STY $nn,X` | $94 | 4 |
| ABS | `STY $nnnn` | $8C | 4 |

### TAX - Transfer A to X

Copies accumulator to X register.

**Opcode**: $AA | **Cycles**: 2 | **Flags**: N, Z

### TAY - Transfer A to Y

Copies accumulator to Y register.

**Opcode**: $A8 | **Cycles**: 2 | **Flags**: N, Z

### TXA - Transfer X to A

Copies X register to accumulator.

**Opcode**: $8A | **Cycles**: 2 | **Flags**: N, Z

### TYA - Transfer Y to A

Copies Y register to accumulator.

**Opcode**: $98 | **Cycles**: 2 | **Flags**: N, Z

### TSX - Transfer SP to X

Copies stack pointer to X register.

**Opcode**: $BA | **Cycles**: 2 | **Flags**: N, Z

### TXS - Transfer X to SP

Copies X register to stack pointer.

**Opcode**: $9A | **Cycles**: 2 | **Flags**: None

## Arithmetic Instructions

### ADC - Add with Carry

Adds memory value and carry flag to accumulator: A = A + M + C

**Flags**: N, V, Z, C

**Binary Mode**: Standard 8-bit addition
**Decimal Mode**: BCD addition (if D flag set and variant supports it)

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles | Page Cross |
|------|--------|--------|--------|------------|
| IMM | `ADC #$nn` | $69 | 2 | - |
| ZP0 | `ADC $nn` | $65 | 3 | - |
| ZPX | `ADC $nn,X` | $75 | 4 | - |
| ABS | `ADC $nnnn` | $6D | 4 | - |
| ABX | `ADC $nnnn,X` | $7D | 4 | +1 |
| ABY | `ADC $nnnn,Y` | $79 | 4 | +1 |
| IZX | `ADC ($nn,X)` | $61 | 6 | - |
| IZY | `ADC ($nn),Y` | $71 | 5 | +1 |

**Example**:

```assembly
CLC         ; Clear carry
LDA #$10    ; A = $10
ADC #$20    ; A = $30, C = 0
ADC #$FF    ; A = $2F, C = 1 (overflow)
```

### SBC - Subtract with Carry

Subtracts memory value from accumulator with borrow: A = A - M - (1 - C)

**Flags**: N, V, Z, C

**Binary Mode**: Standard 8-bit subtraction
**Decimal Mode**: BCD subtraction (if D flag set and variant supports it)

**Addressing Modes**: Same as ADC (opcodes $E9, $E5, $F5, $ED, $FD, $F9, $E1, $F1)

**Example**:

```assembly
SEC         ; Set carry (no borrow)
LDA #$50    ; A = $50
SBC #$20    ; A = $30, C = 1 (no borrow)
SBC #$40    ; A = $F0, C = 0 (borrow occurred)
```

## Logical Instructions

### AND - Logical AND

Performs bitwise AND: A = A & M

**Opcode Range**: $29, $25, $35, $2D, $3D, $39, $21, $31
**Flags**: N, Z

### ORA - Logical OR

Performs bitwise OR: A = A | M

**Opcode Range**: $09, $05, $15, $0D, $1D, $19, $01, $11
**Flags**: N, Z

### EOR - Exclusive OR

Performs bitwise XOR: A = A ^ M

**Opcode Range**: $49, $45, $55, $4D, $5D, $59, $41, $51
**Flags**: N, Z

### BIT - Bit Test

Tests bits in memory with accumulator without storing result.

**Flags**:

- Z = (A & M) == 0
- N = M bit 7
- V = M bit 6

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ZP0 | `BIT $nn` | $24 | 3 |
| ABS | `BIT $nnnn` | $2C | 4 |

## Shift/Rotate Instructions

### ASL - Arithmetic Shift Left

Shifts all bits left one position. Bit 0 = 0, bit 7 → C.

**Flags**: N, Z, C

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| IMP | `ASL A` | $0A | 2 |
| ZP0 | `ASL $nn` | $06 | 5 |
| ZPX | `ASL $nn,X` | $16 | 6 |
| ABS | `ASL $nnnn` | $0E | 6 |
| ABX | `ASL $nnnn,X` | $1E | 7 |

### LSR - Logical Shift Right

Shifts all bits right one position. Bit 7 = 0, bit 0 → C.

**Flags**: N (always 0), Z, C

**Addressing Modes**: Same as ASL (opcodes $4A, $46, $56, $4E, $5E)

### ROL - Rotate Left

Rotates all bits left one position through carry. C → bit 0, bit 7 → C.

**Flags**: N, Z, C

**Addressing Modes**: Same as ASL (opcodes $2A, $26, $36, $2E, $3E)

### ROR - Rotate Right

Rotates all bits right one position through carry. C → bit 7, bit 0 → C.

**Flags**: N, Z, C

**Addressing Modes**: Same as ASL (opcodes $6A, $66, $76, $6E, $7E)

**Note**: NMOS 6502 Rev A has a hardware quirk where ROR behaves like ASL.

## Increment/Decrement Instructions

### INC - Increment Memory

Adds 1 to memory value.

**Flags**: N, Z

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ZP0 | `INC $nn` | $E6 | 5 |
| ZPX | `INC $nn,X` | $F6 | 6 |
| ABS | `INC $nnnn` | $EE | 6 |
| ABX | `INC $nnnn,X` | $FE | 7 |

### INX - Increment X

Adds 1 to X register.

**Opcode**: $E8 | **Cycles**: 2 | **Flags**: N, Z

### INY - Increment Y

Adds 1 to Y register.

**Opcode**: $C8 | **Cycles**: 2 | **Flags**: N, Z

### DEC - Decrement Memory

Subtracts 1 from memory value.

**Flags**: N, Z

**Addressing Modes**: Same as INC (opcodes $C6, $D6, $CE, $DE)

### DEX - Decrement X

Subtracts 1 from X register.

**Opcode**: $CA | **Cycles**: 2 | **Flags**: N, Z

### DEY - Decrement Y

Subtracts 1 from Y register.

**Opcode**: $88 | **Cycles**: 2 | **Flags**: N, Z

## Compare Instructions

### CMP - Compare Accumulator

Compares accumulator with memory: A - M (result not stored).

**Flags**: N, Z, C (set if A >= M)

**Addressing Modes**: Same as ADC (opcodes $C9, $C5, $D5, $CD, $DD, $D9, $C1, $D1)

### CPX - Compare X Register

Compares X with memory: X - M (result not stored).

**Flags**: N, Z, C (set if X >= M)

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| IMM | `CPX #$nn` | $E0 | 2 |
| ZP0 | `CPX $nn` | $E4 | 3 |
| ABS | `CPX $nnnn` | $EC | 4 |

### CPY - Compare Y Register

Compares Y with memory: Y - M (result not stored).

**Flags**: N, Z, C (set if Y >= M)

**Addressing Modes**: Same as CPX (opcodes $C0, $C4, $CC)

## Branch Instructions

All branch instructions use relative addressing and take 2 cycles if not taken, 3 if taken (same page), 4 if taken (page cross).

### BCC - Branch if Carry Clear

Branches if C = 0.

**Opcode**: $90 | **Syntax**: `BCC label`

### BCS - Branch if Carry Set

Branches if C = 1.

**Opcode**: $B0 | **Syntax**: `BCS label`

### BEQ - Branch if Equal

Branches if Z = 1.

**Opcode**: $F0 | **Syntax**: `BEQ label`

### BNE - Branch if Not Equal

Branches if Z = 0.

**Opcode**: $D0 | **Syntax**: `BNE label`

### BMI - Branch if Minus

Branches if N = 1.

**Opcode**: $30 | **Syntax**: `BMI label`

### BPL - Branch if Plus

Branches if N = 0.

**Opcode**: $10 | **Syntax**: `BPL label`

### BVC - Branch if Overflow Clear

Branches if V = 0.

**Opcode**: $50 | **Syntax**: `BVC label`

### BVS - Branch if Overflow Set

Branches if V = 1.

**Opcode**: $70 | **Syntax**: `BVS label`

## Jump/Call Instructions

### JMP - Jump

Jumps to specified address.

**Flags**: None

**Addressing Modes**:

| Mode | Syntax | Opcode | Cycles |
|------|--------|--------|--------|
| ABS | `JMP $nnnn` | $4C | 3 |
| IND | `JMP ($nnnn)` | $6C | 5 |

**Note**: NMOS variants have indirect JMP bug at page boundaries.

### JSR - Jump to Subroutine

Pushes return address (PC-1) to stack, then jumps.

**Opcode**: $20 | **Cycles**: 6 | **Flags**: None

**Example**:

```assembly
JSR subroutine  ; Call subroutine
; ... continues here after RTS
```

### RTS - Return from Subroutine

Pulls return address from stack and jumps to PC+1.

**Opcode**: $60 | **Cycles**: 6 | **Flags**: None

## Stack Instructions

### PHA - Push Accumulator

Pushes A onto stack.

**Opcode**: $48 | **Cycles**: 3 | **Flags**: None

### PHP - Push Processor Status

Pushes P onto stack (with B and U flags set).

**Opcode**: $08 | **Cycles**: 3 | **Flags**: None

### PLA - Pull Accumulator

Pulls A from stack.

**Opcode**: $68 | **Cycles**: 4 | **Flags**: N, Z

### PLP - Pull Processor Status

Pulls P from stack.

**Opcode**: $28 | **Cycles**: 4 | **Flags**: All

## Flag Instructions

### CLC - Clear Carry

Sets C = 0.

**Opcode**: $18 | **Cycles**: 2

### CLD - Clear Decimal

Sets D = 0.

**Opcode**: $D8 | **Cycles**: 2

### CLI - Clear Interrupt Disable

Sets I = 0 (enables IRQ).

**Opcode**: $58 | **Cycles**: 2

### CLV - Clear Overflow

Sets V = 0.

**Opcode**: $B8 | **Cycles**: 2

### SEC - Set Carry

Sets C = 1.

**Opcode**: $38 | **Cycles**: 2

### SED - Set Decimal

Sets D = 1 (enables BCD mode).

**Opcode**: $F8 | **Cycles**: 2

**Note**: Has no effect on Ricoh 2A03/2A07 variants.

### SEI - Set Interrupt Disable

Sets I = 1 (disables IRQ).

**Opcode**: $78 | **Cycles**: 2

## System Instructions

### BRK - Break

Software interrupt. Pushes PC+2 and P (with B set) to stack, loads PC from IRQ vector.

**Opcode**: $00 | **Cycles**: 7 | **Flags**: I (set)

### RTI - Return from Interrupt

Pulls P and PC from stack.

**Opcode**: $40 | **Cycles**: 6 | **Flags**: All

### NOP - No Operation

Does nothing.

**Opcode**: $EA | **Cycles**: 2 | **Flags**: None

## Illegal/Unofficial Opcodes

The emulator supports many illegal NOPs with various addressing modes:

- **Implied NOPs**: $1A, $3A, $5A, $7A, $DA, $FA
- **Immediate NOPs**: $80, $82, $89, $C2, $E2
- **Zero Page NOPs**: $04, $44, $64
- **Zero Page,X NOPs**: $14, $34, $54, $74, $D4, $F4
- **Absolute NOP**: $0C
- **Absolute,X NOPs**: $1C, $3C, $5C, $7C, $DC, $FC (with page cross penalty)

All illegal opcodes are marked with `*` prefix (e.g., `*NOP`).

## Quick Reference Tables

### Instructions by Category

**Data Movement**: LDA, LDX, LDY, STA, STX, STY, TAX, TAY, TXA, TYA, TSX, TXS

**Arithmetic**: ADC, SBC, INC, INX, INY, DEC, DEX, DEY

**Logical**: AND, ORA, EOR, BIT

**Shift/Rotate**: ASL, LSR, ROL, ROR

**Compare**: CMP, CPX, CPY

**Branch**: BCC, BCS, BEQ, BNE, BMI, BPL, BVC, BVS

**Jump/Call**: JMP, JSR, RTS, RTI

**Stack**: PHA, PHP, PLA, PLP

**Flags**: CLC, CLD, CLI, CLV, SEC, SED, SEI

**System**: BRK, NOP

### Cycle Counts Summary

| Cycles | Instructions |
|--------|--------------|
| 2 | IMM loads, register transfers, flag ops, NOP, INX, INY, DEX, DEY |
| 3 | ZP0 loads/stores, JMP ABS, PHA, PHP, BCC/BCS/etc (not taken) |
| 4 | ABS loads, ZPX/ZPY, PLA, PLP, BCC/BCS/etc (taken, same page) |
| 5 | IZY loads, ZP0 RMW, JMP IND, ABX/ABY stores |
| 6 | IZX, ABS RMW, JSR, RTS, IZY stores |
| 7 | ABX RMW, BRK, IRQ, NMI |

## See Also

- [API Reference](api-reference.md) - Complete API documentation
- [Addressing Modes](addressing-modes.md) - Detailed addressing mode reference
- [CPU Variants](compatibility.md) - Variant-specific instruction behavior
- [Performance Guide](performance.md) - Instruction performance characteristics
