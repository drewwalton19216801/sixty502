# Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing the modularization plan. Follow these steps carefully to ensure a smooth transition with no breaking changes.

---

## Prerequisites

### 1. Verify Current Branch

```bash
# Confirm we're on the feat/split-modules branch
git branch --show-current
# Should output: feat/split-modules
```

### 2. Verify Tests Pass

```bash
go test -v ./...
go test -race ./...
```

### 3. Create Working Branch

```bash
git checkout -b refactor/modularize-cpu
```

---

## Phase 1: Foundation Modules (No Dependencies)

### Step 1.1: Create types.go

**Estimated Time**: 30 minutes

```bash
# Create file
touch types.go
```

**Actions**:

1. Add package declaration and documentation
2. Copy AddrModeType and constants (lines 78-106)
3. Copy CPUVariant and methods (lines 108-166)
4. Copy Flags and constants (lines 228-250)
5. Copy Instruction struct (lines 382-391)
6. Copy State struct and String method (lines 1891-1937)

**Verification**:

```bash
go build
go test -v
```

**Remove from cpu.go**:

- Delete copied type definitions
- Keep only references/usage

---

### Step 1.2: Create errors.go

**Estimated Time**: 20 minutes

**Actions**:

1. Add package declaration and imports
2. Copy ErrorType and constants (lines 33-40)
3. Copy CPUError struct and Error method (lines 42-52)
4. Copy ErrorHandler interface (lines 54-57)
5. Copy StrictErrorHandler (lines 59-64)
6. Copy LoggingErrorHandler (lines 66-76)

**Verification**:

```bash
go build
go test -v -run TestError
```

---

### Step 1.3: Create bus.go

**Estimated Time**: 15 minutes

**Actions**:

1. Add package declaration and documentation
2. Copy Bus interface (lines 168-193)
3. Add comprehensive documentation with examples

**Verification**:

```bash
go build
```

---

## Phase 2: Configuration Module

### Step 2.1: Create config.go

**Estimated Time**: 45 minutes

**Actions**:

1. Add package declaration and imports
2. Copy CPUConfig struct (lines 195-214)
3. Copy DefaultConfig function (lines 217-226)
4. Copy CPUBuilder struct (lines 465-469)
5. Copy all builder methods (lines 472-518)
6. Copy factory functions (lines 393-463)

**Important Notes**:

- Factory functions create CPU instances
- Keep NewCPUWithConfig implementation in cpu.go initially
- Builder.Build() calls NewCPUWithConfig

**Verification**:

```bash
go build
go test -v -run TestNew
go test -v -run TestBuilder
```

---

## Phase 3: Support Modules

### Step 3.1: Create cache.go

**Estimated Time**: 30 minutes

**Actions**:

1. Copy InstructionCacheEntry struct (lines 252-257)
2. Copy InstructionCache struct (lines 259-264)
3. Copy NewInstructionCache (lines 266-269)
4. Copy all cache methods (lines 271-312)

**Verification**:

```bash
go build
go test -v -run TestCache
```

---

### Step 3.2: Create helpers.go

**Estimated Time**: 20 minutes

**Actions**:

1. Copy fetchDataIfNeeded (lines 703-713)
2. Copy setZNFlags (lines 716-719)
3. Add documentation for each helper

**Verification**:

```bash
go build
```

---

## Phase 4: Addressing Modes

### Step 4.1: Create addressing.go

**Estimated Time**: 45 minutes

**Actions**:

1. Copy all 12 addressing mode methods (lines 575-697)
2. Add comprehensive documentation
3. Document page crossing behavior
4. Note variant-specific behavior (IND)

**Methods to Copy**:

- IMP, IMM, ZP0, ZPX, ZPY, REL
- ABS, ABX, ABY, IND, IZX, IZY

**Verification**:

```bash
go build
go test -v -run TestAddressing
```

---

## Phase 5: Instruction Modules

### Step 5.1: Create instructions_flags.go

**Estimated Time**: 20 minutes

**Actions**:

1. Copy flag manipulation instructions
2. CLC, CLD, CLI, CLV (lines 930-933)
3. SEC, SED, SEI (lines 1224-1226)

**Verification**:

```bash
go build
go test -v -run TestFlags
```

---

### Step 5.2: Create instructions_stack.go

**Estimated Time**: 25 minutes

**Actions**:

1. Copy PHA, PHP (lines 1052-1056)
2. Copy PLA, PLP (lines 1057-1063)

**Verification**:

```bash
go build
go test -v -run TestStack
```

---

### Step 5.3: Create instructions_logical.go

**Estimated Time**: 30 minutes

**Actions**:

