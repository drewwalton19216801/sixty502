package cpu6502

import (
	"testing"
)

// BenchmarkWithCache benchmarks CPU execution with instruction cache enabled
func BenchmarkWithCache(b *testing.B) {
	cpu, bus := setupCPU()
	cpu.EnableInstructionCache()

	// Tight loop program
	program := []uint8{
		0xA9, 0x00, // LDA #$00
		0x69, 0x01, // ADC #$01  ; Loop start
		0xC9, 0x10, // CMP #$10
		0xD0, 0xFA, // BNE Loop
		0x00, // BRK
	}
	bus.load(0x8000, program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		cpu.A = 0
		runUntilBrk(cpu, bus, 1000)
	}
}

// BenchmarkWithoutCache benchmarks CPU execution with instruction cache disabled
func BenchmarkWithoutCache(b *testing.B) {
	cpu, bus := setupCPU()
	cpu.DisableInstructionCache()

	// Same program as above
	program := []uint8{
		0xA9, 0x00, // LDA #$00
		0x69, 0x01, // ADC #$01  ; Loop start
		0xC9, 0x10, // CMP #$10
		0xD0, 0xFA, // BNE Loop
		0x00, // BRK
	}
	bus.load(0x8000, program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		cpu.A = 0
		runUntilBrk(cpu, bus, 1000)
	}
}

// BenchmarkCacheLookup benchmarks just the cache lookup operation
func BenchmarkCacheLookup(b *testing.B) {
	cache := NewInstructionCache()
	instr := &Instruction{Name: "LDA", Cycles: 2}

	// Pre-populate cache
	for i := uint16(0); i < 256; i++ {
		cache.Store(i, uint8(i), instr)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc := uint16(i & 0xFF)
		opcode := uint8(i & 0xFF)
		cache.Lookup(pc, opcode)
	}
}

// BenchmarkCacheStore benchmarks cache store operations
func BenchmarkCacheStore(b *testing.B) {
	cache := NewInstructionCache()
	instr := &Instruction{Name: "LDA", Cycles: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc := uint16(i & 0xFF)
		opcode := uint8(i & 0xFF)
		cache.Store(pc, opcode, instr)
	}
}

// BenchmarkTightLoop benchmarks a tight loop with cache
func BenchmarkTightLoop(b *testing.B) {
	cpu, bus := setupCPU()
	cpu.EnableInstructionCache()

	// Very tight 2-instruction loop
	program := []uint8{
		0xEA,             // NOP
		0x4C, 0x00, 0x80, // JMP $8000
	}
	bus.load(0x8000, program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		// Run for a fixed number of cycles
		for j := 0; j < 100; j++ {
			cpu.Clock()
		}
	}
}

// BenchmarkBranchingCode benchmarks branching code with cache
func BenchmarkBranchingCode(b *testing.B) {
	cpu, bus := setupCPU()
	cpu.EnableInstructionCache()

	// Program with branches
	program := []uint8{
		0xA9, 0x00, // LDA #$00
		0xC9, 0x05, // CMP #$05  ; Loop1
		0xF0, 0x04, // BEQ Skip
		0x69, 0x01, // ADC #$01
		0x4C, 0x02, 0x80, // JMP Loop1
		0xC9, 0x10, // CMP #$10  ; Skip
		0xD0, 0xF3, // BNE Loop1
		0x00, // BRK
	}
	bus.load(0x8000, program)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.PC = 0x8000
		cpu.SetCycles(0)
		cpu.A = 0
		runUntilBrk(cpu, bus, 1000)
	}
}

// BenchmarkCacheInvalidation benchmarks cache invalidation
func BenchmarkCacheInvalidation(b *testing.B) {
	cpu, _ := setupCPU()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.InvalidateInstructionCache()
	}
}

// BenchmarkCacheStats benchmarks getting cache statistics
func BenchmarkCacheStats(b *testing.B) {
	cpu, _ := setupCPU()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cpu.InstructionCacheStats()
	}
}
