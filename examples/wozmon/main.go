package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	cpu6502 "github.com/drewwalton19216801/sixty502"
)

// SimpleBus implements a 64KB memory bus for the 6502
type SimpleBus struct {
	ram [65536]uint8
}

func (b *SimpleBus) Read(addr uint16) uint8 {
	return b.ram[addr]
}

func (b *SimpleBus) Write(addr uint16, data uint8) {
	b.ram[addr] = data
}

// Emulator holds the CPU and bus
type Emulator struct {
	cpu *cpu6502.CPU
	bus *SimpleBus
}

// NewEmulator creates a new 6502 emulator instance
func NewEmulator() *Emulator {
	bus := &SimpleBus{}
	cpu := cpu6502.NewCPU(bus)

	// Set up vectors
	bus.Write(0xFFFC, 0x00) // Reset vector low byte
	bus.Write(0xFFFD, 0x80) // Reset vector high byte -> PC = $8000
	bus.Write(0xFFFE, 0x00) // BRK/IRQ vector low
	bus.Write(0xFFFF, 0xFF) // BRK/IRQ vector high

	// Load example program at $8000
	// Simple counter: counts from 0 to 16 and stores at $0200
	program := []uint8{
		0xA9, 0x00, // LDA #$00
		0x18,       // CLC
		0x69, 0x01, // ADC #$01
		0x8D, 0x00, 0x02, // STA $0200
		0xC9, 0x10, // CMP #$10
		0xD0, 0xF7, // BNE -9 (back to CLC)
		0x00, // BRK
	}

	for i, b := range program {
		bus.Write(0x8000+uint16(i), b)
	}

	cpu.Reset()

	return &Emulator{
		cpu: cpu,
		bus: bus,
	}
}

// parseHex parses a hex string (with or without 0x prefix)
func parseHex(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	s = strings.TrimPrefix(s, "$")

	val, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid hex value: %s", s)
	}
	return uint16(val), nil
}

// parseHexByte parses a hex byte (2 digits)
func parseHexByte(s string) (uint8, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	s = strings.TrimPrefix(s, "$")

	val, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid hex byte: %s", s)
	}
	return uint8(val), nil
}

// examineMemory displays memory at a single address
func (e *Emulator) examineMemory(addr uint16) {
	value := e.bus.Read(addr)
	fmt.Printf("%04X: %02X\n", addr, value)
}

// depositMemory writes a byte to memory
func (e *Emulator) depositMemory(addr uint16, value uint8) {
	e.bus.Write(addr, value)
	fmt.Printf("%04X: %02X\n", addr, value)
}

// displayRange shows memory contents from start to end address
func (e *Emulator) displayRange(start, end uint16) {
	if start > end {
		fmt.Println("Error: start address must be <= end address")
		return
	}

	addr := start
	for addr <= end {
		fmt.Printf("%04X:", addr)

		// Display up to 8 bytes per line
		for i := 0; i < 8 && addr <= end; i++ {
			fmt.Printf(" %02X", e.bus.Read(addr))
			addr++
		}
		fmt.Println()
	}
}

// runProgram executes from the specified address
func (e *Emulator) runProgram(addr uint16) {
	// Set reset vector to point to the desired address
	e.bus.Write(0xFFFC, uint8(addr&0xFF))      // Low byte
	e.bus.Write(0xFFFD, uint8((addr>>8)&0xFF)) // High byte

	// Set up a simple BRK handler that just halts (RTI would loop back)
	// We'll put it at a recognizable address
	e.bus.Write(0xFF00, 0x00) // BRK at handler location to stop

	// Reset CPU to load PC from the new reset vector
	e.cpu.Reset()

	// Execute reset cycles
	for e.cpu.RemainingCycles() > 0 {
		e.cpu.Clock()
	}

	fmt.Printf("Running from $%04X...\n", addr)

	// Track where BRK was called from
	var brkLocation uint16
	lastPC := addr

	// Execute until BRK or max cycles
	maxCycles := 100000
	for i := 0; i < maxCycles; i++ {
		// Track PC before execution
		if e.cpu.RemainingCycles() == 0 {
			state := e.cpu.GetStateSnapshot()
			lastPC = state.PC
		}

		if err := e.cpu.Clock(); err != nil {
			fmt.Printf("Error during execution: %v\n", err)
			break
		}

		// Check if we hit a BRK instruction
		if e.cpu.RemainingCycles() == 0 && e.cpu.CurrentOpcode() == 0x00 {
			state := e.cpu.GetStateSnapshot()
			// If we're at the BRK handler, we came from the program
			if state.PC == 0xFF00 {
				brkLocation = lastPC
				fmt.Printf("Program halted at $%04X (BRK)\n", brkLocation)
			} else {
				fmt.Printf("Program halted at $%04X (BRK)\n", state.PC)
			}
			break
		}
	}

	e.showStatus()
}

