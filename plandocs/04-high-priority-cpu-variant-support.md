# High Priority: Add CPU Variant Support

## Problem Statement

The current implementation assumes a single 6502 variant, but different systems use different variants with distinct behaviors:

- **NMOS 6502**: Original MOS Technology chip (Apple II, Commodore 64, Atari)
- **CMOS 65C02**: Enhanced version with bug fixes and new instructions (Apple IIc, IIe)
- **Ricoh 2A03/2A07**: NES/Famicom CPU with no decimal mode and APU integration

Key differences:

1. Decimal mode support (disabled on Ricoh)
2. N/V flag behavior in decimal mode
3. Indirect JMP bug (fixed in CMOS)
4. Additional instructions in CMOS
5. Different timing characteristics

## Proposed Solution

Add comprehensive variant support with configurable behavior.

### Implementation Steps

#### Step 1: Define Variant Types

```go
// CPUVariant represents different 6502 processor variants
type CPUVariant int

const (
    // VariantNMOS6502 is the original NMOS 6502 (1975)
    // Used in: Apple II, Commodore 64, Atari 2600/800, BBC Micro
    // Features: All documented bugs, decimal mode supported
    VariantNMOS6502 CPUVariant = iota
    
    // VariantCMOS65C02 is the CMOS 65C02 (1982)
    // Used in: Apple IIc, Apple IIe (enhanced), later systems
    // Features: Bug fixes, additional instructions, lower power
    VariantCMOS65C02
    
    // VariantRicoh2A03 is the NES/Famicom CPU (1983)
    // Used in: Nintendo Entertainment System, Famicom
    // Features: No decimal mode, integrated APU, different timing
    VariantRicoh2A03
    
    // VariantRicoh2A07 is the PAL NES CPU
    // Same as 2A03 but with PAL timing
    VariantRicoh2A07
)

// String returns the variant name
func (v CPUVariant) String() string {
    names := []string{
        "NMOS 6502",
        "CMOS 65C02",
        "Ricoh 2A03 (NTSC)",
        "Ricoh 2A07 (PAL)",
    }
    if int(v) < len(names) {
        return names[v]
    }
    return "Unknown"
}

// SupportsDecimalMode returns true if the variant supports decimal mode
func (v CPUVariant) SupportsDecimalMode() bool {
    switch v {
    case VariantRicoh2A03, VariantRicoh2A07:
        return false
    default:
        return true
    }
}

// HasIndirectJMPBug returns true if the variant has the indirect JMP page boundary bug
func (v CPUVariant) HasIndirectJMPBug() bool {
    switch v {
    case VariantNMOS6502, VariantRicoh2A03, VariantRicoh2A07:
        return true
    case VariantCMOS65C02:
        return false
    default:
        return true
    }
}
```

#### Step 2: Update CPU Struct

```go
type CPU struct {
    // Registers
    A  uint8
    X  uint8
    Y  uint8
    SP uint8
    PC uint16
    P  Flags

    // Bus connection
    bus Bus

    // Internal state
    Cycles             uint8
    opcode             uint8
    fetchedData        uint8
    addrAbs            uint16
    addrRel            uint16
    currentInstruction *Instruction
    lookup             [256]Instruction
    totalCycles        uint64
    
    // NEW: Variant configuration
    variant CPUVariant
}
```

#### Step 3: Update Constructors

```go
// NewCPU creates a new CPU with default NMOS 6502 variant
func NewCPU(bus Bus) *CPU {
    return NewCPUWithVariant(bus, VariantNMOS6502)
}

// NewCPUWithVariant creates a new CPU with specified variant
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

// Variant returns the CPU variant
func (c *CPU) Variant() CPUVariant {
    return c.variant
}
```

#### Step 4: Update IND Addressing Mode

```go
// IND implements indirect addressing mode
// Handles the page boundary bug on NMOS variants
func (c *CPU) IND() uint8 {
    ptrLo := uint16(c.read(c.PC))
    c.PC++
    ptrHi := uint16(c.read(c.PC))
    c.PC++
    ptr := (ptrHi << 8) | ptrLo

    if c.variant.HasIndirectJMPBug() && ptrLo == 0x00FF {
        // NMOS bug: If low byte is $FF, high byte is fetched from $xx00
        c.addrAbs = uint16(c.read(ptr)) | (uint16(c.read(ptr&0xFF00)) << 8)
    } else {
        // CMOS fix: Normal behavior
        c.addrAbs = (uint16(c.read(ptr+1)) << 8) | uint16(c.read(ptr))
    }
    return 0
}
```

#### Step 5: Update Decimal Mode Instructions

