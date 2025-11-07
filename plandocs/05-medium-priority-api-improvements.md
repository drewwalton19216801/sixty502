# Medium Priority: API Improvements

## Problem Statement

Several API design issues affect usability and maintainability:

1. **Exported Internal Fields**: Fields like `Cycles`, `lookup`, `totalCycles` are exported but should be private
2. **No Configuration Options**: No way to configure CPU behavior at creation time
3. **Limited Accessor Methods**: Missing methods to safely inspect CPU state
4. **No Builder Pattern**: Complex initialization requires multiple steps

## Proposed Solution

Improve API design with proper encapsulation and configuration options.

### Implementation Steps

#### Step 1: Make Internal Fields Private

```go
type CPU struct {
    // Public registers (keep exported for direct access)
    A  uint8
    X  uint8
    Y  uint8
    SP uint8
    PC uint16
    P  Flags

    // Bus connection (keep exported)
    bus Bus

    // PRIVATE: Internal state (rename with lowercase)
    cycles             uint8        // Was: Cycles
    opcode             uint8        // Already private
    fetchedData        uint8        // Already private
    addrAbs            uint16       // Already private
    addrRel            uint16       // Already private
    currentInstruction *Instruction // Already private
    lookup             [256]Instruction // Was: lookup (make private)
    totalCycles        uint64       // Was: totalCycles (make private)
    
    // Configuration
    variant      CPUVariant
    errorHandler ErrorHandler
    lastError    *CPUError
}
```

#### Step 2: Add Accessor Methods

```go
// RemainingCycles returns the number of cycles remaining for the current instruction
func (c *CPU) RemainingCycles() uint8 {
    return c.cycles
}

// TotalCycles returns the total number of cycles executed
func (c *CPU) TotalCycles() uint64 {
    return c.totalCycles
}

// CurrentOpcode returns the opcode of the currently executing instruction
func (c *CPU) CurrentOpcode() uint8 {
    return c.opcode
}

// LookupInstruction returns the instruction definition for a given opcode
func (c *CPU) LookupInstruction(opcode uint8) Instruction {
    return c.lookup[opcode]
}

// IsIllegalOpcode returns true if the given opcode is illegal/unofficial
func (c *CPU) IsIllegalOpcode(opcode uint8) bool {
    return c.lookup[opcode].Illegal
}
```

#### Step 3: Create Configuration Struct

```go
// CPUConfig holds configuration options for CPU creation
type CPUConfig struct {
    // Variant specifies the CPU variant (NMOS, CMOS, Ricoh)
    Variant CPUVariant
    
    // ErrorHandler defines how errors are handled
    ErrorHandler ErrorHandler
    
    // StrictMode halts execution on illegal opcodes
    StrictMode bool
    
    // EnableDecimalMode allows disabling decimal mode even on variants that support it
    EnableDecimalMode bool
    
    // LogInstructions enables instruction-level logging
    LogInstructions bool
    
    // InstructionLogger is called for each instruction if LogInstructions is true
    InstructionLogger func(pc uint16, opcode uint8, mnemonic string)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() CPUConfig {
    return CPUConfig{
        Variant:           VariantNMOS6502,
        ErrorHandler:      &LoggingErrorHandler{Logger: log.Default()},
        StrictMode:        false,
        EnableDecimalMode: true,
        LogInstructions:   false,
        InstructionLogger: nil,
    }
}
```

#### Step 4: Update Constructors

```go
// NewCPU creates a new CPU with default configuration
func NewCPU(bus Bus) *CPU {
    return NewCPUWithConfig(bus, DefaultConfig())
}

// NewCPUWithVariant creates a new CPU with specified variant
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU {
    config := DefaultConfig()
    config.Variant = variant
    return NewCPUWithConfig(bus, config)
}

// NewCPUWithConfig creates a new CPU with full configuration
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU {
    // Apply strict mode to error handler if requested
    errorHandler := config.ErrorHandler
    if config.StrictMode {
        errorHandler = &StrictErrorHandler{}
    }
    
    c := &CPU{
        bus:          bus,
        P:            U | I,
        SP:           0xFD,
        variant:      config.Variant,
        errorHandler: errorHandler,
    }
    
    c.buildLookupTable()
    
    // Apply decimal mode configuration
    if !config.EnableDecimalMode {
        c.P &^= D // Clear decimal flag
    }
    
    return c
}
```

#### Step 5: Add Builder Pattern (Optional)

