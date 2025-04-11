package cpu6502

import (
	"fmt"
	"testing"
)

// --- Mock Bus Implementation ---

// MockBus provides a simple RAM-like implementation of the Bus interface for testing.
type MockBus struct {
	ram [64 * 1024]uint8 // Simulate 64KB address space
	// Optionally add fields to track reads/writes if needed for specific tests
	// readLog  []uint16
	// writeLog map[uint16]uint8
}

// NewMockBus creates an initialized MockBus.
func NewMockBus() *MockBus {
	return &MockBus{
		// writeLog: make(map[uint16]uint8),
	}
}

// Read implements the Bus interface for MockBus.
func (b *MockBus) Read(addr uint16) uint8 {
	// b.readLog = append(b.readLog, addr) // Optional logging
	return b.ram[addr]
}

// Write implements the Bus interface for MockBus.
func (b *MockBus) Write(addr uint16, data uint8) {
	// b.writeLog[addr] = data // Optional logging
	b.ram[addr] = data
}

// Helper to load a program into the mock RAM.
func (b *MockBus) load(startAddr uint16, program []uint8) {
	for i, byt := range program {
		b.Write(startAddr+uint16(i), byt)
	}
}

// --- Test Setup Helper ---

// setupCPU creates a CPU instance with a mock bus for testing.
func setupCPU() (*CPU, *MockBus) {
	bus := NewMockBus()
	cpu := NewCPU(bus)
	return cpu, bus
}

// runUntilBrk runs the CPU until it encounters a BRK (0x00) instruction or maxCycles is reached.
// Returns the number of cycles executed.
func runUntilBrk(cpu *CPU, bus *MockBus, maxCycles uint64) uint64 {
	initialCycles := cpu.totalCycles
	for cpu.totalCycles < initialCycles+maxCycles {
		// Peek ahead to see if the next instruction is BRK
		// Note: This check happens *before* executing the BRK instruction itself.
		// The Clock() call will execute the BRK if pc points to it.
		if bus.Read(cpu.PC) == 0x00 {
			cpu.Clock() // Execute the BRK
			break
		}
		cpu.Clock()
	}
	return cpu.totalCycles - initialCycles
}

// --- Test Cases ---

// TestReset verifies the CPU state after a Reset().
func TestReset(t *testing.T) {
	cpu, bus := setupCPU()

	// Set up reset vector
	resetAddr := uint16(0x8000)
	bus.Write(0xFFFC, uint8(resetAddr&0x00FF)) // Low byte
	bus.Write(0xFFFD, uint8(resetAddr>>8))     // High byte

	cpu.Reset()

	if cpu.PC != resetAddr {
		t.Errorf("Reset() failed: Expected PC=0x%04X, got 0x%04X", resetAddr, cpu.PC)
	}
	if cpu.A != 0 {
		t.Errorf("Reset() failed: Expected A=0, got 0x%02X", cpu.A)
	}
	if cpu.X != 0 {
		t.Errorf("Reset() failed: Expected X=0, got 0x%02X", cpu.X)
	}
	if cpu.Y != 0 {
		t.Errorf("Reset() failed: Expected Y=0, got 0x%02X", cpu.Y)
	}
	if cpu.SP != 0xFD {
		t.Errorf("Reset() failed: Expected SP=0xFD, got 0x%02X", cpu.SP)
	}
	// Expect I and U flags to be set, others clear
	expectedFlags := U | I
	// Clear B flag just in case it was somehow set before reset (shouldn't happen)
	cpu.P &= ^B
	if cpu.P != expectedFlags {
		t.Errorf("Reset() failed: Expected P=0x%02X (%s), got 0x%02X (%s)",
			expectedFlags, fmt.Sprintf("%08b", expectedFlags), cpu.P, fmt.Sprintf("%08b", cpu.P))
	}
	// Reset should take cycles (nominally 8, can vary slightly by model/source)
	if cpu.cycles < 7 || cpu.cycles > 8 {
		t.Errorf("Reset() failed: Expected cycles remaining ~8, got %d", cpu.cycles)
	}
}

