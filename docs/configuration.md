# Configuration Guide

This guide covers all configuration options for the sixty502 emulator.

## Configuration Overview

The sixty502 emulator provides flexible configuration through the [`CPUConfig`](config.go:42) struct and builder pattern.

## CPUConfig Structure

```go
type CPUConfig struct {
    Variant                CPUVariant
    ErrorHandler           ErrorHandler
    StrictMode             bool
    EnableDecimalMode      bool
    EnableInstructionCache bool
    InstructionCacheSize   int
}
```

### Variant

Selects the CPU variant to emulate.

**Options**:

- `VariantNMOS6502` - Original NMOS 6502 (Rev B+) [default]
- `VariantNMOS6502RevA` - Early NMOS 6502 Rev A (ROR quirk)
- `VariantCMOS65C02` - CMOS 65C02 (enhanced)
- `VariantRicoh2A03` - NES/Famicom CPU (NTSC)
- `VariantRicoh2A07` - PAL NES CPU

**Impact**: Affects instruction behavior, bug emulation, and decimal mode support.

**Example**:

```go
config := cpu6502.DefaultConfig()
config.Variant = cpu6502.VariantCMOS65C02
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

See [CPU Variants](compatibility.md) for detailed variant differences.

### ErrorHandler

Defines how errors are handled during execution.

**Built-in Handlers**:

1. **LoggingErrorHandler** [default]
   - Logs errors but continues execution
   - Suitable for most emulation scenarios

2. **StrictErrorHandler**
   - Halts execution on any error
   - Useful for debugging and testing

**Example**:

```go
config := cpu6502.DefaultConfig()
config.ErrorHandler = &cpu6502.StrictErrorHandler{}
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

**Custom Handler**:

```go
type MyErrorHandler struct{}

func (h *MyErrorHandler) HandleError(err *cpu6502.CPUError) error {
    if err.Type == cpu6502.ErrorIllegalOpcode {
        log.Printf("Illegal opcode $%02X at $%04X", err.Opcode, err.PC)
        return nil // Continue execution
    }
    return err // Halt on other errors
}

config.ErrorHandler = &MyErrorHandler{}
```

### StrictMode

Halts execution on illegal opcodes, overriding the error handler.

**Default**: `false`

**When to use**:

- Testing with known-good code
- Debugging illegal opcode issues
- Strict compatibility testing

**Example**:

```go
config := cpu6502.DefaultConfig()
config.StrictMode = true
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

### EnableDecimalMode

Controls decimal mode (BCD) support.

**Default**: `true`

**When to disable**:

- Emulating systems without decimal mode (Ricoh 2A03/2A07)
- Slight performance improvement
- Testing binary-only code

**Note**: Ricoh variants automatically disable decimal mode regardless of this setting.

**Example**:

```go
config := cpu6502.DefaultConfig()
config.EnableDecimalMode = false
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

### EnableInstructionCache

Enables instruction caching for performance.

**Default**: `true`

**Performance Impact**: 5-15% improvement for code with loops

**When to disable**:

- Self-modifying code
- Dynamic code generation
- Cache invalidation is too complex

**Example**:

```go
config := cpu6502.DefaultConfig()
config.EnableInstructionCache = false
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

### InstructionCacheSize

Sets the cache size (currently fixed at 256 entries).

**Default**: `256`

**Note**: This option is reserved for future use. The cache size is currently fixed.

## Default Configuration

```go
func DefaultConfig() CPUConfig
```

Returns a configuration with sensible defaults:

```go
CPUConfig{
    Variant:                VariantNMOS6502,
    ErrorHandler:           &LoggingErrorHandler{Logger: log.Default()},
    StrictMode:             false,
    EnableDecimalMode:      true,
    EnableInstructionCache: true,
    InstructionCacheSize:   256,
}
```

## Configuration Methods

### Direct Configuration

```go
config := cpu6502.DefaultConfig()
config.Variant = cpu6502.VariantCMOS65C02
config.StrictMode = true
cpu := cpu6502.NewCPUWithConfig(bus, config)
```

### Convenience Functions

#### NewCPU

Default configuration.

```go
cpu := cpu6502.NewCPU(bus)
```

#### NewCPUWithVariant

Change only the variant.

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
```

#### NewCPUWithErrorHandler

Change only the error handler.

```go
handler := &MyErrorHandler{}
cpu := cpu6502.NewCPUWithErrorHandler(bus, handler)
```

#### NewCPUWithVariantAndErrorHandler

Change variant and error handler.

```go
cpu := cpu6502.NewCPUWithVariantAndErrorHandler(bus,
    cpu6502.VariantCMOS65C02, &MyErrorHandler{})
```

### Builder Pattern

Fluent interface for readable configuration.

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantCMOS65C02).
    WithStrictMode().
    DisableDecimalMode().
    DisableInstructionCache().
    Build()
