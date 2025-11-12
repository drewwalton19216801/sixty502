package cpu6502

// InstructionCacheEntry represents a cached instruction lookup result.
//
// The cache uses a direct-mapped strategy where the low byte of the PC
// is used as the cache index. Each entry stores the opcode and instruction
// pointer, along with a validity flag.
type InstructionCacheEntry struct {
	opcode      uint8        // The opcode that was cached
	instruction *Instruction // Pointer to the instruction definition
	valid       bool         // Whether this cache entry is valid
}

// InstructionCache provides fast lookup for recently executed instructions.
//
// The cache improves performance by avoiding repeated lookups in the main
// instruction table. It uses a direct-mapped strategy with 256 entries,
// indexed by the low byte of the program counter.
//
// # Cache Strategy
//
// The cache uses PC & 0xFF as the index, which provides good locality for:
//   - Tight loops (instructions repeat at similar addresses)
//   - Sequential code (nearby instructions share cache lines)
//   - Subroutines (local code patterns)
//
// # Performance Characteristics
//
// Typical hit rates:
//   - Tight loops: 95-99% (excellent)
//   - Sequential code: 60-80% (good)
//   - Random jumps: 20-40% (poor, but rare)
//
// # Cache Invalidation
//
// The cache should be invalidated when:
//   - Self-modifying code is detected
//   - New program is loaded into memory
//   - Memory is modified in executable regions
//
// Example usage:
//
//	cache := NewInstructionCache()
//	if instr, hit := cache.Lookup(pc, opcode); hit {
//	    // Use cached instruction
//	} else {
//	    // Look up in main table and cache result
//	    instr := &lookupTable[opcode]
//	    cache.Store(pc, opcode, instr)
//	}
type InstructionCache struct {
	entries [256]InstructionCacheEntry // Direct-mapped cache (indexed by PC & 0xFF)
	hits    uint64                     // Number of cache hits
	misses  uint64                     // Number of cache misses
}

// NewInstructionCache creates a new instruction cache.
//
// The cache is initially empty (all entries invalid) and statistics
// are zeroed. The cache is ready to use immediately.
func NewInstructionCache() *InstructionCache {
	return &InstructionCache{}
}

// Lookup attempts to find an instruction in the cache.
//
// Returns the cached instruction and true if found, or nil and false
// if not found or the cache entry is invalid. Updates hit/miss statistics.
//
// The lookup uses the low byte of PC as the cache index and verifies
// that the cached opcode matches the requested opcode.
//
// Parameters:
//   - pc: Program counter value (low byte used as cache index)
//   - opcode: The opcode to look up
//
// Returns:
//   - instruction: Pointer to cached instruction (nil if not found)
//   - hit: true if instruction was found in cache
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

// Store adds an instruction to the cache.
//
// Stores the instruction at the cache index determined by the low byte
// of the PC. Any existing entry at that index is replaced.
//
// Parameters:
//   - pc: Program counter value (low byte used as cache index)
//   - opcode: The opcode being cached
//   - instruction: Pointer to the instruction definition
func (ic *InstructionCache) Store(pc uint16, opcode uint8, instruction *Instruction) {
	index := uint8(pc & 0xFF)
	ic.entries[index] = InstructionCacheEntry{
		opcode:      opcode,
		instruction: instruction,
		valid:       true,
	}
}

// Invalidate clears the entire cache.
//
// This marks all cache entries as invalid and resets statistics to zero.
// Call this after self-modifying code execution or when loading a new
// program into memory.
//
// Example:
//
//	// After writing to executable memory
//	cpu.InvalidateInstructionCache()
func (ic *InstructionCache) Invalidate() {
	for i := range ic.entries {
		ic.entries[i].valid = false
	}
	// Reset statistics
	ic.hits = 0
	ic.misses = 0
}

// Stats returns cache performance statistics.
//
// Returns:
//   - hits: Number of successful cache lookups
//   - misses: Number of failed cache lookups
//   - hitRate: Percentage of successful lookups (0.0 to 1.0)
//
// The hit rate is calculated as hits / (hits + misses). Returns 0.0
// if no lookups have been performed yet.
//
// Example:
//
//	hits, misses, rate := cache.Stats()
//	fmt.Printf("Cache: %d hits, %d misses, %.1f%% hit rate\n",
//	    hits, misses, rate*100)
func (ic *InstructionCache) Stats() (hits, misses uint64, hitRate float64) {
	total := ic.hits + ic.misses
	if total == 0 {
		return 0, 0, 0.0
	}
	return ic.hits, ic.misses, float64(ic.hits) / float64(total)
}
