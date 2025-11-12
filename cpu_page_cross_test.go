package cpu6502

import (
	"testing"
)

// TestPageCrossPenalty tests that instructions correctly add or don't add
// the +1 cycle penalty when crossing page boundaries
func TestPageCrossPenalty(t *testing.T) {
	tests := []struct {
		name          string
		opcode        uint8
		baseAddr      uint16
		indexReg      string // "X" or "Y"
		indexValue    uint8
		expectCross   bool
		baseCycles    uint8
		expectedTotal uint8
	}{
		// LDA tests - should add penalty
		{
			name:   "LDA ABX Page Cross",
			opcode: 0xBD, baseAddr: 0x20FF, indexReg: "X", indexValue: 0x02,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "LDA ABX No Cross",
			opcode: 0xBD, baseAddr: 0x2000, indexReg: "X", indexValue: 0x10,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},
		{
			name:   "LDA ABY Page Cross",
			opcode: 0xB9, baseAddr: 0x30FF, indexReg: "Y", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "LDA ABY No Cross",
			opcode: 0xB9, baseAddr: 0x3000, indexReg: "Y", indexValue: 0x50,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},

		// STA tests - should NOT add penalty (always full cycles)
		{
			name:   "STA ABX Page Cross No Penalty",
			opcode: 0x9D, baseAddr: 0x20FF, indexReg: "X", indexValue: 0x02,
			expectCross: true, baseCycles: 5, expectedTotal: 5,
		},
		{
			name:   "STA ABY Page Cross No Penalty",
			opcode: 0x99, baseAddr: 0x20FF, indexReg: "Y", indexValue: 0x02,
			expectCross: true, baseCycles: 5, expectedTotal: 5,
		},

		// ADC tests - should add penalty
		{
			name:   "ADC ABX Page Cross",
			opcode: 0x7D, baseAddr: 0x40FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "ADC ABY No Cross",
			opcode: 0x79, baseAddr: 0x4000, indexReg: "Y", indexValue: 0x20,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},

		// SBC tests - should add penalty
		{
			name:   "SBC ABX Page Cross",
			opcode: 0xFD, baseAddr: 0x50FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},

		// AND tests - should add penalty
		{
			name:   "AND ABX Page Cross",
			opcode: 0x3D, baseAddr: 0x60FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "AND ABY No Cross",
			opcode: 0x39, baseAddr: 0x6000, indexReg: "Y", indexValue: 0x10,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},

		// EOR tests - should add penalty
		{
			name:   "EOR ABX Page Cross",
			opcode: 0x5D, baseAddr: 0x70FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},

		// ORA tests - should add penalty
		{
			name:   "ORA ABY Page Cross",
			opcode: 0x19, baseAddr: 0x80FF, indexReg: "Y", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},

		// CMP tests - should add penalty
		{
			name:   "CMP ABX Page Cross",
			opcode: 0xDD, baseAddr: 0x90FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "CMP ABY No Cross",
			opcode: 0xD9, baseAddr: 0x9000, indexReg: "Y", indexValue: 0x30,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},

		// LDX tests - should add penalty
		{
			name:   "LDX ABY Page Cross",
			opcode: 0xBE, baseAddr: 0xA0FF, indexReg: "Y", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},

		// LDY tests - should add penalty
		{
			name:   "LDY ABX Page Cross",
			opcode: 0xBC, baseAddr: 0xB0FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},

		// Read-Modify-Write tests - should NOT add penalty
		{
			name:   "ASL ABX Page Cross No Penalty",
			opcode: 0x1E, baseAddr: 0xC0FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},
		{
			name:   "LSR ABX Page Cross No Penalty",
			opcode: 0x5E, baseAddr: 0xD0FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},
		{
			name:   "ROL ABX Page Cross No Penalty",
			opcode: 0x3E, baseAddr: 0xE0FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},
		{
			name:   "ROR ABX Page Cross No Penalty",
			opcode: 0x7E, baseAddr: 0xF0FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},
		{
			name:   "INC ABX Page Cross No Penalty",
			opcode: 0xFE, baseAddr: 0x10FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},
		{
			name:   "DEC ABX Page Cross No Penalty",
			opcode: 0xDE, baseAddr: 0x11FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 7, expectedTotal: 7,
		},

		// Unofficial NOP tests - should add penalty for ABX
		{
			name:   "NOP ABX Page Cross (0x1C)",
			opcode: 0x1C, baseAddr: 0x12FF, indexReg: "X", indexValue: 0x01,
			expectCross: true, baseCycles: 4, expectedTotal: 5,
		},
		{
			name:   "NOP ABX No Cross (0x3C)",
			opcode: 0x3C, baseAddr: 0x1300, indexReg: "X", indexValue: 0x10,
			expectCross: false, baseCycles: 4, expectedTotal: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple RAM bus
			bus := NewMockBus()
			cpu := NewCPU(bus)

			// Set index register
			if tt.indexReg == "X" {
				cpu.X = tt.indexValue
			} else {
				cpu.Y = tt.indexValue
			}

			// Write instruction at PC
			cpu.PC = 0x8000
			bus.Write(0x8000, tt.opcode)
			bus.Write(0x8001, uint8(tt.baseAddr&0xFF))
			bus.Write(0x8002, uint8(tt.baseAddr>>8))

			// Write some data at the target address for read instructions
			effectiveAddr := tt.baseAddr + uint16(tt.indexValue)
			bus.Write(effectiveAddr, 0x42)

			// Execute instruction
			startCycles := cpu.TotalCycles()
			for cpu.RemainingCycles() > 0 || cpu.TotalCycles() == startCycles {
				if err := cpu.Clock(); err != nil {
					t.Fatalf("Clock error: %v", err)
				}
			}
			cyclesUsed := uint8(cpu.TotalCycles() - startCycles)

			// Verify cycle count
			if cyclesUsed != tt.expectedTotal {
				t.Errorf("Expected %d cycles, got %d cycles", tt.expectedTotal, cyclesUsed)
			}

			// Verify page cross detection
			actualCross := (effectiveAddr & 0xFF00) != (tt.baseAddr & 0xFF00)
			if actualCross != tt.expectCross {
				t.Errorf("Page cross detection mismatch: expected %v, got %v", tt.expectCross, actualCross)
			}
		})
	}
}

