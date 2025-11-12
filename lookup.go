package cpu6502

// buildLookupTable initializes the CPU's instruction lookup table.
//
// This method populates the 256-entry lookup table with instruction
// definitions for all opcodes. Each entry contains:
//   - Name: Mnemonic (e.g., "LDA", "STA")
//   - Operate: Function pointer to the instruction implementation
//   - AddrMode: Function pointer to the addressing mode
//   - AddrModeType: Enum identifying the addressing mode
//   - Cycles: Base cycle count
//   - Length: Instruction length in bytes
//   - PageCrossPenalty: Whether to add cycle on page cross
//   - Illegal: Whether this is an unofficial opcode
//
// # Page Cross Penalty Reference
//
// The 6502 CPU has different cycle timing behavior when indexed addressing
// modes (ABX, ABY, IZY) cross page boundaries. A page boundary is crossed
// when the high byte of the effective address differs from the high byte
// of the base address.
//
// Instructions that ADD +1 cycle on page boundary cross:
//   - Load: LDA, LDX, LDY (ABX, ABY, IZY modes)
//   - Logic: AND, EOR, ORA (ABX, ABY, IZY modes)
//   - Arithmetic: ADC, SBC (ABX, ABY, IZY modes)
//   - Compare: CMP (ABX, ABY, IZY modes)
//   - Unofficial: *NOP with ABX addressing
//
// Instructions that DO NOT add cycle on page cross:
//   - Store: STA, STX, STY (always use full cycles)
//   - Read-Modify-Write: ASL, LSR, ROL, ROR, INC, DEC (always use full cycles)
//   - CPX, CPY (no indexed modes that can cross pages)
//
// Addressing modes that can cross pages:
//   - ABX (Absolute,X): effective = base + X
//   - ABY (Absolute,Y): effective = base + Y
//   - IZY (Indirect,Y): effective = [zp] + Y
//
// Page cross detection: (effective & 0xFF00) != (base & 0xFF00)
// --- Lookup Table Initialization ---
// *Use method expressions for assignment*
func (c *CPU) buildLookupTable() {
	IMP := (*CPU).IMP
	IMM := (*CPU).IMM
	ZP0 := (*CPU).ZP0
	ZPX := (*CPU).ZPX
	ZPY := (*CPU).ZPY
	REL := (*CPU).REL
	ABS := (*CPU).ABS
	ABX := (*CPU).ABX
	ABY := (*CPU).ABY
	IND := (*CPU).IND
	IZX := (*CPU).IZX
	IZY := (*CPU).IZY

	ADC := (*CPU).ADC
	AND := (*CPU).AND
	ASL := (*CPU).ASL
	BCC := (*CPU).BCC
	BCS := (*CPU).BCS
	BEQ := (*CPU).BEQ
	BIT := (*CPU).BIT
	BMI := (*CPU).BMI
	BNE := (*CPU).BNE
	BPL := (*CPU).BPL
	BRK := (*CPU).BRK
	BVC := (*CPU).BVC
	BVS := (*CPU).BVS
	CLC := (*CPU).CLC
	CLD := (*CPU).CLD
	CLI := (*CPU).CLI
	CLV := (*CPU).CLV
	CMP := (*CPU).CMP
	CPX := (*CPU).CPX
	CPY := (*CPU).CPY
	DEC := (*CPU).DEC
	DEX := (*CPU).DEX
	DEY := (*CPU).DEY
	EOR := (*CPU).EOR
	INC := (*CPU).INC
	INX := (*CPU).INX
	INY := (*CPU).INY
	JMP := (*CPU).JMP
	JSR := (*CPU).JSR
	LDA := (*CPU).LDA
	LDX := (*CPU).LDX
	LDY := (*CPU).LDY
	LSR := (*CPU).LSR
	NOP := (*CPU).NOP
	ORA := (*CPU).ORA
	PHA := (*CPU).PHA
	PHP := (*CPU).PHP
	PLA := (*CPU).PLA
	PLP := (*CPU).PLP
	ROL := (*CPU).ROL
	ROR := (*CPU).ROR
	RTI := (*CPU).RTI
	RTS := (*CPU).RTS
	SBC := (*CPU).SBC
	SEC := (*CPU).SEC
	SED := (*CPU).SED
	SEI := (*CPU).SEI
	STA := (*CPU).STA
	STX := (*CPU).STX
	STY := (*CPU).STY
	TAX := (*CPU).TAX
	TAY := (*CPU).TAY
	TSX := (*CPU).TSX
	TXA := (*CPU).TXA
	TXS := (*CPU).TXS
	TYA := (*CPU).TYA
	XXX := (*CPU).XXX

	// Fill with illegal opcodes first
	for i := range c.lookup {
		c.lookup[i] = Instruction{Name: "XXX", Operate: XXX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Illegal: true, PageCrossPenalty: false}
	}

	// --- Official Opcodes --- Using the helper variables above
	// Load instructions - ADD page cross penalty for indexed modes
	c.lookup[0xA9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB5] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBD] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xB9] = Instruction{Name: "LDA", Operate: LDA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xA1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB1] = Instruction{Name: "LDA", Operate: LDA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0xA2] = Instruction{Name: "LDX", Operate: LDX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB6] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBE] = Instruction{Name: "LDX", Operate: LDX, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}

	c.lookup[0xA0] = Instruction{Name: "LDY", Operate: LDY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xA4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB4] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xAC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xBC] = Instruction{Name: "LDY", Operate: LDY, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}

	// Store instructions - NO page cross penalty (always use full cycles)
	c.lookup[0x85] = Instruction{Name: "STA", Operate: STA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x95] = Instruction{Name: "STA", Operate: STA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x9D] = Instruction{Name: "STA", Operate: STA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x99] = Instruction{Name: "STA", Operate: STA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x81] = Instruction{Name: "STA", Operate: STA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x91] = Instruction{Name: "STA", Operate: STA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 6, Length: 2, PageCrossPenalty: false}

	c.lookup[0x86] = Instruction{Name: "STX", Operate: STX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x96] = Instruction{Name: "STX", Operate: STX, AddrMode: ZPY, AddrModeType: AddrModeZPY, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8E] = Instruction{Name: "STX", Operate: STX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	c.lookup[0x84] = Instruction{Name: "STY", Operate: STY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x94] = Instruction{Name: "STY", Operate: STY, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x8C] = Instruction{Name: "STY", Operate: STY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Transfer and stack instructions - no page cross possible
	c.lookup[0xAA] = Instruction{Name: "TAX", Operate: TAX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xA8] = Instruction{Name: "TAY", Operate: TAY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xBA] = Instruction{Name: "TSX", Operate: TSX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x8A] = Instruction{Name: "TXA", Operate: TXA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x9A] = Instruction{Name: "TXS", Operate: TXS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x98] = Instruction{Name: "TYA", Operate: TYA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0x48] = Instruction{Name: "PHA", Operate: PHA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1, PageCrossPenalty: false}
	c.lookup[0x08] = Instruction{Name: "PHP", Operate: PHP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 3, Length: 1, PageCrossPenalty: false}
	c.lookup[0x68] = Instruction{Name: "PLA", Operate: PLA, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1, PageCrossPenalty: false}
	c.lookup[0x28] = Instruction{Name: "PLP", Operate: PLP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 4, Length: 1, PageCrossPenalty: false}

	// Arithmetic instructions - ADD page cross penalty for indexed modes
	c.lookup[0x69] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x65] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x75] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x6D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x7D] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x79] = Instruction{Name: "ADC", Operate: ADC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x61] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x71] = Instruction{Name: "ADC", Operate: ADC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0xE9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xE5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF5] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xED] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xFD] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xF9] = Instruction{Name: "SBC", Operate: SBC, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xE1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF1] = Instruction{Name: "SBC", Operate: SBC, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// Read-modify-write instructions - NO page cross penalty (always use full cycles)
	c.lookup[0xE6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF6] = Instruction{Name: "INC", Operate: INC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xEE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0xFE] = Instruction{Name: "INC", Operate: INC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}
	c.lookup[0xE8] = Instruction{Name: "INX", Operate: INX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xC8] = Instruction{Name: "INY", Operate: INY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0xC6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD6] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0xDE] = Instruction{Name: "DEC", Operate: DEC, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}
	c.lookup[0xCA] = Instruction{Name: "DEX", Operate: DEX, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x88] = Instruction{Name: "DEY", Operate: DEY, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0x0A] = Instruction{Name: "ASL", Operate: ASL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x06] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x16] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x0E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x1E] = Instruction{Name: "ASL", Operate: ASL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x4A] = Instruction{Name: "LSR", Operate: LSR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x46] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x56] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x4E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x5E] = Instruction{Name: "LSR", Operate: LSR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x2A] = Instruction{Name: "ROL", Operate: ROL, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x26] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x36] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x3E] = Instruction{Name: "ROL", Operate: ROL, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	c.lookup[0x6A] = Instruction{Name: "ROR", Operate: ROR, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x66] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 5, Length: 2, PageCrossPenalty: false}
	c.lookup[0x76] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x6E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x7E] = Instruction{Name: "ROR", Operate: ROR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 7, Length: 3, PageCrossPenalty: false}

	// Logical instructions - ADD page cross penalty for indexed modes
	c.lookup[0x29] = Instruction{Name: "AND", Operate: AND, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x25] = Instruction{Name: "AND", Operate: AND, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x35] = Instruction{Name: "AND", Operate: AND, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x3D] = Instruction{Name: "AND", Operate: AND, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x39] = Instruction{Name: "AND", Operate: AND, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x21] = Instruction{Name: "AND", Operate: AND, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x31] = Instruction{Name: "AND", Operate: AND, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0x49] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x45] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x55] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x4D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x5D] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x59] = Instruction{Name: "EOR", Operate: EOR, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x41] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x51] = Instruction{Name: "EOR", Operate: EOR, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	c.lookup[0x09] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x05] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x15] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0x0D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0x1D] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x19] = Instruction{Name: "ORA", Operate: ORA, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0x01] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0x11] = Instruction{Name: "ORA", Operate: ORA, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// BIT instruction - no page cross possible
	c.lookup[0x24] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0x2C] = Instruction{Name: "BIT", Operate: BIT, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Compare instructions - ADD page cross penalty for indexed modes
	c.lookup[0xC9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xC5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD5] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}
	c.lookup[0xDD] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xD9] = Instruction{Name: "CMP", Operate: CMP, AddrMode: ABY, AddrModeType: AddrModeABY, Cycles: 4, Length: 3, PageCrossPenalty: true}
	c.lookup[0xC1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZX, AddrModeType: AddrModeIZX, Cycles: 6, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD1] = Instruction{Name: "CMP", Operate: CMP, AddrMode: IZY, AddrModeType: AddrModeIZY, Cycles: 5, Length: 2, PageCrossPenalty: true}

	// CPX, CPY - no indexed modes that can cross pages
	c.lookup[0xE0] = Instruction{Name: "CPX", Operate: CPX, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xE4] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xEC] = Instruction{Name: "CPX", Operate: CPX, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	c.lookup[0xC0] = Instruction{Name: "CPY", Operate: CPY, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xC4] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, PageCrossPenalty: false}
	c.lookup[0xCC] = Instruction{Name: "CPY", Operate: CPY, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, PageCrossPenalty: false}

	// Branch instructions - no page cross penalty (handled by branch logic)
	c.lookup[0x90] = Instruction{Name: "BCC", Operate: BCC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xB0] = Instruction{Name: "BCS", Operate: BCS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xF0] = Instruction{Name: "BEQ", Operate: BEQ, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x30] = Instruction{Name: "BMI", Operate: BMI, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0xD0] = Instruction{Name: "BNE", Operate: BNE, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x10] = Instruction{Name: "BPL", Operate: BPL, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x50] = Instruction{Name: "BVC", Operate: BVC, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}
	c.lookup[0x70] = Instruction{Name: "BVS", Operate: BVS, AddrMode: REL, AddrModeType: AddrModeREL, Cycles: 2, Length: 2, PageCrossPenalty: false}

	// Jump and control flow instructions - no page cross penalty
	c.lookup[0x4C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 3, Length: 3, PageCrossPenalty: false}
	c.lookup[0x6C] = Instruction{Name: "JMP", Operate: JMP, AddrMode: IND, AddrModeType: AddrModeIND, Cycles: 5, Length: 3, PageCrossPenalty: false}
	c.lookup[0x20] = Instruction{Name: "JSR", Operate: JSR, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 6, Length: 3, PageCrossPenalty: false}
	c.lookup[0x60] = Instruction{Name: "RTS", Operate: RTS, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1, PageCrossPenalty: false}

	c.lookup[0x00] = Instruction{Name: "BRK", Operate: BRK, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 7, Length: 1, PageCrossPenalty: false}
	c.lookup[0x40] = Instruction{Name: "RTI", Operate: RTI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 6, Length: 1, PageCrossPenalty: false}

	// Flag instructions - no page cross penalty
	c.lookup[0x18] = Instruction{Name: "CLC", Operate: CLC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xD8] = Instruction{Name: "CLD", Operate: CLD, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x58] = Instruction{Name: "CLI", Operate: CLI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xB8] = Instruction{Name: "CLV", Operate: CLV, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x38] = Instruction{Name: "SEC", Operate: SEC, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0xF8] = Instruction{Name: "SED", Operate: SED, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}
	c.lookup[0x78] = Instruction{Name: "SEI", Operate: SEI, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	c.lookup[0xEA] = Instruction{Name: "NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, PageCrossPenalty: false}

	// Illegal NOPs
	// https://www.masswerk.at/6502/6502_instruction_set.html#NOPs
	c.lookup[0x1A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x3A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x5A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x7A] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xDA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xFA] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMP, AddrModeType: AddrModeIMP, Cycles: 2, Length: 1, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x80] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x82] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x89] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xC2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xE2] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: IMM, AddrModeType: AddrModeIMM, Cycles: 2, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x04] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x44] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x64] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZP0, AddrModeType: AddrModeZP0, Cycles: 3, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x14] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x34] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x54] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0x74] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xD4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}
	c.lookup[0xF4] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ZPX, AddrModeType: AddrModeZPX, Cycles: 4, Length: 2, Illegal: true, PageCrossPenalty: false}

	c.lookup[0x0C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABS, AddrModeType: AddrModeABS, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: false}

	// Unofficial NOPs with ABX addressing - ADD page cross penalty
	c.lookup[0x1C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x3C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x5C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0x7C] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0xDC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
	c.lookup[0xFC] = Instruction{Name: "*NOP", Operate: NOP, AddrMode: ABX, AddrModeType: AddrModeABX, Cycles: 4, Length: 3, Illegal: true, PageCrossPenalty: true}
}
