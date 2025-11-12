# Remaining Modules Plan

## 1. bus.go (~30 lines)

### Purpose

Define the Bus interface for memory access abstraction.

### Contents (Lines 168-193)

```go
// Bus defines the interface for memory access
type Bus interface {
    Read(addr uint16) uint8
    Write(addr uint16, data uint8)
}
```

### Documentation

- Comprehensive interface documentation
- Usage examples
- Implementation guidelines
- Memory mapping patterns

### Benefits

- Clear separation of memory abstraction
- Easy to reference for implementers
- Focused documentation

---

## 2. config.go (~200 lines)

### Purpose

CPU configuration, builder pattern, and factory functions.

### Contents

#### Configuration Structure (Lines 195-226)

```go
type CPUConfig struct {
    Variant                CPUVariant
    ErrorHandler           ErrorHandler
    StrictMode             bool
    EnableDecimalMode      bool
    EnableInstructionCache bool
    InstructionCacheSize   int
}

func DefaultConfig() CPUConfig
```

#### Builder Pattern (Lines 465-518)

```go
type CPUBuilder struct {
    bus    Bus
    config CPUConfig
}

func NewBuilder(bus Bus) *CPUBuilder
func (b *CPUBuilder) WithVariant(variant CPUVariant) *CPUBuilder
func (b *CPUBuilder) WithStrictMode() *CPUBuilder
func (b *CPUBuilder) WithErrorHandler(handler ErrorHandler) *CPUBuilder
func (b *CPUBuilder) DisableDecimalMode() *CPUBuilder
func (b *CPUBuilder) DisableInstructionCache() *CPUBuilder
func (b *CPUBuilder) WithInstructionCacheSize(size int) *CPUBuilder
func (b *CPUBuilder) Build() *CPU
```

#### Factory Functions (Lines 393-463)

```go
func NewCPU(bus Bus) *CPU
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU
func NewCPUWithErrorHandler(bus Bus, handler ErrorHandler) *CPU
func NewCPUWithVariantAndErrorHandler(bus Bus, variant CPUVariant, handler ErrorHandler) *CPU
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU
```

### Dependencies

- Bus interface
- CPUVariant type
- ErrorHandler interface
- CPU struct (forward reference)

### Migration Notes

- Keep factory functions that create CPU instances
- Builder pattern provides fluent configuration
- DefaultConfig provides sensible defaults

---

## 3. cache.go (~100 lines)

### Purpose

Instruction cache implementation for performance optimization.

### Contents (Lines 252-312)

#### Cache Entry Structure

```go
type InstructionCacheEntry struct {
    opcode      uint8
    instruction *Instruction
    valid       bool
}
```

#### Cache Implementation

```go
type InstructionCache struct {
    entries [256]InstructionCacheEntry
    hits    uint64
    misses  uint64
}

func NewInstructionCache() *InstructionCache
func (ic *InstructionCache) Lookup(pc uint16, opcode uint8) (*Instruction, bool)
func (ic *InstructionCache) Store(pc uint16, opcode uint8, instruction *Instruction)
func (ic *InstructionCache) Invalidate()
func (ic *InstructionCache) Stats() (hits, misses uint64, hitRate float64)
```

### Dependencies

- Instruction type

### Benefits

- Performance optimization isolated
- Easy to understand cache behavior
- Statistics tracking separate from CPU logic

---

## 4. addressing.go (~150 lines)

### Purpose

All addressing mode implementations.

### Contents (Lines 571-697)

#### Addressing Mode Methods

```go
func (c *CPU) IMP() uint8  // Implied
func (c *CPU) IMM() uint8  // Immediate
func (c *CPU) ZP0() uint8  // Zero Page
func (c *CPU) ZPX() uint8  // Zero Page, X
func (c *CPU) ZPY() uint8  // Zero Page, Y
func (c *CPU) REL() uint8  // Relative
func (c *CPU) ABS() uint8  // Absolute
func (c *CPU) ABX() uint8  // Absolute, X
func (c *CPU) ABY() uint8  // Absolute, Y
func (c *CPU) IND() uint8  // Indirect
func (c *CPU) IZX() uint8  // Indexed Indirect
func (c *CPU) IZY() uint8  // Indirect Indexed
```