// TestFlagsInstruction verifies instructions that directly set/clear flags.
func TestFlagsInstructions(t *testing.T) {
	cpu, bus := setupCPU()

	tests := []struct {
		name     string
		program  []uint8
		flag     Flags
		expected bool
	}{
		{"CLC", []uint8{0x18, 0x00}, C, false}, // CLC, BRK
		{"SEC", []uint8{0x38, 0x00}, C, true},  // SEC, BRK
		{"CLI", []uint8{0x58, 0x00}, I, false}, // CLI, BRK
		{"SEI", []uint8{0x78, 0x00}, I, true},  // SEI, BRK
		{"CLD", []uint8{0xD8, 0x00}, D, false}, // CLD, BRK
		{"SED", []uint8{0xF8, 0x00}, D, true},  // SED, BRK
		{"CLV", []uint8{0xB8, 0x00}, V, false}, // CLV, BRK
	}

	startAddr := uint16(0x8000)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus.load(startAddr, tt.program)
			// Set flags opposite to expected before running
			cpu.setFlag(tt.flag, !tt.expected)
			// Ensure U flag is set correctly before execution
			cpu.P = U | I
			cpu.PC = startAddr
			cpu.cycles = 0 // Start execution immediately

			runUntilBrk(cpu, bus, 10)

			if cpu.getFlag(tt.flag) != tt.expected {
				t.Errorf("%s failed: Expected flag %v to be %v, got %v (P=0x%02X)",
					tt.name, tt.flag, tt.expected, cpu.getFlag(tt.flag), cpu.P)
			}
		})
	}
}

