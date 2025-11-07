# Medium Priority: Page Cross Cycle Accuracy

## Problem Statement

Page boundary crossing detection in addressing modes like [`ABX()`](../cpu.go:178) and [`ABY()`](../cpu.go:192) is correct, but not all instructions add the extra cycle:

1. **Read Instructions**: LDA, LDX, LDY, EOR, AND, ORA, ADC, SBC, CMP add +1 cycle on page cross
2. **Write Instructions**: STA, STX, STY do NOT add extra cycle (always take full cycles)
3. **Read-Modify-Write**: ASL, LSR, ROL, ROR, INC, DEC do NOT add extra cycle

Current implementation returns 1 from addressing mode on page cross, but all instructions add it.

## Proposed Solution

Add instruction-specific page cross behavior to the `Instruction` struct.

### Implementation Steps

#### Step 1: Add Page Cross Field to Instruction

```go
type Instruction struct {
    Name             string           // Mnemonic (e.g., "LDA")
    Operate          func(*CPU) uint8 // Function to execute the instruction's logic
    AddrMode         func(*CPU) uint8 // Function to calculate the address and fetch data
    AddrModeType     AddrModeType     // Type of addressing mode
    Cycles           uint8            // Base cycles for this instruction/mode
    Length           uint8            // Length of the instruction in bytes
    Illegal          bool             // Whether this is an official or unofficial/illegal opcode
    PageCrossPenalty bool             // NEW: Whether to add +1 cycle on page boundary cross
}
```

#### Step 2: Update Clock() to Respect Page Cross Penalty

```go
// Clock executes one clock cycle of the CPU
func (c *CPU) Clock() error {
    if c.cycles == 0 {
        // ... interrupt handling ...
        
        c.opcode = c.read(c.PC)
        c.PC++
        
        c.setFlag(U, true)
        c.currentInstruction = &c.lookup[c.opcode]
        
        // ... illegal opcode handling ...
        
        baseCycles := c.currentInstruction.Cycles
        addrModeCycles := c.currentInstruction.AddrMode(c)
        opCycles := c.currentInstruction.Operate(c)
        
        // NEW: Only add addressing mode cycles if instruction supports page cross penalty
        if c.currentInstruction.PageCrossPenalty {
            c.cycles = baseCycles + addrModeCycles + opCycles
        } else {
            // Ignore addressing mode cycles (page cross doesn't add cycle)
            c.cycles = baseCycles + opCycles
        }
        
        c.setFlag(U, true)
    }
    
    c.cycles--
    c.totalCycles++
    return nil
}
```

#### Step 3: Update buildLookupTable() with Page Cross Info

