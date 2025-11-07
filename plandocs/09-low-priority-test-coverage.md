# Low Priority: Expand Test Coverage

## Problem Statement

Current test suite in [`cpu_test.go`](../cpu_test.go) has gaps:

1. **Missing Simultaneous Interrupt Tests**: No tests for NMI during IRQ
2. **Limited Decimal Mode Coverage**: Missing edge cases like 99+1, 00-1
3. **No Instruction Length Validation**: PC advancement not verified for all opcodes
4. **Incomplete Unofficial Opcode Tests**: Only NOPs tested, missing other illegal opcodes
5. **No Self-Modifying Code Tests**: Cache invalidation scenarios not tested
6. **Missing Cycle Accuracy Tests**: Page cross timing not comprehensively tested

## Proposed Solution

Add comprehensive test coverage for all identified gaps.

### Implementation Steps

#### Step 1: Add Simultaneous Interrupt Tests

```go
// TestSimultaneousInterrupts verifies interrupt priority and hijacking
func TestSimultaneousInterrupts(t *testing.T) {
    t.Run("NMI Hijacks IRQ", func(t *testing.T) {
        cpu, bus := setupCPU()
        
        // Set up vectors
        bus.Write(0xFFFA, 0x00) // NMI vector -> $F000
        bus.Write(0xFFFB, 0xF0)
        bus.Write(0xFFFE, 0x00) // IRQ vector -> $F200
        bus.Write(0xFFFF, 0xF2)
        
        // Start at known location
        cpu.PC = 0x8000
        cpu.setFlag(I, false) // Enable IRQ
        
        // Assert IRQ
        cpu.SetIRQ(true)
        
        // Run a few cycles to start IRQ sequence
        runCycles(cpu, 3)
        
        // Generate NMI during IRQ sequence
        cpu.SetNMI(true)
        cpu.SetNMI(false) // Falling edge
        
        // Run to completion
        runCycles(cpu, 10)
        
        // Verify PC points to NMI vector, not IRQ vector
        if cpu.PC != 0xF000 {
            t.Errorf("NMI hijack failed: Expected PC=$F000, got $%04X", cpu.PC)
        }
    })
    
    t.Run("NMI and IRQ Both Pending", func(t *testing.T) {
        cpu, bus := setupCPU()
        
        cpu.PC = 0x8000
        cpu.setFlag(I, false)
        
        // Assert both interrupts
        cpu.SetIRQ(true)
        cpu.SetNMI(true)
        cpu.SetNMI(false) // NMI edge
        
        // Run one instruction
        runCycles(cpu, 10)
        
        // NMI should be serviced first (higher priority)
        if cpu.PC != 0xF000 {
            t.Errorf("Expected NMI to be serviced first, PC=$%04X", cpu.PC)
        }
        
        // After RTI, IRQ should be serviced (if I flag cleared)
        cpu.P &^= I // Clear I flag
        bus.Write(0xF000, 0x40) // RTI at NMI handler
        runCycles(cpu, 10)
        
        // Should now be in IRQ handler
        if cpu.PC != 0xF200 {
            t.Errorf("Expected IRQ after NMI, PC=$%04X", cpu.PC)
        }
    })
}
```

#### Step 2: Add Comprehensive Decimal Mode Tests