// showStatus displays the current CPU state
func (e *Emulator) showStatus() {
	state := e.cpu.GetStateSnapshot()
	fmt.Printf("PC:%04X A:%02X X:%02X Y:%02X SP:%02X P:%02X [%s]\n",
		state.PC, state.A, state.X, state.Y, state.SP, state.P,
		formatFlags(state.P))
	fmt.Printf("Total Cycles: %d\n", e.cpu.TotalCycles())
}

// formatFlags converts processor status flags to string
func formatFlags(p cpu6502.Flags) string {
	flags := ""
	if p&cpu6502.N != 0 {
		flags += "N"
	} else {
		flags += "n"
	}
	if p&cpu6502.V != 0 {
		flags += "V"
	} else {
		flags += "v"
	}
	flags += "-"
	if p&cpu6502.B != 0 {
		flags += "B"
	} else {
		flags += "b"
	}
	if p&cpu6502.D != 0 {
		flags += "D"
	} else {
		flags += "d"
	}
	if p&cpu6502.I != 0 {
		flags += "I"
	} else {
		flags += "i"
	}
	if p&cpu6502.Z != 0 {
		flags += "Z"
	} else {
		flags += "z"
	}
	if p&cpu6502.C != 0 {
		flags += "C"
	} else {
		flags += "c"
	}
	return flags
}

// resetCPU resets the CPU to initial state
func (e *Emulator) resetCPU() {
	e.cpu.Reset()
	// Execute reset cycles
	for e.cpu.RemainingCycles() > 0 {
		e.cpu.Clock()
	}
	fmt.Println("CPU reset")
	e.showStatus()
}

// showHelp displays available commands
func showHelp() {
	fmt.Println("\nWozMon Commands:")
	fmt.Println("  AAAA          - Examine memory at address AAAA (hex)")
	fmt.Println("  AAAA: DD      - Deposit byte DD at address AAAA")
	fmt.Println("  AAAA.BBBB     - Display memory range from AAAA to BBBB")
	fmt.Println("  AAAAR         - Run/execute from address AAAA")
	fmt.Println("  RESET         - Reset the CPU")
	fmt.Println("  STATUS        - Display CPU state")
	fmt.Println("  HELP          - Show this help message")
	fmt.Println("  QUIT or EXIT  - Exit the emulator")
	fmt.Println("\nExample program loaded at $8000 (counts 0-16 to $0200)")
	fmt.Println("Try: 8000R to run, then 0200 to see result")
}

// processCommand parses and executes a WozMon command
func (e *Emulator) processCommand(input string) bool {
	input = strings.TrimSpace(strings.ToUpper(input))

	if input == "" {
		return true
	}

	// Handle special commands
	switch input {
	case "HELP", "?":
		showHelp()
		return true
	case "QUIT", "EXIT":
		return false
	case "RESET":
		e.resetCPU()
		return true
	case "STATUS":
		e.showStatus()
		return true
	}

	// Check for run command (ends with R)
	if strings.HasSuffix(input, "R") {
		addrStr := strings.TrimSuffix(input, "R")
		addr, err := parseHex(addrStr)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}
		e.runProgram(addr)
		return true
	}

	// Check for deposit command (contains :)
	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) != 2 {
			fmt.Println("Error: invalid deposit syntax (use AAAA: DD)")
			return true
		}

		addr, err := parseHex(parts[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}

		value, err := parseHexByte(parts[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}

		e.depositMemory(addr, value)
		return true
	}

	// Check for range display (contains .)
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts) != 2 {
			fmt.Println("Error: invalid range syntax (use AAAA.BBBB)")
			return true
		}

		start, err := parseHex(parts[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}

		end, err := parseHex(parts[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}

		e.displayRange(start, end)
		return true
	}

	// Otherwise, treat as examine command
	addr, err := parseHex(input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Type HELP for available commands")
		return true
	}

	e.examineMemory(addr)
	return true
}

func main() {
	fmt.Println("6502 Emulator with WozMon Interface")
	fmt.Println("====================================")
	fmt.Println("Type 'HELP' for available commands")

	emulator := NewEmulator()

	// REPL loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if !emulator.processCommand(input) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nGoodbye!")
}
