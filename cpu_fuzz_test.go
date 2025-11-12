package cpu6502

import (
	"testing"
)

// FuzzCPUExecution fuzzes CPU execution with random programs
func FuzzCPUExecution(f *testing.F) {
	// Add seed corpus - valid 6502 programs
	f.Add([]byte{0xA9, 0x42, 0x00})             // LDA #$42, BRK
	f.Add([]byte{0x69, 0x01, 0x00})             // ADC #$01, BRK
	f.Add([]byte{0xE9, 0x01, 0x00})             // SBC #$01, BRK
	f.Add([]byte{0xA2, 0x10, 0xE8, 0x00})       // LDX #$10, INX, BRK
	f.Add([]byte{0xA0, 0x20, 0xC8, 0x00})       // LDY #$20, INY, BRK
	f.Add([]byte{0x18, 0x69, 0xFF, 0x00})       // CLC, ADC #$FF, BRK
	f.Add([]byte{0x38, 0xE9, 0x01, 0x00})       // SEC, SBC #$01, BRK
	f.Add([]byte{0xA9, 0x00, 0x29, 0xFF, 0x00}) // LDA #$00, AND #$FF, BRK

	f.Fuzz(func(t *testing.T, program []byte) {
		// Skip empty or very large programs
		if len(program) == 0 || len(program) > 256 {
			return
		}

		cpu, bus := setupCPU()
		bus.load(0x8000, program)
		cpu.PC = 0x8000
		cpu.SetCycles(0)

		// Execute with timeout to prevent infinite loops
		maxCycles := uint64(len(program) * 10)
		for i := uint64(0); i < maxCycles; i++ {
			if err := cpu.Clock(); err != nil {
				// Error is acceptable (illegal opcode, etc.)
				return
			}

			// Stop on BRK
			if cpu.PC < 0x8000 || cpu.PC >= 0x8000+uint16(len(program)) {
				// PC went outside program bounds
				return
			}

			// Check for BRK opcode at current PC
			if bus.Read(cpu.PC) == 0x00 && cpu.RemainingCycles() == 0 {
				return
			}
		}

		// If we get here, the program ran for maxCycles without hitting BRK
		// This is acceptable - just means it's a long-running program
	})
}

// FuzzDecimalMode fuzzes decimal mode arithmetic
func FuzzDecimalMode(f *testing.F) {
	// Seed with valid BCD values
	f.Add(uint8(0x00), uint8(0x00), false) // 00 + 00
	f.Add(uint8(0x09), uint8(0x01), false) // 09 + 01
	f.Add(uint8(0x99), uint8(0x01), false) // 99 + 01
	f.Add(uint8(0x50), uint8(0x50), false) // 50 + 50
	f.Add(uint8(0x99), uint8(0x99), true)  // 99 + 99 + carry

	f.Fuzz(func(t *testing.T, a uint8, operand uint8, carryIn bool) {
		cpu, bus := setupCPU()

		// Enable decimal mode
		cpu.setFlag(D, true)
		cpu.setFlag(C, carryIn)
		cpu.A = a

		// ADC immediate
		program := []uint8{0x69, operand, 0x00}
		bus.load(0x8000, program)
		cpu.PC = 0x8000
		cpu.SetCycles(0)

		// Execute ADC
		if err := cpu.Clock(); err != nil {
			// Error is acceptable
			return
		}
		if err := cpu.Clock(); err != nil {
			return
		}

		// Just verify the CPU didn't crash
		// We don't validate the result because invalid BCD inputs
		// have undefined behavior
	})
}

// FuzzAddressingModes fuzzes different addressing modes
func FuzzAddressingModes(f *testing.F) {
	// Seed with various addressing mode combinations
	f.Add([]byte{0xA9, 0x42})       // LDA immediate
	f.Add([]byte{0xA5, 0x10})       // LDA zero page
	f.Add([]byte{0xAD, 0x00, 0x20}) // LDA absolute

	f.Fuzz(func(t *testing.T, program []byte) {
		if len(program) == 0 || len(program) > 10 {
			return
		}

		cpu, bus := setupCPU()

		// Add BRK at the end
		fullProgram := append([]byte{}, program...)
		fullProgram = append(fullProgram, 0x00)

		bus.load(0x8000, fullProgram)
		cpu.PC = 0x8000
		cpu.SetCycles(0)

		// Execute with cycle limit
		maxCycles := uint64(20)
		for i := uint64(0); i < maxCycles; i++ {
			if err := cpu.Clock(); err != nil {
				// Error is acceptable
				return
			}

			// Stop if we hit BRK or PC goes out of bounds
			if cpu.PC < 0x8000 || cpu.PC >= 0x8000+uint16(len(fullProgram)) {
				return
			}
		}
	})
}