```go
func (c *CPU) buildLookupTable() {
    // ... helper variables ...
    
    // Fill with illegal opcodes first
    for i := range c.lookup {
        c.lookup[i] = Instruction{
            Name:             "XXX",
            Operate:          XXX,
            AddrMode:         IMP,
            AddrModeType:     AddrModeIMP,
            Cycles:           2,
            Illegal:          true,
            PageCrossPenalty: false,
        }
    }
    
    // --- Load Instructions (ADD page cross penalty) ---
    c.lookup[0xA9] = Instruction{
        Name: "LDA", Operate: LDA, AddrMode: IMM, AddrModeType: AddrModeIMM,
        Cycles: 2, Length: 2, PageCrossPenalty: false, // Immediate has no page cross
    }
    c.lookup[0xBD] = Instruction{
        Name: "LDA", Operate: LDA, AddrMode: ABX, AddrModeType: AddrModeABX,
        Cycles: 4, Length: 3, PageCrossPenalty: true, // ABX can cross page
    }
    c.lookup[0xB9] = Instruction{
        Name: "LDA", Operate: LDA, AddrMode: ABY, AddrModeType: AddrModeABY,
        Cycles: 4, Length: 3, PageCrossPenalty: true, // ABY can cross page
    }
    c.lookup[0xB1] = Instruction{
        Name: "LDA", Operate: LDA, AddrMode: IZY, AddrModeType: AddrModeIZY,
        Cycles: 5, Length: 2, PageCrossPenalty: true, // IZY can cross page
    }
    
    // --- Store Instructions (NO page cross penalty) ---
    c.lookup[0x9D] = Instruction{
        Name: "STA", Operate: STA, AddrMode: ABX, AddrModeType: AddrModeABX,
        Cycles: 5, Length: 3, PageCrossPenalty: false, // Always 5 cycles
    }
    c.lookup[0x99] = Instruction{
        Name: "STA", Operate: STA, AddrMode: ABY, AddrModeType: AddrModeABY,
        Cycles: 5, Length: 3, PageCrossPenalty: false, // Always 5 cycles
    }
    c.lookup[0x91] = Instruction{
        Name: "STA", Operate: STA, AddrMode: IZY, AddrModeType: AddrModeIZY,
        Cycles: 6, Length: 2, PageCrossPenalty: false, // Always 6 cycles
    }
    
    // --- Arithmetic Instructions (ADD page cross penalty) ---
    c.lookup[0x7D] = Instruction{
        Name: "ADC", Operate: ADC, AddrMode: ABX, AddrModeType: AddrModeABX,
        Cycles: 4, Length: 3, PageCrossPenalty: true,
    }
    c.lookup[0x79] = Instruction{
        Name: "ADC", Operate: ADC, AddrMode: ABY, AddrModeType: AddrModeABY,
        Cycles: 4, Length: 3, PageCrossPenalty: true,
    }
    c.lookup[0x71] = Instruction{
        Name: "ADC", Operate: ADC, AddrMode: IZY, AddrModeType: AddrModeIZY,
        Cycles: 5, Length: 2, PageCrossPenalty: true,
    }
    
    // --- Read-Modify-Write Instructions (NO page cross penalty) ---
    c.lookup[0x1E] = Instruction{
        Name: "ASL", Operate: ASL, AddrMode: ABX, AddrModeType: AddrModeABX,
        Cycles: 7, Length: 3, PageCrossPenalty: false, // Always 7 cycles
    }
    c.lookup[0xFE] = Instruction{
        Name: "INC", Operate: INC, AddrMode: ABX, AddrModeType: AddrModeABX,
        Cycles: 7, Length: 3, PageCrossPenalty: false, // Always 7 cycles
    }
    
    // ... continue for all 256 opcodes ...
}
```

#### Step 4: Document Page Cross Behavior

Create a reference table for page cross behavior:

```go
// Page Cross Penalty Reference
// 
// Instructions that ADD +1 cycle on page boundary cross:
// - Load: LDA, LDX, LDY (ABX, ABY, IZY modes)
// - Logic: AND, EOR, ORA (ABX, ABY, IZY modes)
// - Arithmetic: ADC, SBC (ABX, ABY, IZY modes)
// - Compare: CMP (ABX, ABY, IZY modes)
// - Bit Test: BIT (ABX mode on 65C02 only)
//
// Instructions that DO NOT add cycle on page cross:
// - Store: STA, STX, STY (always use full cycles)
// - Read-Modify-Write: ASL, LSR, ROL, ROR, INC, DEC (always use full cycles)
// - CPX, CPY (no indexed modes that can cross pages)
//
// Addressing modes that can cross pages:
// - ABX (Absolute,X): baseAddr + X crosses page if (baseAddr & 0xFF00) != ((baseAddr + X) & 0xFF00)
// - ABY (Absolute,Y): baseAddr + Y crosses page if (baseAddr & 0xFF00) != ((baseAddr + Y) & 0xFF00)
// - IZY (Indirect,Y): baseAddr + Y crosses page if (baseAddr & 0xFF00) != ((baseAddr + Y) & 0xFF00)
```

#### Step 5: Update Operate Functions

Some operate functions currently return 1 to indicate potential page cross. Update them:

```go
// ADC - OLD version
func (c *CPU) ADC() uint8 {
    // ... implementation ...
    return 1 // Indicates potential page cross cycle
}

// ADC - NEW version
func (c *CPU) ADC() uint8 {
    // ... implementation ...
    return 0 // Page cross handled by Clock() based on PageCrossPenalty field
}

// Similar updates for: AND, EOR, ORA, SBC, CMP, LDA, LDX, LDY
```

