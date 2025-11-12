# Addressing Modes Reference

This document provides a comprehensive reference for all 6502 addressing modes supported by the sixty502 emulator.

## Overview

The 6502 supports 13 different addressing modes that determine how instructions access their operands. Each mode has different characteristics regarding:

- Memory access patterns
- Instruction length (1-3 bytes)
- Cycle timing
- Page boundary crossing behavior

## Addressing Mode Summary

| Mode | Syntax | Example | Bytes | Description |
|------|--------|---------|-------|-------------|
| IMP | - | `TAX` | 1 | Implied/Implicit |
| IMM | #$nn | `LDA #$42` | 2 | Immediate |
| ZP0 | $nn | `LDA $42` | 2 | Zero Page |
| ZPX | $nn,X | `LDA $42,X` | 2 | Zero Page,X |
| ZPY | $nn,Y | `LDX $42,Y` | 2 | Zero Page,Y |
| REL | label | `BNE label` | 2 | Relative |
| ABS | $nnnn | `LDA $1234` | 3 | Absolute |
| ABX | $nnnn,X | `LDA $1234,X` | 3 | Absolute,X |
| ABY | $nnnn,Y | `LDA $1234,Y` | 3 | Absolute,Y |
| IND | ($nnnn) | `JMP ($1234)` | 3 | Indirect |
| IZX | ($nn,X) | `LDA ($40,X)` | 2 | Indexed Indirect |
| IZY | ($nn),Y | `LDA ($40),Y` | 2 | Indirect Indexed |

## Detailed Mode Descriptions

### IMP - Implied

**Syntax**: No operand

**Description**: The operand is implied by the instruction itself. No additional bytes are read from memory.

**Used by**:

- Register transfers: `TAX`, `TXA`, `TAY`, `TYA`, `TSX`, `TXS`
- Stack operations: `PHA`, `PLA`, `PHP`, `PLP`
- Flag operations: `CLC`, `SEC`, `CLD`, `SED`, `CLI`, `SEI`, `CLV`
- Control: `NOP`, `BRK`, `RTI`, `RTS`
- Shift/rotate on accumulator: `ASL A`, `LSR A`, `ROL A`, `ROR A`

**Example**:

```text
TAX     ; Transfer A to X (no operand needed)
```

**Characteristics**:

- Instruction length: 1 byte
- No memory access for operand
- No page cross possible
- Fastest addressing mode

### IMM - Immediate

**Syntax**: `#$nn`

**Description**: The operand is the byte immediately following the opcode. The value is used directly, not as an address.

**Used by**: Load, arithmetic, logical, and compare instructions

**Example**:

```text
LDA #$42    ; Load literal value $42 into A
ADC #$10    ; Add literal value $10 to A
CMP #$FF    ; Compare A with literal value $FF
```

**Characteristics**:

- Instruction length: 2 bytes
- Operand at PC+1
- No page cross possible
- Fast execution

**Memory Layout**:

```text
Address  | Byte
---------|------
$8000    | $A9    (LDA opcode)
$8001    | $42    (operand)
```

### ZP0 - Zero Page

**Syntax**: `$nn`

**Description**: The operand address is in the zero page ($0000-$00FF). Only one byte is needed to specify the address, making this mode faster and more compact than absolute addressing.

**Example**:

```text
LDA $42     ; Load from address $0042
STA $80     ; Store to address $0080
```

**Characteristics**:

- Instruction length: 2 bytes
- Address range: $0000-$00FF
- No page cross possible (always in zero page)
- Faster than absolute addressing

**Memory Layout**:

```text
Address  | Byte
---------|------
$8000    | $A5    (LDA ZP0 opcode)
$8001    | $42    (zero page address)
         |
$0042    | $FF    (data loaded from here)
```

### ZPX - Zero Page,X

**Syntax**: `$nn,X`

**Description**: The X register is added to the zero page address. The result wraps around within the zero page (no carry to high byte).

**Example**:

```text
LDA $40,X   ; Load from ($40 + X) & $FF
```

**Wrapping Example**:

```text
; X = $10
LDA $40,X   ; Loads from $0050

; X = $FF
LDA $FF,X   ; Loads from $00FE (wraps: ($FF + $FF) & $FF = $FE)

; X = $02
LDA $FF,X   ; Loads from $0001 (wraps: ($FF + $02) & $FF = $01)
```

**Characteristics**:

- Instruction length: 2 bytes
- Wraps within zero page
- No page cross possible
- One extra cycle vs ZP0

### ZPY - Zero Page,Y

**Syntax**: `$nn,Y`

**Description**: Similar to ZPX but uses Y register. Only used by `LDX` and `STX` instructions.

**Example**:

```text
LDX $40,Y   ; Load X from ($40 + Y) & $FF
STX $80,Y   ; Store X to ($80 + Y) & $FF
```

**Characteristics**:

- Instruction length: 2 bytes
- Wraps within zero page
- Limited instruction support
- No page cross possible

### REL - Relative