```go
// CPUBuilder provides a fluent interface for CPU configuration
type CPUBuilder struct {
    bus    Bus
    config CPUConfig
}

// NewBuilder creates a new CPU builder
func NewBuilder(bus Bus) *CPUBuilder {
    return &CPUBuilder{
        bus:    bus,
        config: DefaultConfig(),
    }
}

// WithVariant sets the CPU variant
func (b *CPUBuilder) WithVariant(variant CPUVariant) *CPUBuilder {
    b.config.Variant = variant
    return b
}

// WithStrictMode enables strict mode
func (b *CPUBuilder) WithStrictMode() *CPUBuilder {
    b.config.StrictMode = true
    return b
}

// WithErrorHandler sets a custom error handler
func (b *CPUBuilder) WithErrorHandler(handler ErrorHandler) *CPUBuilder {
    b.config.ErrorHandler = handler
    return b
}

// WithInstructionLogging enables instruction logging
func (b *CPUBuilder) WithInstructionLogging(logger func(pc uint16, opcode uint8, mnemonic string)) *CPUBuilder {
    b.config.LogInstructions = true
    b.config.InstructionLogger = logger
    return b
}

// DisableDecimalMode disables decimal mode
func (b *CPUBuilder) DisableDecimalMode() *CPUBuilder {
    b.config.EnableDecimalMode = false
    return b
}

// Build creates the configured CPU
func (b *CPUBuilder) Build() *CPU {
    return NewCPUWithConfig(b.bus, b.config)
}
```

#### Step 6: Add State Inspection Methods

```go
// State represents a snapshot of CPU state
type State struct {
    A           uint8
    X           uint8
    Y           uint8
    SP          uint8
    PC          uint16
    P           Flags
    Cycles      uint8
    TotalCycles uint64
    Opcode      uint8
    Instruction string
}

// GetStateSnapshot returns a snapshot of the current CPU state
func (c *CPU) GetStateSnapshot() State {
    instrName := "???"
    if c.currentInstruction != nil {
        instrName = c.currentInstruction.Name
    }
    
    return State{
        A:           c.A,
        X:           c.X,
        Y:           c.Y,
        SP:          c.SP,
        PC:          c.PC,
        P:           c.P,
        Cycles:      c.cycles,
        TotalCycles: c.totalCycles,
        Opcode:      c.opcode,
        Instruction: instrName,
    }
}

// String returns a human-readable representation of the state
func (s State) String() string {
    return fmt.Sprintf(
        "PC:%04X A:%02X X:%02X Y:%02X P:%02X[%s] SP:%02X CYC:%d (%s $%02X)",
        s.PC, s.A, s.X, s.Y, uint8(s.P), FormatFlags(s.P),
        s.SP, s.TotalCycles, s.Instruction, s.Opcode,
    )
}
```

## Usage Examples

### Example 1: Simple Creation (Backward Compatible)

```go
bus := &SimpleBus{}
cpu := NewCPU(bus)
cpu.Reset()
```

### Example 2: With Configuration

```go
bus := &SimpleBus{}
config := DefaultConfig()
config.Variant = VariantRicoh2A03
config.StrictMode = true
cpu := NewCPUWithConfig(bus, config)
cpu.Reset()
```

### Example 3: Using Builder Pattern

```go
bus := &SimpleBus{}
cpu := NewBuilder(bus).
    WithVariant(VariantCMOS65C02).
    WithStrictMode().
    WithInstructionLogging(func(pc uint16, opcode uint8, mnemonic string) {
        fmt.Printf("$%04X: %s ($%02X)\n", pc, mnemonic, opcode)
    }).
    Build()
cpu.Reset()
```

### Example 4: State Inspection

```go
// Get current state
state := cpu.GetStateSnapshot()
fmt.Println(state)

// Check specific values
if state.Cycles == 0 {
    fmt.Println("Ready for next instruction")
}

// Access total cycles
fmt.Printf("Total cycles: %d\n", cpu.TotalCycles())
```

## Migration Guide

### Breaking Changes

1. `cpu.Cycles` → `cpu.RemainingCycles()`
2. `cpu.totalCycles` → `cpu.TotalCycles()`
3. `cpu.lookup` → `cpu.LookupInstruction(opcode)`

### Migration Script

```go
// Before
cycles := cpu.Cycles
total := cpu.totalCycles
instr := cpu.lookup[0xA9]

// After
cycles := cpu.RemainingCycles()
total := cpu.TotalCycles()
instr := cpu.LookupInstruction(0xA9)
```

## Testing Strategy

1. **Accessor Tests**: Verify all accessor methods return correct values
2. **Configuration Tests**: Test all configuration options
3. **Builder Tests**: Verify builder pattern works correctly
4. **Backward Compatibility**: Ensure existing code still works
5. **State Snapshot Tests**: Verify state capture is accurate

## Success Criteria

- [ ] Internal fields made private
- [ ] Accessor methods implemented
- [ ] Configuration struct created
- [ ] Builder pattern implemented
- [ ] State inspection methods added
- [ ] All tests pass
- [ ] Migration guide provided
- [ ] Backward compatibility maintained where possible
