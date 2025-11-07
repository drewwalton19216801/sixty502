# High Priority: Remove Reflection from Hot Paths

## Problem Statement

The current implementation uses reflection extensively via [`getFuncPtr()`](../cpu.go:260) to compare addressing mode function pointers. This occurs in critical hot paths:

- [`fetchDataIfNeeded()`](../cpu.go:265) - called for every instruction
- [`ASL()`](../cpu.go:368), [`LSR()`](../cpu.go:545), [`ROL()`](../cpu.go:604), [`ROR()`](../cpu.go:629) - called for shift/rotate operations
- [`Disassemble()`](../cpu.go:1257) - called during disassembly

**Performance Impact**: ~256+ reflection calls per instruction in worst case scenarios.

## Proposed Solution

Add an `AddrModeType` enum to eliminate reflection-based comparisons.

### Implementation Steps

#### Step 1: Define Addressing Mode Enum

```go
// AddrModeType represents the addressing mode of an instruction
type AddrModeType uint8

const (
    AddrModeIMP AddrModeType = iota // Implied
    AddrModeIMM                     // Immediate
    AddrModeZP0                     // Zero Page
    AddrModeZPX                     // Zero Page, X
    AddrModeZPY                     // Zero Page, Y
    AddrModeREL                     // Relative
    AddrModeABS                     // Absolute
    AddrModeABX                     // Absolute, X
    AddrModeABY                     // Absolute, Y
    AddrModeIND                     // Indirect
    AddrModeIZX                     // Indexed Indirect
    AddrModeIZY                     // Indirect Indexed
)

// String returns the addressing mode name for debugging
func (a AddrModeType) String() string {
    names := []string{
        "IMP", "IMM", "ZP0", "ZPX", "ZPY", "REL",
        "ABS", "ABX", "ABY", "IND", "IZX", "IZY",
    }
    if int(a) < len(names) {
        return names[a]
    }
    return "UNKNOWN"
}
```

#### Step 2: Update Instruction Struct

```go
type Instruction struct {
    Name         string           // Mnemonic (e.g., "LDA")
    Operate      func(*CPU) uint8 // Function to execute the instruction's logic
    AddrMode     func(*CPU) uint8 // Function to calculate the address and fetch data
    AddrModeType AddrModeType     // NEW: Type of addressing mode
    Cycles       uint8            // Base cycles for this instruction/mode
    Length       uint8            // Length of the instruction in bytes
    Illegal      bool             // Whether this is an official or unofficial/illegal opcode
}
```

#### Step 3: Update buildLookupTable()

Modify [`buildLookupTable()`](../cpu.go:774) to include `AddrModeType` for each instruction:

```go
func (c *CPU) buildLookupTable() {
    // Helper to get method expression pointer
    IMP := (*CPU).IMP
    IMM := (*CPU).IMM
    // ... etc
    
    // Fill with illegal opcodes first
    for i := range c.lookup {
        c.lookup[i] = Instruction{
            Name:         "XXX",
            Operate:      XXX,
            AddrMode:     IMP,
            AddrModeType: AddrModeIMP, // NEW
            Cycles:       2,
            Illegal:      true,
        }
    }
    
    // Example official opcode
    c.lookup[0xA9] = Instruction{
        Name:         "LDA",
        Operate:      LDA,
        AddrMode:     IMM,
        AddrModeType: AddrModeIMM, // NEW
        Cycles:       2,
        Length:       2,
    }
    
    // ... continue for all 256 opcodes
}
```

#### Step 4: Replace Reflection Calls

**In fetchDataIfNeeded():**

```go
// OLD (uses reflection)
func (c *CPU) fetchDataIfNeeded() {
    if getFuncPtr(c.currentInstruction.AddrMode) != getFuncPtr((*CPU).IMP) {
        c.fetchedData = c.read(c.addrAbs)
    }
}

// NEW (uses enum)
func (c *CPU) fetchDataIfNeeded() {
    if c.currentInstruction.AddrModeType != AddrModeIMP {
        c.fetchedData = c.read(c.addrAbs)
    }
}
```