// TestLDA tests various Load Accumulator instructions and flags.
func TestLDA(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0200) // Use a different start address

	tests := []struct {
		name     string
		program  []uint8
		value    uint8  // Value expected in A
		zero     bool   // Expected Z flag state
		negative bool   // Expected N flag state
		setup    func() // Optional setup before running (e.g., set X/Y)
	}{
		{
			"Immediate",
			[]uint8{0xA9, 0x42, 0x00}, // LDA #$42, BRK
			0x42, false, false, nil,
		},
		{
			"Immediate Zero",
			[]uint8{0xA9, 0x00, 0x00}, // LDA #$00, BRK
			0x00, true, false, nil,
		},
		{
			"Immediate Negative",
			[]uint8{0xA9, 0x88, 0x00}, // LDA #$88, BRK
			0x88, false, true, nil,
		},
		{
			"ZeroPage",
			[]uint8{0xA5, 0x30, 0x00}, // LDA $30, BRK
			0x55, false, false, func() { bus.Write(0x0030, 0x55) },
		},
		{
			"ZeroPage,X",
			[]uint8{0xB5, 0x30, 0x00}, // LDA $30,X, BRK
			0x66, false, false, func() { cpu.X = 0x05; bus.Write(0x0035, 0x66) },
		},
		{
			"ZeroPage,X Wrap",
			[]uint8{0xB5, 0xFE, 0x00},                                            // LDA $FE,X, BRK
			0x77, false, false, func() { cpu.X = 0x03; bus.Write(0x0001, 0x77) }, // FE + 03 = 101 -> wraps to 01
		},
		{
			"Absolute",
			[]uint8{0xAD, 0xCD, 0xAB, 0x00}, // LDA $ABCD, BRK
			0x88, false, true, func() { bus.Write(0xABCD, 0x88) },
		},
		{
			"Absolute,X",
			[]uint8{0xBD, 0xCD, 0xAB, 0x00}, // LDA $ABCD,X, BRK
			0x99, false, true, func() { cpu.X = 0x10; bus.Write(0xABCD+0x10, 0x99) },
		},
		{
			"Absolute,Y",
			[]uint8{0xB9, 0xCD, 0xAB, 0x00}, // LDA $ABCD,Y, BRK
			0xAA, false, true, func() { cpu.Y = 0x20; bus.Write(0xABCD+0x20, 0xAA) },
		},
		{
			"Indirect,X",              // (ZP,X)
			[]uint8{0xA1, 0x40, 0x00}, // LDA ($40,X), BRK
			0xBB, false, true, func() {
				cpu.X = 0x04
				// Vector address calculation: ($40 + $04) & 0xFF = $44
				bus.Write(0x0044, 0x34) // Low byte of target address
				bus.Write(0x0045, 0x12) // High byte of target address ($1234)
				bus.Write(0x1234, 0xBB) // Value to load
			},
		},
		{
			"Indirect,Y",              // (ZP),Y
			[]uint8{0xB1, 0x50, 0x00}, // LDA ($50),Y, BRK
			0xCC, false, true, func() {
				cpu.Y = 0x05
				// Base address pointer at $50
				bus.Write(0x0050, 0xBC) // Low byte of base address
				bus.Write(0x0051, 0x9A) // High byte of base address ($9ABC)
				// Effective address: $9ABC + Y = $9ABC + $05 = $9AC1
				bus.Write(0x9AC1, 0xCC) // Value to load
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset CPU state for each test
			cpu, bus = setupCPU() // Get fresh instances
			if tt.setup != nil {
				tt.setup()
			}

			bus.load(startAddr, tt.program)
			cpu.PC = startAddr
			cpu.cycles = 0 // Start execution immediately

			runUntilBrk(cpu, bus, 20) // Allow enough cycles

			if cpu.A != tt.value {
				t.Errorf("LDA %s failed: Expected A=0x%02X, got 0x%02X", tt.name, tt.value, cpu.A)
			}
			if cpu.getFlag(Z) != tt.zero {
				t.Errorf("LDA %s failed: Expected Z flag=%v, got %v", tt.name, tt.zero, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.negative {
				t.Errorf("LDA %s failed: Expected N flag=%v, got %v", tt.name, tt.negative, cpu.getFlag(N))
			}
		})
	}
}

// TestSTA tests various Store Accumulator instructions.
func TestSTA(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0300)
	valueToStore := uint8(0xDA)

	tests := []struct {
		name    string
		program []uint8
		addr    uint16 // Address where value should be stored
		setup   func() // Optional setup before running
	}{
		{
			"ZeroPage",
			[]uint8{0x85, 0x40, 0x00}, // STA $40, BRK
			0x0040, nil,
		},
		{
			"ZeroPage,X",
			[]uint8{0x95, 0x40, 0x00}, // STA $40,X, BRK
			0x0045, func() { cpu.X = 0x05 },
		},
		{
			"ZeroPage,X Wrap",
			[]uint8{0x95, 0xFF, 0x00},       // STA $FF,X, BRK
			0x0001, func() { cpu.X = 0x02 }, // FF + 02 = 101 -> wraps to 01
		},
		{
			"Absolute",
			[]uint8{0x8D, 0x34, 0x12, 0x00}, // STA $1234, BRK
			0x1234, nil,
		},
		{
			"Absolute,X",
			[]uint8{0x9D, 0x34, 0x12, 0x00}, // STA $1234,X, BRK
			0x1234 + 0x11, func() { cpu.X = 0x11 },
		},
		{
			"Absolute,Y",
			[]uint8{0x99, 0x34, 0x12, 0x00}, // STA $1234,Y, BRK
			0x1234 + 0x22, func() { cpu.Y = 0x22 },
		},
		{
			"Indirect,X",              // (ZP,X)
			[]uint8{0x81, 0x60, 0x00}, // STA ($60,X), BRK
			0x4567, func() {
				cpu.X = 0x03
				// Vector address calculation: ($60 + $03) & 0xFF = $63
				bus.Write(0x0063, 0x67) // Low byte of target address
				bus.Write(0x0064, 0x45) // High byte of target address ($4567)
			},
		},
		{
			"Indirect,Y",              // (ZP),Y
			[]uint8{0x91, 0x70, 0x00}, // STA ($70),Y, BRK
			0x89AB + 0x0F, func() {
				cpu.Y = 0x0F
				// Base address pointer at $70
				bus.Write(0x0070, 0xAB) // Low byte of base address
				bus.Write(0x0071, 0x89) // High byte of base address ($89AB)
				// Effective address: $89AB + Y = $89AB + $0F = $89BA
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus = setupCPU() // Fresh instances
			cpu.A = valueToStore  // Set accumulator value

			if tt.setup != nil {
				tt.setup()
			}

			bus.load(startAddr, tt.program)
			cpu.PC = startAddr
			cpu.cycles = 0

			runUntilBrk(cpu, bus, 20)

			storedValue := bus.Read(tt.addr)
			if storedValue != valueToStore {
				t.Errorf("STA %s failed: Expected 0x%02X at 0x%04X, got 0x%02X",
					tt.name, valueToStore, tt.addr, storedValue)
			}
		})
	}
}