```go
// TestDecimalModeEdgeCases tests all BCD edge cases
func TestDecimalModeEdgeCases(t *testing.T) {
    tests := []struct {
        name      string
        instr     string // "ADC" or "SBC"
        opcode    uint8
        a         uint8
        operand   uint8
        carryIn   bool
        expectedA uint8
        expectedC bool
        expectedZ bool
        expectedN bool // Based on binary intermediate result
        expectedV bool // Based on binary intermediate result
    }{
        // ADC edge cases
        {"ADC 99+1 C=0", "ADC", 0x69, 0x99, 0x01, false, 0x00, true, true, true, false},
        {"ADC 99+0 C=1", "ADC", 0x69, 0x99, 0x00, true, 0x00, true, true, true, false},
        {"ADC 50+50 C=0", "ADC", 0x69, 0x50, 0x50, false, 0x00, true, true, true, true},
        {"ADC 09+1 C=0", "ADC", 0x69, 0x09, 0x01, false, 0x10, false, false, false, false},
        {"ADC 09+1 C=1", "ADC", 0x69, 0x09, 0x01, true, 0x11, false, false, false, false},
        {"ADC 19+1 C=0", "ADC", 0x69, 0x19, 0x01, false, 0x20, false, false, false, false},
        {"ADC 00+00 C=0", "ADC", 0x69, 0x00, 0x00, false, 0x00, false, true, false, false},
        {"ADC 00+00 C=1", "ADC", 0x69, 0x00, 0x00, true, 0x01, false, false, false, false},
        {"ADC 49+49 C=0", "ADC", 0x69, 0x49, 0x49, false, 0x98, false, false, true, false},
        {"ADC 50+49 C=0", "ADC", 0x69, 0x50, 0x49, false, 0x99, false, false, true, false},
        
        // SBC edge cases
        {"SBC 00-1 C=1", "SBC", 0xE9, 0x00, 0x01, true, 0x99, false, false, true, false},
        {"SBC 00-1 C=0", "SBC", 0xE9, 0x00, 0x01, false, 0x98, false, false, true, false},
        {"SBC 10-1 C=1", "SBC", 0xE9, 0x10, 0x01, true, 0x09, true, false, false, false},
        {"SBC 10-1 C=0", "SBC", 0xE9, 0x10, 0x01, false, 0x08, true, false, false, false},
        {"SBC 00-0 C=1", "SBC", 0xE9, 0x00, 0x00, true, 0x00, true, true, false, false},
        {"SBC 00-0 C=0", "SBC", 0xE9, 0x00, 0x00, false, 0x99, false, false, true, false},
        {"SBC 50-50 C=1", "SBC", 0xE9, 0x50, 0x50, true, 0x00, true, true, false, false},
        {"SBC 32-2 C=1", "SBC", 0xE9, 0x32, 0x02, true, 0x30, true, false, false, false},
        {"SBC 12-21 C=1", "SBC", 0xE9, 0x12, 0x21, true, 0x91, false, false, true, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cpu, bus := setupCPU()
            
            // Enable decimal mode
            cpu.setFlag(D, true)
            cpu.setFlag(C, tt.carryIn)
            cpu.A = tt.a
            
            // Load instruction
            program := []uint8{tt.opcode, tt.operand, 0x00}
            bus.load(0x8000, program)
            cpu.PC = 0x8000
            cpu.Cycles = 0
            
            // Execute
            runCycles(cpu, 2)
            
            // Verify result
            if cpu.A != tt.expectedA {
                t.Errorf("Expected A=$%02X, got $%02X", tt.expectedA, cpu.A)
            }
            if cpu.getFlag(C) != tt.expectedC {
                t.Errorf("Expected C=%v, got %v", tt.expectedC, cpu.getFlag(C))
            }
            if cpu.getFlag(Z) != tt.expectedZ {
                t.Errorf("Expected Z=%v, got %v", tt.expectedZ, cpu.getFlag(Z))
            }
            if cpu.getFlag(N) != tt.expectedN {
                t.Errorf("Expected N=%v, got %v", tt.expectedN, cpu.getFlag(N))
            }
            if cpu.getFlag(V) != tt.expectedV {
                t.Errorf("Expected V=%v, got %v", tt.expectedV, cpu.getFlag(V))
            }
        })
    }
}
```

#### Step 3: Add Instruction Length Validation Tests

```go
// TestInstructionLengthValidation verifies PC advances correctly for all opcodes
func TestInstructionLengthValidation(t *testing.T) {
    cpu, bus := setupCPU()
    
    // Test all 256 opcodes
    for opcode := 0; opcode <= 255; opcode++ {
        t.Run(fmt.Sprintf("Opcode_%02X", opcode), func(t *testing.T) {
            cpu, bus = setupCPU()
            
            instr := cpu.LookupInstruction(uint8(opcode))
            
            // Skip if no length defined
            if instr.Length == 0 {
                t.Skip("Instruction length not defined")
            }
            
            // Load instruction with dummy operands
            startPC := uint16(0x8000)
            bus.Write(startPC, uint8(opcode))
            for i := uint8(1); i < instr.Length; i++ {
                bus.Write(startPC+uint16(i), 0x00)
            }
            bus.Write(startPC+uint16(instr.Length), 0x00) // BRK after
            
            cpu.PC = startPC
            cpu.Cycles = 0
            
            // Execute instruction
            maxCycles := instr.Cycles + 5 // Allow for page cross
            runCycles(cpu, uint(maxCycles))
            
            // Verify PC advanced by instruction length
            expectedPC := startPC + uint16(instr.Length)
            if cpu.PC != expectedPC {
                t.Errorf("Opcode $%02X (%s): Expected PC=$%04X, got $%04X (Length=%d)",
                    opcode, instr.Name, expectedPC, cpu.PC, instr.Length)
            }
        })
    }
}
```

