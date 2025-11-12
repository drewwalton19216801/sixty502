package cpu6502

import (
	"testing"
)

// TestDecimalModeEdgeCases tests all BCD edge cases for ADC and SBC
func TestDecimalModeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		instr     string // "ADC" or "SBC"
		opcode    uint8
		a         uint8
		operand   uint8
		carryIn   bool
		expectedA uint8
		expectedC bool
		expectedZ bool
		expectedN bool // Based on binary intermediate result
		expectedV bool // Based on binary intermediate result
	}{
		// ADC edge cases
		{"ADC 99+1 C=0", "ADC", 0x69, 0x99, 0x01, false, 0x00, true, true, true, false},
		{"ADC 99+0 C=1", "ADC", 0x69, 0x99, 0x00, true, 0x00, true, true, true, false},
		{"ADC 50+50 C=0", "ADC", 0x69, 0x50, 0x50, false, 0x00, true, true, true, true},
		{"ADC 09+1 C=0", "ADC", 0x69, 0x09, 0x01, false, 0x10, false, false, false, false},
		{"ADC 09+1 C=1", "ADC", 0x69, 0x09, 0x01, true, 0x11, false, false, false, false},
		{"ADC 19+1 C=0", "ADC", 0x69, 0x19, 0x01, false, 0x20, false, false, false, false},
		{"ADC 00+00 C=0", "ADC", 0x69, 0x00, 0x00, false, 0x00, false, true, false, false},
		{"ADC 00+00 C=1", "ADC", 0x69, 0x00, 0x00, true, 0x01, false, false, false, false},
		{"ADC 49+49 C=0", "ADC", 0x69, 0x49, 0x49, false, 0x98, false, false, true, true},
		{"ADC 50+49 C=0", "ADC", 0x69, 0x50, 0x49, false, 0x99, false, false, true, true},

		// SBC edge cases
		{"SBC 00-1 C=1", "SBC", 0xE9, 0x00, 0x01, true, 0x99, false, false, true, false},
		{"SBC 00-1 C=0", "SBC", 0xE9, 0x00, 0x01, false, 0x98, false, false, true, false},
		{"SBC 10-1 C=1", "SBC", 0xE9, 0x10, 0x01, true, 0x09, true, false, false, false},
		{"SBC 10-1 C=0", "SBC", 0xE9, 0x10, 0x01, false, 0x08, true, false, false, false},
		{"SBC 00-0 C=1", "SBC", 0xE9, 0x00, 0x00, true, 0x00, true, true, false, false},
		{"SBC 00-0 C=0", "SBC", 0xE9, 0x00, 0x00, false, 0x99, false, false, true, false},
		{"SBC 50-50 C=1", "SBC", 0xE9, 0x50, 0x50, true, 0x00, true, true, false, false},
		{"SBC 32-2 C=1", "SBC", 0xE9, 0x32, 0x02, true, 0x30, true, false, false, false},
		{"SBC 12-21 C=1", "SBC", 0xE9, 0x12, 0x21, true, 0x91, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()

			// Enable decimal mode
			cpu.setFlag(D, true)
			cpu.setFlag(C, tt.carryIn)
			cpu.A = tt.a

			// Load instruction
			program := []uint8{tt.opcode, tt.operand, 0x00}
			bus.load(0x8000, program)
			cpu.PC = 0x8000
			cpu.SetCycles(0)

			// Execute
			runCycles(cpu, 2)

			// Verify result
			if cpu.A != tt.expectedA {
				t.Errorf("Expected A=$%02X, got $%02X", tt.expectedA, cpu.A)
			}
			if cpu.getFlag(C) != tt.expectedC {
				t.Errorf("Expected C=%v, got %v", tt.expectedC, cpu.getFlag(C))
			}
			if cpu.getFlag(Z) != tt.expectedZ {
				t.Errorf("Expected Z=%v, got %v", tt.expectedZ, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.expectedN {
				t.Errorf("Expected N=%v, got %v", tt.expectedN, cpu.getFlag(N))
			}
			if cpu.getFlag(V) != tt.expectedV {
				t.Errorf("Expected V=%v, got %v", tt.expectedV, cpu.getFlag(V))
			}
		})
	}
}