### Dependencies

- CPU struct
- CPUVariant (for IND bug behavior)

### Documentation

- Each addressing mode explained
- Page crossing behavior documented
- Variant-specific behavior noted

---

## 5. lookup.go (~350 lines)

### Purpose

Instruction lookup table initialization.

### Contents (Lines 1248-1581)

#### Main Function

```go
func (c *CPU) buildLookupTable()
```

#### Structure

- Method expression assignments
- Official opcode definitions
- Unofficial/illegal opcode definitions
- Page cross penalty documentation

### Dependencies

- All instruction methods
- All addressing mode methods
- Instruction type
- AddrModeType enum

### Migration Notes

- Large but straightforward
- Mostly data initialization
- Includes comprehensive comments about page crossing

---

## 6. interrupts.go (~150 lines)

### Purpose

Interrupt handling (IRQ, NMI, BRK).

### Contents

#### Interrupt State Management (Lines 1614-1661)

```go
func (c *CPU) SetIRQ(asserted bool)
func (c *CPU) SetNMI(asserted bool)
func (c *CPU) ClearNMI()
func (c *CPU) HasPendingInterrupt() bool
```

#### Deprecated Methods (Lines 1614-1633)

```go
func (c *CPU) InterruptRequest()
func (c *CPU) NonMaskableInterrupt()
```

#### Interrupt Handlers (Lines 1663-1730)

```go
func (c *CPU) handleNMI() error
func (c *CPU) handleIRQ() error
```

### Dependencies

- CPU struct
- Flags
- Stack operations

### Documentation

- Interrupt timing explained
- Edge vs level triggering
- Vector addresses documented

---

## 7. state.go (~150 lines)

### Purpose

State inspection and snapshot functionality.

### Contents (Lines 1826-1937, 1958-2009)

#### State Methods

```go
func (c *CPU) RemainingCycles() uint8
func (c *CPU) SetCycles(cycles uint8)
func (c *CPU) TotalCycles() uint64
func (c *CPU) CurrentOpcode() uint8
func (c *CPU) LookupInstruction(opcode uint8) Instruction
func (c *CPU) IsIllegalOpcode(opcode uint8) bool
func (c *CPU) GetStateSnapshot() State
func (c *CPU) GetState() string
func (c *CPU) LastError() *CPUError
func (c *CPU) Variant() CPUVariant
```

#### Helper Functions

```go
func FormatFlags(p Flags) string
```

### Dependencies

- CPU struct
- State type
- Flags type

---

## 8. debug.go (~200 lines)

### Purpose

Debugging and disassembly tools.

### Contents (Lines 1939-2129)

#### Debug Methods

```go
func (c *CPU) GetCurrentInstruction() *Instruction
func (c *CPU) Opcode() uint8  // Deprecated
func (c *CPU) LookupTable() [256]Instruction
```

#### Disassembler

```go
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string
```

### Dependencies

- CPU struct
- Instruction type
- AddrModeType enum

### Documentation

- Disassembly format explained
- Usage examples
- Debugging workflows

---

## 9. cpu.go (Core - ~300 lines)

### Purpose

Main CPU struct and core execution logic.

### Contents

#### CPU Structure (Lines 336-379)

```go
type CPU struct {
    // Public registers
    A  uint8
    X  uint8
    Y  uint8
    SP uint8
    PC uint16
    P  Flags
    
    // Bus connection
    bus Bus
    
    // Private internal state
    cycles             uint8
    opcode             uint8
    fetchedData        uint8
    addrAbs            uint16
    addrRel            uint16
    currentInstruction *Instruction
    lookup             [256]Instruction
    totalCycles        uint64
    
    // Configuration
    errorHandler ErrorHandler
    lastError    *CPUError
    variant      CPUVariant
    
    // Interrupt state
    irqLine         bool
    nmiLine         bool
    nmiPrevious     bool
    nmiPending      bool
    inInterrupt     bool
    interruptVector uint16
    
    // Performance
    instrCache *InstructionCache
}
```

