// Package main demonstrates memory-mapped I/O
package main

import (
	"fmt"
	"log"

	cpu6502 "github.com/drewwalton19216801/sixty502"
)

// MemoryMappedBus implements a system with RAM, ROM, and memory-mapped I/O
type MemoryMappedBus struct {
	ram    [0x6000]uint8 // RAM: $0000-$5FFF
	io     [0x2000]uint8 // I/O: $6000-$7FFF
	rom    [0x8000]uint8 // ROM: $8000-$FFFF
	output []uint8       // Captured output from I/O port
}

func (b *MemoryMappedBus) Read(addr uint16) uint8 {
	switch {
	case addr < 0x6000:
		return b.ram[addr]
	case addr < 0x8000:
		// Memory-mapped I/O read
		return b.io[addr-0x6000]
	default:
		// ROM area
		return b.rom[addr-0x8000]
	}
}

func (b *MemoryMappedBus) Write(addr uint16, data uint8) {
	switch {
	case addr < 0x6000:
		b.ram[addr] = data
	case addr < 0x8000:
		// Memory-mapped I/O write
		b.io[addr-0x6000] = data
		// Special handling for output port at $6000
		if addr == 0x6000 {
			b.output = append(b.output, data)
			if data >= 0x20 && data < 0x7F {
				fmt.Printf("Output: '%c' ($%02X)\n", data, data)
			} else {
				fmt.Printf("Output: $%02X\n", data)
			}
		}
	default:
		// ROM writes are ignored
	}
}

func main() {
	bus := &MemoryMappedBus{}
	cpu := cpu6502.NewCPU(bus)

	// Program: Output "HELLO" to I/O port at $6000
	program := []uint8{
		// Output 'H'
		0xA9, 0x48, // LDA #$48      ; Load 'H'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		// Output 'E'
		0xA9, 0x45, // LDA #$45      ; Load 'E'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		// Output 'L'
		0xA9, 0x4C, // LDA #$4C      ; Load 'L'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		// Output 'L'
		0xA9, 0x4C, // LDA #$4C      ; Load 'L'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		// Output 'O'
		0xA9, 0x4F, // LDA #$4F      ; Load 'O'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		// Output newline
		0xA9, 0x0A, // LDA #$0A      ; Load '\n'
		0x8D, 0x00, 0x60, // STA $6000     ; Write to I/O port

		0x00, // BRK           ; Halt
	}

	// Load program into ROM at $8000
	copy(bus.rom[:], program)

	// Set reset vector
	bus.rom[0x7FFC] = 0x00 // Low byte ($FFFC in address space)
	bus.rom[0x7FFD] = 0x80 // High byte ($FFFD in address space)

	// Set BRK vector
	bus.rom[0x7FFE] = 0x00 // Low byte ($FFFE in address space)
	bus.rom[0x7FFF] = 0xFF // High byte ($FFFF in address space)

	// Reset and execute
	cpu.Reset()

	fmt.Println("Starting memory-mapped I/O example...")
	fmt.Println("Program will output 'HELLO' via I/O port at $6000")

	// Execute until BRK or timeout
	maxCycles := 10000
	for i := 0; i < maxCycles; i++ {
		if err := cpu.Clock(); err != nil {
			log.Printf("Error: %v\n", err)
			break
		}

		if cpu.CurrentOpcode() == 0x00 && cpu.RemainingCycles() == 0 {
			break
		}
	}

	fmt.Printf("\nProgram completed\n")
	fmt.Printf("Total cycles: %d\n", cpu.TotalCycles())
	fmt.Printf("Output bytes: %v\n", bus.output)
}
