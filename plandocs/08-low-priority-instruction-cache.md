# Low Priority: Add Instruction Cache

## Problem Statement

The CPU repeatedly looks up the same instructions from the same memory locations:

- Tight loops execute the same instructions repeatedly
- Lookup table access happens for every instruction fetch
- No optimization for frequently executed code paths

**Performance Impact**: Moderate - most time is spent in instruction execution, not lookup.

## Proposed Solution

Add a small instruction cache to optimize repeated instruction fetches.

### Implementation Steps

#### Step 1: Define Cache Structure

```go
// InstructionCacheEntry represents a cached instruction
type InstructionCacheEntry struct {
    opcode      uint8
    instruction *Instruction
    valid       bool
}

// InstructionCache provides fast lookup for recently executed instructions
type InstructionCache struct {
    entries [256]InstructionCacheEntry // Direct-mapped cache
    hits    uint64
    misses  uint64
}

// NewInstructionCache creates a new instruction cache
func NewInstructionCache() *InstructionCache {
    return &InstructionCache{}
}

// Lookup attempts to find an instruction in the cache
func (ic *InstructionCache) Lookup(pc uint16, opcode uint8) (*Instruction, bool) {
    index := uint8(pc & 0xFF) // Use low byte of PC as cache index
    entry := &ic.entries[index]
    
    if entry.valid && entry.opcode == opcode {
        ic.hits++
        return entry.instruction, true
    }
    
    ic.misses++
    return nil, false
}

// Store adds an instruction to the cache
func (ic *InstructionCache) Store(pc uint16, opcode uint8, instruction *Instruction) {
    index := uint8(pc & 0xFF)
    ic.entries[index] = InstructionCacheEntry{
        opcode:      opcode,
        instruction: instruction,
        valid:       true,
    }
}

// Invalidate clears the cache (e.g., after self-modifying code)
func (ic *InstructionCache) Invalidate() {
    for i := range ic.entries {
        ic.entries[i].valid = false
    }
}

// Stats returns cache statistics
func (ic *InstructionCache) Stats() (hits, misses uint64, hitRate float64) {
    total := ic.hits + ic.misses
    if total == 0 {
        return 0, 0, 0.0
    }
    return ic.hits, ic.misses, float64(ic.hits) / float64(total)
}
```

#### Step 2: Add Cache to CPU

```go
type CPU struct {
    // ... existing fields ...
    
    // Performance optimization
    instrCache *InstructionCache
}

// NewCPU creates a new CPU with instruction cache enabled
func NewCPU(bus Bus) *CPU {
    return NewCPUWithConfig(bus, DefaultConfig())
}

// NewCPUWithConfig creates a new CPU with full configuration
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU {
    c := &CPU{
        bus:          bus,
        P:            U | I,
        SP:           0xFD,
        variant:      config.Variant,
        errorHandler: config.ErrorHandler,
        instrCache:   NewInstructionCache(), // NEW
    }
    
    c.buildLookupTable()
    return c
}
```

#### Step 3: Update Clock() to Use Cache

```go
// Clock executes one clock cycle of the CPU
func (c *CPU) Clock() error {
    if c.cycles == 0 {
        // ... interrupt handling ...
        
        // Fetch opcode
        c.opcode = c.read(c.PC)
        fetchPC := c.PC // Save PC for cache lookup
        c.PC++
        
        c.setFlag(U, true)
        
        // Try cache lookup first
        if instr, hit := c.instrCache.Lookup(fetchPC, c.opcode); hit {
            c.currentInstruction = instr
        } else {
            // Cache miss - use lookup table
            c.currentInstruction = &c.lookup[c.opcode]
            // Store in cache for next time
            c.instrCache.Store(fetchPC, c.opcode, c.currentInstruction)
        }
        
        // ... rest of execution ...
    }
    
    c.cycles--
    c.totalCycles++
    return nil
}
```

#### Step 4: Add Cache Control Methods

