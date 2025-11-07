# High Priority: Fix Decimal Mode Edge Cases

## Problem Statement

The decimal mode implementation in [`ADC()`](../cpu.go:283) and [`SBC()`](../cpu.go:669) has accuracy issues:

1. **Line 316 in ADC**: Carry detection from lower nibble may be incorrect
2. **N/V Flag Behavior**: Varies between 6502 variants but implementation assumes one behavior
3. **Edge Cases**: 99+1, 00-1, and other boundary conditions need verification

## Current Implementation Issues

### Issue 1: ADC Lower Nibble Carry Detection

```go
// Current code at line 310-318
low := (c.A & 0x0F) + (c.fetchedData & 0x0F) + uint8(carry)
if low > 9 {
    low += 6 // BCD adjustment for lower nibble
}
high := (c.A >> 4) + (c.fetchedData >> 4)
if low > 0x0F { // ISSUE: This check is after adjustment
    high++
}
```

**Problem**: The check `if low > 0x0F` happens after BCD adjustment, which may not correctly detect carry.

### Issue 2: N/V Flags in Decimal Mode

Different 6502 variants handle N/V flags differently in decimal mode:

- **NMOS 6502**: N/V flags are undefined/unreliable
- **CMOS 65C02**: N/V flags are set based on binary result
- **Ricoh 2A03** (NES): Decimal mode is disabled entirely

Current implementation assumes CMOS behavior for all variants.

## Proposed Solution

### Step 1: Fix ADC Lower Nibble Carry

```go
func (c *CPU) ADC() uint8 {
    c.fetchDataIfNeeded()
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }

    if c.getFlag(D) {
        // Calculate intermediate binary sum for N/V flags
        binarySum := uint16(c.A) + uint16(c.fetchedData) + carry
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)

        // Perform BCD addition
        low := (c.A & 0x0F) + (c.fetchedData & 0x0F) + uint8(carry)
        lowCarry := uint8(0)
        if low > 9 {
            low += 6
            lowCarry = 1  // Carry generated from lower nibble
        }
        
        high := (c.A >> 4) + (c.fetchedData >> 4) + lowCarry
        if high > 9 {
            high += 6
            c.setFlag(C, true)  // Carry out from high nibble
        } else {
            c.setFlag(C, false)
        }

        result := ((high & 0x0F) << 4) | (low & 0x0F)
        c.A = result
        c.setFlag(Z, c.A == 0)
    } else {
        // Binary mode (unchanged)
        temp := uint16(c.A) + uint16(c.fetchedData) + carry
        c.setFlag(C, temp > 0xFF)
        result := uint8(temp & 0x00FF)
        c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^temp)&0x0080) > 0)
        c.A = result
        c.setZNFlags(c.A)
    }
    return 1
}
```

### Step 2: Fix SBC Lower Nibble Borrow

```go
func (c *CPU) SBC() uint8 {
    c.fetchDataIfNeeded()
    value := uint16(c.fetchedData) ^ 0x00FF
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }

    if c.getFlag(D) {
        // Calculate intermediate binary result for N/V flags
        binarySum := uint16(c.A) + value + carry
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)

        // Perform BCD subtraction
        borrow_in := 1 - uint8(carry)
        
        low := int16(c.A&0x0F) - int16(c.fetchedData&0x0F) - int16(borrow_in)
        borrow_low := uint8(0)
        if low < 0 {
            low += 10  // BCD adjustment (not 6!)
            borrow_low = 1
        }
        
        high := int16(c.A>>4) - int16(c.fetchedData>>4) - int16(borrow_low)
        if high < 0 {
            high += 10  // BCD adjustment (not 6!)
            c.setFlag(C, false)  // Borrow occurred
        } else {
            c.setFlag(C, true)   // No borrow
        }
        
        result := (uint8(high&0x0F) << 4) | uint8(low&0x0F)
        c.A = result
        c.setFlag(Z, c.A == 0)
    } else {
        // Binary mode (unchanged)
        temp := uint16(c.A) + value + carry
        c.setFlag(C, temp > 0xFF)
        result := uint8(temp & 0x00FF)
        c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^temp)&0x0080) > 0)
        c.A = result
        c.setZNFlags(c.A)
    }
    return 1
}
```

### Step 3: Add CPU Variant Support