**In ASL(), LSR(), ROL(), ROR():**

```go
// OLD (uses reflection)
func (c *CPU) ASL() uint8 {
    var temp uint16
    if getFuncPtr(c.currentInstruction.AddrMode) == getFuncPtr((*CPU).IMP) {
        // Accumulator mode
        temp = uint16(c.A) << 1
        c.setFlag(C, (temp&0xFF00) > 0)
        c.A = uint8(temp & 0x00FF)
        c.setZNFlags(c.A)
    } else {
        // Memory mode
        c.fetchDataIfNeeded()
        temp = uint16(c.fetchedData) << 1
        c.setFlag(C, (temp&0xFF00) > 0)
        result := uint8(temp & 0x00FF)
        c.write(c.addrAbs, result)
        c.setZNFlags(result)
    }
    return 0
}

// NEW (uses enum)
func (c *CPU) ASL() uint8 {
    var temp uint16
    if c.currentInstruction.AddrModeType == AddrModeIMP {
        // Accumulator mode
        temp = uint16(c.A) << 1
        c.setFlag(C, (temp&0xFF00) > 0)
        c.A = uint8(temp & 0x00FF)
        c.setZNFlags(c.A)
    } else {
        // Memory mode
        c.fetchDataIfNeeded()
        temp = uint16(c.fetchedData) << 1
        c.setFlag(C, (temp&0xFF00) > 0)
        result := uint8(temp & 0x00FF)
        c.write(c.addrAbs, result)
        c.setZNFlags(result)
    }
    return 0
}
```

**In Disassemble():**

```go
// OLD (uses reflection)
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string {
    disassembly := make(map[uint16]string)
    addr := startAddr
    
    impPtr := getFuncPtr((*CPU).IMP)
    immPtr := getFuncPtr((*CPU).IMM)
    // ... etc
    
    for addr <= endAddr && addr >= startAddr {
        // ...
        addrModePtr := getFuncPtr(instr.AddrMode)
        
        switch addrModePtr {
        case impPtr:
            // No operand bytes
        case immPtr:
            // ...
        }
    }
    return disassembly
}

// NEW (uses enum)
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string {
    disassembly := make(map[uint16]string)
    addr := startAddr
    
    for addr <= endAddr && addr >= startAddr {
        // ...
        switch instr.AddrModeType {
        case AddrModeIMP:
            // No operand bytes
        case AddrModeIMM:
            // ...
        }
    }
    return disassembly
}
```

#### Step 5: Remove getFuncPtr()

Once all reflection calls are replaced, remove the [`getFuncPtr()`](../cpu.go:260) function and the `reflect` import.

## Testing Strategy

1. **Unit Tests**: Verify all addressing modes work correctly with enum
2. **Benchmark Tests**: Compare performance before/after
3. **Integration Tests**: Run full test suite to ensure no regressions

### Expected Performance Improvement

- **Before**: ~1000ns per instruction (with reflection)
- **After**: ~500ns per instruction (with enum)
- **Improvement**: ~50% reduction in instruction execution time

## Migration Path

1. Add `AddrModeType` field to `Instruction` struct
2. Update `buildLookupTable()` to populate new field
3. Update all reflection-based comparisons to use enum
4. Run full test suite
5. Remove `getFuncPtr()` and `reflect` import
6. Update benchmarks to verify improvement

## Risks & Mitigation

**Risk**: Breaking existing code that relies on reflection
**Mitigation**: This is internal implementation; no public API changes

**Risk**: Incorrect enum assignments in lookup table
**Mitigation**: Add validation test to verify all 256 opcodes have correct `AddrModeType`

## Success Criteria

- [ ] All reflection calls removed from hot paths
- [ ] All tests pass
- [ ] Benchmark shows >40% performance improvement
- [ ] No public API changes
- [ ] Code coverage maintained or improved
