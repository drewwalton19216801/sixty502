package cpu6502

import "fmt"

// Debug and Disassembly Tools
//
// This file provides debugging utilities including instruction
// disassembly and lookup table access.

// Disassemble disassembles instructions in the specified memory range.
//
// This method reads memory and decodes instructions, producing a map
// of addresses to disassembled instruction strings. The format matches
// common 6502 assembler syntax.
//
// Parameters:
//   - startAddr: Starting address to disassemble
//   - endAddr: Ending address to disassemble (inclusive)
//
// Returns a map where keys are instruction addresses and values are
// disassembled instruction strings.
//
// Format examples:
//   - "LDA #$42" (immediate)
//   - "STA $1234" (absolute)
//   - "BNE $8010" (relative, shows target address)
//   - "JMP ($FFFC)" (indirect)
//
// The disassembler handles:
//   - All addressing modes correctly
//   - Relative branch target calculation
//   - Multi-byte operands (little-endian)
//   - Illegal opcodes (marked with *)
//
// Example:
//
//	disasm := cpu.Disassemble(0x8000, 0x8010)
//	for addr := uint16(0x8000); addr <= 0x8010; {
//	    if instr, ok := disasm[addr]; ok {
//	        fmt.Printf("$%04X: %s\n", addr, instr)
//	        // Advance by instruction length
//	        addr += uint16(cpu.LookupInstruction(cpu.read(addr)).Length)
//	    }
//	}
func (c *CPU) Disassemble(startAddr, endAddr uint16) map[uint16]string {
	disassembly := make(map[uint16]string)
	addr := startAddr

	for addr <= endAddr && addr >= startAddr { // Check for wrap around too
		lineAddr := addr
		opcode := c.read(addr)
		addr++ // Consume opcode byte
		instr := c.lookup[opcode]
		operandStr := ""

		switch instr.AddrModeType { // Use enum comparison
		case AddrModeIMP:
			// No operand bytes
		case AddrModeIMM:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("#$%02X", val)
		case AddrModeZP0:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X", val)
		case AddrModeZPX:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,X", val)
		case AddrModeZPY:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("$%02X,Y", val)
		case AddrModeREL:
			val := c.read(addr)
			addr++            // Consume operand byte
			relTarget := addr // Relative jump target is relative to the *next* instruction
			if val&0x80 > 0 {
				relTarget += uint16(int16(val)) // Handle negative offset
			} else {
				relTarget += uint16(val) // Positive offset
			}
			operandStr = fmt.Sprintf("$%04X", relTarget)
		case AddrModeABS:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X", (hi<<8)|lo)
		case AddrModeABX:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,X", (hi<<8)|lo)
		case AddrModeABY:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("$%04X,Y", (hi<<8)|lo)
		case AddrModeIND:
			lo := uint16(c.read(addr))
			addr++
			hi := uint16(c.read(addr))
			addr++
			operandStr = fmt.Sprintf("($%04X)", (hi<<8)|lo)
		case AddrModeIZX:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("($%02X,X)", val)
		case AddrModeIZY:
			val := c.read(addr)
			addr++
			operandStr = fmt.Sprintf("($%02X),Y", val)
		default:
			operandStr = "???"
		}

		// Add safety checks in case addr went beyond bounds during operand read
		if addr > endAddr+4 {
			disassembly[lineAddr] = fmt.Sprintf("%s ??? ; Address bounds exceeded", instr.Name)
			break
		}

		disassembly[lineAddr] = fmt.Sprintf("%s %s", instr.Name, operandStr)

		// Safety break for very large ranges or infinite loops from bad data
		if addr < lineAddr && lineAddr != 0 {
			break
		} // Address wrapped around negatively

	}
	return disassembly
}

// LookupTable exposes the instruction lookup table.
//
// This provides direct access to the 256-entry instruction table,
// primarily intended for tools like disassemblers or UI displays.
//
// Each entry contains:
//   - Name: Instruction mnemonic
//   - Operate: Function pointer to implementation
//   - AddrMode: Function pointer to addressing mode
//   - AddrModeType: Addressing mode enum
//   - Cycles: Base cycle count
//   - Length: Instruction length in bytes
//   - PageCrossPenalty: Whether page cross adds a cycle
//   - Illegal: Whether this is an unofficial opcode
//
// Returns a copy of the lookup table array.
//
// Example:
//
//	table := cpu.LookupTable()
//	for opcode, instr := range table {
//	    if !instr.Illegal {
//	        fmt.Printf("$%02X: %s (%d cycles)\n",
//	            opcode, instr.Name, instr.Cycles)
//	    }
//	}
//
// Warning: Use with caution. Modifying the returned array does not
// affect the CPU's internal table (it's a copy).
func (c *CPU) LookupTable() [256]Instruction {
	return c.lookup
}