1. Copy AND (line 833)
2. Copy ORA (line 1045)
3. Copy EOR (line 970)
4. Copy BIT (line 882)

**Verification**:

```bash
go build
go test -v -run TestLogical
```

---

### Step 5.4: Create instructions_branch.go

**Estimated Time**: 30 minutes

**Actions**:

1. Copy branchIf helper (lines 861-871)
2. Copy all 8 branch instructions (lines 873-880)

**Verification**:

```bash
go build
go test -v -run TestBranch
```

---

### Step 5.5: Create instructions_transfer.go

**Estimated Time**: 35 minutes

**Actions**:

1. Copy LDA, LDX, LDY (lines 1000-1018)
2. Copy STA, STX, STY (lines 1228-1234)
3. Copy TAX, TAY, TSX, TXA, TXS, TYA (lines 1236-1241)

**Verification**:

```bash
go build
go test -v -run TestTransfer
go test -v -run TestLoad
go test -v -run TestStore
```

---

### Step 5.6: Create instructions_shift.go

**Estimated Time**: 40 minutes

**Actions**:

1. Copy ASL (lines 840-858)
2. Copy LSR (lines 1021-1037)
3. Copy ROL (lines 1065-1088)
4. Copy ROR (lines 1090-1114)

**Verification**:

```bash
go build
go test -v -run TestShift
go test -v -run TestRotate
```

---

### Step 5.7: Create instructions_control.go

**Estimated Time**: 35 minutes

**Actions**:

1. Copy JMP (lines 988-991)
2. Copy JSR (lines 993-998)
3. Copy RTS (lines 1129-1133)
4. Copy RTI (lines 1116-1127)
5. Copy BRK (lines 891-928)
6. Copy NOP (lines 1039-1043)
7. Copy XXX (lines 1243-1246)

**Verification**:

```bash
go build
go test -v -run TestControl
go test -v -run TestJump
go test -v -run TestSubroutine
```

---

### Step 5.8: Create instructions_arithmetic.go

**Estimated Time**: 60 minutes

**Actions**:

1. Copy ADC and helpers (lines 721-831)
   - ADC main function
   - adcBinary helper
   - adcDecimal helper
2. Copy SBC and helpers (lines 1135-1222)
   - SBC main function
   - sbcBinary helper
   - sbcDecimal helper
3. Copy INC, INX, INY (lines 977-986)
4. Copy DEC, DEX, DEY (lines 959-968)
5. Copy CMP, CPX, CPY (lines 935-956)

**Verification**:

```bash
go build
go test -v -run TestArithmetic
go test -v -run TestDecimal
go test -v -run TestCompare
```

---

## Phase 6: Lookup Table

### Step 6.1: Create lookup.go

**Estimated Time**: 45 minutes

**Actions**:

1. Copy buildLookupTable method (lines 1250-1581)
2. Keep all comments about page crossing
3. Ensure all method expressions are correct

**Important**:

- This is the largest single function
- Contains all opcode definitions
- Critical for correct operation

**Verification**:

```bash
go build
go test -v -run TestLookup
go test -v -run TestOpcode
```

---

## Phase 7: Interrupt Handling

### Step 7.1: Create interrupts.go

**Estimated Time**: 40 minutes

**Actions**:

1. Copy SetIRQ (lines 1635-1639)
2. Copy SetNMI (lines 1641-1650)
3. Copy ClearNMI (lines 1652-1655)
4. Copy HasPendingInterrupt (lines 1657-1661)
5. Copy handleNMI (lines 1663-1696)
6. Copy handleIRQ (lines 1698-1730)
7. Copy deprecated methods (lines 1614-1633)

**Verification**:

```bash
go build
go test -v -run TestInterrupt
go test -v -run TestNMI
go test -v -run TestIRQ
```

---

## Phase 8: State and Debug

### Step 8.1: Create state.go

**Estimated Time**: 35 minutes

**Actions**:

1. Copy RemainingCycles (line 1829)
2. Copy SetCycles (lines 1833-1835)
3. Copy TotalCycles (lines 1840-1842)
4. Copy CurrentOpcode (lines 1844-1846)
5. Copy LookupInstruction (lines 1849-1851)
6. Copy IsIllegalOpcode (lines 1853-1855)
7. Copy GetStateSnapshot (lines 1908-1927)
8. Copy GetState (lines 1959-2009)
9. Copy FormatFlags (lines 2011-2029)
10. Copy LastError (lines 1954-1956)
11. Copy Variant (lines 520-522)

**Verification**:

```bash
go build
go test -v -run TestState
```

---

### Step 8.2: Create debug.go

**Estimated Time**: 40 minutes

**Actions**:

