# Instructions Module Plan

## Overview

The instruction implementations (lines 699-1246) represent the largest and most complex section of cpu.go. This plan breaks them into logical, manageable modules based on instruction categories.

## Module Structure

### 1. instructions_arithmetic.go (~200 lines)

**Purpose**: Arithmetic operations including BCD support

**Contents**:

- `ADC()` - Add with Carry (lines 721-831)
  - `adcBinary()` helper
  - `adcDecimal()` helper
- `SBC()` - Subtract with Carry (lines 1135-1222)
  - `sbcBinary()` helper
  - `sbcDecimal()` helper
- `INC()` - Increment Memory (line 977)
- `INX()` - Increment X (line 985)
- `INY()` - Increment Y (line 986)
- `DEC()` - Decrement Memory (line 959)
- `DEX()` - Decrement X (line 967)
- `DEY()` - Decrement Y (line 968)
- `CMP()` - Compare Accumulator (line 935)
- `CPX()` - Compare X (line 943)
- `CPY()` - Compare Y (line 951)

**Dependencies**: CPU struct, Flags, helper functions

---

### 2. instructions_logical.go (~100 lines)

**Purpose**: Logical operations

**Contents**:

- `AND()` - Logical AND (line 833)
- `ORA()` - Logical OR (line 1045)
- `EOR()` - Exclusive OR (line 970)
- `BIT()` - Bit Test (line 882)

**Dependencies**: CPU struct, Flags

---

### 3. instructions_shift.go (~150 lines)

**Purpose**: Shift and rotate operations

**Contents**:

- `ASL()` - Arithmetic Shift Left (lines 840-858)
- `LSR()` - Logical Shift Right (lines 1021-1037)
- `ROL()` - Rotate Left (lines 1065-1088)
- `ROR()` - Rotate Right (lines 1090-1114)

**Dependencies**: CPU struct, Flags, AddrModeType

---

### 4. instructions_branch.go (~80 lines)

**Purpose**: Branch instructions

**Contents**:

- `branchIf()` - Branch helper (lines 861-871)
- `BCC()` - Branch if Carry Clear (line 873)
- `BCS()` - Branch if Carry Set (line 874)
- `BEQ()` - Branch if Equal (line 875)
- `BMI()` - Branch if Minus (line 876)
- `BNE()` - Branch if Not Equal (line 877)
- `BPL()` - Branch if Plus (line 878)
- `BVC()` - Branch if Overflow Clear (line 879)
- `BVS()` - Branch if Overflow Set (line 880)

**Dependencies**: CPU struct, Flags

---

### 5. instructions_transfer.go (~120 lines)

**Purpose**: Load, store, and transfer operations

**Contents**:

- `LDA()` - Load Accumulator (line 1000)
- `LDX()` - Load X (line 1007)
- `LDY()` - Load Y (line 1014)
- `STA()` - Store Accumulator (line 1228)
- `STX()` - Store X (line 1233)
- `STY()` - Store Y (line 1234)
- `TAX()` - Transfer A to X (line 1236)
- `TAY()` - Transfer A to Y (line 1237)
- `TSX()` - Transfer SP to X (line 1238)
- `TXA()` - Transfer X to A (line 1239)
- `TXS()` - Transfer X to SP (line 1240)
- `TYA()` - Transfer Y to A (line 1241)

**Dependencies**: CPU struct, Flags

---

### 6. instructions_stack.go (~60 lines)

**Purpose**: Stack operations

**Contents**:

- `PHA()` - Push Accumulator (line 1052)
- `PHP()` - Push Processor Status (lines 1053-1056)
- `PLA()` - Pull Accumulator (line 1057)
- `PLP()` - Pull Processor Status (lines 1058-1063)

**Dependencies**: CPU struct, Flags, stack operations

---

### 7. instructions_control.go (~100 lines)

**Purpose**: Control flow instructions

**Contents**:

- `JMP()` - Jump (lines 988-991)
- `JSR()` - Jump to Subroutine (lines 993-998)
- `RTS()` - Return from Subroutine (lines 1129-1133)
- `RTI()` - Return from Interrupt (lines 1116-1127)
- `BRK()` - Break (lines 891-928)
- `NOP()` - No Operation (lines 1039-1043)
- `XXX()` - Illegal Opcode Handler (lines 1243-1246)

**Dependencies**: CPU struct, Flags, interrupt state

---

