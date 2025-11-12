package cpu6502

import (
	"testing"
)

// TestUnofficialOpcodes tests behavior of illegal/unofficial opcodes
// Note: This is a framework for future implementation of unofficial opcodes
func TestUnofficialOpcodes(t *testing.T) {
	// Test cases for documented unofficial opcodes
	tests := []struct {
		name     string
		opcode   uint8
		behavior string
		verify   func(t *testing.T, cpu *CPU, bus *MockBus)
	}{
		{
			name:     "LAX (Load A and X)",
			opcode:   0xA7, // LAX Zero Page
			behavior: "Loads value into both A and X",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("LAX not yet implemented")
			},
		},
		{
			name:     "SAX (Store A AND X)",
			opcode:   0x87, // SAX Zero Page
			behavior: "Stores A & X to memory",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("SAX not yet implemented")
			},
		},
		{
			name:     "DCP (Decrement and Compare)",
			opcode:   0xC7, // DCP Zero Page
			behavior: "Decrements memory then compares with A",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("DCP not yet implemented")
			},
		},
		{
			name:     "ISC (Increment and Subtract with Carry)",
			opcode:   0xE7, // ISC Zero Page
			behavior: "Increments memory then performs SBC",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("ISC not yet implemented")
			},
		},
		{
			name:     "SLO (Shift Left and OR)",
			opcode:   0x07, // SLO Zero Page
			behavior: "ASL then ORA with accumulator",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("SLO not yet implemented")
			},
		},
		{
			name:     "RLA (Rotate Left and AND)",
			opcode:   0x27, // RLA Zero Page
			behavior: "ROL then AND with accumulator",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("RLA not yet implemented")
			},
		},
		{
			name:     "SRE (Shift Right and EOR)",
			opcode:   0x47, // SRE Zero Page
			behavior: "LSR then EOR with accumulator",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("SRE not yet implemented")
			},
		},
		{
			name:     "RRA (Rotate Right and ADC)",
			opcode:   0x67, // RRA Zero Page
			behavior: "ROR then ADC with accumulator",
			verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
				t.Skip("RRA not yet implemented")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()
			tt.verify(t, cpu, bus)
		})
	}
}

// TestUnofficialNOPs verifies that unofficial NOP variants work correctly
func TestUnofficialNOPs(t *testing.T) {
	// These are already tested in TestNopInstructions in cpu_test.go
	// This test just verifies they are marked as illegal but functional

	unofficialNOPs := []uint8{
		0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA, // Implied NOPs
		0x80, 0x82, 0x89, 0xC2, 0xE2, // Immediate NOPs
		0x04, 0x44, 0x64, // Zero Page NOPs
		0x14, 0x34, 0x54, 0x74, 0xD4, 0xF4, // Zero Page,X NOPs
		0x0C,                               // Absolute NOP
		0x1C, 0x3C, 0x5C, 0x7C, 0xDC, 0xFC, // Absolute,X NOPs
	}

	for _, opcode := range unofficialNOPs {
		t.Run("NOP_"+string(rune(opcode)), func(t *testing.T) {
			cpu, _ := setupCPU()

			// Verify it's marked as illegal
			if !cpu.IsIllegalOpcode(opcode) {
				t.Errorf("Opcode $%02X should be marked as illegal", opcode)
			}

			// Verify it has the NOP operation
			instr := cpu.LookupInstruction(opcode)
			if instr.Name != "*NOP" {
				t.Errorf("Opcode $%02X should be *NOP, got %s", opcode, instr.Name)
			}
		})
	}
}