1. Copy GetCurrentInstruction (lines 1943-1945)
2. Copy Opcode (deprecated) (lines 1948-1950)
3. Copy LookupTable (lines 2127-2129)
4. Copy Disassemble (lines 2032-2123)

**Verification**:

```bash
go build
go test -v -run TestDisassemble
go test -v -run TestDebug
```

---

## Phase 9: Final CPU Core

### Step 9.1: Refactor cpu.go

**Estimated Time**: 60 minutes

**What Remains**:

1. CPU struct definition (lines 336-379)
2. Core memory operations (lines 526-532)
3. Stack operations (lines 534-556)
4. Flag operations (lines 558-569)
5. Reset (lines 1599-1612)
6. Clock (lines 1751-1824)
7. Cache control methods (lines 1861-1887)

**Actions**:

1. Remove all extracted code
2. Keep only core execution logic
3. Ensure all imports are correct
4. Add imports for new modules (none needed - same package)

**Verification**:

```bash
go build
go test -v ./...
go test -race ./...
```

---

## Phase 10: Final Validation

### Step 10.1: Comprehensive Testing

**Estimated Time**: 45 minutes

```bash
# Run all tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem

# Test examples
cd examples/basic && go run main.go
cd ../memory-mapped && go run main.go
```

---

### Step 10.2: Documentation Update

**Estimated Time**: 30 minutes

**Actions**:

1. Update package documentation
2. Add module overview to README
3. Update architecture documentation
4. Add migration guide for contributors

---

### Step 10.3: Code Review Checklist

- [ ] All tests pass
- [ ] No race conditions detected
- [ ] Examples work correctly
- [ ] Benchmarks show no regression
- [ ] Documentation is complete
- [ ] No circular dependencies
- [ ] All exports are intentional
- [ ] Code formatting is consistent
- [ ] Comments are clear and accurate

---

## Troubleshooting

### Common Issues

#### Import Cycles

**Problem**: Circular dependency between modules
**Solution**: Move shared types to types.go or create separate file

#### Missing Methods

**Problem**: Method not found after extraction
**Solution**: Verify method is in correct file and exported

#### Test Failures

**Problem**: Tests fail after refactoring
**Solution**: Check that all code was copied correctly, no logic changed

#### Build Errors

**Problem**: Undefined types or functions
**Solution**: Ensure all dependencies are in place before extraction

---

## Rollback Plan

If issues arise:

```bash
# Revert specific commits
git revert <commit-hash>

# Or reset to before modularization (use with caution)
git reset --hard HEAD~N

# Or switch back to main and restart
git checkout main
git branch -D feat/split-modules
git checkout -b feat/split-modules
```

---

## Success Metrics

### Must Have

- ✅ All tests pass
- ✅ No breaking API changes
- ✅ Examples work
- ✅ Documentation complete

### Nice to Have

- ✅ Improved test coverage
- ✅ Better performance
- ✅ Enhanced documentation
- ✅ Additional examples

---

## Post-Implementation

### 1. Update CI/CD

Ensure build pipeline works with new structure

### 2. Update Documentation

- Architecture diagrams
- Module dependency graphs
- Contribution guidelines

### 3. Announce Changes

- Update CHANGELOG
- Create migration guide
- Notify users of improvements

### 4. Monitor

- Watch for issues
- Gather feedback
- Plan improvements

---

## Estimated Total Time

| Phase | Time |
|-------|------|
| Phase 1: Foundation | 1.5 hours |
| Phase 2: Configuration | 1 hour |
| Phase 3: Support | 1 hour |
| Phase 4: Addressing | 1 hour |
| Phase 5: Instructions | 4 hours |
| Phase 6: Lookup | 1 hour |
| Phase 7: Interrupts | 1 hour |
| Phase 8: State/Debug | 1.5 hours |
| Phase 9: CPU Core | 1 hour |
| Phase 10: Validation | 1.5 hours |
| **Total** | **14.5 hours** |

Add 20% buffer for unexpected issues: **~17-18 hours**

---

## Tips for Success

1. **Work incrementally** - Complete one module at a time
2. **Test frequently** - Run tests after each module
3. **Commit often** - Small commits are easier to revert
4. **Document as you go** - Add comments while code is fresh
5. **Ask for help** - Review with team members
6. **Take breaks** - Fresh eyes catch more issues
7. **Use tools** - Let IDE help with refactoring
8. **Stay organized** - Keep track of what's done

---

## Next Steps

After completing this implementation:

1. **Review with team** - Get feedback on structure
2. **Plan improvements** - Identify areas for enhancement
3. **Update roadmap** - Plan future features
4. **Celebrate** - Acknowledge the improved codebase!