// TestPageCrossPenaltyIZY tests indirect indexed (IZY) mode page crossing
func TestPageCrossPenaltyIZY(t *testing.T) {
	tests := []struct {
		name          string
		opcode        uint8
		zpAddr        uint8
		baseAddr      uint16
		yValue        uint8
		expectCross   bool
		baseCycles    uint8
		expectedTotal uint8
	}{
		{
			name:   "LDA IZY Page Cross",
			opcode: 0xB1, zpAddr: 0x10, baseAddr: 0x20FF, yValue: 0x02,
			expectCross: true, baseCycles: 5, expectedTotal: 6,
		},
		{
			name:   "LDA IZY No Cross",
			opcode: 0xB1, zpAddr: 0x20, baseAddr: 0x2000, yValue: 0x10,
			expectCross: false, baseCycles: 5, expectedTotal: 5,
		},
		{
			name:   "STA IZY Page Cross No Penalty",
			opcode: 0x91, zpAddr: 0x30, baseAddr: 0x30FF, yValue: 0x01,
			expectCross: true, baseCycles: 6, expectedTotal: 6,
		},
		{
			name:   "ADC IZY Page Cross",
			opcode: 0x71, zpAddr: 0x40, baseAddr: 0x40FF, yValue: 0x01,
			expectCross: true, baseCycles: 5, expectedTotal: 6,
		},
		{
			name:   "CMP IZY Page Cross",
			opcode: 0xD1, zpAddr: 0x50, baseAddr: 0x50FF, yValue: 0x01,
			expectCross: true, baseCycles: 5, expectedTotal: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewMockBus()
			cpu := NewCPU(bus)

			cpu.Y = tt.yValue
			cpu.PC = 0x8000

			// Write instruction
			bus.Write(0x8000, tt.opcode)
			bus.Write(0x8001, tt.zpAddr)

			// Write base address to zero page
			bus.Write(uint16(tt.zpAddr), uint8(tt.baseAddr&0xFF))
			bus.Write(uint16(tt.zpAddr)+1, uint8(tt.baseAddr>>8))

			// Write data at effective address
			effectiveAddr := tt.baseAddr + uint16(tt.yValue)
			bus.Write(effectiveAddr, 0x42)

			// Execute instruction
			startCycles := cpu.TotalCycles()
			for cpu.RemainingCycles() > 0 || cpu.TotalCycles() == startCycles {
				if err := cpu.Clock(); err != nil {
					t.Fatalf("Clock error: %v", err)
				}
			}
			cyclesUsed := uint8(cpu.TotalCycles() - startCycles)

			if cyclesUsed != tt.expectedTotal {
				t.Errorf("Expected %d cycles, got %d cycles", tt.expectedTotal, cyclesUsed)
			}
		})
	}
}
