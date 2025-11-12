# CPU.go Modularization Plan - Overview

## Executive Summary

This document outlines a comprehensive plan to refactor the monolithic `cpu.go` file (2,129 lines) into a well-organized, modular structure. The refactoring will improve maintainability, testability, and code organization while maintaining backward compatibility.

## Current State Analysis

### File Statistics

- **Total Lines**: 2,129
- **Package**: cpu6502
- **Primary Concerns**:
  - Single file contains all CPU functionality
  - Difficult to navigate and maintain
  - Testing requires understanding entire file
  - Hard to extend with new features

### Major Components Identified

1. **Error Handling** (Lines 33-76)
   - Error types and structures
   - Error handlers (Strict, Logging)

2. **Type Definitions** (Lines 78-154)
   - Addressing modes
   - CPU variants
   - Flags

3. **Bus Interface** (Lines 168-193)
   - Memory access abstraction

4. **Configuration** (Lines 195-518)
   - CPU configuration structures
   - Builder pattern implementation
   - Factory functions

5. **Instruction Cache** (Lines 252-312)
   - Cache implementation
   - Statistics tracking

6. **CPU Core Structure** (Lines 314-379)
   - Main CPU struct
   - Instruction definition

7. **Stack Operations** (Lines 534-556)
   - Push/pop operations

8. **Flag Operations** (Lines 558-569)
   - Flag manipulation helpers

9. **Addressing Modes** (Lines 571-697)
   - 12 addressing mode implementations

10. **Instruction Operations** (Lines 699-1246)
    - All 6502 instruction implementations
    - Helper functions

11. **Lookup Table** (Lines 1248-1581)
    - Instruction table initialization

12. **Core Execution** (Lines 1583-1824)
    - Reset, Clock, Interrupt handling

13. **Accessors & Debug** (Lines 1826-2129)
    - State inspection
    - Disassembly
    - Debug helpers

## Proposed Module Structure

```text
cpu6502/
├── cpu.go                    # Main CPU struct and core execution
├── types.go                  # Type definitions (Flags, AddrModeType, CPUVariant)
├── errors.go                 # Error types and handlers
├── config.go                 # Configuration and builder pattern
├── bus.go                    # Bus interface definition
├── cache.go                  # Instruction cache implementation
├── addressing.go             # All addressing mode implementations
├── instructions.go           # Instruction operation implementations
├── instructions_arithmetic.go # ADC, SBC, INC, DEC, etc.
├── instructions_logical.go   # AND, ORA, EOR, BIT
├── instructions_shift.go     # ASL, LSR, ROL, ROR
├── instructions_branch.go    # BCC, BCS, BEQ, etc.
├── instructions_transfer.go  # LDA, LDX, LDY, STA, STX, STY, TAX, etc.
├── instructions_stack.go     # PHA, PHP, PLA, PLP
├── instructions_control.go   # JMP, JSR, RTS, RTI, BRK
├── instructions_flags.go     # CLC, CLD, CLI, CLV, SEC, SED, SEI
├── lookup.go                 # Instruction lookup table
├── interrupts.go             # Interrupt handling (IRQ, NMI)
├── state.go                  # State inspection and snapshots
├── debug.go                  # Disassembly and debug helpers
└── helpers.go                # Internal helper functions
```

## Key Design Principles

### 1. Backward Compatibility

- **All public APIs remain unchanged**
- Existing code using the library continues to work
- No breaking changes to exported types or methods

### 2. Logical Grouping

- Related functionality grouped together
- Clear separation of concerns
- Easy to locate specific functionality

### 3. Maintainability

- Smaller, focused files (150-300 lines each)
- Clear file naming conventions
- Consistent organization patterns

### 4. Testability

- Each module can be tested independently
- Easier to write focused unit tests
- Better test coverage visibility

### 5. Extensibility

- Easy to add new instructions
- Simple to extend with new variants
- Clear patterns for new features

## Migration Strategy

### Phase 1: Preparation

1. Create new module files with package declaration
2. Document dependencies between modules
3. Identify shared internal functions

### Phase 2: Type Extraction

1. Move type definitions to `types.go`
2. Move error types to `errors.go`
3. Move bus interface to `bus.go`
4. Update imports in remaining code

### Phase 3: Configuration Extraction

1. Move configuration to `config.go`
2. Ensure builder pattern works correctly
3. Test factory functions

### Phase 4: Instruction Extraction

1. Move addressing modes to `addressing.go`
2. Split instructions into logical groups
3. Move lookup table to `lookup.go`
4. Verify all instruction references

### Phase 5: Support Systems

1. Move cache to `cache.go`
2. Move interrupts to `interrupts.go`
3. Move state/debug to respective files
4. Extract helper functions

### Phase 6: Core Refinement

1. Keep only core execution in `cpu.go`
2. Ensure all cross-references work
3. Update documentation
4. Run full test suite

## Benefits

### For Developers

- **Easier Navigation**: Find code quickly by file name
- **Focused Changes**: Modify specific functionality without touching unrelated code
- **Better Understanding**: Smaller files are easier to comprehend
- **Parallel Development**: Multiple developers can work on different modules

### For Maintainers

- **Reduced Complexity**: Each file has a single, clear purpose
- **Easier Debugging**: Isolate issues to specific modules
- **Better Testing**: Write focused tests for each module
- **Clear Dependencies**: Understand relationships between components

### For Users

- **No Breaking Changes**: Existing code continues to work
- **Better Documentation**: Each module can have focused documentation
- **Easier Learning**: Understand the CPU piece by piece
- **Clearer Examples**: Reference specific modules in examples

## Risk Mitigation

### Potential Risks

1. **Build Errors**: Circular dependencies or missing imports
2. **Test Failures**: Tests may need updates for new structure
3. **Performance Impact**: Additional file overhead (minimal)
4. **Documentation Drift**: Docs may need updates

### Mitigation Strategies

1. **Incremental Approach**: Move one module at a time
2. **Continuous Testing**: Run tests after each module extraction
3. **Dependency Mapping**: Document all dependencies before moving
4. **Automated Checks**: Use go build and go test throughout

## Success Criteria

1. ✅ All existing tests pass without modification
2. ✅ No breaking changes to public API
3. ✅ Each file is under 400 lines
4. ✅ Clear, logical organization
5. ✅ Improved code navigation
6. ✅ Documentation updated
7. ✅ Examples still work correctly

## Timeline Estimate

- **Phase 1 (Preparation)**: 1-2 hours
- **Phase 2 (Type Extraction)**: 2-3 hours
- **Phase 3 (Configuration)**: 1-2 hours
- **Phase 4 (Instructions)**: 4-6 hours
- **Phase 5 (Support Systems)**: 2-3 hours
- **Phase 6 (Core Refinement)**: 2-3 hours
- **Testing & Documentation**: 2-3 hours

**Total Estimated Time**: 14-22 hours

## Next Steps

1. Review this plan with stakeholders
2. Get approval for the proposed structure
3. Begin Phase 1 implementation
4. Proceed incrementally through each phase
5. Validate at each step with tests
