# Types Module (types.go)

## Purpose

Extract all type definitions, constants, and enums into a dedicated types file. This provides a central location for understanding the CPU's data structures.

## Contents

### 1. Addressing Mode Types (Lines 78-106)

```go
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

func (a AddrModeType) String() string
```

### 2. CPU Variants (Lines 108-166)

```go
type CPUVariant int

const (
    VariantNMOS6502 CPUVariant = iota
    VariantCMOS65C02
    VariantRicoh2A03
    VariantRicoh2A07
)

func (v CPUVariant) String() string
func (v CPUVariant) SupportsDecimalMode() bool
func (v CPUVariant) HasIndirectJMPBug() bool
```

### 3. Processor Flags (Lines 228-250)

```go
type Flags uint8

const (
    C Flags = 1 << 0 // Carry Bit
    Z Flags = 1 << 1 // Zero
    I Flags = 1 << 2 // Disable Interrupts
    D Flags = 1 << 3 // Decimal Mode
    B Flags = 1 << 4 // Break Command
    U Flags = 1 << 5 // Unused (always 1)
    V Flags = 1 << 6 // Overflow
    N Flags = 1 << 7 // Negative
)
```

### 4. Instruction Structure (Lines 382-391)

```go
type Instruction struct {
    Name             string
    Operate          func(*CPU) uint8
    AddrMode         func(*CPU) uint8
    AddrModeType     AddrModeType
    Cycles           uint8
    Length           uint8
    Illegal          bool
    PageCrossPenalty bool
}
```

### 5. State Structure (Lines 1891-1937)

```go
type State struct {
    A               uint8
    X               uint8
    Y               uint8
    SP              uint8
    PC              uint16
    P               Flags
    Cycles          uint8
    TotalCycles     uint64
    Opcode          uint8
    Instruction     string
    InInterrupt     bool
    InterruptVector uint16
}

func (s State) String() string
```

## Dependencies

- None (pure type definitions)

## Exports

- All types and constants are exported
- All methods on types are exported

## File Size Estimate

~150-200 lines

## Migration Notes

### Step 1: Create File

```go
package cpu6502

// Type definitions for the 6502 CPU emulator
```

### Step 2: Copy Type Definitions

- Copy all type definitions in order
- Include all associated constants
- Include all methods on types

### Step 3: Update cpu.go

- Remove type definitions from cpu.go
- Verify no duplicate definitions remain

### Step 4: Verify

- Run `go build`
- Ensure no compilation errors
- Run test suite

## Testing Impact

- No test changes required
- Types remain in same package
- All references work unchanged

## Documentation

- Add package-level documentation
- Document each type's purpose
- Include usage examples for complex types

## Benefits

1. **Single Source of Truth**: All types in one place
2. **Easy Reference**: Developers can quickly find type definitions
3. **Better Documentation**: Each type can have focused docs
4. **Reduced Clutter**: Main CPU file becomes cleaner