// TestRegisterTransfers tests TAX, TAY, TXA, TYA, TSX, TXS.
func TestRegisterTransfers(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0400)
	testVal := uint8(0xB7) // A value that is non-zero and negative

	tests := []struct {
		name    string
		program []uint8
		setup   func()
		verify  func()
	}{
		{
			"TAX", []uint8{0xAA, 0x00}, // TAX, BRK
			func() { cpu.A = testVal; cpu.X = 0 },
			func() {
				if cpu.X != testVal {
					t.Errorf("TAX failed: Expected X=0x%02X, got 0x%02X", testVal, cpu.X)
				}
				if cpu.getFlag(Z) {
					t.Errorf("TAX failed: Expected Z=false, got true")
				}
				if !cpu.getFlag(N) {
					t.Errorf("TAX failed: Expected N=true, got false")
				}
			},
		},
		{
			"TAY", []uint8{0xA8, 0x00}, // TAY, BRK
			func() { cpu.A = testVal; cpu.Y = 0 },
			func() {
				if cpu.Y != testVal {
					t.Errorf("TAY failed: Expected Y=0x%02X, got 0x%02X", testVal, cpu.Y)
				}
				if cpu.getFlag(Z) {
					t.Errorf("TAY failed: Expected Z=false, got true")
				}
				if !cpu.getFlag(N) {
					t.Errorf("TAY failed: Expected N=true, got false")
				}
			},
		},
		{
			"TXA", []uint8{0x8A, 0x00}, // TXA, BRK
			func() { cpu.X = testVal; cpu.A = 0 },
			func() {
				if cpu.A != testVal {
					t.Errorf("TXA failed: Expected A=0x%02X, got 0x%02X", testVal, cpu.A)
				}
				if cpu.getFlag(Z) {
					t.Errorf("TXA failed: Expected Z=false, got true")
				}
				if !cpu.getFlag(N) {
					t.Errorf("TXA failed: Expected N=true, got false")
				}
			},
		},
		{
			"TYA", []uint8{0x98, 0x00}, // TYA, BRK
			func() { cpu.Y = testVal; cpu.A = 0 },
			func() {
				if cpu.A != testVal {
					t.Errorf("TYA failed: Expected A=0x%02X, got 0x%02X", testVal, cpu.A)
				}
				if cpu.getFlag(Z) {
					t.Errorf("TYA failed: Expected Z=false, got true")
				}
				if !cpu.getFlag(N) {
					t.Errorf("TYA failed: Expected N=true, got false")
				}
			},
		},
		{
			"TSX", []uint8{0xBA, 0x00}, // TSX, BRK
			func() { cpu.SP = testVal; cpu.X = 0 },
			func() {
				if cpu.X != testVal {
					t.Errorf("TSX failed: Expected X=0x%02X, got 0x%02X", testVal, cpu.X)
				}
				if cpu.getFlag(Z) {
					t.Errorf("TSX failed: Expected Z=false, got true")
				}
				if !cpu.getFlag(N) { // N flag reflects bit 7 of SP
					t.Errorf("TSX failed: Expected N=true, got false")
				}
			},
		},
		{
			"TXS", []uint8{0x9A, 0x00}, // TXS, BRK
			func() { cpu.X = testVal; cpu.SP = 0 },
			func() {
				if cpu.SP != testVal {
					t.Errorf("TXS failed: Expected SP=0x%02X, got 0x%02X", testVal, cpu.SP)
				}
				// TXS does NOT affect flags
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus = setupCPU() // Fresh instances
			if tt.setup != nil {
				tt.setup()
			}
			// Save initial flags to check if TXS modified them incorrectly
			initialFlags := cpu.P

			bus.load(startAddr, tt.program)
			cpu.PC = startAddr
			cpu.cycles = 0

			runUntilBrk(cpu, bus, 10)

			if tt.verify != nil {
				tt.verify()
			}
			// Special check for TXS flags
			if tt.name == "TXS" && cpu.P != initialFlags {
				// Allow U flag to be set if it wasn't (it should always end up set)
				expectedP := initialFlags | U
				if cpu.P != expectedP {
					t.Errorf("TXS failed: Flags were modified. Expected P=0x%02X, got 0x%02X", expectedP, cpu.P)
				}
			}
		})
	}
}

