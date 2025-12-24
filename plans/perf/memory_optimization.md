# Performance Improvement Plan: Memory and Bus Optimization

## Current State

Memory access is performed through the `Bus` interface, which requires an interface method call for every read and write.

### Bottlenecks

1. **Interface Dispatch**: Every `read()` and `write()` call on the CPU goes through the `Bus` interface. In Go, interface method calls are more expensive than direct function calls or array accesses.
2. **Zero Page Access**: The 6502 heavily relies on Zero Page (addresses $00-$FF). Currently, these are treated like any other memory access.
3. **Redundant Calculations**: Addressing modes like `ABX`, `ABY`, and `IZY` perform page-cross checks and additions every time, even when the results could be partially pre-calculated.

## Proposed Improvements

### 1. Specialized Zero Page Access

Since the Zero Page is frequently accessed, providing a faster path for it can yield significant gains.

**Action:**

- Add a `DirectPage` pointer (or slice) to the `CPU` struct that points to the first 256 bytes of RAM.
- If the `Bus` implementation allows it, the CPU can read/write directly to this slice, bypassing the interface.

### 2. Bus "Fast Path" for Simple RAM

For many emulations, the majority of the 64KB address space is simple RAM.

**Action:**

- Allow the `Bus` to expose a raw slice of memory via an optional interface (e.g., `FastBus`).
- If the `Bus` implements `FastBus`, the CPU can cache the slice and perform direct array indexing for most operations.

### 3. Inline Addressing Mode Logic

Currently, addressing modes are separate functions.

**Action:**

- Inline the logic for common addressing modes (Immediate, Zero Page, Absolute) directly into the instruction execution path.
- This reduces the call stack depth and allows for better register allocation by the compiler.

## Expected Impact

- 15-25% improvement in memory-heavy instructions (LDA, STA, etc.).
- Reduced CPU cache pressure by avoiding interface lookup tables (itables).
