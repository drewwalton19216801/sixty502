# CPU Variant Compatibility Matrix

This document details the differences between the various 6502 CPU variants supported by this emulator.

## Variant Comparison Table

| Feature | NMOS 6502 | CMOS 65C02 | Ricoh 2A03 | Ricoh 2A07 |
|---------|-----------|------------|------------|------------|
| **Year Released** | 1975 | 1982 | 1983 | 1983 |
| **Decimal Mode** | ✓ | ✓ | ✗ | ✗ |
| **Indirect JMP Bug** | ✓ | ✗ | ✓ | ✓ |
| **N/V Flags in BCD** | Undefined | Defined | N/A | N/A |
| **Additional Instructions** | ✗ | ✓ | ✗ | ✗ |
| **Power Consumption** | Higher | Lower | Higher | Higher |

## System Compatibility

### NMOS 6502 (Original)

**Used in:**

- Apple II, II+, IIe (original)
- Commodore 64, VIC-20, PET
- Atari 2600, 400, 800, 5200
- BBC Micro
- Acorn Electron

**Characteristics:**

- Full decimal mode support with NMOS-specific behavior
- N and V flags have undefined values after BCD operations
- Has the indirect JMP page boundary bug at $xxFF
- 151 official opcodes
- Higher power consumption

**Example:**

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)
```

### CMOS 65C02 (Enhanced)

**Used in:**

- Apple IIc
- Apple IIe (enhanced)
- Later Commodore systems
- Various embedded systems

**Characteristics:**

- Full decimal mode support with improved behavior
- N and V flags properly defined after BCD operations
- Fixes the indirect JMP page boundary bug
- Additional instructions (BRA, PHX, PHY, PLX, PLY, STZ, TRB, TSB, etc.)
- Lower power consumption
- Some instructions have different cycle counts

**Example:**

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
```

### Ricoh 2A03 (NTSC)

**Used in:**

- Nintendo Entertainment System (NTSC)
- Nintendo Famicom

**Characteristics:**

- Decimal mode disabled (SED/CLD are effectively NOPs)
- D flag can be set but has no effect on ADC/SBC
- Has the indirect JMP page boundary bug
- Same instruction set as NMOS 6502
- Integrated APU (Audio Processing Unit) - not emulated by CPU core
- NTSC timing (1.789773 MHz)

**Example:**

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
```

### Ricoh 2A07 (PAL)

**Used in:**

- Nintendo Entertainment System (PAL)

**Characteristics:**

- Same as 2A03 but with PAL timing
- Decimal mode disabled
- Has the indirect JMP page boundary bug
- PAL timing (1.662607 MHz)

**Example:**

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A07)
```

## Behavioral Differences

### Decimal Mode

#### NMOS 6502

```go
// N and V flags are based on binary intermediate result
// This means they may not reflect the BCD result accurately
cpu.A = 0x09
// ADC #$01 in decimal mode
// Binary: 0x09 + 0x01 = 0x0A (N=0, V=0)
// BCD: 0x09 + 0x01 = 0x10 (but N and V based on 0x0A)
```

#### CMOS 65C02

```go
// N and V flags are properly defined for BCD operations
// They still reflect the binary intermediate result (same as NMOS)
// but the behavior is more consistent and documented
```

#### Ricoh 2A03/2A07

```go
// Decimal mode is completely disabled
// SED and CLD instructions execute but have no effect
// ADC and SBC always perform binary arithmetic
cpu.P |= cpu6502.D  // D flag can be set
// But ADC/SBC ignore it and use binary mode
```

### Indirect JMP Bug

#### NMOS 6502 and Ricoh Variants

The indirect JMP instruction has a bug when the indirect address is at a page boundary:

```go
// If the indirect address is $xxFF:
// JMP ($10FF) should read address from $10FF and $1100
// But NMOS bug reads from $10FF and $1000 instead

bus.Write(0x10FF, 0x00)
bus.Write(0x1000, 0x80)  // Bug: reads this instead of $1100
bus.Write(0x1100, 0x90)  // Should read this

// JMP ($10FF) will jump to $8000 instead of $9000
```

#### CMOS 65C02

The bug is fixed:

```go
// JMP ($10FF) correctly reads from $10FF and $1100
bus.Write(0x10FF, 0x00)
bus.Write(0x1100, 0x90)

// JMP ($10FF) correctly jumps to $9000
```

### Checking Variant Capabilities

```go
// Check if decimal mode is supported
if cpu.Variant().SupportsDecimalMode() {
    fmt.Println("Decimal mode available")
}

// Check for indirect JMP bug
if cpu.Variant().HasIndirectJMPBug() {
    fmt.Println("Has page boundary bug in JMP ($xxFF)")
}

// Get variant name
fmt.Printf("CPU Variant: %s\n", cpu.Variant())
```

## Timing Differences

### Clock Speeds

| System | Variant | Clock Speed | Notes |
|--------|---------|-------------|-------|
| Apple II | NMOS 6502 | 1.023 MHz | NTSC systems |
| Commodore 64 | NMOS 6502 | 1.023 MHz (NTSC) | 0.985 MHz (PAL) |
| Atari 2600 | NMOS 6502 | 1.19 MHz | |
| BBC Micro | NMOS 6502 | 2.0 MHz | |
| Apple IIc | CMOS 65C02 | 1.023 MHz | |

Note: This emulator provides cycle-accurate timing but does not enforce specific clock speeds. The host application is responsible for timing control.

## Instruction Set Differences

### CMOS 65C02 Additional Instructions

The CMOS 65C02 adds several new instructions not present in the NMOS 6502:

- **BRA** - Branch Always (relative)
- **PHX** - Push X Register
- **PHY** - Push Y Register
- **PLX** - Pull X Register
- **PLY** - Pull Y Register
- **STZ** - Store Zero
- **TRB** - Test and Reset Bits
- **TSB** - Test and Set Bits
- **BBR** - Branch on Bit Reset (Rockwell/WDC variants)
- **BBS** - Branch on Bit Set (Rockwell/WDC variants)

Note: This emulator currently focuses on the core 6502 instruction set. CMOS-specific instructions may be added in future versions.

## Choosing the Right Variant

### For Accurate System Emulation

Choose the variant that matches your target system:

```go
// Apple II emulation
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)

// Apple IIc emulation
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)

// Commodore 64 emulation
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)

// Atari 2600 emulation
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)
```

### For General 6502 Development

Use NMOS 6502 for maximum compatibility:

```go
cpu := cpu6502.NewCPU(bus)  // Defaults to NMOS 6502
```

### For Performance

Ricoh variants are slightly faster due to disabled decimal mode:

```go
// No decimal mode overhead
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
```

## Testing Variant-Specific Behavior

```go
func TestVariantBehavior(t *testing.T) {
    bus := &SimpleBus{}
    
    // Test NMOS decimal mode
    nmos := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)
    nmos.P |= cpu6502.D
    // Test BCD arithmetic...
    
    // Test Ricoh without decimal mode
    ricoh := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
    ricoh.P |= cpu6502.D  // Set D flag
    // Verify ADC/SBC still use binary mode...
    
    // Test CMOS bug fix
    cmos := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
    // Test JMP ($xxFF) behavior...
}
```

## References

- [6502.org](http://www.6502.org/) - Comprehensive 6502 documentation
- [Visual 6502](http://visual6502.org/) - Visual transistor-level simulation
- [Nesdev Wiki](https://wiki.nesdev.com/) - NES/Ricoh 2A03 documentation
- [WDC 65C02 Datasheet](https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf) - Official CMOS documentation