```go
// InvalidateInstructionCache clears the instruction cache
// Call this after self-modifying code or when loading new programs
func (c *CPU) InvalidateInstructionCache() {
    if c.instrCache != nil {
        c.instrCache.Invalidate()
    }
}

// InstructionCacheStats returns cache performance statistics
func (c *CPU) InstructionCacheStats() (hits, misses uint64, hitRate float64) {
    if c.instrCache != nil {
        return c.instrCache.Stats()
    }
    return 0, 0, 0.0
}

// DisableInstructionCache disables the instruction cache
func (c *CPU) DisableInstructionCache() {
    c.instrCache = nil
}

// EnableInstructionCache enables the instruction cache
func (c *CPU) EnableInstructionCache() {
    if c.instrCache == nil {
        c.instrCache = NewInstructionCache()
    }
}
```

#### Step 5: Add Cache Configuration

```go
type CPUConfig struct {
    // ... existing fields ...
    
    // EnableInstructionCache enables instruction caching for performance
    EnableInstructionCache bool
    
    // InstructionCacheSize sets the cache size (default: 256 entries)
    InstructionCacheSize int
}

func DefaultConfig() CPUConfig {
    return CPUConfig{
        // ... existing defaults ...
        EnableInstructionCache: true,
        InstructionCacheSize:   256,
    }
}
```

## Alternative: Simpler Approach

If full cache is too complex, use a simpler "last instruction" cache:

```go
type CPU struct {
    // ... existing fields ...
    
    // Simple last-instruction cache
    lastPC          uint16
    lastOpcode      uint8
    lastInstruction *Instruction
}

func (c *CPU) Clock() error {
    if c.cycles == 0 {
        // ... interrupt handling ...
        
        c.opcode = c.read(c.PC)
        fetchPC := c.PC
        c.PC++
        
        // Check if same as last instruction
        if fetchPC == c.lastPC && c.opcode == c.lastOpcode && c.lastInstruction != nil {
            c.currentInstruction = c.lastInstruction
        } else {
            c.currentInstruction = &c.lookup[c.opcode]
            c.lastPC = fetchPC
            c.lastOpcode = c.opcode
            c.lastInstruction = c.currentInstruction
        }
        
        // ... rest of execution ...
    }
    
    c.cycles--
    c.totalCycles++
    return nil
}
```

## Performance Analysis

### Expected Cache Hit Rates

- **Tight Loops**: 90-95% hit rate
- **General Code**: 60-70% hit rate
- **Branching Code**: 40-50% hit rate

### Expected Performance Improvement

- **Best Case** (tight loops): 10-15% faster
- **Average Case**: 5-8% faster
- **Worst Case** (no loops): 0-2% faster

### Memory Overhead

- **Full Cache**: 256 entries × 24 bytes = 6KB per CPU instance
- **Simple Cache**: 1 entry × 24 bytes = 24 bytes per CPU instance

## Testing Strategy

1. **Cache Hit Tests**: Verify cache hits on repeated instructions
2. **Cache Miss Tests**: Verify cache misses on new instructions
3. **Invalidation Tests**: Verify cache invalidation works
4. **Performance Tests**: Benchmark with/without cache
5. **Correctness Tests**: Ensure cache doesn't affect behavior

### Benchmark Tests

```go
func BenchmarkWithCache(b *testing.B) {
    cpu, bus := setupCPU()
    cpu.EnableInstructionCache()
    
    // Tight loop program
    program := []uint8{
        0xA9, 0x00,       // LDA #$00
        0x69, 0x01,       // ADC #$01  ; Loop start
        0xC9, 0x10,       // CMP #$10
        0xD0, 0xFA,       // BNE Loop
        0x00,             // BRK
    }
    bus.load(0x8000, program)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cpu.PC = 0x8000
        cpu.Cycles = 0
        runUntilBrk(cpu, bus, 1000)
    }
}

func BenchmarkWithoutCache(b *testing.B) {
    cpu, bus := setupCPU()
    cpu.DisableInstructionCache()
    
    // Same program as above
    // ... 
}
```

## Success Criteria

- [ ] Cache structure implemented
- [ ] Cache lookup integrated into Clock()
- [ ] Cache invalidation methods added
- [ ] Cache statistics tracking implemented
- [ ] Configuration option added
- [ ] Benchmarks show performance improvement
- [ ] All tests pass with cache enabled/disabled
- [ ] No behavioral changes (cache is transparent)

## Notes

- This is LOW priority because the performance gain is modest
- Consider implementing only if profiling shows lookup is a bottleneck
- The simpler "last instruction" cache may be sufficient
- Cache invalidation is critical for self-modifying code support