### 8. instructions_flags.go (~50 lines)

**Purpose**: Flag manipulation instructions

**Contents**:

- `CLC()` - Clear Carry (line 930)
- `CLD()` - Clear Decimal (line 931)
- `CLI()` - Clear Interrupt Disable (line 932)
- `CLV()` - Clear Overflow (line 933)
- `SEC()` - Set Carry (line 1224)
- `SED()` - Set Decimal (line 1225)
- `SEI()` - Set Interrupt Disable (line 1226)

**Dependencies**: CPU struct, Flags

---

### 9. helpers.go (~50 lines)

**Purpose**: Shared helper functions used by instructions

**Contents**:

- `fetchDataIfNeeded()` (lines 703-713)
- `setZNFlags()` (lines 716-719)

**Dependencies**: CPU struct, Flags, AddrModeType

---

## Migration Strategy

### Phase 1: Create Helper Module

1. Create `helpers.go`
2. Move `fetchDataIfNeeded()` and `setZNFlags()`
3. Verify all instruction references work

### Phase 2: Extract Simple Modules

1. Create `instructions_flags.go` (smallest, simplest)
2. Create `instructions_stack.go`
3. Create `instructions_logical.go`
4. Test after each extraction

### Phase 3: Extract Complex Modules

1. Create `instructions_branch.go`
2. Create `instructions_transfer.go`
3. Create `instructions_shift.go`
4. Test after each extraction

### Phase 4: Extract Arithmetic Module

1. Create `instructions_arithmetic.go` (largest, most complex)
2. Include BCD helper functions
3. Thorough testing of decimal mode

### Phase 5: Extract Control Module

1. Create `instructions_control.go`
2. Handle interrupt-related instructions carefully
3. Final comprehensive testing

## Common Patterns

### Instruction Method Signature

```go
func (c *CPU) InstructionName() uint8 {
    // Returns extra cycles needed
}
```

### Typical Instruction Structure

```go
func (c *CPU) LDA() uint8 {
    c.fetchDataIfNeeded()
    c.A = c.fetchedData
    c.setZNFlags(c.A)
    return 0
}
```

### Instructions with Mode-Specific Behavior

```go
func (c *CPU) ASL() uint8 {
    if c.currentInstruction.AddrModeType == AddrModeIMP {
        // Operate on accumulator
    } else {
        // Operate on memory
    }
    return 0
}
```

## Testing Strategy

### Per-Module Testing

Each instruction module should have:

1. Unit tests for each instruction
2. Flag behavior tests
3. Edge case tests
4. Cycle count verification

### Integration Testing

After all modules extracted:

1. Run full instruction test suite
2. Verify all addressing modes work
3. Test instruction combinations
4. Validate cycle accuracy

## Documentation Requirements

### Each Module Should Include

1. Package-level documentation
2. Instruction group description
3. Individual instruction documentation
4. Usage examples for complex instructions

### Example Documentation

```go
// Package cpu6502 provides arithmetic instruction implementations
// for the 6502 CPU emulator.
//
// This module includes:
//   - Addition and subtraction (ADC, SBC)
//   - Increment and decrement (INC, INX, INY, DEC, DEX, DEY)
//   - Comparison operations (CMP, CPX, CPY)
//   - Binary Coded Decimal (BCD) support
```

## Benefits of This Structure

1. **Logical Grouping**: Related instructions together
2. **Manageable Size**: Each file 50-200 lines
3. **Easy Navigation**: Find instructions by category
4. **Focused Testing**: Test instruction groups independently
5. **Clear Dependencies**: Each module's needs are explicit
6. **Extensibility**: Easy to add new instructions to appropriate module

## File Size Estimates

| Module | Estimated Lines | Complexity |
|--------|----------------|------------|
| helpers.go | 50 | Low |
| instructions_flags.go | 50 | Low |
| instructions_stack.go | 60 | Low |
| instructions_logical.go | 100 | Low |
| instructions_branch.go | 80 | Medium |
| instructions_transfer.go | 120 | Medium |
| instructions_shift.go | 150 | Medium |
| instructions_control.go | 100 | High |
| instructions_arithmetic.go | 200 | High |
| **Total** | **910** | - |

## Backward Compatibility

All instruction methods remain as methods on the `*CPU` type:

- No API changes
- No signature changes
- No behavior changes
- Existing code continues to work

The only change is file organization - all methods are still accessible through the CPU struct.
