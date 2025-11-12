package cpu6502

import (
	"testing"
)

// TestInstructionCacheHit verifies cache hits on repeated instructions
func TestInstructionCacheHit(t *testing.T) {
	cpu, bus := setupCPU()

	// Load a simple loop program
	program := []uint8{
		0xA9, 0x00, // LDA #$00  ; $8000
		0x69, 0x01, // ADC #$01  ; $8002 - Loop start
		0xC9, 0x10, // CMP #$10  ; $8004
		0xD0, 0xFA, // BNE $8002 ; $8006 - Branch back to $8002
		0x00, // BRK       ; $8008
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Clear cache stats
	cpu.InvalidateInstructionCache()

	// Run the program until BRK
	runUntilBrk(cpu, bus, 1000)

	// Check cache stats
	hits, misses, hitRate := cpu.InstructionCacheStats()

	// We should have some hits since the loop executes multiple times
	// The loop runs 16 times (0 to 15), with 4 instructions per iteration
	// First iteration: 4 misses, subsequent 15 iterations: mostly hits
	if hits == 0 {
		t.Errorf("Expected cache hits, got 0")
	}

	// Hit rate should be reasonable for a tight loop (at least 50%)
	if hitRate < 0.5 {
		t.Errorf("Expected hit rate > 0.5, got %.2f (hits=%d, misses=%d)", hitRate, hits, misses)
	}

	t.Logf("Cache stats: hits=%d, misses=%d, hit rate=%.2f%%", hits, misses, hitRate*100)
}

// TestInstructionCacheMiss verifies cache misses on new instructions
func TestInstructionCacheMiss(t *testing.T) {
	cpu, bus := setupCPU()

	// Load a program with unique instructions at different addresses
	program := []uint8{
		0xA9, 0x01, // LDA #$01  ; $8000
		0xA2, 0x02, // LDX #$02  ; $8002
		0xA0, 0x03, // LDY #$03  ; $8004
		0x00, // BRK       ; $8006
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Clear cache stats
	cpu.InvalidateInstructionCache()

	// Execute until BRK
	runUntilBrk(cpu, bus, 50)

	hits, misses, _ := cpu.InstructionCacheStats()

	// First time through, all should be misses
	if misses == 0 {
		t.Errorf("Expected cache misses for new instructions")
	}

	// Since this is a straight-line program (no loops), we shouldn't have many hits
	t.Logf("Straight-line program: hits=%d, misses=%d", hits, misses)
}

// TestInstructionCacheInvalidation verifies cache invalidation works
func TestInstructionCacheInvalidation(t *testing.T) {
	cpu, bus := setupCPU()

	// Load a simple program
	program := []uint8{
		0xA9, 0x00, // LDA #$00  ; $8000
		0x69, 0x01, // ADC #$01  ; $8002
		0x00, // BRK       ; $8004
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Run once to populate cache
	runUntilBrk(cpu, bus, 50)

	hits1, misses1, _ := cpu.InstructionCacheStats()

	// Invalidate cache
	cpu.InvalidateInstructionCache()

	// Check that stats were cleared
	hitsAfterInvalidate, missesAfterInvalidate, _ := cpu.InstructionCacheStats()
	if hitsAfterInvalidate != 0 || missesAfterInvalidate != 0 {
		t.Errorf("Expected stats to be cleared after invalidation, got hits=%d, misses=%d", hitsAfterInvalidate, missesAfterInvalidate)
	}

	// Reset and run again
	bus.load(0x8000, program) // Reload program
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	runUntilBrk(cpu, bus, 50)

	hits2, misses2, _ := cpu.InstructionCacheStats()

	// After invalidation and re-run, we should have misses again
	if misses2 == 0 {
		t.Errorf("Expected cache misses after invalidation and re-run")
	}

	t.Logf("Before invalidation: hits=%d, misses=%d", hits1, misses1)
	t.Logf("After invalidation and re-run: hits=%d, misses=%d", hits2, misses2)
}

// TestInstructionCacheDisable verifies cache can be disabled
func TestInstructionCacheDisable(t *testing.T) {
	cpu, bus := setupCPU()

	// Disable cache
	cpu.DisableInstructionCache()

	// Load a simple loop program
	program := []uint8{
		0xA9, 0x00, // LDA #$00  ; $8000
		0x69, 0x01, // ADC #$01  ; $8002
		0xC9, 0x05, // CMP #$05  ; $8004
		0xD0, 0xFA, // BNE $8002 ; $8006
		0x00, // BRK       ; $8008
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Run the program
	runUntilBrk(cpu, bus, 200)

	// Check cache stats - should be all zeros
	hits, misses, hitRate := cpu.InstructionCacheStats()

	if hits != 0 || misses != 0 || hitRate != 0 {
		t.Errorf("Expected zero cache stats with disabled cache, got hits=%d, misses=%d, rate=%.2f", hits, misses, hitRate)
	}
}

// TestInstructionCacheEnable verifies cache can be re-enabled
func TestInstructionCacheEnable(t *testing.T) {
	cpu, bus := setupCPU()

	// Disable then re-enable cache
	cpu.DisableInstructionCache()
	cpu.EnableInstructionCache()

	// Load a simple loop program
	program := []uint8{
		0xA9, 0x00, // LDA #$00  ; $8000
		0x69, 0x01, // ADC #$01  ; $8002
		0xC9, 0x05, // CMP #$05  ; $8004
		0xD0, 0xFA, // BNE $8002 ; $8006
		0x00, // BRK       ; $8008
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Run the program
	runUntilBrk(cpu, bus, 200)

	// Check cache stats - should have hits
	hits, misses, hitRate := cpu.InstructionCacheStats()

	if hits == 0 {
		t.Errorf("Expected cache hits after re-enabling cache")
	}

	if hitRate < 0.3 {
		t.Errorf("Expected reasonable hit rate after re-enabling cache, got %.2f", hitRate)
	}

	t.Logf("Cache stats after re-enable: hits=%d, misses=%d, rate=%.2f%%", hits, misses, hitRate*100)
}

// TestInstructionCacheConfig verifies cache can be disabled via config
func TestInstructionCacheConfig(t *testing.T) {
	bus := NewMockBus()

	// Create CPU with cache disabled
	config := DefaultConfig()
	config.EnableInstructionCache = false
	cpu := NewCPUWithConfig(bus, config)

	// Load a simple loop program
	program := []uint8{
		0xA9, 0x00, // LDA #$00  ; $8000
		0x69, 0x01, // ADC #$01  ; $8002
		0xC9, 0x05, // CMP #$05  ; $8004
		0xD0, 0xFA, // BNE $8002 ; $8006
		0x00, // BRK       ; $8008
	}
	bus.load(0x8000, program)
	cpu.PC = 0x8000
	cpu.SetCycles(0)

	// Run the program
	runUntilBrk(cpu, bus, 200)

	// Check cache stats - should be all zeros
	hits, misses, hitRate := cpu.InstructionCacheStats()

	if hits != 0 || misses != 0 || hitRate != 0 {
		t.Errorf("Expected zero cache stats with disabled cache via config, got hits=%d, misses=%d, rate=%.2f", hits, misses, hitRate)
	}
}

// TestInstructionCacheCorrectness verifies cache doesn't affect CPU behavior
func TestInstructionCacheCorrectness(t *testing.T) {
	bus1 := NewMockBus()
	bus2 := NewMockBus()

	// Create two CPUs - one with cache, one without
	config1 := DefaultConfig()
	config1.EnableInstructionCache = true
	cpu1 := NewCPUWithConfig(bus1, config1)

	config2 := DefaultConfig()
	config2.EnableInstructionCache = false
	cpu2 := NewCPUWithConfig(bus2, config2)

	// Load same program to both
	program := []uint8{
		0xA9, 0x05, // LDA #$05  ; $8000
		0x69, 0x03, // ADC #$03  ; $8002
		0x85, 0x10, // STA $10   ; $8004
		0xA2, 0x02, // LDX #$02  ; $8006
		0xE8,       // INX       ; $8008
		0x86, 0x11, // STX $11   ; $8009
		0x00, // BRK       ; $800B
	}
	bus1.load(0x8000, program)
	bus2.load(0x8000, program)
	cpu1.PC = 0x8000
	cpu1.SetCycles(0)
	cpu2.PC = 0x8000
	cpu2.SetCycles(0)

	// Run both CPUs
	runUntilBrk(cpu1, bus1, 100)
	runUntilBrk(cpu2, bus2, 100)

	// Compare final states
	if cpu1.A != cpu2.A {
		t.Errorf("A register mismatch: cached=%02X, uncached=%02X", cpu1.A, cpu2.A)
	}
	if cpu1.X != cpu2.X {
		t.Errorf("X register mismatch: cached=%02X, uncached=%02X", cpu1.X, cpu2.X)
	}
	if cpu1.Y != cpu2.Y {
		t.Errorf("Y register mismatch: cached=%02X, uncached=%02X", cpu1.Y, cpu2.Y)
	}
	if cpu1.P != cpu2.P {
		t.Errorf("P register mismatch: cached=%02X, uncached=%02X", cpu1.P, cpu2.P)
	}
	if bus1.Read(0x10) != bus2.Read(0x10) {
		t.Errorf("Memory $10 mismatch: cached=%02X, uncached=%02X", bus1.Read(0x10), bus2.Read(0x10))
	}
	if bus1.Read(0x11) != bus2.Read(0x11) {
		t.Errorf("Memory $11 mismatch: cached=%02X, uncached=%02X", bus1.Read(0x11), bus2.Read(0x11))
	}
}