// TestStackOperations tests PHA, PLA, PHP, PLP.
func TestStackOperations(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0500)
	initialSP := uint8(0xFD) // Default SP after reset

	t.Run("PHA", func(t *testing.T) {
		cpu, bus = setupCPU()
		valueToPush := uint8(0xAB)
		cpu.A = valueToPush
		program := []uint8{0x48, 0x00} // PHA, BRK
		bus.load(startAddr, program)
		cpu.PC = startAddr
		cpu.cycles = 0
		cpu.SP = initialSP

		runUntilBrk(cpu, bus, 10)

		if cpu.SP != initialSP-1 {
			t.Errorf("PHA failed: Expected SP=0x%02X, got 0x%02X", initialSP-1, cpu.SP)
		}
		pushedValue := bus.Read(stackBase + uint16(initialSP))
		if pushedValue != valueToPush {
			t.Errorf("PHA failed: Expected value 0x%02X on stack at 0x%04X, got 0x%02X",
				valueToPush, stackBase+uint16(initialSP), pushedValue)
		}
	})

	t.Run("PLA", func(t *testing.T) {
		cpu, bus = setupCPU()
		valueToPull := uint8(0xCD) // Non-zero, negative
		bus.Write(stackBase+uint16(initialSP), valueToPull)
		program := []uint8{0x68, 0x00} // PLA, BRK
		bus.load(startAddr, program)
		cpu.PC = startAddr
		cpu.cycles = 0
		cpu.SP = initialSP - 1 // SP points to the last used location *before* pull

		runUntilBrk(cpu, bus, 10)

		if cpu.SP != initialSP {
			t.Errorf("PLA failed: Expected SP=0x%02X, got 0x%02X", initialSP, cpu.SP)
		}
		if cpu.A != valueToPull {
			t.Errorf("PLA failed: Expected A=0x%02X, got 0x%02X", valueToPull, cpu.A)
		}
		if cpu.getFlag(Z) {
			t.Errorf("PLA failed: Expected Z=false, got true")
		}
		if !cpu.getFlag(N) {
			t.Errorf("PLA failed: Expected N=true, got false")
		}
	})

	t.Run("PHP", func(t *testing.T) {
		cpu, bus = setupCPU()
		// Set some flags C=1, Z=0, I=0, D=0, V=1, N=1
		cpu.P = C | V | N | U          // Start with I=0, D=0, Z=0; U always 1
		expectedPushedP := cpu.P | B   // PHP pushes with B flag set
		program := []uint8{0x08, 0x00} // PHP, BRK
		bus.load(startAddr, program)
		cpu.PC = startAddr
		cpu.cycles = 0
		cpu.SP = initialSP

		runUntilBrk(cpu, bus, 10)

		if cpu.SP != initialSP-1 {
			t.Errorf("PHP failed: Expected SP=0x%02X, got 0x%02X", initialSP-1, cpu.SP)
		}
		pushedValue := bus.Read(stackBase + uint16(initialSP))
		if Flags(pushedValue) != expectedPushedP {
			t.Errorf("PHP failed: Expected P value 0x%02X on stack at 0x%04X, got 0x%02X",
				expectedPushedP, stackBase+uint16(initialSP), pushedValue)
		}
		// Ensure original P register didn't gain the B flag permanently
		if cpu.getFlag(B) {
			t.Errorf("PHP failed: B flag was incorrectly set in P register after PHP.")
		}
	})

	t.Run("PLP", func(t *testing.T) {
		cpu, bus = setupCPU()
		// Value to pull from stack: C=0, Z=1, I=1, D=1, V=0, N=0
		// Make sure B and U are set as PHP would have pushed them
		valueToPull := Z | I | D | B | U
		bus.Write(stackBase+uint16(initialSP), uint8(valueToPull))
		program := []uint8{0x28, 0x00} // PLP, BRK
		bus.load(startAddr, program)
		cpu.PC = startAddr
		cpu.cycles = 0
		cpu.SP = initialSP - 1 // SP points to the last used location *before* pull
		// Start with different flags to ensure they change
		cpu.P = C | V | N | U

		runUntilBrk(cpu, bus, 10)

		if cpu.SP != initialSP {
			t.Errorf("PLP failed: Expected SP=0x%02X, got 0x%02X", initialSP, cpu.SP)
		}
		// Expected flags after PLP: pulled flags excluding B, but ensuring U is set
		expectedP := (valueToPull & ^B) | U
		if cpu.P != expectedP {
			t.Errorf("PLP failed: Expected P=0x%02X (%s), got 0x%02X (%s)",
				expectedP, fmt.Sprintf("%08b", expectedP),
				cpu.P, fmt.Sprintf("%08b", cpu.P))
		}
	})
}