#### Step 4: Add Unofficial Opcode Tests

```go
// TestUnofficialOpcodes tests behavior of illegal/unofficial opcodes
func TestUnofficialOpcodes(t *testing.T) {
    // Test cases for documented unofficial opcodes
    tests := []struct {
        name     string
        opcode   uint8
        behavior string
        verify   func(t *testing.T, cpu *CPU, bus *MockBus)
    }{
        {
            name:     "LAX (Load A and X)",
            opcode:   0xA7, // LAX Zero Page
            behavior: "Loads value into both A and X",
            verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
                // Implementation would go here when LAX is implemented
                t.Skip("LAX not yet implemented")
            },
        },
        {
            name:     "SAX (Store A AND X)",
            opcode:   0x87, // SAX Zero Page
            behavior: "Stores A & X to memory",
            verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
                t.Skip("SAX not yet implemented")
            },
        },
        {
            name:     "DCP (Decrement and Compare)",
            opcode:   0xC7, // DCP Zero Page
            behavior: "Decrements memory then compares with A",
            verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
                t.Skip("DCP not yet implemented")
            },
        },
        {
            name:     "ISC (Increment and Subtract with Carry)",
            opcode:   0xE7, // ISC Zero Page
            behavior: "Increments memory then performs SBC",
            verify: func(t *testing.T, cpu *CPU, bus *MockBus) {
                t.Skip("ISC not yet implemented")
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cpu, bus := setupCPU()
            tt.verify(t, cpu, bus)
        })
    }
}
```

#### Step 5: Add Self-Modifying Code Tests

```go
// TestSelfModifyingCode verifies behavior with self-modifying code
func TestSelfModifyingCode(t *testing.T) {
    t.Run("Modify Next Instruction", func(t *testing.T) {
        cpu, bus := setupCPU()
        
        // Program that modifies the next instruction
        program := []uint8{
            0xA9, 0xEA,       // LDA #$EA (NOP opcode)
            0x8D, 0x06, 0x80, // STA $8006 (modify next instruction)
            0x00,             // This will become NOP
            0x00,             // BRK
        }
        
        bus.load(0x8000, program)
        cpu.PC = 0x8000
        cpu.Cycles = 0
        
        // Invalidate cache before execution
        cpu.InvalidateInstructionCache()
        
        // Execute
        runUntilBrk(cpu, bus, 50)
        
        // Verify the instruction was modified and executed
        if bus.Read(0x8006) != 0xEA {
            t.Errorf("Instruction not modified")
        }
    })
    
    t.Run("Cache Invalidation", func(t *testing.T) {
        cpu, bus := setupCPU()
        
        // Execute instruction from location
        bus.Write(0x8000, 0xA9) // LDA #$42
        bus.Write(0x8001, 0x42)
        bus.Write(0x8002, 0x00) // BRK
        
        cpu.PC = 0x8000
        cpu.Cycles = 0
        runCycles(cpu, 2)
        
        if cpu.A != 0x42 {
            t.Errorf("First execution failed")
        }
        
        // Modify the instruction
        bus.Write(0x8001, 0x99) // Change operand
        
        // Invalidate cache
        cpu.InvalidateInstructionCache()
        
        // Execute again
        cpu.PC = 0x8000
        cpu.Cycles = 0
        runCycles(cpu, 2)
        
        if cpu.A != 0x99 {
            t.Errorf("Modified instruction not executed: A=$%02X", cpu.A)
        }
    })
}
```

#### Step 6: Add Comprehensive Cycle Accuracy Tests