#### Core Operations (Lines 525-569)

```go
func (c *CPU) read(addr uint16) uint8
func (c *CPU) write(addr uint16, data uint8)
func (c *CPU) push(data uint8)
func (c *CPU) pop() uint8
func (c *CPU) push16(data uint16)
func (c *CPU) pop16() uint16
func (c *CPU) setFlag(flag Flags, value bool)
func (c *CPU) getFlag(flag Flags) bool
```

#### Core Execution (Lines 1585-1824)

```go
func (c *CPU) Reset()
func (c *CPU) Clock() error
```

#### Cache Control (Lines 1859-1887)

```go
func (c *CPU) InvalidateInstructionCache()
func (c *CPU) InstructionCacheStats() (hits, misses uint64, hitRate float64)
func (c *CPU) DisableInstructionCache()
func (c *CPU) EnableInstructionCache()
```

### Dependencies

- All other modules (imports them)
- Bus interface
- All types

### What Remains in cpu.go

1. CPU struct definition
2. Core execution loop (Clock, Reset)
3. Basic memory/stack operations
4. Flag operations
5. Cache control methods

---

## Migration Order

### Recommended Sequence

1. **types.go** - No dependencies
2. **errors.go** - No dependencies
3. **bus.go** - No dependencies
4. **cache.go** - Depends on types
5. **config.go** - Depends on types, errors, bus
6. **helpers.go** - Depends on CPU struct
7. **addressing.go** - Depends on CPU struct
8. **instructions_*.go** - Depends on CPU struct, helpers
9. **lookup.go** - Depends on all instructions
10. **interrupts.go** - Depends on CPU struct
11. **state.go** - Depends on CPU struct
12. **debug.go** - Depends on CPU struct
13. **cpu.go** - Final cleanup, imports all modules

---

## Testing Strategy

### After Each Module

1. Run `go build` to check compilation
2. Run `go test` to verify tests pass
3. Check for import cycles
4. Verify no duplicate definitions

### Final Validation

1. Full test suite passes
2. All examples work
3. Benchmarks show no regression
4. Documentation builds correctly

---

## File Size Summary

| Module | Lines | Status |
|--------|-------|--------|
| types.go | ~150 | New |
| errors.go | ~100 | New |
| bus.go | ~30 | New |
| config.go | ~200 | New |
| cache.go | ~100 | New |
| addressing.go | ~150 | New |
| helpers.go | ~50 | New |
| instructions_flags.go | ~50 | New |
| instructions_stack.go | ~60 | New |
| instructions_logical.go | ~100 | New |
| instructions_branch.go | ~80 | New |
| instructions_transfer.go | ~120 | New |
| instructions_shift.go | ~150 | New |
| instructions_control.go | ~100 | New |
| instructions_arithmetic.go | ~200 | New |
| lookup.go | ~350 | New |
| interrupts.go | ~150 | New |
| state.go | ~150 | New |
| debug.go | ~200 | New |
| cpu.go | ~300 | Refactored |
| **Total** | **~2,890** | - |

Note: Total is higher than original due to added documentation and spacing for readability.

---

## Benefits Summary

### Maintainability

- Each file has single, clear purpose
- Easy to locate specific functionality
- Smaller files easier to understand

### Testability

- Test individual modules in isolation
- Focused test files per module
- Better test organization

### Extensibility

- Clear patterns for adding features
- Easy to add new instructions
- Simple to extend with new variants

### Documentation

- Each module can have focused docs
- Better code examples
- Clearer API surface

### Development

- Multiple developers can work in parallel
- Reduced merge conflicts
- Faster code reviews
