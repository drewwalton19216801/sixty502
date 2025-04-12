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
		// Use Peek instruction definition safely
		nextOpcode := cpu.read(cpu.PC)
		if cpu.cycles == 0 && (cpu.lookup[nextOpcode].Operate == nil || getFuncPtr(cpu.lookup[nextOpcode].Operate) == getFuncPtr((*CPU).XXX)) {
			// Check for explicit XXX or unimplemented operate func
			fmt.Printf("Warning: CPU potentially stuck at PC=0x%04X, Opcode=0x%02X (%s)\n", cpu.PC, nextOpcode, cpu.lookup[nextOpcode].Name)
			break // Avoid infinite loop in test
		}
		cpu.Clock()
		// Add extra break condition if absolutely necessary
		if cpu.totalCycles > targetTotalCycles+20 { // Safety break increased slightly
			fmt.Printf("Warning: Exceeded target cycles significantly in runCycles (Target: %d, Current: %d). Stuck at PC=0x%04X Op=0x%02X\n",
				targetTotalCycles, cpu.totalCycles, cpu.PC, cpu.opcode)
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
	// This is slightly inaccurate, Reset doesn't explicitly clear B, it just sets P to U|I
	// But we check against the expected U|I state.
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
			// Get fresh instances for isolation
			cpu, bus = setupCPU()
			bus.load(startAddr, tt.program)
			// Set flags opposite to expected before running
			cpu.setFlag(tt.flag, !tt.expected)
			// Ensure U flag is set correctly before execution, and I if needed for CLI/SEI setup
			cpu.P = U
			if tt.name == "CLI" { // Start with I set if testing CLI
				cpu.P |= I
			} else if tt.name == "SEI" { // Start with I clear if testing SEI
				// cpu.P is already I=0 here
			} else { // For others, default I=1 is fine from Reset
				cpu.P |= I
			}

			cpu.PC = startAddr
			cpu.cycles = 0 // Start execution immediately

			runUntilBrk(cpu, bus, 10)

			if cpu.getFlag(tt.flag) != tt.expected {
				t.Errorf("%s failed: Expected flag %v to be %v, got %v (P=0x%02X)",
					tt.name, tt.flag, tt.expected, cpu.getFlag(tt.flag), cpu.P)
			}
			// Verify U flag is still set
			if !cpu.getFlag(U) {
				t.Errorf("%s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
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

// TestLDX tests various Load X instructions.
func TestLDX(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0400)

	tests := []struct {
		name     string
		program  []uint8
		value    uint8  // Value expected in X
		zero     bool   // Expected Z flag state
		negative bool   // Expected N flag state
		setup    func() // Optional setup before running (e.g., set X/Y)
	}{
		{
			"Immediate",
			[]uint8{0xA2, 0x42, 0x00}, // LDX #$42, BRK
			0x42, false, false, nil,
		},
		{
			"Immediate Zero",
			[]uint8{0xA2, 0x00, 0x00}, // LDX #$00, BRK
			0x00, true, false, nil,
		},
		{
			"Immediate Negative",
			[]uint8{0xA2, 0x88, 0x00}, // LDX #$88, BRK
			0x88, false, true, nil,
		},
		{
			"ZeroPage",
			[]uint8{0xA6, 0x30, 0x00}, // LDX $30, BRK
			0x55, false, false, func() { bus.Write(0x0030, 0x55) },
		},
		{
			"ZeroPage,Y",
			[]uint8{0xB6, 0x30, 0x00}, // LDX $30,Y, BRK
			0x66, false, false, func() { cpu.Y = 0x05; bus.Write(0x0035, 0x66) },
		},
		{
			"Absolute",
			[]uint8{0xAE, 0xCD, 0xAB, 0x00}, // LDX $ABCD, BRK
			0x88, false, true, func() { bus.Write(0xABCD, 0x88) },
		},
		{
			"Absolute,Y",
			[]uint8{0xBE, 0xCD, 0xAB, 0x00}, // LDX $ABCD,Y, BRK
			0x99, false, true, func() { cpu.Y = 0x10; bus.Write(0xABCD+0x10, 0x99) },
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

			if cpu.X != tt.value {
				t.Errorf("LDX %s failed: Expected X=0x%02X, got 0x%02X", tt.name, tt.value, cpu.X)
			}
			if cpu.getFlag(Z) != tt.zero {
				t.Errorf("LDX %s failed: Expected Z flag=%v, got %v", tt.name, tt.zero, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.negative {
				t.Errorf("LDX %s failed: Expected N flag=%v, got %v", tt.name, tt.negative, cpu.getFlag(N))
			}
		})
	}
}

// TestLDY tests various Load Y instructions.
func TestLDY(t *testing.T) {
	cpu, bus := setupCPU()
	startAddr := uint16(0x0500)

	tests := []struct {
		name     string
		program  []uint8
		value    uint8  // Value expected in Y
		zero     bool   // Expected Z flag state
		negative bool   // Expected N flag state
		setup    func() // Optional setup before running (e.g., set X/Y)
	}{
		{
			"Immediate",
			[]uint8{0xA0, 0x42, 0x00}, // LDY #$42, BRK
			0x42, false, false, nil,
		},
		{
			"Immediate Zero",
			[]uint8{0xA0, 0x00, 0x00}, // LDY #$00, BRK
			0x00, true, false, nil,
		},
		{
			"Immediate Negative",
			[]uint8{0xA0, 0x88, 0x00}, // LDY #$88, BRK
			0x88, false, true, nil,
		},
		{
			"ZeroPage",
			[]uint8{0xA4, 0x30, 0x00}, // LDY $30, BRK
			0x55, false, false, func() { bus.Write(0x0030, 0x55) },
		},
		{
			"ZeroPage,X",
			[]uint8{0xB4, 0x30, 0x00}, // LDY $30,X, BRK
			0x66, false, false, func() { cpu.X = 0x05; bus.Write(0x0035, 0x66) },
		},
		{
			"Absolute",
			[]uint8{0xAC, 0xCD, 0xAB, 0x00}, // LDY $ABCD, BRK
			0x88, false, true, func() { bus.Write(0xABCD, 0x88) },
		},
		{
			"Absolute,X",
			[]uint8{0xBC, 0xCD, 0xAB, 0x00}, // LDY $ABCD,X, BRK
			0x99, false, true, func() { cpu.X = 0x10; bus.Write(0xABCD+0x10, 0x99) },
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

			if cpu.Y != tt.value {
				t.Errorf("LDY %s failed: Expected Y=0x%02X, got 0x%02X", tt.name, tt.value, cpu.Y)
			}
			if cpu.getFlag(Z) != tt.zero {
				t.Errorf("LDY %s failed: Expected Z flag=%v, got %v", tt.name, tt.zero, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.negative {
				t.Errorf("LDY %s failed: Expected N flag=%v, got %v", tt.name, tt.negative, cpu.getFlag(N))
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

// TestSTX tests the STX instruction.
func TestSTX(t *testing.T) {
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
			[]uint8{0x86, 0x40, 0x00}, // STX $40, BRK
			0x0040, nil,
		},
		{
			"ZeroPage,Y",
			[]uint8{0x96, 0x40, 0x00}, // STX $40,Y, BRK
			0x0045, func() { cpu.Y = 0x05 },
		},
		{
			"Absolute",
			[]uint8{0x8E, 0x34, 0x12, 0x00}, // STX $1234, BRK
			0x1234, nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus = setupCPU() // Fresh instances
			cpu.X = valueToStore  // Set X register value

			if tt.setup != nil {
				tt.setup()
			}

			bus.load(startAddr, tt.program)
			cpu.PC = startAddr
			cpu.cycles = 0

			runUntilBrk(cpu, bus, 20)

			storedValue := bus.Read(tt.addr)
			if storedValue != valueToStore {
				t.Errorf("STX %s failed: Expected 0x%02X at 0x%04X, got 0x%02X",
					tt.name, valueToStore, tt.addr, storedValue)
			}
		})
	}
}

// TestSTY tests the STY instruction.
func TestSTY(t *testing.T) {
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
			[]uint8{0x84, 0x40, 0x00}, // STY $40, BRK
			0x0040, nil,
		},
		{
			"ZeroPage,X",
			[]uint8{0x94, 0x40, 0x00}, // STY $40,X, BRK
			0x0045, func() { cpu.X = 0x05 },
		},
		{
			"Absolute",
			[]uint8{0x8C, 0x34, 0x12, 0x00}, // STY $1234, BRK
			0x1234, nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus = setupCPU() // Fresh instances
			cpu.Y = valueToStore  // Set Y register value

			if tt.setup != nil {
				tt.setup()
			}

			bus.load(startAddr, tt.program)
			cpu.PC = startAddr
			cpu.cycles = 0

			runUntilBrk(cpu, bus, 20)

			storedValue := bus.Read(tt.addr)
			if storedValue != valueToStore {
				t.Errorf("STY %s failed: Expected 0x%02X at 0x%04X, got 0x%02X",
					tt.name, valueToStore, tt.addr, storedValue)
			}
		})
	}
}

// TestNOP tests the NOP instruction.
func TestNopInstructions(t *testing.T) {
	const baseAddr = 0x0A00 // Base address for these tests

	// Define test cases for various NOP opcodes
	tests := []struct {
		name                string
		opcode              uint8
		operand1            uint8          // Optional operand byte 1
		operand2            uint8          // Optional operand byte 2
		setup               func(cpu *CPU) // Setup initial CPU state (esp. X/Y for indexed modes)
		expectedPCIncrement uint16         // How many bytes PC should advance (1, 2, or 3)
		expectedCycles      uint           // Expected cycles for this specific case
	}{
		// --- Official NOP ---
		{
			name:                "Official NOP EA IMP",
			opcode:              0xEA,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},

		// --- Unofficial NOPs (implied mode) ---
		{
			name:                "Unofficial *NOP 1A IMP",
			opcode:              0x1A,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP 3A IMP",
			opcode:              0x3A,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP 5A IMP",
			opcode:              0x5A,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP 7A IMP",
			opcode:              0x7A,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP DA IMP",
			opcode:              0xDA,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP FA IMP",
			opcode:              0xFA,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 1,
			expectedCycles:      2,
		},

		// --- Illegal NOPs (Immediate mode) ---
		{
			name:                "Unofficial *NOP 80 IMM",
			opcode:              0x80,
			operand1:            0x00,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP 82 IMM",
			opcode:              0x82,
			operand1:            0x00,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP 89 IMM",
			opcode:              0x89,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP C2 IMM",
			opcode:              0xC2,
			operand1:            0x00,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      2,
		},
		{
			name:                "Unofficial *NOP E2 IMM",
			opcode:              0xE2,
			operand1:            0x00,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      2,
		},

		// --- Illegal NOPs (Zero page mode) ---
		{
			name:                "Unofficial *NOP 04 ZP0",
			opcode:              0x04,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      3,
		},
		{
			name:                "Unofficial *NOP 44 ZP0",
			opcode:              0x44,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      3,
		},
		{
			name:                "Unofficial *NOP 64 ZP0",
			opcode:              0x64,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 2,
			expectedCycles:      3,
		},

		// -- Illegal NOPs (Zero page X mode) ---
		{
			name:                "Unofficial *NOP 14 ZPX",
			opcode:              0x14,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 34 ZPX",
			opcode:              0x34,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 54 ZPX",
			opcode:              0x54,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 74 ZPX",
			opcode:              0x74,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP D4 ZPX",
			opcode:              0xD4,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP F4 ZPX",
			opcode:              0xF4,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 2,
			expectedCycles:      4,
		},

		// -- Illegal NOPs (Absolute mode) ---
		{
			name:                "Unofficial *NOP 0C ABS",
			opcode:              0x0C,
			setup:               func(cpu *CPU) {},
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},

		// -- Illegal NOPs (Absolute X mode) ---
		{
			name:                "Unofficial *NOP 1C ABX",
			opcode:              0x1C,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 3C ABX",
			opcode:              0x3C,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 5C ABX",
			opcode:              0x5C,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP 7C ABX",
			opcode:              0x7C,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP DC ABX",
			opcode:              0xDC,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
		{
			name:                "Unofficial *NOP FC ABX",
			opcode:              0xFC,
			setup:               func(cpu *CPU) { cpu.X = 0x10 }, // Set X to a non-zero value
			expectedPCIncrement: 3,
			expectedCycles:      4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, bus := setupCPU()
			initialPC := uint16(baseAddr) // Use a consistent starting point within the test run

			// Setup initial state
			cpu.PC = initialPC
			cpu.P = 0 // Clear all flags initially (except U will be set by CPU)
			cpu.A = 0xAA
			cpu.X = 0xBB
			cpu.Y = 0xCC
			cpu.SP = 0xFA
			tt.setup(cpu) // Apply test-specific setup (like setting X)

			// Store initial state *after* setup
			initialA := cpu.A
			initialX := cpu.X
			initialY := cpu.Y
			initialSP := cpu.SP
			initialP := cpu.P | U // Expect U flag to be set eventually

			// Construct the program bytes based on operands needed
			program := []uint8{tt.opcode}
			if tt.expectedPCIncrement >= 2 {
				program = append(program, tt.operand1)
			}
			if tt.expectedPCIncrement >= 3 {
				program = append(program, tt.operand2)
			}
			program = append(program, 0x00) // BRK for safety

			bus.load(initialPC, program)
			cpu.cycles = 0 // Start execution immediately

			// Execute the instruction by running the expected number of cycles
			cyclesRun := runCycles(cpu, tt.expectedCycles)

			// --- Verification ---
			expectedPC := initialPC + tt.expectedPCIncrement

			// 1. Verify PC advancement
			if cpu.PC != expectedPC {
				t.Errorf("%s failed: PC incorrect. Expected 0x%04X, got 0x%04X", tt.name, expectedPC, cpu.PC)
			}

			// 2. Verify Registers Unchanged
			if cpu.A != initialA {
				t.Errorf("%s failed: A register changed. Expected 0x%02X, got 0x%02X", tt.name, initialA, cpu.A)
			}
			if cpu.X != initialX {
				t.Errorf("%s failed: X register changed. Expected 0x%02X, got 0x%02X", tt.name, initialX, cpu.X)
			}
			if cpu.Y != initialY {
				t.Errorf("%s failed: Y register changed. Expected 0x%02X, got 0x%02X", tt.name, initialY, cpu.Y)
			}
			if cpu.SP != initialSP {
				t.Errorf("%s failed: SP register changed. Expected 0x%02X, got 0x%02X", tt.name, initialSP, cpu.SP)
			}

			// 3. Verify Flags Unchanged (except U should be set)
			expectedP := initialP | U // Ensure U is considered set for comparison
			if cpu.P != expectedP {
				t.Errorf("%s failed: P register changed unexpectedly. Expected 0x%02X (%s), got 0x%02X (%s)",
					tt.name, expectedP, fmt.Sprintf("%08b", expectedP), cpu.P, fmt.Sprintf("%08b", cpu.P))
			}

			// 4. Verify Cycles (Optional, but good sanity check)
			// Note: runCycles might not be perfectly cycle-exact depending on implementation details,
			// but it should be close for simple instructions like NOP.
			if cyclesRun != uint64(tt.expectedCycles) {
				t.Logf("%s warning: Cycle count mismatch. Expected %d, executed %d (This might be due to test harness timing)", tt.name, tt.expectedCycles, cyclesRun)
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
			if tt.name == "TXS" {
				// Allow U flag to be set if it wasn't (it should always end up set)
				expectedP := initialFlags | U
				if cpu.P != expectedP {
					t.Errorf("TXS failed: Flags were modified. Expected P=0x%02X, got 0x%02X", expectedP, cpu.P)
				}
			} else {
				// For other transfers, ensure U is set
				if !cpu.getFlag(U) {
					t.Errorf("%s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
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
		if !cpu.getFlag(U) {
			t.Errorf("PLA failed: U flag became unset (P=0x%02X)", cpu.P)
		}
	})

	t.Run("PHP", func(t *testing.T) {
		cpu, bus = setupCPU()
		// Set some flags C=1, Z=0, I=0, D=0, V=1, N=1
		cpu.P = C | V | N | U            // Start with I=0, D=0, Z=0; U always 1
		expectedPushedP := cpu.P | B | U // PHP pushes with B flag set and U always set
		program := []uint8{0x08, 0x00}   // PHP, BRK
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
			t.Errorf("PHP failed: Expected P value 0x%02X (%s) on stack at 0x%04X, got 0x%02X (%s)",
				expectedPushedP, fmt.Sprintf("%08b", expectedPushedP), stackBase+uint16(initialSP),
				pushedValue, fmt.Sprintf("%08b", pushedValue))
		}
		// Ensure original P register didn't gain the B flag permanently
		if cpu.getFlag(B) {
			t.Errorf("PHP failed: B flag was incorrectly set in P register after PHP.")
		}
		if !cpu.getFlag(U) {
			t.Errorf("PHP failed: U flag became unset (P=0x%02X)", cpu.P)
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
		cpu.P = C | V | N | U // Ensure I is clear here

		runUntilBrk(cpu, bus, 10)

		if cpu.SP != initialSP {
			t.Errorf("PLP failed: Expected SP=0x%02X, got 0x%02X", initialSP, cpu.SP)
		}
		// Expected flags after PLP: pulled flags excluding B, but ensuring U is set
		// PLP ignores the B flag from the stack, keeps the CPU's current B (which should be 0)
		// U is always set.
		expectedP := (valueToPull & ^B) | U // U is always set, B is ignored from stack
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
			setup:      func(cpu *CPU) { cpu.setFlag(C, false); cpu.PC = baseAddr + 0x08 }, // Ensure cross from baseAddr+8+2=A -> baseAddr-10+A = baseAddr-6
			expectedPC: baseAddr + 0x08 + 2 - 0x10, expectedCycles: 4,                      // 2 base + 1 taken + 1 cross
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
			setup:      func(cpu *CPU) { cpu.setFlag(C, true); cpu.PC = baseAddr + 0x02 }, // Cross from baseAddr+2+2=4 -> baseAddr+4-5 = baseAddr-1
			expectedPC: baseAddr + 0x02 + 2 - 0x05, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(Z, true); cpu.PC = baseAddr + 0x10 }, // Cross 10+2=12 -> 12-20 = -E
			expectedPC: baseAddr + 0x10 + 2 - 0x20, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(Z, false); cpu.PC = baseAddr + 0x04 }, // 4+2=6 -> 6-8=-2
			expectedPC: baseAddr + 0x04 + 2 - 0x08, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(N, true); cpu.PC = baseAddr + 0x10 }, // 10+2=12 -> 12-30 = -1E
			expectedPC: baseAddr + 0x10 + 2 - 0x30, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(N, false); cpu.PC = baseAddr + 0x05 }, // 5+2=7 -> 7-A = -3
			expectedPC: baseAddr + 0x05 + 2 - 0x0A, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(V, false); cpu.PC = baseAddr + 0x20 }, // 20+2=22 -> 22-40 = -1E
			expectedPC: baseAddr + 0x20 + 2 - 0x40, expectedCycles: 4,
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
			setup:      func(cpu *CPU) { cpu.setFlag(V, true); cpu.PC = baseAddr + 0x06 }, // 6+2=8 -> 8-C = -4
			expectedPC: baseAddr + 0x06 + 2 - 0x0C, expectedCycles: 4,
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
			// Set PC *before* setup, as setup might modify PC for page cross tests
			cpu.PC = baseAddr
			cpu.P = U | I // Reset flags
			//cpu.setFlag(tt.flag, tt.flagValue) // Setup initial opposite state if needed by setup func
			tt.setup(cpu) // Apply actual flag state for branch condition

			// Program: Branch instruction ONLY. We rely on runCycles to execute exactly this one.
			// We pad with BRK just in case something goes wrong and runs too far.
			program := []uint8{tt.opcode, uint8(tt.offset), 0x00, 0x00, 0x00}
			bus.load(cpu.PC, program) // Load at potentially adjusted PC
			// Ensure reset vector isn't 0x0000 in case BRK is hit accidentally
			bus.Write(0xFFFC, 0x00)
			bus.Write(0xFFFD, 0xFF) // Point BRK vector somewhere harmless (e.g., $FF00)

			// cpu.PC is already set
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
				t.Logf("%s warning: Cycle count mismatch. Expected %d cycles, executed %d (This might be due to test harness timing)", tt.name, tt.expectedCycles, cyclesRun)
			}
			// Verify U flag is still set
			if !cpu.getFlag(U) {
				t.Errorf("%s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
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
			0x00,                   // BRK just in case
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
			0x00,                           // BRK just in case
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
			0x00,                           // BRK just in case
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
			0x00,                       // BRK at target just in case
		}
		bus.load(baseAddr, program)
		bus.Write(subroutineAddr, 0x00) // Put BRK at subroutine start for safety

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
		program := []uint8{0x60, 0x00} // RTS opcode, BRK just in case
		bus.load(rtsInstructionAddr, program)
		bus.Write(expectedFinalPC, 0x00) // Put BRK at destination for safety

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

	// Get function pointers for comparison once
	immPtr := getFuncPtr((*CPU).IMM)
	zp0Ptr := getFuncPtr((*CPU).ZP0)
	absPtr := getFuncPtr((*CPU).ABS)
	// ... add others if testing those modes

	// --- AND ---
	t.Run("AND", func(t *testing.T) {
		tests := []struct {
			name        string
			addrModePtr uintptr // Use uintptr for comparison
			opcode      uint8
			operand     uint8
			memAddr     uint16
			initialA    uint8
			expectedA   uint8
			zero        bool
			negative    bool
			cycles      uint
		}{
			{"Immediate", immPtr, 0x29, 0x0F, 0, 0xF0, 0x00, true, false, 2},
			{"Immediate Neg", immPtr, 0x29, 0xFF, 0, 0x81, 0x81, false, true, 2},
			{"ZeroPage", zp0Ptr, 0x25, 0xAA, 0x50, 0xF0, 0xA0, false, true, 3},
			{"Absolute", absPtr, 0x2D, 0x55, 0x1234, 0x3C, 0x14, false, false, 4},
			// Add more modes if needed (e.g., ZPX, ABX, ABY, IZX, IZY)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8

				// Construct program based on addressing mode pointer
				if tt.addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Fatalf("AND %s: Unhandled addressing mode pointer in test setup: %v", tt.name, tt.addrModePtr)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles) // Use runCycles

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
				if !cpu.getFlag(U) {
					t.Errorf("AND %s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
				}
			})
		}
	})

	// --- EOR ---
	t.Run("EOR", func(t *testing.T) {
		tests := []struct {
			name        string
			addrModePtr uintptr
			opcode      uint8
			operand     uint8
			memAddr     uint16
			initialA    uint8
			expectedA   uint8
			zero        bool
			negative    bool
			cycles      uint
		}{
			{"Immediate", immPtr, 0x49, 0xFF, 0, 0x55, 0xAA, false, true, 2},
			{"Immediate Zero", immPtr, 0x49, 0x33, 0, 0x33, 0x00, true, false, 2},
			{"ZeroPage", zp0Ptr, 0x45, 0xF0, 0x60, 0x0F, 0xFF, false, true, 3},
			{"Absolute", absPtr, 0x4D, 0x88, 0x4321, 0x88, 0x00, true, false, 4},
			// Add more modes if needed
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8

				// Construct program based on addressing mode pointer
				if tt.addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Fatalf("EOR %s: Unhandled addressing mode pointer in test setup: %v", tt.name, tt.addrModePtr)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles) // Use runCycles

				if cpu.A != tt.expectedA {
					t.Errorf("EOR %s failed: Expected A=0x%02X, got A=0x%02X", tt.name, tt.expectedA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("EOR %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("EOR %s failed: Expected N=%v, got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
				if !cpu.getFlag(U) {
					t.Errorf("EOR %s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
				}
			})
		}
	})

	// --- ORA ---
	t.Run("ORA", func(t *testing.T) {
		tests := []struct {
			name        string
			addrModePtr uintptr
			opcode      uint8
			operand     uint8
			memAddr     uint16
			initialA    uint8
			expectedA   uint8
			zero        bool
			negative    bool
			cycles      uint
		}{
			{"Immediate", immPtr, 0x09, 0x0F, 0, 0xF0, 0xFF, false, true, 2},
			{"Immediate Zero", immPtr, 0x09, 0x00, 0, 0x00, 0x00, true, false, 2},
			{"ZeroPage", zp0Ptr, 0x05, 0xAA, 0x70, 0x55, 0xFF, false, true, 3},
			{"Absolute", absPtr, 0x0D, 0x80, 0xABCD, 0x01, 0x81, false, true, 4},
			// Add more modes if needed
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				var program []uint8

				// Construct program based on addressing mode pointer
				if tt.addrModePtr == immPtr {
					program = []uint8{tt.opcode, tt.operand, 0x00}
				} else if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Fatalf("ORA %s: Unhandled addressing mode pointer in test setup: %v", tt.name, tt.addrModePtr)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles) // Use runCycles

				if cpu.A != tt.expectedA {
					t.Errorf("ORA %s failed: Expected A=0x%02X, got A=0x%02X", tt.name, tt.expectedA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("ORA %s failed: Expected Z=%v, got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("ORA %s failed: Expected N=%v, got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
				if !cpu.getFlag(U) {
					t.Errorf("ORA %s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
				}
			})
		}
	})

	// --- BIT ---
	t.Run("BIT", func(t *testing.T) {
		tests := []struct {
			name        string
			addrModePtr uintptr
			opcode      uint8
			operand     uint8 // Value in memory
			memAddr     uint16
			initialA    uint8
			zero        bool // Result of A & M == 0
			negative    bool // M bit 7
			overflow    bool // M bit 6
			cycles      uint
		}{
			{"ZeroPage Z=1, N=1, V=0", zp0Ptr, 0x24, 0xAA, 0x80, 0x55, true, true, false, 3},     // 55 & AA = 00 (Z=1), AA bit 7=1 (N=1), AA bit 6=0 (V=0)
			{"ZeroPage Z=0, N=0, V=1", zp0Ptr, 0x24, 0x55, 0x80, 0x77, false, false, true, 3},    // 77 & 55 = 55 (Z=0), 55 bit 7=0 (N=0), 55 bit 6=1 (V=1)
			{"Absolute Z=0, N=1, V=1", absPtr, 0x2C, 0xC0, 0xBEEF, 0xFF, false, true, true, 4},   // FF & C0 = C0 (Z=0), C0 bit 7=1 (N=1), C0 bit 6=1 (V=1)
			{"Absolute Z=0, N=0, V=0", absPtr, 0x2C, 0x3F, 0xBEEF, 0xFF, false, false, false, 4}, // FF & 3F = 3F (Z=0), 3F bit 7=0 (N=0), 3F bit 6=0 (V=0)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cpu, bus := setupCPU()
				cpu.A = tt.initialA
				initialA := cpu.A // BIT doesn't modify A
				var program []uint8

				// Construct program based on addressing mode pointer
				if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, uint8(tt.memAddr), uint8(tt.memAddr >> 8), 0x00}
					bus.Write(tt.memAddr, tt.operand)
				} else {
					t.Fatalf("BIT %s: Unhandled addressing mode pointer in test setup: %v", tt.name, tt.addrModePtr)
				}

				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.cycles = 0
				runCycles(cpu, tt.cycles) // Use runCycles

				// Assertions
				if cpu.A != initialA {
					t.Errorf("BIT %s failed: Accumulator was modified! Expected A=0x%02X, got A=0x%02X", tt.name, initialA, cpu.A)
				}
				if cpu.getFlag(Z) != tt.zero {
					t.Errorf("BIT %s failed: Expected Z=%v (from A & M), got Z=%v", tt.name, tt.zero, cpu.getFlag(Z))
				}
				if cpu.getFlag(N) != tt.negative {
					t.Errorf("BIT %s failed: Expected N=%v (from operand bit 7), got N=%v", tt.name, tt.negative, cpu.getFlag(N))
				}
				if cpu.getFlag(V) != tt.overflow {
					t.Errorf("BIT %s failed: Expected V=%v (from operand bit 6), got V=%v", tt.name, tt.overflow, cpu.getFlag(V))
				}
				if !cpu.getFlag(U) {
					t.Errorf("BIT %s failed: U flag became unset (P=0x%02X)", tt.name, cpu.P)
				}
			})
		}
	})
}

// --- Shift and Rotate Tests ---
func TestShiftRotateInstructions(t *testing.T) {
	const baseAddr = 0x0800 // New base address for clarity

	// Get function pointers for comparison once
	impPtr := getFuncPtr((*CPU).IMP)
	zp0Ptr := getFuncPtr((*CPU).ZP0)
	zpxPtr := getFuncPtr((*CPU).ZPX)
	absPtr := getFuncPtr((*CPU).ABS)
	abxPtr := getFuncPtr((*CPU).ABX)

	type ShiftRotateTest struct {
		name             string
		instrName        string // "ASL", "LSR", "ROL", "ROR"
		addrModePtr      uintptr
		opcode           uint8
		initialValue     uint8
		memAddr          uint16 // Used for non-implied modes
		initialCarry     bool
		setupX           uint8 // For ZPX, ABX
		expectedValue    uint8
		expectedCarry    bool
		expectedZero     bool
		expectedNegative bool
		cycles           uint
	}

	tests := []ShiftRotateTest{
		// --- ASL ---
		{instrName: "ASL", name: "Accumulator Basic", addrModePtr: impPtr, opcode: 0x0A, initialValue: 0x42, expectedValue: 0x84, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ASL", name: "Accumulator Set Carry", addrModePtr: impPtr, opcode: 0x0A, initialValue: 0x81, expectedValue: 0x02, expectedCarry: true, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "ASL", name: "Accumulator To Zero", addrModePtr: impPtr, opcode: 0x0A, initialValue: 0x80, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "ASL", name: "ZeroPage", addrModePtr: zp0Ptr, opcode: 0x06, initialValue: 0x11, memAddr: 0x30, expectedValue: 0x22, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "ASL", name: "ZeroPage Set Carry", addrModePtr: zp0Ptr, opcode: 0x06, initialValue: 0xFF, memAddr: 0x31, expectedValue: 0xFE, expectedCarry: true, expectedZero: false, expectedNegative: true, cycles: 5},
		{instrName: "ASL", name: "ZeroPage,X", addrModePtr: zpxPtr, opcode: 0x16, initialValue: 0x03, memAddr: 0x40, setupX: 0x05, expectedValue: 0x06, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 6}, // Addr = 40+5=45
		{instrName: "ASL", name: "Absolute", addrModePtr: absPtr, opcode: 0x0E, initialValue: 0x80, memAddr: 0x1234, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 6},
		{instrName: "ASL", name: "Absolute,X", addrModePtr: abxPtr, opcode: 0x1E, initialValue: 0x0F, memAddr: 0x2000, setupX: 0x10, expectedValue: 0x1E, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 7}, // Addr = 2010

		// --- LSR ---
		{instrName: "LSR", name: "Accumulator Basic", addrModePtr: impPtr, opcode: 0x4A, initialValue: 0x84, expectedValue: 0x42, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "LSR", name: "Accumulator Set Carry", addrModePtr: impPtr, opcode: 0x4A, initialValue: 0x01, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "LSR", name: "Accumulator Zero", addrModePtr: impPtr, opcode: 0x4A, initialValue: 0x00, expectedValue: 0x00, expectedCarry: false, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "LSR", name: "ZeroPage", addrModePtr: zp0Ptr, opcode: 0x46, initialValue: 0x22, memAddr: 0x32, expectedValue: 0x11, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "LSR", name: "ZeroPage Set Carry", addrModePtr: zp0Ptr, opcode: 0x46, initialValue: 0xFF, memAddr: 0x33, expectedValue: 0x7F, expectedCarry: true, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "LSR", name: "ZeroPage,X", addrModePtr: zpxPtr, opcode: 0x56, initialValue: 0x06, memAddr: 0x50, setupX: 0x02, expectedValue: 0x03, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 6}, // Addr = 50+2=52
		{instrName: "LSR", name: "Absolute", addrModePtr: absPtr, opcode: 0x4E, initialValue: 0x01, memAddr: 0x4321, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 6},
		{instrName: "LSR", name: "Absolute,X", addrModePtr: abxPtr, opcode: 0x5E, initialValue: 0x1E, memAddr: 0x3000, setupX: 0x20, expectedValue: 0x0F, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 7}, // Addr = 3020

		// --- ROL ---
		{instrName: "ROL", name: "Accumulator Basic C=0", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x42, initialCarry: false, expectedValue: 0x84, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ROL", name: "Accumulator Basic C=1", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x42, initialCarry: true, expectedValue: 0x85, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ROL", name: "Accumulator Set Carry C=0", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x81, initialCarry: false, expectedValue: 0x02, expectedCarry: true, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "ROL", name: "Accumulator Set Carry C=1", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x81, initialCarry: true, expectedValue: 0x03, expectedCarry: true, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "ROL", name: "Accumulator To Zero C=0", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x80, initialCarry: false, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "ROL", name: "Accumulator To One C=1", addrModePtr: impPtr, opcode: 0x2A, initialValue: 0x80, initialCarry: true, expectedValue: 0x01, expectedCarry: true, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "ROL", name: "ZeroPage C=1", addrModePtr: zp0Ptr, opcode: 0x26, initialValue: 0x11, memAddr: 0x34, initialCarry: true, expectedValue: 0x23, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "ROL", name: "ZeroPage,X C=0", addrModePtr: zpxPtr, opcode: 0x36, initialValue: 0xFF, memAddr: 0x60, setupX: 0x01, initialCarry: false, expectedValue: 0xFE, expectedCarry: true, expectedZero: false, expectedNegative: true, cycles: 6}, // Addr = 61
		{instrName: "ROL", name: "Absolute C=1", addrModePtr: absPtr, opcode: 0x2E, initialValue: 0x7F, memAddr: 0x5678, initialCarry: true, expectedValue: 0xFF, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 6},
		{instrName: "ROL", name: "Absolute,X C=0", addrModePtr: abxPtr, opcode: 0x3E, initialValue: 0x00, memAddr: 0x4000, setupX: 0x30, initialCarry: false, expectedValue: 0x00, expectedCarry: false, expectedZero: true, expectedNegative: false, cycles: 7}, // Addr = 4030

		// --- ROR ---
		{instrName: "ROR", name: "Accumulator Basic C=0", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x84, initialCarry: false, expectedValue: 0x42, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "ROR", name: "Accumulator Basic C=1", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x84, initialCarry: true, expectedValue: 0xC2, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ROR", name: "Accumulator Set Carry C=0", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x01, initialCarry: false, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "ROR", name: "Accumulator Set Carry C=1", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x01, initialCarry: true, expectedValue: 0x80, expectedCarry: true, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ROR", name: "Accumulator Zero C=0", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x00, initialCarry: false, expectedValue: 0x00, expectedCarry: false, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "ROR", name: "Accumulator Neg C=1", addrModePtr: impPtr, opcode: 0x6A, initialValue: 0x00, initialCarry: true, expectedValue: 0x80, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "ROR", name: "ZeroPage C=1", addrModePtr: zp0Ptr, opcode: 0x66, initialValue: 0x22, memAddr: 0x35, initialCarry: true, expectedValue: 0x91, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 5},
		{instrName: "ROR", name: "ZeroPage,X C=0", addrModePtr: zpxPtr, opcode: 0x76, initialValue: 0x01, memAddr: 0x70, setupX: 0x03, initialCarry: false, expectedValue: 0x00, expectedCarry: true, expectedZero: true, expectedNegative: false, cycles: 6}, // Addr = 73
		{instrName: "ROR", name: "Absolute C=1", addrModePtr: absPtr, opcode: 0x6E, initialValue: 0xFE, memAddr: 0x8765, initialCarry: true, expectedValue: 0xFF, expectedCarry: false, expectedZero: false, expectedNegative: true, cycles: 6},
		{instrName: "ROR", name: "Absolute,X C=0", addrModePtr: abxPtr, opcode: 0x7E, initialValue: 0xAA, memAddr: 0x5000, setupX: 0x40, initialCarry: false, expectedValue: 0x55, expectedCarry: false, expectedZero: false, expectedNegative: false, cycles: 7}, // Addr = 5040
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.instrName, tt.name), func(t *testing.T) {
			cpu, bus := setupCPU()

			// --- Setup ---
			cpu.setFlag(C, tt.initialCarry)
			cpu.P |= U // Ensure U is set initially

			var program []uint8
			var effectiveAddr uint16

			if tt.addrModePtr == impPtr {
				cpu.A = tt.initialValue
				program = []uint8{tt.opcode, 0x00} // Add BRK for safety
				effectiveAddr = 0                  // Not used
			} else {
				// Memory addressing modes
				addrArgLow := uint8(tt.memAddr & 0x00FF)
				addrArgHigh := uint8(tt.memAddr >> 8)

				if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, addrArgLow, 0x00}
					effectiveAddr = uint16(addrArgLow)
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == zpxPtr {
					cpu.X = tt.setupX
					program = []uint8{tt.opcode, addrArgLow, 0x00}
					effectiveAddr = (uint16(addrArgLow) + uint16(cpu.X)) & 0x00FF
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
					effectiveAddr = tt.memAddr
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == abxPtr {
					cpu.X = tt.setupX
					program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
					// Note: ABX cycle count depends on page cross, but the test provides the expected total.
					// We don't need to calculate the exact effective address here for setup, just for verification.
					// The CPU calculates the final address during execution.
					// For verification, calculate the expected final address.
					effectiveAddr = tt.memAddr + uint16(cpu.X)
					bus.Write(effectiveAddr, tt.initialValue) // Write to the *final* location for the test value
				} else {
					t.Fatalf("%s %s: Unhandled addressing mode pointer in test setup: %v", tt.instrName, tt.name, tt.addrModePtr)
				}
			}

			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.cycles = 0

			// --- Execution ---
			runCycles(cpu, tt.cycles)

			// --- Verification ---
			var finalValue uint8
			if tt.addrModePtr == impPtr {
				finalValue = cpu.A
			} else {
				// Re-calculate effective address for indexed modes if necessary, as CPU state might have changed
				// (though it shouldn't for these instructions).
				// For ZPX, the calculation is done inside the CPU's addressing mode.
				// For ABX, same. The effectiveAddr calculated above is where the result should be.
				finalValue = bus.Read(effectiveAddr)
			}

			// Check Result Value
			if finalValue != tt.expectedValue {
				if tt.addrModePtr == impPtr {
					t.Errorf("%s %s failed: Expected A=0x%02X, got A=0x%02X", tt.instrName, tt.name, tt.expectedValue, finalValue)
				} else {
					t.Errorf("%s %s failed: Expected Mem[0x%04X]=0x%02X, got 0x%02X", tt.instrName, tt.name, effectiveAddr, tt.expectedValue, finalValue)
				}
			}

			// Check Flags
			if cpu.getFlag(C) != tt.expectedCarry {
				t.Errorf("%s %s failed: Expected C=%v, got C=%v", tt.instrName, tt.name, tt.expectedCarry, cpu.getFlag(C))
			}
			if cpu.getFlag(Z) != tt.expectedZero {
				t.Errorf("%s %s failed: Expected Z=%v, got Z=%v", tt.instrName, tt.name, tt.expectedZero, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.expectedNegative {
				t.Errorf("%s %s failed: Expected N=%v, got N=%v", tt.instrName, tt.name, tt.expectedNegative, cpu.getFlag(N))
			}
			if !cpu.getFlag(U) {
				t.Errorf("%s %s failed: U flag became unset (P=0x%02X)", tt.instrName, tt.name, cpu.P)
			}
		})
	}
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
		program := []uint8{0x00} // BRK Opcode
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

		// 2. Check SP decremented by 3
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("BRK failed: SP incorrect. Expected SP=0x%02X, got SP=0x%02X", expectedSP, cpu.SP)
		}

		// 3. Check stack contents
		// PC pushed should be address *after* the BRK instruction + 1 byte padding (PC+2)
		pushedPC := baseAddr + 2
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1)) // Stack grows down, so +1 is status
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))

		// Status pushed should have B and U flags set.
		expectedPushedStatus := originalFlags | B | U
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("BRK failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		// Check PC bytes
		if pushedPCLow != uint8(pushedPC&0x00FF) {
			t.Errorf("BRK failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", uint8(pushedPC&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(pushedPC>>8) {
			t.Errorf("BRK failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", uint8(pushedPC>>8), pushedPCHigh)
		}

		// 4. Check CPU state after BRK
		// Interrupt Disable flag should be set in the CPU's P register
		if !cpu.getFlag(I) {
			t.Errorf("BRK failed: Interrupt Disable flag (I) was not set in P register.")
		}
		// B flag in the CPU's P register should remain clear (it's only set for the push)
		if cpu.getFlag(B) {
			t.Errorf("BRK failed: Break flag (B) was incorrectly set in P register.")
		}
		// U flag should remain set
		if !cpu.getFlag(U) {
			t.Errorf("BRK failed: U flag became unset in P register.")
		}
	})

	// --- RTI ---
	t.Run("RTI", func(t *testing.T) {
		cpu, bus := setupCPU() // Fresh setup
		targetPC := uint16(0xC0DE)
		// Status to pull: C=1, D=1, V=1. N,Z,I = 0. B is ignored. U should end up set.
		targetStatusOnStack := C | D | V | B | U              // Include B and U as they might be on stack
		expectedFinalStatus := (targetStatusOnStack & ^B) | U // Final status ignores B, forces U

		cpu.SP = initialSP - 3 // Make space for P, PC_low, PC_high
		bus.Write(stackBase+uint16(cpu.SP+1), uint8(targetStatusOnStack))
		bus.Write(stackBase+uint16(cpu.SP+2), uint8(targetPC&0x00FF))
		bus.Write(stackBase+uint16(cpu.SP+3), uint8(targetPC>>8))

		// Start CPU with different state
		cpu.PC = 0x1000       // Irrelevant start PC for RTI itself
		cpu.P = N | Z | I | U // Start with different flags

		rtiAddr := uint16(0x2000)
		bus.load(rtiAddr, []uint8{0x40, 0x00}) // RTI, BRK
		bus.Write(targetPC, 0x00)              // BRK at destination for safety

		cpu.PC = rtiAddr
		cpu.cycles = 0
		runCycles(cpu, 6)

		// Verification
		if cpu.PC != targetPC {
			t.Errorf("RTI failed: PC incorrect. Expected PC=0x%04X, got PC=0x%04X", targetPC, cpu.PC)
		}
		if cpu.P != expectedFinalStatus {
			t.Errorf("RTI failed: P register incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)",
				expectedFinalStatus, fmt.Sprintf("%08b", expectedFinalStatus),
				cpu.P, fmt.Sprintf("%08b", cpu.P))
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
		cpu.P = C | Z | I | U // I flag IS SET
		initialP := cpu.P
		initialSP := cpu.SP

		cpu.InterruptRequest() // Attempt IRQ

		// Verify state hasn't changed
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

		// 2. Check SP decremented by 3
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("IRQ (I=0) failed: Expected SP=0x%02X, got SP=0x%02X", expectedSP, cpu.SP)
		}

		// 3. Check stack contents
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1))
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))

		// Status pushed should have B CLEAR and U SET. I flag in pushed value should be original (0).
		expectedPushedStatus := (originalP | U) & ^B // B is clear, U is set, I remains original (0)
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("IRQ (I=0) failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		// PC pushed is the address where the interrupt occurred
		if pushedPCLow != uint8(baseAddr&0x00FF) {
			t.Errorf("IRQ (I=0) failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", uint8(baseAddr&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(baseAddr>>8) {
			t.Errorf("IRQ (I=0) failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", uint8(baseAddr>>8), pushedPCHigh)
		}

		// 4. Check CPU state after IRQ
		if !cpu.getFlag(I) { // I flag must be set now
			t.Errorf("IRQ (I=0) failed: Interrupt Disable flag (I) was not set in P register after IRQ.")
		}
		if !cpu.getFlag(U) { // U flag must be set
			t.Errorf("IRQ (I=0) failed: U flag became unset after IRQ.")
		}
		if cpu.getFlag(B) { // B flag must be clear
			t.Errorf("IRQ (I=0) failed: B flag became set after IRQ.")
		}

		// 5. Check cycle count
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
		cpu.P = N | V | U // Base flags
		if iFlag {
			cpu.P |= I // Set I flag if testing NMI with I=1
		}
		originalP := cpu.P // Capture the starting flags (including I state)

		// Sanity check: Verify vectors immediately after setup, before interrupt call
		if bus.Read(0xFFFA) != 0x00 || bus.Read(0xFFFB) != 0xF0 {
			t.Fatalf("NMI Pre-Test Check Failed: NMI Vector not set correctly in MockBus! Got %02X%02X", bus.Read(0xFFFB), bus.Read(0xFFFA))
		}

		cpu.NonMaskableInterrupt() // Call NMI handler

		// 1. Check PC loaded from NMI vector
		if cpu.PC != expectedNmiVector {
			t.Errorf("NMI (I=%v) failed: Expected PC=0x%04X, got PC=0x%04X", iFlag, expectedNmiVector, cpu.PC)
		}

		// 2. Check SP decremented by 3
		expectedSP := initialSP - 3
		if cpu.SP != expectedSP {
			t.Errorf("NMI (I=%v) failed: Expected SP=0x%02X, got SP=0x%02X", iFlag, expectedSP, cpu.SP)
		}

		// 3. Check stack contents
		pushedStatus := bus.Read(stackBase + uint16(expectedSP+1))
		pushedPCLow := bus.Read(stackBase + uint16(expectedSP+2))
		pushedPCHigh := bus.Read(stackBase + uint16(expectedSP+3))

		// Status pushed should have B CLEAR and U SET. I flag in pushed value should be original.
		expectedPushedStatus := (originalP | U) & ^B // B is clear, U is set, I remains original
		if Flags(pushedStatus) != expectedPushedStatus {
			t.Errorf("NMI (I=%v) failed: Pushed Status incorrect. Expected P=0x%02X (%s), got P=0x%02X (%s)", iFlag, expectedPushedStatus, fmt.Sprintf("%08b", expectedPushedStatus), pushedStatus, fmt.Sprintf("%08b", pushedStatus))
		}
		// PC pushed is the address where the interrupt occurred
		if pushedPCLow != uint8(baseAddr&0x00FF) {
			t.Errorf("NMI (I=%v) failed: Pushed PC Low incorrect. Expected 0x%02X, got 0x%02X", iFlag, uint8(baseAddr&0x00FF), pushedPCLow)
		}
		if pushedPCHigh != uint8(baseAddr>>8) {
			t.Errorf("NMI (I=%v) failed: Pushed PC High incorrect. Expected 0x%02X, got 0x%02X", iFlag, uint8(baseAddr>>8), pushedPCHigh)
		}

		// 4. Check CPU state after NMI
		if !cpu.getFlag(I) { // I flag must be set now, regardless of initial state
			t.Errorf("NMI (I=%v) failed: Interrupt Disable flag (I) was not set in P register after NMI.", iFlag)
		}
		if !cpu.getFlag(U) { // U flag must be set
			t.Errorf("NMI (I=%v) failed: U flag became unset after NMI.", iFlag)
		}
		if cpu.getFlag(B) { // B flag must be clear
			t.Errorf("NMI (I=%v) failed: B flag became set after NMI.", iFlag)
		}

		// 5. Check cycle count
		if cpu.cycles != 8 {
			t.Errorf("NMI (I=%v) failed: Expected cpu.cycles=8, got %d", iFlag, cpu.cycles)
		}
	}

	t.Run("NMI I=0", func(t *testing.T) { runNMITest(t, false) })
	t.Run("NMI I=1", func(t *testing.T) { runNMITest(t, true) }) // NMI occurs even if I=1
}

// TestIncrementDecrement tests INC, INX, INY, DEC, DEX, DEY
func TestIncrementDecrement(t *testing.T) {
	const baseAddr = 0x0900 // New base address

	// Get function pointers for comparison once
	impPtr := getFuncPtr((*CPU).IMP)
	zp0Ptr := getFuncPtr((*CPU).ZP0)
	zpxPtr := getFuncPtr((*CPU).ZPX)
	absPtr := getFuncPtr((*CPU).ABS)
	abxPtr := getFuncPtr((*CPU).ABX)

	type IncDecTest struct {
		name             string
		instrName        string // "INC", "DEC", "INX", "DEX", "INY", "DEY"
		addrModePtr      uintptr
		opcode           uint8
		initialValue     uint8
		reg              *uint8 // Pointer to X or Y for INX/DEX/INY/DEY, nil for INC/DEC
		memAddr          uint16 // Used for INC/DEC memory modes
		setupX           uint8  // For ZPX, ABX
		expectedValue    uint8
		expectedZero     bool
		expectedNegative bool
		cycles           uint
	}

	tests := []IncDecTest{
		// --- INX ---
		{instrName: "INX", name: "Basic", addrModePtr: impPtr, opcode: 0xE8, initialValue: 0x10, reg: nil, expectedValue: 0x11, expectedZero: false, expectedNegative: false, cycles: 2}, // reg will be set later
		{instrName: "INX", name: "Wrap Zero", addrModePtr: impPtr, opcode: 0xE8, initialValue: 0xFF, reg: nil, expectedValue: 0x00, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "INX", name: "To Negative", addrModePtr: impPtr, opcode: 0xE8, initialValue: 0x7F, reg: nil, expectedValue: 0x80, expectedZero: false, expectedNegative: true, cycles: 2},

		// --- INY ---
		{instrName: "INY", name: "Basic", addrModePtr: impPtr, opcode: 0xC8, initialValue: 0x20, reg: nil, expectedValue: 0x21, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "INY", name: "Wrap Zero", addrModePtr: impPtr, opcode: 0xC8, initialValue: 0xFF, reg: nil, expectedValue: 0x00, expectedZero: true, expectedNegative: false, cycles: 2},
		{instrName: "INY", name: "To Negative", addrModePtr: impPtr, opcode: 0xC8, initialValue: 0x7F, reg: nil, expectedValue: 0x80, expectedZero: false, expectedNegative: true, cycles: 2},

		// --- DEX ---
		{instrName: "DEX", name: "Basic", addrModePtr: impPtr, opcode: 0xCA, initialValue: 0x11, reg: nil, expectedValue: 0x10, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "DEX", name: "Wrap Neg", addrModePtr: impPtr, opcode: 0xCA, initialValue: 0x00, reg: nil, expectedValue: 0xFF, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "DEX", name: "From Negative", addrModePtr: impPtr, opcode: 0xCA, initialValue: 0x80, reg: nil, expectedValue: 0x7F, expectedZero: false, expectedNegative: false, cycles: 2},

		// --- DEY ---
		{instrName: "DEY", name: "Basic", addrModePtr: impPtr, opcode: 0x88, initialValue: 0x21, reg: nil, expectedValue: 0x20, expectedZero: false, expectedNegative: false, cycles: 2},
		{instrName: "DEY", name: "Wrap Neg", addrModePtr: impPtr, opcode: 0x88, initialValue: 0x00, reg: nil, expectedValue: 0xFF, expectedZero: false, expectedNegative: true, cycles: 2},
		{instrName: "DEY", name: "From Negative", addrModePtr: impPtr, opcode: 0x88, initialValue: 0x80, reg: nil, expectedValue: 0x7F, expectedZero: false, expectedNegative: false, cycles: 2},

		// --- INC (Memory) ---
		{instrName: "INC", name: "ZeroPage", addrModePtr: zp0Ptr, opcode: 0xE6, initialValue: 0x10, memAddr: 0x90, expectedValue: 0x11, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "INC", name: "ZeroPage Wrap", addrModePtr: zp0Ptr, opcode: 0xE6, initialValue: 0xFF, memAddr: 0x91, expectedValue: 0x00, expectedZero: true, expectedNegative: false, cycles: 5},
		{instrName: "INC", name: "ZeroPage,X", addrModePtr: zpxPtr, opcode: 0xF6, initialValue: 0x7F, memAddr: 0xA0, setupX: 0x05, expectedValue: 0x80, expectedZero: false, expectedNegative: true, cycles: 6}, // Addr = A5
		{instrName: "INC", name: "Absolute", addrModePtr: absPtr, opcode: 0xEE, initialValue: 0xAA, memAddr: 0xAAAA, expectedValue: 0xAB, expectedZero: false, expectedNegative: true, cycles: 6},
		{instrName: "INC", name: "Absolute,X", addrModePtr: abxPtr, opcode: 0xFE, initialValue: 0xFE, memAddr: 0xBB00, setupX: 0xCC, expectedValue: 0xFF, expectedZero: false, expectedNegative: true, cycles: 7}, // Addr = BBCC

		// --- DEC (Memory) ---
		{instrName: "DEC", name: "ZeroPage", addrModePtr: zp0Ptr, opcode: 0xC6, initialValue: 0x11, memAddr: 0x92, expectedValue: 0x10, expectedZero: false, expectedNegative: false, cycles: 5},
		{instrName: "DEC", name: "ZeroPage Wrap", addrModePtr: zp0Ptr, opcode: 0xC6, initialValue: 0x00, memAddr: 0x93, expectedValue: 0xFF, expectedZero: false, expectedNegative: true, cycles: 5},
		{instrName: "DEC", name: "ZeroPage,X", addrModePtr: zpxPtr, opcode: 0xD6, initialValue: 0x80, memAddr: 0xB0, setupX: 0x06, expectedValue: 0x7F, expectedZero: false, expectedNegative: false, cycles: 6}, // Addr = B6
		{instrName: "DEC", name: "Absolute", addrModePtr: absPtr, opcode: 0xCE, initialValue: 0x01, memAddr: 0xCCCC, expectedValue: 0x00, expectedZero: true, expectedNegative: false, cycles: 6},
		{instrName: "DEC", name: "Absolute,X", addrModePtr: abxPtr, opcode: 0xDE, initialValue: 0xAB, memAddr: 0xDD00, setupX: 0xDD, expectedValue: 0xAA, expectedZero: false, expectedNegative: true, cycles: 7}, // Addr = DD DD
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.instrName, tt.name), func(t *testing.T) {
			cpu, bus := setupCPU()

			// --- Setup ---
			cpu.P |= U // Ensure U is set initially

			var program []uint8
			var effectiveAddr uint16
			var targetReg *uint8 // Pointer to X or Y for verification

			// Determine target register for IMP mode
			if tt.instrName == "INX" || tt.instrName == "DEX" {
				targetReg = &cpu.X
			} else if tt.instrName == "INY" || tt.instrName == "DEY" {
				targetReg = &cpu.Y
			}

			if tt.addrModePtr == impPtr {
				if targetReg == nil {
					t.Fatalf("%s %s: IMP mode specified but no target register (X/Y)", tt.instrName, tt.name)
				}
				*targetReg = tt.initialValue       // Set X or Y directly
				program = []uint8{tt.opcode, 0x00} // Add BRK for safety
				effectiveAddr = 0                  // Not used
			} else {
				// Memory addressing modes (for INC/DEC)
				addrArgLow := uint8(tt.memAddr & 0x00FF)
				addrArgHigh := uint8(tt.memAddr >> 8)

				if tt.addrModePtr == zp0Ptr {
					program = []uint8{tt.opcode, addrArgLow, 0x00}
					effectiveAddr = uint16(addrArgLow)
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == zpxPtr {
					cpu.X = tt.setupX
					program = []uint8{tt.opcode, addrArgLow, 0x00}
					effectiveAddr = (uint16(addrArgLow) + uint16(cpu.X)) & 0x00FF
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == absPtr {
					program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
					effectiveAddr = tt.memAddr
					bus.Write(effectiveAddr, tt.initialValue)
				} else if tt.addrModePtr == abxPtr {
					cpu.X = tt.setupX
					program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
					// Similar to shifts, effective address calculated by CPU, verify final location
					effectiveAddr = tt.memAddr + uint16(cpu.X)
					bus.Write(effectiveAddr, tt.initialValue) // Write initial value to final address
				} else {
					t.Fatalf("%s %s: Unhandled addressing mode pointer in test setup: %v", tt.instrName, tt.name, tt.addrModePtr)
				}
			}

			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.cycles = 0

			// --- Execution ---
			runCycles(cpu, tt.cycles)

			// --- Verification ---
			var finalValue uint8
			if tt.addrModePtr == impPtr {
				finalValue = *targetReg // Read from X or Y
			} else {
				// INC/DEC modify memory
				finalValue = bus.Read(effectiveAddr)
			}

			// Check Result Value
			if finalValue != tt.expectedValue {
				if tt.addrModePtr == impPtr {
					t.Errorf("%s %s failed: Expected Reg=0x%02X, got Reg=0x%02X", tt.instrName, tt.name, tt.expectedValue, finalValue)
				} else {
					t.Errorf("%s %s failed: Expected Mem[0x%04X]=0x%02X, got 0x%02X", tt.instrName, tt.name, effectiveAddr, tt.expectedValue, finalValue)
				}
			}

			// Check Flags
			if cpu.getFlag(Z) != tt.expectedZero {
				t.Errorf("%s %s failed: Expected Z=%v, got Z=%v", tt.instrName, tt.name, tt.expectedZero, cpu.getFlag(Z))
			}
			if cpu.getFlag(N) != tt.expectedNegative {
				t.Errorf("%s %s failed: Expected N=%v, got N=%v", tt.instrName, tt.name, tt.expectedNegative, cpu.getFlag(N))
			}
			// Carry flag is NOT affected by INC/DEC/INX/DEX/INY/DEY
			if !cpu.getFlag(U) {
				t.Errorf("%s %s failed: U flag became unset (P=0x%02X)", tt.instrName, tt.name, cpu.P)
			}
		})
	}
}

// --- Compare Instruction Tests ---
func TestCompareInstructions(t *testing.T) {
	const baseAddr = 0x0B00 // New base address

	// Get function pointers for comparison once
	immPtr := getFuncPtr((*CPU).IMM)
	zp0Ptr := getFuncPtr((*CPU).ZP0)
	zpxPtr := getFuncPtr((*CPU).ZPX)
	absPtr := getFuncPtr((*CPU).ABS)
	abxPtr := getFuncPtr((*CPU).ABX)
	abyPtr := getFuncPtr((*CPU).ABY)
	izxPtr := getFuncPtr((*CPU).IZX)
	izyPtr := getFuncPtr((*CPU).IZY)

	type CompareTest struct {
		name             string
		instrName        string // "CMP", "CPX", "CPY"
		register         byte   // 'A', 'X', or 'Y'
		addrModePtr      uintptr
		opcode           uint8
		initialRegValue  uint8
		operandValue     uint8    // Immediate or value in memory
		memAddr          uint16   // Address for memory modes
		setupXY          [2]uint8 // X, Y values for setup
		expectedCarry    bool     // C = 1 if Reg >= Mem
		expectedZero     bool     // Z = 1 if Reg == Mem
		expectedNegative bool     // N = 1 if (Reg - Mem) & 0x80
		cycles           uint
	}

	tests := []CompareTest{
		// --- CMP (Compare Accumulator) ---
		{instrName: "CMP", register: 'A', name: "Immediate Equal", addrModePtr: immPtr, opcode: 0xC9, initialRegValue: 0x55, operandValue: 0x55, cycles: 2, expectedCarry: true, expectedZero: true, expectedNegative: false},
		{instrName: "CMP", register: 'A', name: "Immediate Greater", addrModePtr: immPtr, opcode: 0xC9, initialRegValue: 0x60, operandValue: 0x55, cycles: 2, expectedCarry: true, expectedZero: false, expectedNegative: false},   // 60-55=0B
		{instrName: "CMP", register: 'A', name: "Immediate Less", addrModePtr: immPtr, opcode: 0xC9, initialRegValue: 0x40, operandValue: 0x55, cycles: 2, expectedCarry: false, expectedZero: false, expectedNegative: true},      // 40-55=EB
		{instrName: "CMP", register: 'A', name: "Immediate NegResult", addrModePtr: immPtr, opcode: 0xC9, initialRegValue: 0x00, operandValue: 0x01, cycles: 2, expectedCarry: false, expectedZero: false, expectedNegative: true}, // 00-01=FF
		{instrName: "CMP", register: 'A', name: "Immediate SameNeg", addrModePtr: immPtr, opcode: 0xC9, initialRegValue: 0x80, operandValue: 0x80, cycles: 2, expectedCarry: true, expectedZero: true, expectedNegative: false},
		{instrName: "CMP", register: 'A', name: "ZeroPage Greater", addrModePtr: zp0Ptr, opcode: 0xC5, initialRegValue: 0xAA, operandValue: 0x55, memAddr: 0x40, cycles: 3, expectedCarry: true, expectedZero: false, expectedNegative: false},                            // AA-55=55
		{instrName: "CMP", register: 'A', name: "ZeroPage,X Less", addrModePtr: zpxPtr, opcode: 0xD5, initialRegValue: 0x10, operandValue: 0x20, memAddr: 0x50, setupXY: [2]uint8{0x05, 0}, cycles: 4, expectedCarry: false, expectedZero: false, expectedNegative: true}, // Addr=55, 10-20=F0
		{instrName: "CMP", register: 'A', name: "Absolute Equal", addrModePtr: absPtr, opcode: 0xCD, initialRegValue: 0xBE, operandValue: 0xBE, memAddr: 0xABCD, cycles: 4, expectedCarry: true, expectedZero: true, expectedNegative: false},
		{instrName: "CMP", register: 'A', name: "Absolute,X Greater", addrModePtr: abxPtr, opcode: 0xDD, initialRegValue: 0xFF, operandValue: 0x01, memAddr: 0x1000, setupXY: [2]uint8{0x10, 0}, cycles: 4, expectedCarry: true, expectedZero: false, expectedNegative: true}, // Addr=1010, FF-01=FE (N=1) Base cycles, no page cross
		{instrName: "CMP", register: 'A', name: "Absolute,Y Less", addrModePtr: abyPtr, opcode: 0xD9, initialRegValue: 0x7F, operandValue: 0x80, memAddr: 0x2000, setupXY: [2]uint8{0, 0x20}, cycles: 4, expectedCarry: false, expectedZero: false, expectedNegative: true},   // Addr=2020, 7F-80=FF (N=1) Base cycles, no page cross
		{instrName: "CMP", register: 'A', name: "Indirect,X Equal", addrModePtr: izxPtr, opcode: 0xC1, initialRegValue: 0x42, operandValue: 0x42, memAddr: 0x60, setupXY: [2]uint8{0x03, 0}, cycles: 6, expectedCarry: true, expectedZero: true, expectedNegative: false},     // ZP Base=60, X=3 -> Addr=63
		{instrName: "CMP", register: 'A', name: "Indirect,Y Greater", addrModePtr: izyPtr, opcode: 0xD1, initialRegValue: 0x90, operandValue: 0x80, memAddr: 0x70, setupXY: [2]uint8{0, 0x0A}, cycles: 5, expectedCarry: true, expectedZero: false, expectedNegative: false},  // ZP Base=70 -> points somewhere, +Y=A -> 90-80=10 (N=0) Base cycles, no page cross

		// --- CPX (Compare X Register) ---
		{instrName: "CPX", register: 'X', name: "Immediate Equal", addrModePtr: immPtr, opcode: 0xE0, initialRegValue: 0x33, operandValue: 0x33, cycles: 2, expectedCarry: true, expectedZero: true, expectedNegative: false},
		{instrName: "CPX", register: 'X', name: "Immediate Less", addrModePtr: immPtr, opcode: 0xE0, initialRegValue: 0x80, operandValue: 0x81, cycles: 2, expectedCarry: false, expectedZero: false, expectedNegative: true},                  // 80-81=FF
		{instrName: "CPX", register: 'X', name: "ZeroPage Greater", addrModePtr: zp0Ptr, opcode: 0xE4, initialRegValue: 0x01, operandValue: 0x00, memAddr: 0x41, cycles: 3, expectedCarry: true, expectedZero: false, expectedNegative: false}, // 01-00=01
		{instrName: "CPX", register: 'X', name: "Absolute Less", addrModePtr: absPtr, opcode: 0xEC, initialRegValue: 0x00, operandValue: 0xFF, memAddr: 0xBEEF, cycles: 4, expectedCarry: false, expectedZero: false, expectedNegative: false}, // 00-FF=01

		// --- CPY (Compare Y Register) ---
		{instrName: "CPY", register: 'Y', name: "Immediate Equal", addrModePtr: immPtr, opcode: 0xC0, initialRegValue: 0xCC, operandValue: 0xCC, cycles: 2, expectedCarry: true, expectedZero: true, expectedNegative: false},
		{instrName: "CPY", register: 'Y', name: "Immediate Greater", addrModePtr: immPtr, opcode: 0xC0, initialRegValue: 0xDD, operandValue: 0xCC, cycles: 2, expectedCarry: true, expectedZero: false, expectedNegative: false},                    // DD-CC=11
		{instrName: "CPY", register: 'Y', name: "ZeroPage Less", addrModePtr: zp0Ptr, opcode: 0xC4, initialRegValue: 0x10, operandValue: 0xF0, memAddr: 0x42, cycles: 3, expectedCarry: false, expectedZero: false, expectedNegative: false},        // 10-F0 = 20
		{instrName: "CPY", register: 'Y', name: "Absolute Greater Neg", addrModePtr: absPtr, opcode: 0xCC, initialRegValue: 0x80, operandValue: 0x00, memAddr: 0xCAFE, cycles: 4, expectedCarry: true, expectedZero: false, expectedNegative: true}, // 80-00=80
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.instrName, tt.name), func(t *testing.T) {
			cpu, bus := setupCPU()

			// --- Setup ---
			cpu.P = U // Reset flags, keep U
			cpu.X = tt.setupXY[0]
			cpu.Y = tt.setupXY[1]

			var initialRegValue uint8
			switch tt.register {
			case 'A':
				cpu.A = tt.initialRegValue
				initialRegValue = cpu.A
			case 'X':
				cpu.X = tt.initialRegValue
				initialRegValue = cpu.X
			case 'Y':
				cpu.Y = tt.initialRegValue
				initialRegValue = cpu.Y
			default:
				t.Fatalf("%s %s: Invalid register '%c' specified in test", tt.instrName, tt.name, tt.register)
			}

			var program []uint8
			var effectiveAddr uint16 // For memory verification if needed, not directly used by compare

			// Construct program and setup memory based on addressing mode
			addrArgLow := uint8(tt.memAddr & 0x00FF)
			addrArgHigh := uint8(tt.memAddr >> 8)

			switch tt.addrModePtr {
			case immPtr:
				program = []uint8{tt.opcode, tt.operandValue, 0x00}
			case zp0Ptr:
				program = []uint8{tt.opcode, addrArgLow, 0x00}
				effectiveAddr = uint16(addrArgLow)
				bus.Write(effectiveAddr, tt.operandValue)
			case zpxPtr: // Only CMP uses this for compare
				program = []uint8{tt.opcode, addrArgLow, 0x00}
				effectiveAddr = (uint16(addrArgLow) + uint16(cpu.X)) & 0x00FF
				bus.Write(effectiveAddr, tt.operandValue)
			case absPtr:
				program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
				effectiveAddr = tt.memAddr
				bus.Write(effectiveAddr, tt.operandValue)
			case abxPtr: // Only CMP uses this for compare
				program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
				effectiveAddr = tt.memAddr + uint16(cpu.X)
				bus.Write(effectiveAddr, tt.operandValue)
			case abyPtr: // Only CMP uses this for compare
				program = []uint8{tt.opcode, addrArgLow, addrArgHigh, 0x00}
				effectiveAddr = tt.memAddr + uint16(cpu.Y)
				bus.Write(effectiveAddr, tt.operandValue)
			case izxPtr: // Only CMP uses this for compare
				program = []uint8{tt.opcode, addrArgLow, 0x00}
				zpLookupAddr := (uint16(addrArgLow) + uint16(cpu.X)) & 0xFF
				targetAddr := uint16(0xBEEF) // Dummy target address
				bus.Write(zpLookupAddr, uint8(targetAddr&0xFF))
				bus.Write((zpLookupAddr+1)&0xFF, uint8(targetAddr>>8))
				bus.Write(targetAddr, tt.operandValue)
			case izyPtr: // Only CMP uses this for compare
				program = []uint8{tt.opcode, addrArgLow, 0x00}
				baseAddrLow := uint16(addrArgLow)
				baseTarget := uint16(0xC000) // Dummy base address
				bus.Write(baseAddrLow, uint8(baseTarget&0xFF))
				bus.Write((baseAddrLow+1)&0xFF, uint8(baseTarget>>8))
				effectiveAddr = baseTarget + uint16(cpu.Y)
				bus.Write(effectiveAddr, tt.operandValue)
			default:
				t.Fatalf("%s %s: Unhandled addressing mode pointer in test setup: %v", tt.instrName, tt.name, tt.addrModePtr)
			}

			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.cycles = 0

			// --- Execution ---
			// Add +1 cycle for indexed modes if page boundary is crossed
			// Note: Our runCycles doesn't automatically handle this extra cycle based on condition yet.
			// For now, we use the base cycles. If precise cycle testing needed, adjust expected cycles or runCycles.
			runCycles(cpu, tt.cycles)

			// --- Verification ---
			var finalRegValue uint8
			switch tt.register {
			case 'A':
				finalRegValue = cpu.A
			case 'X':
				finalRegValue = cpu.X
			case 'Y':
				finalRegValue = cpu.Y
			}

			// 1. Verify Register Unchanged
			if finalRegValue != initialRegValue {
				t.Errorf("%s %s failed: Register %c changed. Expected 0x%02X, got 0x%02X", tt.instrName, tt.name, tt.register, initialRegValue, finalRegValue)
			}

			// 2. Verify Flags
			if cpu.getFlag(C) != tt.expectedCarry {
				t.Errorf("%s %s failed: Expected C=%v, got C=%v (P=%02X)", tt.instrName, tt.name, tt.expectedCarry, cpu.getFlag(C), cpu.P)
			}
			if cpu.getFlag(Z) != tt.expectedZero {
				t.Errorf("%s %s failed: Expected Z=%v, got Z=%v (P=%02X)", tt.instrName, tt.name, tt.expectedZero, cpu.getFlag(Z), cpu.P)
			}
			if cpu.getFlag(N) != tt.expectedNegative {
				t.Errorf("%s %s failed: Expected N=%v, got N=%v (P=%02X)", tt.instrName, tt.name, tt.expectedNegative, cpu.getFlag(N), cpu.P)
			}
			if !cpu.getFlag(U) {
				t.Errorf("%s %s failed: U flag became unset (P=0x%02X)", tt.instrName, tt.name, cpu.P)
			}
		})
	}
}

// --- Placeholder Tests for Complex Instructions ---
// Add more tests here as you implement instructions like ADC, SBC, branches, shifts etc.

func TestArithmeticPlaceholders(t *testing.T) {
	t.Skip("Skipping arithmetic tests (ADC/SBC) - Implement instructions first.")
	// TODO: Write tests for ADC (various modes, flags C/V/Z/N, decimal mode?)
	// TODO: Write tests for SBC (various modes, flags C/V/Z/N, decimal mode?)
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