```go
// CPUVariant represents different 6502 processor variants
type CPUVariant int

const (
    VariantNMOS6502  CPUVariant = iota // Original NMOS 6502
    VariantCMOS65C02                   // CMOS 65C02 with fixes
    VariantRicoh2A03                   // NES/Famicom CPU (no decimal mode)
)

type CPU struct {
    // ... existing fields
    variant CPUVariant
}

// NewCPU creates a CPU with default NMOS 6502 variant
func NewCPU(bus Bus) *CPU {
    return NewCPUWithVariant(bus, VariantNMOS6502)
}

// NewCPUWithVariant creates a CPU with specified variant
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU {
    c := &CPU{
        bus:     bus,
        P:       U | I,
        SP:      0xFD,
        variant: variant,
    }
    c.buildLookupTable()
    return c
}
```

### Step 4: Variant-Specific Decimal Mode Handling

```go
func (c *CPU) ADC() uint8 {
    c.fetchDataIfNeeded()
    
    // Ricoh 2A03 has no decimal mode
    if c.variant == VariantRicoh2A03 {
        return c.adcBinary()
    }
    
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }

    if c.getFlag(D) {
        // Calculate binary result for N/V flags
        binarySum := uint16(c.A) + uint16(c.fetchedData) + carry
        
        // Variant-specific flag handling
        switch c.variant {
        case VariantNMOS6502:
            // N/V flags are undefined in decimal mode on NMOS
            // Set them based on binary result (common behavior)
            c.setFlag(N, (binarySum&0x80) > 0)
            c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
        case VariantCMOS65C02:
            // CMOS always sets N/V based on binary result
            c.setFlag(N, (binarySum&0x80) > 0)
            c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
        }
        
        // BCD arithmetic (same for all variants that support it)
        // ... (use corrected algorithm from Step 1)
    } else {
        return c.adcBinary()
    }
    return 1
}

// Helper for binary mode ADC
func (c *CPU) adcBinary() uint8 {
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }
    
    temp := uint16(c.A) + uint16(c.fetchedData) + carry
    c.setFlag(C, temp > 0xFF)
    result := uint8(temp & 0x00FF)
    c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^temp)&0x0080) > 0)
    c.A = result
    c.setZNFlags(c.A)
    return 1
}
```

## Test Cases to Add

### ADC Decimal Mode Tests

```go
// Edge cases that need testing
tests := []struct {
    name      string
    a         uint8
    operand   uint8
    carryIn   bool
    expectedA uint8
    expectedC bool
    expectedZ bool
    expectedN bool
    expectedV bool
}{
    {"99+1 C=0", 0x99, 0x01, false, 0x00, true, true, true, false},
    {"99+0 C=1", 0x99, 0x00, true, 0x00, true, true, true, false},
    {"50+50 C=0", 0x50, 0x50, false, 0x00, true, true, true, true},
    {"09+1 C=0", 0x09, 0x01, false, 0x10, false, false, false, false},
    {"09+1 C=1", 0x09, 0x01, true, 0x11, false, false, false, false},
    {"00+00 C=0", 0x00, 0x00, false, 0x00, false, true, false, false},
    {"00+00 C=1", 0x00, 0x00, true, 0x01, false, false, false, false},
}
```

### SBC Decimal Mode Tests

```go
tests := []struct {
    name      string
    a         uint8
    operand   uint8
    carryIn   bool
    expectedA uint8
    expectedC bool
    expectedZ bool
}{
    {"00-1 C=1", 0x00, 0x01, true, 0x99, false, false},
    {"00-1 C=0", 0x00, 0x01, false, 0x98, false, false},
    {"10-1 C=1", 0x10, 0x01, true, 0x09, true, false},
    {"10-1 C=0", 0x10, 0x01, false, 0x08, true, false},
    {"00-0 C=1", 0x00, 0x00, true, 0x00, true, true},
}
```

## Migration Path

1. Add `CPUVariant` type and field
2. Fix ADC/SBC algorithms with corrected carry/borrow detection
3. Add variant-specific behavior
4. Add comprehensive decimal mode tests
5. Verify against known test suites (e.g., Klaus Dormann's tests)

## Success Criteria

- [ ] ADC lower nibble carry detection fixed
- [ ] SBC lower nibble borrow detection fixed
- [ ] CPU variant support added
- [ ] All decimal mode edge cases pass
- [ ] N/V flag behavior correct for each variant
- [ ] Backward compatible (default NMOS behavior)