// --- Placeholder Tests for Complex Instructions ---
// Add more tests here as you implement instructions like ADC, SBC, branches, shifts etc.

func TestArithmeticPlaceholders(t *testing.T) {
	t.Skip("Skipping arithmetic tests (ADC/SBC) - Implement instructions first.")
	// TODO: Write tests for ADC (various modes, flags C/V/Z/N, decimal mode?)
	// TODO: Write tests for SBC (various modes, flags C/V/Z/N, decimal mode?)
}

func TestLogicPlaceholders(t *testing.T) {
	t.Skip("Skipping some logic tests (AND/EOR/ORA/BIT) - Verify implementation.")
	// Example structure:
	// cpu, bus := setupCPU()
	// cpu.A = 0b01010101
	// bus.Write(0x10, 0b11001100) // Value for operation
	// program := []uint8{0x25, 0x10, 0x00} // AND $10, BRK
	// ... run ...
	// expectedA := 0b01000100
	// Check cpu.A, cpu.getFlag(Z), cpu.getFlag(N)
}

func TestShiftRotatePlaceholders(t *testing.T) {
	t.Skip("Skipping shift/rotate tests (ASL/LSR/ROL/ROR) - Implement instructions first.")
	// TODO: Write tests for ASL (Acc & Mem, Flags C/Z/N)
	// TODO: Write tests for LSR (Acc & Mem, Flags C/Z/N)
	// TODO: Write tests for ROL (Acc & Mem, Flags C/Z/N)
	// TODO: Write tests for ROR (Acc & Mem, Flags C/Z/N)
}

func TestComparePlaceholders(t *testing.T) {
	t.Skip("Skipping compare tests (CMP/CPX/CPY) - Implement instructions first.")
	// TODO: Write tests for CMP (A=M, A<M, A>M -> Flags C/Z/N)
	// TODO: Write tests for CPX (X=M, X<M, X>M -> Flags C/Z/N)
	// TODO: Write tests for CPY (Y=M, Y<M, Y>M -> Flags C/Z/N)
}

func TestBranchPlaceholders(t *testing.T) {
	t.Skip("Skipping branch tests - Implement instructions first.")
	// TODO: Write tests for each branch type:
	// - Condition met, no page cross
	// - Condition met, page cross
	// - Condition not met
	// Verify PC and potentially cycle counts.
}

func TestJumpSubroutinePlaceholders(t *testing.T) {
	t.Skip("Skipping jump/subroutine tests - Implement instructions first.")
	// TODO: Write tests for JMP (ABS/IND, including indirect bug)
	// TODO: Write tests for JSR (check PC, SP, stack content)
	// TODO: Write tests for RTS (check PC, SP)
}

func TestInterruptPlaceholders(t *testing.T) {
	t.Skip("Skipping interrupt tests (IRQ/NMI/BRK) - Verify implementation.")
	// TODO: Test BRK thoroughly (flags, stack, PC)
	// TODO: Test IRQ (when I=0 and I=1, flags, stack, PC)
	// TODO: Test NMI (flags, stack, PC)
}

func TestIllegalOpcode(t *testing.T) {
	t.Skip("Skipping illegal opcode test - Ensure XXX handler is robust.")
	// cpu, bus := setupCPU()
	// illegalOpcode := uint8(0x02) // Example KIL/JAM opcode
	// bus.load(0x8000, []uint8{illegalOpcode})
	// cpu.PC = 0x8000
	// cpu.cycles = 0
	// // Running this might require a timeout or checking logs if XXX prints an error
	// // Or, if XXX sets a specific "halted" state in the CPU struct.
	// cpu.Clock() // Execute the illegal instruction
	// // Add assertions here based on how XXX behaves (e.g., logged error, halted state)
}