**Syntax**: `label` (assembler calculates offset)

**Description**: Used exclusively by branch instructions. The operand is a signed 8-bit offset (-128 to +127) relative to the address of the next instruction.

**Example**:

```text
BEQ forward     ; Branch if equal
BNE backward    ; Branch if not equal
```

**Offset Calculation**:

```text
; At address $8000:
$8000: BEQ $10  ; Offset = $10 (positive)
               ; Target = $8002 + $10 = $8012

; At address $8000:
$8000: BEQ $FE  ; Offset = $FE (negative: -2)
               ; Target = $8002 + (-2) = $8000
```

**Characteristics**:

- Instruction length: 2 bytes
- Range: -128 to +127 bytes from next instruction
- Branch taken: +1 cycle
- Page cross on branch: +1 additional cycle

**Cycle Timing**:

- Branch not taken: 2 cycles
- Branch taken (same page): 3 cycles
- Branch taken (page cross): 4 cycles

### ABS - Absolute

**Syntax**: `$nnnn`

**Description**: The full 16-bit address is specified in the next two bytes (little-endian: low byte first, then high byte).

**Example**:

```text
LDA $1234   ; Load from address $1234
JMP $8000   ; Jump to address $8000
```

**Memory Layout**:

```text
Address  | Byte
---------|------
$8000    | $AD    (LDA ABS opcode)
$8001    | $34    (address low byte)
$8002    | $12    (address high byte)
         |
$1234    | $42    (data loaded from here)
```

**Characteristics**:

- Instruction length: 3 bytes
- Full 64KB address space
- No page cross possible (address is explicit)

### ABX - Absolute,X

**Syntax**: `$nnnn,X`

**Description**: The X register is added to the 16-bit base address to form the effective address. Page boundary crossing may add an extra cycle.

**Example**:

```text
LDA $1234,X   ; Load from $1234 + X
```

**Page Cross Example**:

```text
; X = $10
LDA $1234,X   ; Effective address: $1244 (no page cross)

; X = $FF
LDA $12FF,X   ; Effective address: $13FE (page cross: $12 → $13)
```

**Characteristics**:

- Instruction length: 3 bytes
- Can access full 64KB space
- Page cross detection: `(base & $FF00) != (effective & $FF00)`
- Some instructions add +1 cycle on page cross

**Page Cross Penalty**:

- Load instructions (LDA, LDX, LDY): +1 cycle
- Store instructions (STA, STX, STY): No penalty (always use full cycles)
- Read-modify-write (INC, DEC, ASL, etc.): No penalty

### ABY - Absolute,Y

**Syntax**: `$nnnn,Y`

**Description**: Similar to ABX but uses Y register.

**Example**:

```text
LDA $1234,Y   ; Load from $1234 + Y
```

**Characteristics**:

- Instruction length: 3 bytes
- Same page cross behavior as ABX
- Used by fewer instructions than ABX

### IND - Indirect

**Syntax**: `($nnnn)`

**Description**: Used only by `JMP` instruction. The operand is a pointer address. The actual jump target is read from that pointer address.

**Example**:

```text
JMP ($1234)   ; Jump to address stored at $1234/$1235
```

**Normal Behavior**:

```text
; Memory:
$1234: $00    (target low byte)
$1235: $80    (target high byte)

JMP ($1234)   ; Jumps to $8000
```

**NMOS Bug** (page boundary):

When the pointer address has $FF as the low byte, the NMOS 6502 has a bug:

```text
; Memory:
$12FF: $00    (target low byte)
$1300: $80    (target high byte - correct)
$1200: $90    (target high byte - bug reads this!)

; NMOS 6502:
JMP ($12FF)   ; Jumps to $9000 (bug: reads $12FF and $1200)

; CMOS 65C02:
JMP ($12FF)   ; Jumps to $8000 (correct: reads $12FF and $1300)
```

**Characteristics**:

- Instruction length: 3 bytes
- Only used by JMP
- NMOS variants have page boundary bug
- CMOS 65C02 fixes the bug

### IZX - Indexed Indirect (Indirect,X)

**Syntax**: `($nn,X)`

**Description**: The X register is added to the zero page address to get a pointer address. The actual operand address is then read from this pointer. All arithmetic wraps within the zero page.

**Example**:

```text
LDA ($40,X)   ; Load from address pointed to by ($40 + X)
```

**Step-by-Step**:

```text
; X = $05
; Memory:
$0045: $00    (pointer low byte)
$0046: $12    (pointer high byte)
$1200: $42    (actual data)

LDA ($40,X)
1. Calculate pointer address: $40 + $05 = $45
2. Read pointer from $45/$46: $1200
3. Load data from $1200: $42
```

**Wrapping Example**:

```text
; X = $02
; Memory:
$0001: $00    (pointer low byte)
$0002: $20    (pointer high byte)

LDA ($FF,X)
1. Pointer address: ($FF + $02) & $FF = $01 (wraps!)
2. Read pointer from $01/$02: $2000
3. Load data from $2000
```