```go
// TestCycleAccuracy verifies exact cycle counts for all instructions
func TestCycleAccuracy(t *testing.T) {
    tests := []struct {
        name           string
        program        []uint8
        setup          func(cpu *CPU, bus *MockBus)
        expectedCycles uint64
        description    string
    }{
        {
            name:           "LDA ABX No Page Cross",
            program:        []uint8{0xBD, 0x00, 0x20, 0x00}, // LDA $2000,X
            setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x10 },
            expectedCycles: 4,
            description:    "Base cycles, no page cross",
        },
        {
            name:           "LDA ABX Page Cross",
            program:        []uint8{0xBD, 0xFF, 0x20, 0x00}, // LDA $20FF,X
            setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x02 },
            expectedCycles: 5,
            description:    "Base cycles + 1 for page cross",
        },
        {
            name:           "STA ABX Page Cross No Penalty",
            program:        []uint8{0x9D, 0xFF, 0x20, 0x00}, // STA $20FF,X
            setup:          func(cpu *CPU, bus *MockBus) { cpu.X = 0x02 },
            expectedCycles: 5,
            description:    "Always 5 cycles, page cross doesn't add cycle",
        },
        {
            name:           "Branch Taken No Cross",
            program:        []uint8{0xD0, 0x10, 0x00}, // BNE +$10
            setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, false) },
            expectedCycles: 3,
            description:    "Base 2 + 1 for taken",
        },
        {
            name:           "Branch Taken Page Cross",
            program:        []uint8{0xD0, 0xFE, 0x00}, // BNE -$02
            setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, false); cpu.PC = 0x8001 },
            expectedCycles: 4,
            description:    "Base 2 + 1 for taken + 1 for page cross",
        },
        {
            name:           "Branch Not Taken",
            program:        []uint8{0xD0, 0x10, 0x00}, // BNE +$10
            setup:          func(cpu *CPU, bus *MockBus) { cpu.setFlag(Z, true) },
            expectedCycles: 2,
            description:    "Base cycles only",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cpu, bus := setupCPU()
            
            if tt.setup != nil {
                tt.setup(cpu, bus)
            }
            
            startPC := cpu.PC
            if startPC == 0 {
                startPC = 0x8000
                cpu.PC = startPC
            }
            
            bus.load(startPC, tt.program)
            cpu.Cycles = 0
            
            startCycles := cpu.TotalCycles()
            runCycles(cpu, uint(tt.expectedCycles))
            actualCycles := cpu.TotalCycles() - startCycles
            
            if actualCycles != tt.expectedCycles {
                t.Errorf("%s: Expected %d cycles, got %d (%s)",
                    tt.name, tt.expectedCycles, actualCycles, tt.description)
            }
        })
    }
}
```

#### Step 7: Add Integration Tests with Real ROMs

```go
// TestKlausDormannSuite runs the Klaus Dormann 6502 functional test
func TestKlausDormannSuite(t *testing.T) {
    t.Skip("Requires Klaus Dormann test ROM - implement when available")
    
    // This would load the actual test ROM and verify behavior
    // The Klaus Dormann test is the gold standard for 6502 accuracy
}

// TestNESTestROMs runs NES test ROMs
func TestNESTestROMs(t *testing.T) {
    t.Skip("Requires NES test ROMs - implement when available")
    
    // Test ROMs like:
    // - nestest.nes (comprehensive instruction test)
    // - instr_test-v5 (instruction timing test)
}
```

#### Step 8: Add Fuzzing Tests

```go
// FuzzCPUExecution fuzzes CPU execution with random programs
func FuzzCPUExecution(f *testing.F) {
    // Add seed corpus
    f.Add([]byte{0xA9, 0x42, 0x00}) // LDA #$42, BRK
    f.Add([]byte{0x69, 0x01, 0x00}) // ADC #$01, BRK
    
    f.Fuzz(func(t *testing.T, program []byte) {
        if len(program) == 0 || len(program) > 256 {
            return
        }
        
        cpu, bus := setupCPU()
        bus.load(0x8000, program)
        cpu.PC = 0x8000
        cpu.Cycles = 0
        
        // Execute with timeout
        maxCycles := uint64(len(program) * 10)
        for i := uint64(0); i < maxCycles; i++ {
            if err := cpu.Clock(); err != nil {
                // Error is acceptable (illegal opcode, etc.)
                return
            }
            
            // Stop on BRK
            if cpu.CurrentOpcode() == 0x00 {
                return
            }
        }
    })
}
```

## Test Coverage Goals

| Category | Current | Target |
|----------|---------|--------|
| Instructions | ~80% | 100% |
| Addressing Modes | ~90% | 100% |
| Edge Cases | ~60% | 95% |
| Interrupts | ~70% | 100% |
| Decimal Mode | ~50% | 100% |
| Cycle Accuracy | ~40% | 90% |

## Testing Tools to Add

1. **Test ROM Loader**: Load and execute test ROMs
2. **Trace Comparison**: Compare execution traces with reference emulator
3. **Cycle Counter**: Verify exact cycle counts
4. **State Dumper**: Dump CPU state at each instruction for debugging

## Success Criteria

- [ ] Simultaneous interrupt tests added
- [ ] All decimal mode edge cases tested
- [ ] Instruction length validation for all 256 opcodes
- [ ] Unofficial opcode tests added
- [ ] Self-modifying code tests added
- [ ] Cycle accuracy tests added
- [ ] Test coverage >95%
- [ ] All tests pass
- [ ] Integration with Klaus Dormann test suite
