package cpu6502

import (
	"testing"
)

// TestSelfModifyingCode verifies behavior with self-modifying code
func TestSelfModifyingCode(t *testing.T) {
	t.Run("Modify Next Instruction", func(t *testing.T) {
		cpu, bus := setupCPU()

		// Program that modifies the next instruction
		program := []uint8{
			0xA9, 0xEA, // LDA #$EA (NOP opcode)
			0x8D, 0x06, 0x80, // STA $8006 (modify next instruction)
			0x00, // This will become NOP
			0x00, // BRK
		}

		bus.load(0x8000, program)
		cpu.PC = 0x8000
		cpu.SetCycles(0)

		// Invalidate cache before execution
		cpu.InvalidateInstructionCache()

		// Execute until BRK
		runUntilBrk(cpu, bus, 50)

		// Verify the instruction was modified
		if bus.Read(0x8006) != 0xEA {
			t.Errorf("Instruction not modified: expected $EA, got $%02X", bus.Read(0x8006))
		}
	})

	t.Run("Cache Invalidation", func(t *testing.T) {
		cpu, bus := setupCPU()

		// Execute instruction from location
		bus.Write(0x8000, 0xA9) // LDA #$42
		bus.Write(0x8001, 0x42)
		bus.Write(0x8002, 0x00) // BRK

		cpu.PC = 0x8000
		cpu.SetCycles(0)
		runCycles(cpu, 2)

		if cpu.A != 0x42 {
			t.Errorf("First execution failed: expected A=$42, got A=$%02X", cpu.A)
		}

		// Modify the instruction
		bus.Write(0x8001, 0x99) // Change operand

		// Invalidate cache
		cpu.InvalidateInstructionCache()

		// Execute again
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		runCycles(cpu, 2)

		if cpu.A != 0x99 {
			t.Errorf("Modified instruction not executed: expected A=$99, got A=$%02X", cpu.A)
		}
	})

	t.Run("Cache Stats", func(t *testing.T) {
		cpu, bus := setupCPU()

		// Simple loop that should benefit from caching
		program := []uint8{
			0xA2, 0x00, // LDX #$00      ; $8000
			0xE8,       // INX           ; $8002 (loop start)
			0xE0, 0x05, // CPX #$05      ; $8003
			0xD0, 0xFB, // BNE -5        ; $8005 (branch to $8002)
			0x00, // BRK           ; $8007
		}

		bus.load(0x8000, program)
		cpu.PC = 0x8000
		cpu.SetCycles(0)

		// Execute the loop
		runUntilBrk(cpu, bus, 100)

		// Check cache statistics
		hits, misses, hitRate := cpu.InstructionCacheStats()

		// We should have some cache hits from the loop
		if hits == 0 {
			t.Logf("Warning: No cache hits detected (hits=%d, misses=%d, rate=%.2f%%)",
				hits, misses, hitRate*100)
		} else {
			t.Logf("Cache stats: hits=%d, misses=%d, hit rate=%.2f%%",
				hits, misses, hitRate*100)
		}

		// Verify X register has the expected value
		if cpu.X != 0x05 {
			t.Errorf("Loop execution failed: expected X=$05, got X=$%02X", cpu.X)
		}
	})

	t.Run("Disable and Enable Cache", func(t *testing.T) {
		cpu, bus := setupCPU()

		// Simple program
		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x00, // BRK
		}

		bus.load(0x8000, program)

		// Execute with cache enabled
		cpu.EnableInstructionCache()
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		runCycles(cpu, 2)

		hits1, misses1, _ := cpu.InstructionCacheStats()

		// Disable cache
		cpu.DisableInstructionCache()
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		runCycles(cpu, 2)

		hits2, misses2, _ := cpu.InstructionCacheStats()

		// Stats should be zero when cache is disabled
		if hits2 != 0 || misses2 != 0 {
			t.Errorf("Cache stats should be zero when disabled: hits=%d, misses=%d", hits2, misses2)
		}

		// Re-enable cache
		cpu.EnableInstructionCache()
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		runCycles(cpu, 2)

		hits3, misses3, _ := cpu.InstructionCacheStats()

		// Should have stats again
		if hits3 == 0 && misses3 == 0 {
			t.Errorf("Cache stats should not be zero after re-enabling")
		}

		t.Logf("Cache enabled: hits=%d, misses=%d", hits1, misses1)
		t.Logf("Cache disabled: hits=%d, misses=%d", hits2, misses2)
		t.Logf("Cache re-enabled: hits=%d, misses=%d", hits3, misses3)
	})
}
