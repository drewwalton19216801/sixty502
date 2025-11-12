package cpu6502

import (
	"fmt"
	"testing"
)

// TestInstructionLengthValidation verifies PC advances correctly for all opcodes
func TestInstructionLengthValidation(t *testing.T) {
	cpu, bus := setupCPU()

	// Instructions that modify PC in ways other than simple advancement
	pcModifyingOpcodes := map[uint8]bool{
		0x00: true, // BRK
		0x20: true, // JSR
		0x4C: true, // JMP ABS
		0x6C: true, // JMP IND
		0x40: true, // RTI
		0x60: true, // RTS
		// Branch instructions
		0x10: true, // BPL
		0x30: true, // BMI
		0x50: true, // BVC
		0x70: true, // BVS
		0x90: true, // BCC
		0xB0: true, // BCS
		0xD0: true, // BNE
		0xF0: true, // BEQ
	}

	// Test all 256 opcodes
	for opcode := 0; opcode <= 255; opcode++ {
		t.Run(fmt.Sprintf("Opcode_%02X", opcode), func(t *testing.T) {
			cpu, bus = setupCPU()

			instr := cpu.LookupInstruction(uint8(opcode))

			// Skip if no length defined
			if instr.Length == 0 {
				t.Skip("Instruction length not defined")
			}

			// Skip instructions that modify PC in special ways
			if pcModifyingOpcodes[uint8(opcode)] {
				t.Skip("Instruction modifies PC (branch/jump/interrupt)")
			}

			// Load instruction with dummy operands
			startPC := uint16(0x8000)
			bus.Write(startPC, uint8(opcode))
			for i := uint8(1); i < instr.Length; i++ {
				bus.Write(startPC+uint16(i), 0x00)
			}
			// Write NOPs after the instruction to prevent accidental BRK execution
			for i := uint16(0); i < 10; i++ {
				bus.Write(startPC+uint16(instr.Length)+i, 0xEA) // NOP
			}

			cpu.PC = startPC
			cpu.SetCycles(0)

			// Execute just one instruction by tracking initial PC
			initialPC := cpu.PC
			maxCycles := uint8(instr.Cycles + 5) // Allow for page cross

			// Run cycles until PC changes (instruction completes)
			cyclesRun := uint8(0)
			for cyclesRun < maxCycles && cpu.PC == initialPC {
				if err := cpu.Clock(); err != nil {
					t.Fatalf("CPU error: %v", err)
				}
				cyclesRun++
			}

			// Verify PC advanced by instruction length
			expectedPC := startPC + uint16(instr.Length)
			if cpu.PC != expectedPC {
				t.Errorf("Opcode $%02X (%s): Expected PC=$%04X, got $%04X (Length=%d)",
					opcode, instr.Name, expectedPC, cpu.PC, instr.Length)
			}
		})
	}
}
