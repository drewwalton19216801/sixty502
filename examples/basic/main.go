// Package main demonstrates basic CPU usage
package main

import (
	"fmt"
	"log"

	cpu6502 "github.com/drewwalton19216801/sixty502"
)

// SimpleBus implements a basic 64KB RAM
type SimpleBus struct {
	ram [65536]uint8
}

func (b *SimpleBus) Read(addr uint16) uint8 {
	return b.ram[addr]
}

func (b *SimpleBus) Write(addr uint16, data uint8) {
	b.ram[addr] = data
}

func main() {
	bus := &SimpleBus{}
	cpu := cpu6502.NewCPU(bus)

	// Load program: Count from 0 to 10
	program := []uint8{
		0xA9, 0x00, // LDA #$00      ; Load 0 into accumulator
		0x69, 0x01, // ADC #$01      ; Loop: Add 1 to accumulator
		0xC9, 0x0A, // CMP #$0A      ; Compare with 10
		0xD0, 0xFA, // BNE Loop      ; Branch if not equal (back to ADC)
		0x00, // BRK           ; Break (halt)
	}

	// Load program at $8000
	for i, b := range program {
		bus.Write(0x8000+uint16(i), b)
	}

	// Set reset vector to point to our program
	bus.Write(0xFFFC, 0x00) // Low byte
	bus.Write(0xFFFD, 0x80) // High byte -> $8000

	// Set BRK vector (for when program halts)
	bus.Write(0xFFFE, 0x00) // Low byte
	bus.Write(0xFFFF, 0xFF) // High byte -> $FF00

	// Reset CPU to start execution
	cpu.Reset()

	fmt.Println("Starting CPU execution...")
	fmt.Printf("Initial state: %s\n", cpu.GetState())

	// Execute until BRK or timeout
	maxCycles := 1000
	for i := 0; i < maxCycles; i++ {
		if err := cpu.Clock(); err != nil {
			log.Printf("Error: %v\n", err)
			break
		}

		// Check if we hit BRK (opcode $00)
		if cpu.CurrentOpcode() == 0x00 && cpu.RemainingCycles() == 0 {
			fmt.Println("\nBRK instruction reached!")
			break
		}
	}

	// Display final state
	fmt.Printf("\nFinal state: %s\n", cpu.GetState())
	fmt.Printf("Accumulator: $%02X (%d)\n", cpu.A, cpu.A)
	fmt.Printf("Total cycles executed: %d\n", cpu.TotalCycles())
}
