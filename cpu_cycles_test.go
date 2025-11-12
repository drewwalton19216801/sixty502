package cpu6502

import (
	"testing"
)

// TestCycleAccuracy verifies exact cycle counts for all instructions
func TestCycleAccuracy(t *testing.T) {
	tests := []struct {
		name           string
		program        []uint8
		setup          func(cpu *CPU, bus *MockBus)
		expectedCycles uint64
		description    string
	}{
		{
			name:           "LDA ABX No Page Cross",
			program:        []uint8{0xBD, 0x00, 0x20, 0x00}, // LDA $2000,X
			setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x10 },
			expectedCycles: 4,
			description:    "Base cycles, no page cross",
		},
		{
			name:           "LDA ABX Page Cross",
			program:        []uint8{0xBD, 0xFF, 0x20, 0x00}, // LDA $20FF,X
			setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x02 },
			expectedCycles: 5,
			description:    "Base cycles + 1 for page cross",
		},
		{
			name:           "STA ABX Page Cross No Penalty",
			program:        []uint8{0x9D, 0xFF, 0x20, 0x00}, // STA $20FF,X
			setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x02 },
			expectedCycles: 5,
			description:    "Always 5 cycles, page cross doesn't add cycle",
		},
		{
			name:           "Branch Taken No Cross",
			program:        []uint8{0xD0, 0x10, 0x00}, // BNE +$10
			setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, false) },
			expectedCycles: 3,
			description:    "Base 2 + 1 for taken",
		},
		{
			name:           "Branch Taken Page Cross",
			program:        []uint8{0xD0, 0xFE, 0x00}, // BNE -$02
			setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, false); cpu.PC = 0x8001 },
			expectedCycles: 4,
			description:    "Base 2 + 1 for taken + 1 for page cross",
		},
		{
			name:           "Branch Not Taken",
			program:        []uint8{0xD0, 0x10, 0x00}, // BNE +$10
			setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, true) },
			expectedCycles: 2,
			description:    "Base cycles only",
		},
		{
			name:           "ADC Immediate",
			program:        []uint8{0x69, 0x01, 0x00}, // ADC #$01
			setup:          func(cpu *CPU, bus *MockBus) { cpu.A = 0x10 },
			expectedCycles: 2,
			description:    "Immediate mode, 2 cycles",
		},
		{
			name:           "ADC Zero Page",
			program:        []uint8{0x65, 0x10, 0x00}, // ADC $10
			setup:          func(cpu *CPU, bus *MockBus) { cpu.A = 0x10; bus.Write(0x10, 0x05) },
			expectedCycles: 3,
			description:    "Zero page, 3 cycles",
		},
		{
			name:           "ADC Absolute",
			program:        []uint8{0x6D, 0x00, 0x20, 0x00}, // ADC $2000
			setup:          func(cpu *CPU, bus *MockBus) { cpu.A = 0x10; bus.Write(0x2000, 0x05) },
			expectedCycles: 4,
			description:    "Absolute, 4 cycles",
		},
		{
			name:           "INC Zero Page",
			program:        []uint8{0xE6, 0x10, 0x00}, // INC $10
			setup:          func(cpu *CPU, bus *MockBus) { bus.Write(0x10, 0x42) },
			expectedCycles: 5,
			description:    "Read-modify-write, 5 cycles",
		},
		{
			name:           "INC Absolute",
			program:        []uint8{0xEE, 0x00, 0x20, 0x00}, // INC $2000
			setup:          func(cpu *CPU, bus *MockBus) { bus.Write(0x2000, 0x42) },
			expectedCycles: 6,
			description:    "Read-modify-write absolute, 6 cycles",
		},
		{
			name:           "JSR",
			program:        []uint8{0x20, 0x00, 0x90, 0x00}, // JSR $9000
			setup:          func(cpu *CPU, bus *MockBus) {},
			expectedCycles: 6,
			description:    "Jump to subroutine, 6 cycles",
		},
		{
			name:    "RTS",
			program: []uint8{0x60, 0x00}, // RTS
			setup: func(cpu *CPU, bus *MockBus) {
				// Set up stack as if JSR happened
				cpu.SP = 0xFD - 2
				bus.Write(stackBase+uint16(cpu.SP+1), 0x00)
				bus.Write(stackBase+uint16(cpu.SP+2), 0x80)
			},
			expectedCycles: 6,
			description:    "Return from subroutine, 6 cycles",
		},
		{
			name:           "PHA",
			program:        []uint8{0x48, 0x00}, // PHA
			setup:          func(cpu *CPU, bus *MockBus) { cpu.A = 0x42 },
			expectedCycles: 3,
			description:    "Push accumulator, 3 cycles",
		},
		{
			name:    "PLA",
			program: []uint8{0x68, 0x00}, // PLA
			setup: func(cpu *CPU, bus *MockBus) {
				cpu.SP = 0xFD - 1
				bus.Write(stackBase+uint16(cpu.SP+1), 0x42)
			},
			expectedCycles: 4,
			description:    "Pull accumulator, 4 cycles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()

			if tt.setup != nil {
				tt.setup(cpu, bus)
			}

			startPC := cpu.PC
			if startPC == 0 {
				startPC = 0x8000
				cpu.PC = startPC
			}

			bus.load(startPC, tt.program)
			cpu.SetCycles(0)

			startCycles := cpu.TotalCycles()
			runCycles(cpu, uint(tt.expectedCycles))
			actualCycles := cpu.TotalCycles() - startCycles

			if actualCycles != tt.expectedCycles {
				t.Errorf("%s: Expected %d cycles, got %d (%s)",
					tt.name, tt.expectedCycles, actualCycles, tt.description)
			}
		})
	}
}

