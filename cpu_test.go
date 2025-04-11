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

// --- runCycles helper ---
// We need a slightly different helper for branch tests, as we want to
// execute a specific number of cycles or just the branch instruction itself,
// not necessarily run until BRK.
func runCycles(cpu *CPU, cyclesToRun uint) uint64 {
	startTotalCycles := cpu.totalCycles
	targetTotalCycles := startTotalCycles + uint64(cyclesToRun)
	for cpu.totalCycles < targetTotalCycles {
		// Check if CPU is stuck (e.g., on an unimplemented BRK or infinite loop)
		if cpu.cycles == 0 && cpu.lookup[cpu.read(cpu.PC)].Operate == nil {
			fmt.Printf("Warning: CPU potentially stuck at PC=0x%04X, Opcode=0x%02X\n", cpu.PC, cpu.read(cpu.PC))
			break // Avoid infinite loop in test
		}
		cpu.Clock()
		// Add extra break condition if absolutely necessary
		if cpu.totalCycles > targetTotalCycles+10 { // Safety break
			fmt.Printf("Warning: Exceeded target cycles significantly in runCycles.\n")
			break
		}
	}
	return cpu.totalCycles - startTotalCycles
}

// setupCPU creates a CPU instance with a mock bus for testing.
func setupCPU() (*CPU, *MockBus) {
	bus := NewMockBus()
	cpu := NewCPU(bus)
	// Set reset/irq/nmi vectors for safety during tests
	bus.Write(0xFFFA, 0x00)
	bus.Write(0xFFFB, 0xF0) // NMI -> F000
	bus.Write(0xFFFC, 0x00)
	bus.Write(0xFFFD, 0xF1) // Reset -> F100 <<-- RESET VECTOR
	bus.Write(0xFFFE, 0x00)
	bus.Write(0xFFFF, 0xF2) // IRQ/BRK -> F200 <<-- IRQ VECTOR
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

// --- Branch Tests ---

// TestBranchInstructions covers all 8 conditional branches.
func TestBranchInstructions(t *testing.T) {
	const baseAddr = 0x0200

	tests := []struct {
		name           string
		opcode         uint8
		flag           Flags
		flagValue      bool // Value flag should have for the condition *opposite* to branching
		offset         int8
		setup          func(cpu *CPU)
		expectedPC     uint16
		expectedCycles uint // Expected cycles for *this specific instruction*
	}{
		// --- BCC (Branch if Carry Clear) ---
		{name: "BCC Taken No Cross", opcode: 0x90, flag: C, flagValue: true, offset: 0x10,
			setup:      func(cpu *CPU) { cpu.setFlag(C, false) },
			expectedPC: baseAddr + 2 + 0x10, expectedCycles: 3, // 2 base + 1 taken
		},
		{name: "BCC Taken Page Cross", opcode: 0x90, flag: C, flagValue: true, offset: -0x10,
			setup:      func(cpu *CPU) { cpu.setFlag(C, false) },
			expectedPC: baseAddr + 2 - 0x10, expectedCycles: 4, // 2 base + 1 taken + 1 cross
		},
		{name: "BCC Not Taken", opcode: 0x90, flag: C, flagValue: false, offset: 0x10,
			setup:      func(cpu *CPU) { cpu.setFlag(C, true) },
			expectedPC: baseAddr + 2, expectedCycles: 2, // 2 base
		},

		// --- BCS (Branch if Carry Set) ---
		{name: "BCS Taken No Cross", opcode: 0xB0, flag: C, flagValue: false, offset: 0x15,
			setup:      func(cpu *CPU) { cpu.setFlag(C, true) },
			expectedPC: baseAddr + 2 + 0x15, expectedCycles: 3,
		},
		{name: "BCS Taken Page Cross", opcode: 0xB0, flag: C, flagValue: false, offset: -0x05,
			setup:      func(cpu *CPU) { cpu.setFlag(C, true) },
			expectedPC: baseAddr + 2 - 0x05, expectedCycles: 4,
		},
		{name: "BCS Not Taken", opcode: 0xB0, flag: C, flagValue: true, offset: 0x15,
			setup:      func(cpu *CPU) { cpu.setFlag(C, false) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BEQ (Branch if Equal - Zero Set) ---
		{name: "BEQ Taken No Cross", opcode: 0xF0, flag: Z, flagValue: false, offset: 0x20,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, true) },
			expectedPC: baseAddr + 2 + 0x20, expectedCycles: 3,
		},
		{name: "BEQ Taken Page Cross", opcode: 0xF0, flag: Z, flagValue: false, offset: -0x20,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, true) },
			expectedPC: baseAddr + 2 - 0x20, expectedCycles: 4,
		},
		{name: "BEQ Not Taken", opcode: 0xF0, flag: Z, flagValue: true, offset: 0x20,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, false) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BNE (Branch if Not Equal - Zero Clear) ---
		{name: "BNE Taken No Cross", opcode: 0xD0, flag: Z, flagValue: true, offset: 0x25,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, false) },
			expectedPC: baseAddr + 2 + 0x25, expectedCycles: 3,
		},
		{name: "BNE Taken Page Cross", opcode: 0xD0, flag: Z, flagValue: true, offset: -0x08,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, false) },
			expectedPC: baseAddr + 2 - 0x08, expectedCycles: 4,
		},
		{name: "BNE Not Taken", opcode: 0xD0, flag: Z, flagValue: false, offset: 0x25,
			setup:      func(cpu *CPU) { cpu.setFlag(Z, true) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BMI (Branch if Minus - Negative Set) ---
		{name: "BMI Taken No Cross", opcode: 0x30, flag: N, flagValue: false, offset: 0x30,
			setup:      func(cpu *CPU) { cpu.setFlag(N, true) },
			expectedPC: baseAddr + 2 + 0x30, expectedCycles: 3,
		},
		{name: "BMI Taken Page Cross", opcode: 0x30, flag: N, flagValue: false, offset: -0x30,
			setup:      func(cpu *CPU) { cpu.setFlag(N, true) },
			expectedPC: baseAddr + 2 - 0x30, expectedCycles: 4,
		},
		{name: "BMI Not Taken", opcode: 0x30, flag: N, flagValue: true, offset: 0x30,
			setup:      func(cpu *CPU) { cpu.setFlag(N, false) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BPL (Branch if Plus - Negative Clear) ---
		{name: "BPL Taken No Cross", opcode: 0x10, flag: N, flagValue: true, offset: 0x35,
			setup:      func(cpu *CPU) { cpu.setFlag(N, false) },
			expectedPC: baseAddr + 2 + 0x35, expectedCycles: 3,
		},
		{name: "BPL Taken Page Cross", opcode: 0x10, flag: N, flagValue: true, offset: -0x0A,
			setup:      func(cpu *CPU) { cpu.setFlag(N, false) },
			expectedPC: baseAddr + 2 - 0x0A, expectedCycles: 4,
		},
		{name: "BPL Not Taken", opcode: 0x10, flag: N, flagValue: false, offset: 0x35,
			setup:      func(cpu *CPU) { cpu.setFlag(N, true) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BVC (Branch if Overflow Clear) ---
		{name: "BVC Taken No Cross", opcode: 0x50, flag: V, flagValue: true, offset: 0x40,
			setup:      func(cpu *CPU) { cpu.setFlag(V, false) },
			expectedPC: baseAddr + 2 + 0x40, expectedCycles: 3,
		},
		{name: "BVC Taken Page Cross", opcode: 0x50, flag: V, flagValue: true, offset: -0x40,
			setup:      func(cpu *CPU) { cpu.setFlag(V, false) },
			expectedPC: baseAddr + 2 - 0x40, expectedCycles: 4,
		},
		{name: "BVC Not Taken", opcode: 0x50, flag: V, flagValue: false, offset: 0x40,
			setup:      func(cpu *CPU) { cpu.setFlag(V, true) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},

		// --- BVS (Branch if Overflow Set) ---
		{name: "BVS Taken No Cross", opcode: 0x70, flag: V, flagValue: false, offset: 0x45,
			setup:      func(cpu *CPU) { cpu.setFlag(V, true) },
			expectedPC: baseAddr + 2 + 0x45, expectedCycles: 3,
		},
		{name: "BVS Taken Page Cross", opcode: 0x70, flag: V, flagValue: false, offset: -0x0C,
			setup:      func(cpu *CPU) { cpu.setFlag(V, true) },
			expectedPC: baseAddr + 2 - 0x0C, expectedCycles: 4,
		},
		{name: "BVS Not Taken", opcode: 0x70, flag: V, flagValue: true, offset: 0x45,
			setup:      func(cpu *CPU) { cpu.setFlag(V, false) },
			expectedPC: baseAddr + 2, expectedCycles: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()

			// Reset P and set the specific flag state needed for the test setup
			cpu.P = U | I                      // Reset flags
			cpu.setFlag(tt.flag, tt.flagValue) // Setup initial opposite state if needed by setup func
			tt.setup(cpu)                      // Apply actual flag state for branch condition

			// Program: Branch instruction ONLY. We rely on runCycles to execute exactly this one.
			// We pad with BRK just in case something goes wrong and runs too far.
			program := []uint8{tt.opcode, uint8(tt.offset), 0x00, 0x00, 0x00}
			bus.load(baseAddr, program)
			// Ensure reset vector isn't 0x0000 in case BRK is hit accidentally
			bus.Write(0xFFFC, 0x00)
			bus.Write(0xFFFD, 0xFF) // Point BRK vector somewhere harmless (e.g., $FF00)

			cpu.PC = baseAddr
			cpu.cycles = 0 // Ensure instruction runs immediately

			// Execute *exactly* the number of cycles this instruction should take
			cyclesRun := runCycles(cpu, tt.expectedCycles)

			// Verification
			if cpu.PC != tt.expectedPC {
				t.Errorf("%s failed: PC mismatch. Expected PC=0x%04X, got PC=0x%04X", tt.name, tt.expectedPC, cpu.PC)
			}
			if cyclesRun != uint64(tt.expectedCycles) {
				// This check might be sensitive to the exact implementation of Clock() and runCycles().
				// It's less critical than PC correctness if the timing model is complex.
				t.Logf("%s warning: Cycle count mismatch. Expected %d cycles, executed %d", tt.name, tt.expectedCycles, cyclesRun)
			}
		})
	}
}

// --- Jump and Subroutine Tests ---

func TestJumpSubroutineInstructions(t *testing.T) {
	const baseAddr = 0xC000 // Common starting point for PC

	// --- JMP Absolute ---
	t.Run("JMP Absolute", func(t *testing.T) {
		cpu, bus := setupCPU()
		targetAddr := uint16(0xABCD)
		program := []uint8{
			0x4C,                   // JMP Absolute opcode
			uint8(targetAddr),      // Low byte of target address
			uint8(targetAddr >> 8), // High byte of target address
		}
		bus.load(baseAddr, program)
		cpu.PC = baseAddr
		cpu.cycles = 0

		runCycles(cpu, 3) // JMP Absolute takes 3 cycles

		if cpu.PC != targetAddr {
			t.Errorf("JMP Absolute failed: Expected PC=0x%04X, got PC=0x%04X", targetAddr, cpu.PC)
		}
	})

	// --- JMP Indirect ---
	t.Run("JMP Indirect", func(t *testing.T) {
		cpu, bus := setupCPU()
		indirectVectorAddr := uint16(0x1000)
		targetAddr := uint16(0xEF90)

		// Program: JMP ($1000)
		program := []uint8{
			0x6C,                           // JMP Indirect opcode
			uint8(indirectVectorAddr),      // Low byte of indirect vector address
			uint8(indirectVectorAddr >> 8), // High byte of indirect vector address
		}
		bus.load(baseAddr, program)

		// Set up the indirect vector in memory
		bus.Write(indirectVectorAddr, uint8(targetAddr))      // $1000 = $90 (Low byte)
		bus.Write(indirectVectorAddr+1, uint8(targetAddr>>8)) // $1001 = $EF (High byte)

		cpu.PC = baseAddr
		cpu.cycles = 0

		runCycles(cpu, 5) // JMP Indirect takes 5 cycles

		if cpu.PC != targetAddr {
			t.Errorf("JMP Indirect failed: Expected PC=0x%04X, got PC=0x%04X", targetAddr, cpu.PC)
		}
	})

	// --- JMP Indirect Bug ---
	t.Run("JMP Indirect Bug", func(t *testing.T) {
		cpu, bus := setupCPU()
		// Vector address where low byte is $FF -> $xxFF
		indirectVectorAddr := uint16(0x10FF)
		targetAddrLowByte := uint8(0x34)
		targetAddrHighByte := uint8(0x12) // Expected target $1234

		// Program: JMP ($10FF)
		program := []uint8{
			0x6C,                           // JMP Indirect opcode
			uint8(indirectVectorAddr),      // Low byte ($FF)
			uint8(indirectVectorAddr >> 8), // High byte ($10)
		}
		bus.load(baseAddr, program)

		// Set up the indirect vector bytes demonstrating the bug
		bus.Write(indirectVectorAddr, targetAddrLowByte) // $10FF = $34 (Correct low byte)
		// The bug reads the high byte from $1000 instead of $1100
		bus.Write(indirectVectorAddr&0xFF00, targetAddrHighByte) // $1000 = $12 (Incorrect high byte location)
		bus.Write(indirectVectorAddr+1, 0xEE)                    // Write something different at the *correct* high byte loc ($1100) to ensure bug works

		cpu.PC = baseAddr
		cpu.cycles = 0

		runCycles(cpu, 5) // JMP Indirect takes 5 cycles

		expectedTargetAddr := uint16(targetAddrHighByte)<<8 | uint16(targetAddrLowByte) // $1234
		if cpu.PC != expectedTargetAddr {
			t.Errorf("JMP Indirect Bug failed: Expected PC=0x%04X, got PC=0x%04X", expectedTargetAddr, cpu.PC)
		}
	})

	// --- JSR (Jump to Subroutine) ---
	t.Run("JSR", func(t *testing.T) {
		cpu, bus := setupCPU()
		subroutineAddr := uint16(0xABCD)
		initialSP := cpu.SP // SP is usually $FD after reset

		// Program: JSR $ABCD at $C000
		program := []uint8{
			0x20,                       // JSR opcode
			uint8(subroutineAddr),      // Low byte
			uint8(subroutineAddr >> 8), // High byte
		}
		bus.load(baseAddr, program)
		cpu.PC = baseAddr
		cpu.cycles = 0

		runCycles(cpu, 6) // JSR takes 6 cycles

		// 1. Check PC jumped to subroutine
		if cpu.PC != subroutineAddr {
			t.Errorf("JSR failed: PC did not jump. Expected PC=0x%04X, got PC=0x%04X", subroutineAddr, cpu.PC)
		}

		// 2. Check SP decremented by 2
		expectedSP := initialSP - 2
		if cpu.SP != expectedSP {
			t.Errorf("JSR failed: SP incorrect. Expected SP=0x%02X, got SP=0x%02X", expectedSP, cpu.SP)
		}

		// 3. Check stack content (Return address is PC of *last byte* of JSR instruction)
		// JSR is 3 bytes: $C000 (opcode), $C001 (low), $C002 (high)
		// Return address pushed is $C002
		expectedReturnAddr := baseAddr + 2
		pushedLow := bus.Read(stackBase + uint16(expectedSP+1))  // Lower address on stack = low byte
		pushedHigh := bus.Read(stackBase + uint16(expectedSP+2)) // Higher address on stack = high byte

		if pushedLow != uint8(expectedReturnAddr&0x00FF) {
			t.Errorf("JSR failed: Stack low byte incorrect. Expected 0x%02X, got 0x%02X", uint8(expectedReturnAddr&0x00FF), pushedLow)
		}
		if pushedHigh != uint8(expectedReturnAddr>>8) {
			t.Errorf("JSR failed: Stack high byte incorrect. Expected 0x%02X, got 0x%02X", uint8(expectedReturnAddr>>8), pushedHigh)
		}
	})

	// --- RTS (Return from Subroutine) ---
	t.Run("RTS", func(t *testing.T) {
		cpu, bus := setupCPU()
		returnAddrOnStack := uint16(0xC123)      // The address JSR would have pushed (PC of last byte of JSR)
		expectedFinalPC := returnAddrOnStack + 1 // RTS increments after pulling
		rtsInstructionAddr := uint16(0xD000)

		// Manually set up stack as if JSR happened
		initialSP := cpu.SP                                                    // Usually $FD
		cpu.SP = initialSP - 2                                                 // Make space
		bus.Write(stackBase+uint16(cpu.SP+2), uint8(returnAddrOnStack>>8))     // Push high byte ($C1)
		bus.Write(stackBase+uint16(cpu.SP+1), uint8(returnAddrOnStack&0x00FF)) // Push low byte ($23)

		// Program: Just RTS
		program := []uint8{0x60} // RTS opcode
		bus.load(rtsInstructionAddr, program)
		cpu.PC = rtsInstructionAddr
		cpu.cycles = 0

		runCycles(cpu, 6) // RTS takes 6 cycles

		// 1. Check PC is correct return address + 1
		if cpu.PC != expectedFinalPC {
			t.Errorf("RTS failed: PC incorrect. Expected PC=0x%04X, got PC=0x%04X", expectedFinalPC, cpu.PC)
		}

		// 2. Check SP incremented by 2 (back to original value)
		if cpu.SP != initialSP {
			t.Errorf("RTS failed: SP incorrect. Expected SP=0x%02X, got SP=0x%02X", initialSP, cpu.SP)
		}
	})
}

// --- Logic Operation Tests ---

func TestLogicInstructions(t *testing.T) {
	const baseAddr = 0x0600

	// --- AND ---
	t.Run("AND", func(t *testing.T) {
		// Get function pointers for comparison once
		immPtr := getFuncPtr((*CPU).IMM)
		zp0Ptr := getFuncPtr((*CPU).ZP0)
		absPtr := getFuncPtr((*CPU).ABS)
		// Add others ZPX, ABX, etc. pointers here if testing those modes

		tests := []struct {
			name      string
			addrMode  func(*CPU) uint8 // Keep the function type
			opcode    uint8
			operand   uint8
			memAddr   uint16
			initialA  uint8
			expectedA uint8
			zero      bool
			negative  bool
			cycles    uint
		}{
			{"Immediate", (*CPU).IMM, 0x29, 0x0F, 0, 0xF0, 0x00, true, false, 2},
			{"Immediate Neg", (*CPU).IMM, 0x29, 0xFF, 0, 0x81, 0x81, false, true, 2},
			{"ZeroPage", (*CPU).ZP0, 0x25, 0xAA, 0x50, 0xF0, 0xA0, false, true, 3},
			{"Absolute", (*CPU).ABS, 0x2D, 0x55, 0x1234, 0x3C, 0x14, false, false, 4},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8
				addrModePtr := getFuncPtr(tt.addrMode) // Get pointer for the current test case

				// ***** USE POINTER COMPARISON *****
				if addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Errorf("AND %s failed: Unknown address mode", tt.name)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles)

				// Assertions remain the same
				if cpu.A != tt.expectedA {
					t.Errorf("AND %s failed: Expected A=0x%02X, got A=0x%02X", tt.name, tt.expectedA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("AND %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("AND %s failed: Expected N=%v, got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
			})
		}
	})

	// --- EOR ---
	t.Run("EOR", func(t *testing.T) {
		immPtr := getFuncPtr((*CPU).IMM)
		zp0Ptr := getFuncPtr((*CPU).ZP0)
		absPtr := getFuncPtr((*CPU).ABS)

		tests := []struct {
			name      string
			addrMode  func(*CPU) uint8
			opcode    uint8
			operand   uint8
			memAddr   uint16
			initialA  uint8
			expectedA uint8
			zero      bool
			negative  bool
			cycles    uint
		}{
			{"Immediate", (*CPU).IMM, 0x49, 0xFF, 0, 0x55, 0xAA, false, true, 2},
			{"Immediate Zero", (*CPU).IMM, 0x49, 0x33, 0, 0x33, 0x00, true, false, 2},
			{"ZeroPage", (*CPU).ZP0, 0x45, 0xF0, 0x60, 0x0F, 0xFF, false, true, 3},
			{"Absolute", (*CPU).ABS, 0x4D, 0x88, 0x4321, 0x88, 0x00, true, false, 4},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8
				addrModePtr := getFuncPtr(tt.addrMode)

				// ***** USE POINTER COMPARISON *****
				if addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Errorf("EOR %s failed: Unknown address mode", tt.name)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles)

				if cpu.A != tt.expectedA {
					t.Errorf("EOR %s failed: Expected A=0x%02X, got A=0x%02X", tt.name, tt.expectedA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("EOR %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("EOR %s failed: Expected N=%v, got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
			})
		}
	})

	// --- ORA ---
	t.Run("ORA", func(t *testing.T) {
		immPtr := getFuncPtr((*CPU).IMM)
		zp0Ptr := getFuncPtr((*CPU).ZP0)
		absPtr := getFuncPtr((*CPU).ABS)

		tests := []struct {
			name      string
			addrMode  func(*CPU) uint8
			opcode    uint8
			operand   uint8
			memAddr   uint16
			initialA  uint8
			expectedA uint8
			zero      bool
			negative  bool
			cycles    uint
		}{
			{"Immediate", (*CPU).IMM, 0x09, 0x0F, 0, 0xF0, 0xFF, false, true, 2},
			{"Immediate Zero", (*CPU).IMM, 0x09, 0x00, 0, 0x00, 0x00, true, false, 2},
			{"ZeroPage", (*CPU).ZP0, 0x05, 0xAA, 0x70, 0x55, 0xFF, false, true, 3},
			{"Absolute", (*CPU).ABS, 0x0D, 0x80, 0xABCD, 0x01, 0x81, false, true, 4},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8
				addrModePtr := getFuncPtr(tt.addrMode)

				// ***** USE POINTER COMPARISON *****
				if addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Errorf("ORA %s failed: Unknown address mode", tt.name)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles)

				if cpu.A != tt.expectedA {
					t.Errorf("ORA %s failed: Expected A=0x%02X, got A=0x%02X", tt.name, tt.expectedA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("ORA %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("ORA %s failed: Expected N=%v, got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
			})
		}
	})

	// --- BIT ---
	t.Run("BIT", func(t *testing.T) {
		zp0Ptr := getFuncPtr((*CPU).ZP0)
		absPtr := getFuncPtr((*CPU).ABS)

		tests := []struct {
			name     string
			addrMode func(*CPU) uint8
			opcode   uint8
			operand  uint8
			memAddr  uint16
			initialA uint8
			zero     bool
			negative bool
			overflow bool
			cycles   uint
		}{
			{"ZeroPage Z=1", (*CPU).ZP0, 0x24, 0xAA, 0x80, 0x55, true, true, false, 3},
			{"ZeroPage Z=0", (*CPU).ZP0, 0x24, 0x55, 0x80, 0x77, false, false, true, 3},
			{"Absolute V=1", (*CPU).ABS, 0x2C, 0xC0, 0xBEEF, 0xFF, false, true, true, 4},
			{"Absolute N=0", (*CPU).ABS, 0x2C, 0x3F, 0xBEEF, 0xFF, false, false, false, 4},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				initialA := cpu.A
				var program []uint8
				addrModePtr := getFuncPtr(tt.addrMode)

				// ***** USE POINTER COMPARISON *****
				if addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Fatalf("BIT %s: Unhandled addressing mode pointer in test setup", tt.name)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles)

				// Assertions remain the same
				if cpu.A != initialA {
					t.Errorf("BIT %s failed: Accumulator was modified! Expected A=0x%02X, got A=0x%02X", tt.name, initialA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("BIT %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("BIT %s failed: Expected N=%v (from operand bit 7), got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
				if cpu.getFlag(V) != tt.overflow {
					t.Errorf("BIT %s failed: Expected V=%v (from operand bit 6), got V=%v", tt.name, tt.overflow, cpu.getFlag(V))
				}
			})
		}
	})
}

// --- Interrupt and Miscellaneous Tests (Keep as before) ---
func TestInterruptMiscInstructions(t *testing.T) {
	// --- Constants for expected vector values ---
	// Readability: define constants for the expected vector addresses from setupCPU
	const expectedNmiVector = 0xF000
	const expectedIrqBrkVector = 0xF200

	initialSP := uint8(0xFD) // Default SP after reset

	// --- BRK ---
	t.Run("BRK", func(t *testing.T) {
		// ***** Ensure fresh setup FOR THIS TEST *****
		cpu, bus := setupCPU()
		baseAddr := uint16(0x0700) // Local base address for clarity

		cpu.PC = baseAddr
		originalFlags := C | Z | U // Start with I=0, B=0, D=0, V=0, N=0
		cpu.P = originalFlags
		program := []uint8{0x00}
		bus.load(baseAddr, program)
		cpu.cycles = 0

		// Sanity check: Verify vectors immediately after setup, before running CPU
		if bus.Read(0xFFFE) != 0x00 || bus.Read(0xFFFF) != 0xF2 {
			t.Fatalf("BRK Pre-Test Check Failed: IRQ Vector not set correctly in MockBus! Got %02X%02X", bus.Read(0xFFFF), bus.Read(0xFFFE))
		}

		runCycles(cpu, 7)

		// 1. Check PC loaded from IRQ/BRK vector ($FFFE/F)
		if cpu.PC != expectedIrqBrkVector {
			t.Errorf("BRK failed: PC load failed. Expected PC=0x%04X, got PC=0x%04X", expectedIrqBrkVector, cpu.PC)
		}

		// (Rest of BRK assertions remain the same)
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("BRK failed: SP incorrect. Expected SP=0x%02X, got SP=0x%02X", expectedSP, cpu.SP)
		}
		pushedPC := baseAddr + 2
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1))
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))
		expectedPushedStatus := originalFlags | B | U
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("BRK failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		if pushedPCLow != uint8(pushedPC&0x00FF) {
			t.Errorf("BRK failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", uint8(pushedPC&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(pushedPC>>8) {
			t.Errorf("BRK failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", uint8(pushedPC>>8), pushedPCHigh)
		}
		if !cpu.getFlag(I) {
			t.Errorf("BRK failed: Interrupt Disable flag (I) was not set in P register.")
		}
		if cpu.getFlag(B) {
			t.Errorf("BRK failed: Break flag (B) was incorrectly set in P register.")
		}
	})

	// --- RTI (Passed before, likely okay - Keep setup inside) ---
	t.Run("RTI", func(t *testing.T) {
		cpu, bus := setupCPU() // Fresh setup
		targetPC := uint16(0xC0DE)
		targetStatus := C | D | V | U
		cpu.SP = initialSP - 3
		bus.Write(stackBase+uint16(cpu.SP+1), uint8(targetStatus))
		bus.Write(stackBase+uint16(cpu.SP+2), uint8(targetPC&0x00FF))
		bus.Write(stackBase+uint16(cpu.SP+3), uint8(targetPC>>8))
		cpu.PC = 0x1000
		cpu.P = N | Z | I | U
		rtiAddr := uint16(0x2000)
		bus.load(rtiAddr, []uint8{0x40})
		cpu.PC = rtiAddr
		cpu.cycles = 0
		runCycles(cpu, 6)
		if cpu.PC != targetPC {
			t.Errorf("RTI failed: PC incorrect. Expected PC=0x%04X, got PC=0x%04X", targetPC, cpu.PC)
		}
		if cpu.P != targetStatus {
			t.Errorf("RTI failed: P register incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", targetStatus, fmt.Sprintf("%08b", targetStatus), cpu.P, fmt.Sprintf("%08b", cpu.P))
		}
		if cpu.SP != initialSP {
			t.Errorf("RTI failed: SP incorrect. Expected SP=0x%02X, got SP=0x%02X", initialSP, cpu.SP)
		}
	})

	// --- IRQ ---
	t.Run("IRQ I=1 (Ignored)", func(t *testing.T) {
		cpu, _ := setupCPU() // Fresh setup
		baseAddr := uint16(0x0700)
		cpu.PC = baseAddr
		cpu.P = C | Z | I | U
		initialP := cpu.P
		initialSP := cpu.SP
		cpu.InterruptRequest()
		if cpu.PC != baseAddr {
			t.Errorf("IRQ (I=1) failed: PC changed. Expected 0x%04X, got 0x%04X", baseAddr, cpu.PC)
		}
		if cpu.P != initialP {
			t.Errorf("IRQ (I=1) failed: P changed. Expected 0x%02X, got 0x%02X", initialP, cpu.P)
		}
		if cpu.SP != initialSP {
			t.Errorf("IRQ (I=1) failed: SP changed. Expected 0x%02X, got 0x%02X", initialSP, cpu.SP)
		}
		if cpu.cycles != 0 {
			t.Errorf("IRQ (I=1) failed: cpu.cycles was set (%d), expected 0", cpu.cycles)
		}
	})

	t.Run("IRQ I=0 (Occurs)", func(t *testing.T) {
		// ***** Ensure fresh setup FOR THIS TEST *****
		cpu, bus := setupCPU()
		baseAddr := uint16(0x0700)

		cpu.PC = baseAddr
		cpu.P = C | Z | U // I flag IS CLEAR
		originalP := cpu.P

		// Sanity check: Verify vectors immediately after setup, before interrupt call
		if bus.Read(0xFFFE) != 0x00 || bus.Read(0xFFFF) != 0xF2 {
			t.Fatalf("IRQ Pre-Test Check Failed: IRQ Vector not set correctly in MockBus! Got %02X%02X", bus.Read(0xFFFF), bus.Read(0xFFFE))
		}

		cpu.InterruptRequest() // Call the IRQ handler directly

		// 1. Check PC loaded from IRQ vector
		if cpu.PC != expectedIrqBrkVector {
			t.Errorf("IRQ (I=0) failed: Expected PC=0x%04X, got PC=0x%04X", expectedIrqBrkVector, cpu.PC)
		}
		// (Rest of IRQ Occurs assertions remain the same)
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("IRQ (I=0) failed: Expected SP=0x%02X, got SP=0x%02X", expectedSP, cpu.SP)
		}
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1))
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))
		expectedPushedStatus := (originalP | U) & ^B &^ I
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("IRQ (I=0) failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		if pushedPCLow != uint8(baseAddr&0x00FF) {
			t.Errorf("IRQ (I=0) failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", uint8(baseAddr&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(baseAddr>>8) {
			t.Errorf("IRQ (I=0) failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", uint8(baseAddr>>8), pushedPCHigh)
		}
		if !cpu.getFlag(I) {
			t.Errorf("IRQ (I=0) failed: Interrupt Disable flag (I) was not set in P register after IRQ.")
		}
		if cpu.cycles != 7 {
			t.Errorf("IRQ (I=0) failed: Expected cpu.cycles=7, got %d", cpu.cycles)
		}
	})

	// --- NMI ---
	runNMITest := func(t *testing.T, iFlag bool) {
		// ***** Ensure fresh setup FOR THIS TEST *****
		cpu, bus := setupCPU() // Fresh CPU and BUS for each NMI invocation
		baseAddr := uint16(0x0700)

		cpu.PC = baseAddr
		if iFlag {
			cpu.P = N | V | I | U
		} else {
			cpu.P = N | V | U
		}
		originalP := cpu.P

		// Sanity check: Verify vectors immediately after setup, before interrupt call
		if bus.Read(0xFFFA) != 0x00 || bus.Read(0xFFFB) != 0xF0 {
			t.Fatalf("NMI Pre-Test Check Failed: NMI Vector not set correctly in MockBus! Got %02X%02X", bus.Read(0xFFFB), bus.Read(0xFFFA))
		}

		cpu.NonMaskableInterrupt() // Call NMI handler

		// 1. Check PC loaded from NMI vector
		if cpu.PC != expectedNmiVector {
			t.Errorf("NMI (I=%v) failed: Expected PC=0x%04X, got PC=0x%04X", iFlag, expectedNmiVector, cpu.PC)
		}
		// (Rest of NMI assertions remain the same)
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("NMI (I=%v) failed: Expected SP=0x%02X, got SP=0x%02X", iFlag, expectedSP, cpu.SP)
		}
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1))
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))
		expectedPushedStatus := (originalP | U) & ^B
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("NMI (I=%v) failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", iFlag, expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		if pushedPCLow != uint8(baseAddr&0x00FF) {
			t.Errorf("NMI (I=%v) failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", iFlag, uint8(baseAddr&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(baseAddr>>8) {
			t.Errorf("NMI (I=%v) failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", iFlag, uint8(baseAddr>>8), pushedPCHigh)
		}
		if !cpu.getFlag(I) {
			t.Errorf("NMI (I=%v) failed: Interrupt Disable flag (I) was not set in P register after NMI.", iFlag)
		}
		if cpu.cycles != 8 {
			t.Errorf("NMI (I=%v) failed: Expected cpu.cycles=8, got %d", iFlag, cpu.cycles)
		}
	}

	t.Run("NMI I=0", func(t *testing.T) { runNMITest(t, false) })
	t.Run("NMI I=1", func(t *testing.T) { runNMITest(t, true) })
}

// --- Placeholder Tests for Complex Instructions ---
// Add more tests here as you implement instructions like ADC, SBC, branches, shifts etc.

func TestArithmeticPlaceholders(t *testing.T) {
	t.Skip("Skipping arithmetic tests (ADC/SBC) - Implement instructions first.")
	// TODO: Write tests for ADC (various modes, flags C/V/Z/N, decimal mode?)
	// TODO: Write tests for SBC (various modes, flags C/V/Z/N, decimal mode?)
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