```go
// ADC with variant-specific decimal mode handling
func (c *CPU) ADC() uint8 {
    c.fetchDataIfNeeded()
    
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }

    // Check if decimal mode is supported and enabled
    if c.getFlag(D) && c.variant.SupportsDecimalMode() {
        return c.adcDecimal(carry)
    }
    
    // Binary mode (or decimal mode disabled)
    return c.adcBinary(carry)
}

func (c *CPU) adcBinary(carry uint16) uint8 {
    temp := uint16(c.A) + uint16(c.fetchedData) + carry
    c.setFlag(C, temp > 0xFF)
    result := uint8(temp & 0x00FF)
    c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^temp)&0x0080) > 0)
    c.A = result
    c.setZNFlags(c.A)
    return 1
}

func (c *CPU) adcDecimal(carry uint16) uint8 {
    // Calculate binary result for N/V flags
    binarySum := uint16(c.A) + uint16(c.fetchedData) + carry
    
    // Variant-specific N/V flag handling
    switch c.variant {
    case VariantNMOS6502:
        // NMOS: N/V based on binary intermediate result
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
    case VariantCMOS65C02:
        // CMOS: N/V based on binary intermediate result (same as NMOS)
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ uint16(c.fetchedData)))&(uint16(c.A)^binarySum)&0x0080) > 0)
    }
    
    // BCD arithmetic (corrected algorithm)
    low := (c.A & 0x0F) + (c.fetchedData & 0x0F) + uint8(carry)
    lowCarry := uint8(0)
    if low > 9 {
        low += 6
        lowCarry = 1
    }
    
    high := (c.A >> 4) + (c.fetchedData >> 4) + lowCarry
    if high > 9 {
        high += 6
        c.setFlag(C, true)
    } else {
        c.setFlag(C, false)
    }
    
    result := ((high & 0x0F) << 4) | (low & 0x0F)
    c.A = result
    c.setFlag(Z, c.A == 0)
    
    return 1
}

// SBC with variant-specific decimal mode handling
func (c *CPU) SBC() uint8 {
    c.fetchDataIfNeeded()
    
    // Check if decimal mode is supported and enabled
    if c.getFlag(D) && c.variant.SupportsDecimalMode() {
        return c.sbcDecimal()
    }
    
    // Binary mode (or decimal mode disabled)
    return c.sbcBinary()
}

func (c *CPU) sbcBinary() uint8 {
    value := uint16(c.fetchedData) ^ 0x00FF
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }
    
    temp := uint16(c.A) + value + carry
    c.setFlag(C, temp > 0xFF)
    result := uint8(temp & 0x00FF)
    c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^temp)&0x0080) > 0)
    c.A = result
    c.setZNFlags(c.A)
    
    return 1
}

func (c *CPU) sbcDecimal() uint8 {
    value := uint16(c.fetchedData) ^ 0x00FF
    var carry uint16 = 0
    if c.getFlag(C) {
        carry = 1
    }
    
    // Calculate binary result for N/V flags
    binarySum := uint16(c.A) + value + carry
    
    // Variant-specific N/V flag handling
    switch c.variant {
    case VariantNMOS6502:
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)
    case VariantCMOS65C02:
        c.setFlag(N, (binarySum&0x80) > 0)
        c.setFlag(V, ((^(uint16(c.A) ^ value))&(uint16(c.A)^binarySum)&0x0080) > 0)
    }
    
    // BCD subtraction (corrected algorithm)
    borrow_in := 1 - uint8(carry)
    
    low := int16(c.A&0x0F) - int16(c.fetchedData&0x0F) - int16(borrow_in)
    borrow_low := uint8(0)
    if low < 0 {
        low += 10
        borrow_low = 1
    }
    
    high := int16(c.A>>4) - int16(c.fetchedData>>4) - int16(borrow_low)
    if high < 0 {
        high += 10
        c.setFlag(C, false)
    } else {
        c.setFlag(C, true)
    }
    
    result := (uint8(high&0x0F) << 4) | uint8(low&0x0F)
    c.A = result
    c.setFlag(Z, c.A == 0)
    
    return 1
}
```

#### Step 6: Add Variant-Specific Tests

```go
func TestVariantBehavior(t *testing.T) {
    variants := []CPUVariant{
        VariantNMOS6502,
        VariantCMOS65C02,
        VariantRicoh2A03,
    }
    
    for _, variant := range variants {
        t.Run(variant.String(), func(t *testing.T) {
            cpu, bus := setupCPUWithVariant(variant)
            
            // Test decimal mode support
            if variant.SupportsDecimalMode() {
                testDecimalMode(t, cpu, bus)
            } else {
                testDecimalModeDisabled(t, cpu, bus)
            }
            
            // Test indirect JMP bug
            if variant.HasIndirectJMPBug() {
                testIndirectJMPBug(t, cpu, bus)
            } else {
                testIndirectJMPFixed(t, cpu, bus)
            }
        })
    }
}
```

## Usage Examples

### Example 1: NES Emulator

```go
// Create Ricoh 2A03 CPU for NES emulation
bus := &NESBus{}
cpu := NewCPUWithVariant(bus, VariantRicoh2A03)

// Decimal mode instructions will execute as binary
cpu.Reset()
```

### Example 2: Apple II Emulator

```go
// Create NMOS 6502 for Apple II
bus := &AppleIIBus{}
cpu := NewCPUWithVariant(bus, VariantNMOS6502)

// Has indirect JMP bug and decimal mode
cpu.Reset()
```

### Example 3: Apple IIc Emulator

```go
// Create CMOS 65C02 for Apple IIc
bus := &AppleIIcBus{}
cpu := NewCPUWithVariant(bus, VariantCMOS65C02)

// No indirect JMP bug, has decimal mode
cpu.Reset()
```

## Testing Strategy

1. **Variant Detection Tests**: Verify variant properties
2. **Decimal Mode Tests**: Test each variant's decimal mode behavior
3. **Indirect JMP Tests**: Verify bug presence/absence
4. **Cross-Variant Tests**: Ensure same binary behavior across variants
5. **Integration Tests**: Test with real ROM images

## Success Criteria

- [ ] All variants defined with correct properties
- [ ] Decimal mode respects variant support
- [ ] Indirect JMP bug correctly implemented/fixed per variant
- [ ] All variant-specific tests pass
- [ ] Backward compatible (default NMOS behavior)
- [ ] Documentation updated with variant differences