#### Step 6: Update NOP for Unofficial Opcodes

```go
// NOP - Updated to not return page cross indicator
func (c *CPU) NOP() uint8 {
    // Unofficial NOPs with indexed addressing don't need special handling
    // Page cross is handled by PageCrossPenalty field in lookup table
    return 0
}
```

## Detailed Page Cross Behavior by Instruction

### Instructions WITH Page Cross Penalty (+1 cycle)

| Instruction | ABX | ABY | IZY | Notes |
|-------------|-----|-----|-----|-------|
| LDA | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| LDX | - | ✓ | - | 4→5 cycles |
| LDY | ✓ | - | - | 4→5 cycles |
| AND | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| EOR | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| ORA | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| ADC | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| SBC | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |
| CMP | ✓ | ✓ | ✓ | 4→5, 4→5, 5→6 cycles |

### Instructions WITHOUT Page Cross Penalty (fixed cycles)

| Instruction | ABX | ABY | IZY | Cycles |
|-------------|-----|-----|-----|--------|
| STA | ✓ | ✓ | ✓ | 5, 5, 6 (always) |
| STX | - | - | - | N/A |
| STY | - | - | - | N/A |
| ASL | ✓ | - | - | 7 (always) |
| LSR | ✓ | - | - | 7 (always) |
| ROL | ✓ | - | - | 7 (always) |
| ROR | ✓ | - | - | 7 (always) |
| INC | ✓ | - | - | 7 (always) |
| DEC | ✓ | - | - | 7 (always) |

## Testing Strategy

### Test Cases

```go
func TestPageCrossPenalty(t *testing.T) {
    tests := []struct {
        name          string
        opcode        uint8
        baseAddr      uint16
        indexReg      uint8
        indexValue    uint8
        expectCross   bool
        baseCycles    uint
        expectedTotal uint
    }{
        // LDA ABX with page cross
        {
            name: "LDA ABX Page Cross",
            opcode: 0xBD, baseAddr: 0x20FF, indexReg: 'X', indexValue: 0x02,
            expectCross: true, baseCycles: 4, expectedTotal: 5,
        },
        // LDA ABX without page cross
        {
            name: "LDA ABX No Cross",
            opcode: 0xBD, baseAddr: 0x2000, indexReg: 'X', indexValue: 0x10,
            expectCross: false, baseCycles: 4, expectedTotal: 4,
        },
        // STA ABX with page cross (no penalty)
        {
            name: "STA ABX Page Cross No Penalty",
            opcode: 0x9D, baseAddr: 0x20FF, indexReg: 'X', indexValue: 0x02,
            expectCross: true, baseCycles: 5, expectedTotal: 5,
        },
        // ASL ABX with page cross (no penalty)
        {
            name: "ASL ABX Page Cross No Penalty",
            opcode: 0x1E, baseAddr: 0x20FF, indexReg: 'X', indexValue: 0x02,
            expectCross: true, baseCycles: 7, expectedTotal: 7,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cpu, bus := setupCPU()
            
            // Set index register
            if tt.indexReg == 'X' {
                cpu.X = tt.indexValue
            } else {
                cpu.Y = tt.indexValue
            }
            
            // Load instruction
            bus.Write(0x8000, tt.opcode)
            bus.Write(0x8001, uint8(tt.baseAddr&0xFF))
            bus.Write(0x8002, uint8(tt.baseAddr>>8))
            
            cpu.PC = 0x8000
            cpu.Cycles = 0
            
            // Execute instruction
            cyclesRun := runCycles(cpu, tt.expectedTotal)
            
            if cyclesRun != uint64(tt.expectedTotal) {
                t.Errorf("Expected %d cycles, got %d", tt.expectedTotal, cyclesRun)
            }
        })
    }
}
```

## Success Criteria

- [ ] PageCrossPenalty field added to Instruction struct
- [ ] Clock() updated to respect page cross penalty
- [ ] All 256 opcodes updated with correct penalty flag
- [ ] Read instructions add penalty on page cross
- [ ] Write instructions do NOT add penalty
- [ ] Read-modify-write instructions do NOT add penalty
- [ ] All page cross tests pass
- [ ] Cycle counts match hardware behavior