// TestPageCrossCycles specifically tests page boundary crossing behavior
func TestPageCrossCycles(t *testing.T) {
	tests := []struct {
		name          string
		opcode        uint8
		baseAddr      uint16
		indexValue    uint8
		expectPenalty bool
		baseCycles    uint64
		description   string
	}{
		// Load instructions - should add cycle on page cross
		{
			name:          "LDA ABX No Cross",
			opcode:        0xBD,
			baseAddr:      0x2000,
			indexValue:    0x10,
			expectPenalty: false,
			baseCycles:    4,
			description:   "LDA $2000,X with X=$10 -> $2010 (no cross)",
		},
		{
			name:          "LDA ABX Cross",
			opcode:        0xBD,
			baseAddr:      0x20FF,
			indexValue:    0x02,
			expectPenalty: true,
			baseCycles:    4,
			description:   "LDA $20FF,X with X=$02 -> $2101 (cross)",
		},
		{
			name:          "LDA ABY No Cross",
			opcode:        0xB9,
			baseAddr:      0x3000,
			indexValue:    0x20,
			expectPenalty: false,
			baseCycles:    4,
			description:   "LDA $3000,Y with Y=$20 -> $3020 (no cross)",
		},
		{
			name:          "LDA ABY Cross",
			opcode:        0xB9,
			baseAddr:      0x30F0,
			indexValue:    0x20,
			expectPenalty: true,
			baseCycles:    4,
			description:   "LDA $30F0,Y with Y=$20 -> $3110 (cross)",
		},
		// Store instructions - should NOT add cycle on page cross
		{
			name:          "STA ABX No Cross",
			opcode:        0x9D,
			baseAddr:      0x2000,
			indexValue:    0x10,
			expectPenalty: false,
			baseCycles:    5,
			description:   "STA $2000,X with X=$10 -> $2010 (no penalty)",
		},
		{
			name:          "STA ABX Cross",
			opcode:        0x9D,
			baseAddr:      0x20FF,
			indexValue:    0x02,
			expectPenalty: false,
			baseCycles:    5,
			description:   "STA $20FF,X with X=$02 -> $2101 (no penalty)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()

			// Set up index register
			if tt.opcode == 0xBD || tt.opcode == 0x9D { // ABX modes
				cpu.X = tt.indexValue
			} else { // ABY modes
				cpu.Y = tt.indexValue
			}

			// Build program
			program := []uint8{
				tt.opcode,
				uint8(tt.baseAddr & 0xFF),
				uint8(tt.baseAddr >> 8),
				0x00, // BRK
			}

			bus.load(0x8000, program)
			cpu.PC = 0x8000
			cpu.SetCycles(0)

			expectedCycles := tt.baseCycles
			if tt.expectPenalty {
				expectedCycles++
			}

			startCycles := cpu.TotalCycles()
			runCycles(cpu, uint(expectedCycles))
			actualCycles := cpu.TotalCycles() - startCycles

			if actualCycles != expectedCycles {
				t.Errorf("%s: Expected %d cycles, got %d (%s)",
					tt.name, expectedCycles, actualCycles, tt.description)
			}
		})
	}
}