**Characteristics**:

- Instruction length: 2 bytes
- Pointer calculation wraps in zero page
- No page cross penalty (pointer is in zero page)
- Commonly used for indexed data structures

### IZY - Indirect Indexed (Indirect),Y

**Syntax**: `($nn),Y`

**Description**: A zero page address points to a base address. The Y register is then added to this base address to get the effective address. Page crossing may add a cycle.

**Example**:

```text
LDA ($40),Y   ; Load from (address at $40) + Y
```

**Step-by-Step**:

```text
; Y = $10
; Memory:
$0040: $00    (base address low byte)
$0041: $12    (base address high byte)
$1210: $42    (actual data)

LDA ($40),Y
1. Read base address from $40/$41: $1200
2. Add Y: $1200 + $10 = $1210
3. Load data from $1210: $42
```

**Page Cross Example**:

```text
; Y = $10
; Memory:
$0040: $FF    (base address low byte)
$0041: $12    (base address high byte)

LDA ($40),Y
1. Base address: $12FF
2. Add Y: $12FF + $10 = $130F (page cross: $12 → $13)
3. +1 cycle penalty for page cross
```

**Characteristics**:

- Instruction length: 2 bytes
- Base pointer in zero page
- Page cross detection on final address
- Load instructions: +1 cycle on page cross
- Store instructions: No penalty

## Page Boundary Crossing

### What is a Page?

A "page" is 256 bytes ($100). The 6502's 64KB address space is divided into 256 pages:

- Page $00: $0000-$00FF (zero page)
- Page $01: $0100-$01FF (stack)
- Page $02: $0200-$02FF
- ...
- Page $FF: $FF00-$FFFF

### When Does Page Crossing Occur?

A page boundary is crossed when the high byte of the effective address differs from the high byte of the base address:

```go
pageCross := (effectiveAddr & 0xFF00) != (baseAddr & 0xFF00)
```

### Which Modes Can Cross Pages?

- **ABX** (Absolute,X)
- **ABY** (Absolute,Y)
- **IZY** (Indirect,Y)

### Which Instructions Add a Cycle?

**Add +1 cycle on page cross**:

- Load: LDA, LDX, LDY
- Arithmetic: ADC, SBC
- Logical: AND, EOR, ORA
- Compare: CMP

**Always use full cycles** (no penalty):

- Store: STA, STX, STY
- Read-modify-write: ASL, LSR, ROL, ROR, INC, DEC

### Example

```text
; No page cross
LDA $1200,X   ; X = $50
; Effective: $1250 (page $12)
; Cycles: 4

; Page cross
LDA $12FF,X   ; X = $02
; Effective: $1301 (page $13)
; Cycles: 5 (+1 for page cross)

; Store - no penalty
STA $12FF,X   ; X = $02
; Effective: $1301 (page $13)
; Cycles: 5 (always, regardless of page cross)
```

## Addressing Mode Selection Guide

### When to Use Each Mode

**IMP**: Instructions that don't need operands

```text
TAX, PHA, RTS, NOP
```

**IMM**: Loading literal values

```text
LDA #$00    ; Clear A
CMP #$FF    ; Compare with constant
```

**ZP0**: Frequently accessed variables (fast!)

```text
LDA $80     ; Load from zero page variable
INC $90     ; Increment counter
```

**ZPX/ZPY**: Indexed zero page access

```text
LDA $80,X   ; Array access in zero page
```

**REL**: Conditional branching

```text
BEQ loop    ; Branch to loop
BNE exit    ; Branch to exit
```

**ABS**: Accessing specific addresses

```text
LDA $2000   ; Load from specific location
JMP $8000   ; Jump to routine
```

**ABX/ABY**: Array/table access

```text
LDA table,X ; Access array element
STA buffer,Y ; Store to buffer
```

**IND**: Indirect jumps (jump tables)

```text
JMP (vector) ; Jump through vector
```

**IZX**: Indexed pointer tables

```text
LDA (ptrs,X) ; Access through indexed pointer
```

**IZY**: Pointer plus offset (common for structures)

```text
LDA (ptr),Y  ; Access structure member
```

## Performance Comparison

| Mode | Cycles (typical) | Speed | Use Case |
|------|------------------|-------|----------|
| IMP | 2 | Fastest | Register operations |
| IMM | 2 | Fastest | Literal values |
| ZP0 | 3 | Fast | Variables |
| ZPX/ZPY | 4 | Fast | Indexed variables |
| ABS | 4 | Medium | Specific addresses |
| ABX/ABY | 4-5 | Medium | Arrays |
| IZX | 6 | Slow | Indexed pointers |
| IZY | 5-6 | Slow | Structures |

## See Also

- [API Reference](api-reference.md) - Complete API documentation
- [Instruction Set](instruction-set.md) - Instructions by addressing mode
- [Performance Guide](performance.md) - Optimization strategies
- [CPU Variants](compatibility.md) - Variant-specific addressing behavior