```

**Builder Methods**:

- `WithVariant(variant CPUVariant) *CPUBuilder`
- `WithStrictMode() *CPUBuilder`
- `WithErrorHandler(handler ErrorHandler) *CPUBuilder`
- `DisableDecimalMode() *CPUBuilder`
- `DisableInstructionCache() *CPUBuilder`
- `WithInstructionCacheSize(size int) *CPUBuilder`
- `Build() *CPU`

## Configuration Examples

### Apple II Emulation

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantNMOS6502).
    Build()
```

### NES Emulation

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantRicoh2A03).
    DisableDecimalMode(). // Already disabled by variant
    Build()
```

### Commodore 64 Emulation

```go
cpu := cpu6502.NewCPU(bus) // Default NMOS 6502
```

### Development/Testing

```go
cpu := cpu6502.NewBuilder(bus).
    WithStrictMode().
    WithErrorHandler(&MyDebugHandler{}).
    Build()
```

### Self-Modifying Code

```go
cpu := cpu6502.NewBuilder(bus).
    DisableInstructionCache().
    Build()
```

### Maximum Performance

```go
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantRicoh2A03). // No decimal mode overhead
    Build()
```

## Runtime Configuration

Some settings can be changed at runtime:

### Instruction Cache

```go
// Disable cache
cpu.DisableInstructionCache()

// Enable cache
cpu.EnableInstructionCache()

// Invalidate cache after loading new code
cpu.InvalidateInstructionCache()

// Check cache performance
hits, misses, rate := cpu.InstructionCacheStats()
```

### Interrupts

```go
// Set IRQ line
cpu.SetIRQ(true)
cpu.SetIRQ(false)

// Trigger NMI
cpu.SetNMI(true)
cpu.SetNMI(false)
```

### Flags

Direct register access allows runtime flag modification:

```go
// Enable decimal mode
cpu.P |= cpu6502.D

// Disable interrupts
cpu.P |= cpu6502.I

// Clear carry flag
cpu.P &^= cpu6502.C
```

## Configuration Best Practices

### 1. Choose the Right Variant

Match the variant to your target system for accurate emulation:

```go
// For Apple IIc
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)

// For NES
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantRicoh2A03)
```

### 2. Use Strict Mode During Development

Catch illegal opcodes early:

```go
cpu := cpu6502.NewBuilder(bus).
    WithStrictMode().
    Build()
```

### 3. Profile Before Disabling Cache

Measure cache effectiveness before disabling:

```go
hits, misses, rate := cpu.InstructionCacheStats()
if rate < 0.5 {
    cpu.DisableInstructionCache()
}
```

### 4. Implement Custom Error Handlers

Tailor error handling to your needs:

```go
type GameErrorHandler struct {
    logger *log.Logger
}

func (h *GameErrorHandler) HandleError(err *cpu6502.CPUError) error {
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        h.logger.Printf("Game used illegal opcode: $%02X", err.Opcode)
        return nil // Continue
    default:
        return err // Halt
    }
}
```

### 5. Document Your Configuration

```go
// NES CPU configuration
// - Ricoh 2A03 variant (no decimal mode)
// - Lenient error handling (some games use illegal opcodes)
// - Instruction cache enabled (tight loops in games)
cpu := cpu6502.NewBuilder(bus).
    WithVariant(cpu6502.VariantRicoh2A03).
    WithErrorHandler(&NESErrorHandler{}).
    Build()
```

## Troubleshooting Configuration Issues

### Issue: Decimal Mode Not Working

**Cause**: Variant doesn't support decimal mode (Ricoh 2A03/2A07)

**Solution**: Use NMOS or CMOS variant

```go
cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantNMOS6502)
```

### Issue: Illegal Opcodes Halting Execution

**Cause**: Strict mode enabled or strict error handler

**Solution**: Use lenient error handler

```go
cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&cpu6502.LoggingErrorHandler{}).
    Build()
```

### Issue: Poor Performance

**Cause**: Instruction cache disabled or ineffective

**Solution**: Enable cache and check hit rate

```go
cpu.EnableInstructionCache()
hits, misses, rate := cpu.InstructionCacheStats()
fmt.Printf("Cache hit rate: %.1f%%\n", rate*100)
```

### Issue: Self-Modifying Code Not Working

**Cause**: Instruction cache not invalidated

**Solution**: Invalidate cache after code modification

```go
// Modify code
bus.Write(0x8000, newOpcode)

// Invalidate cache
cpu.InvalidateInstructionCache()
```

## See Also

- [API Reference](api-reference.md) - Complete API documentation
- [CPU Variants](compatibility.md) - Variant-specific behavior
- [Performance Guide](performance.md) - Optimization strategies
- [Error Handling](error-handling.md) - Error handling patterns
