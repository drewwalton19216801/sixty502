# Performance Improvement Plan: Instruction Dispatch Optimization

## Current State

The CPU uses a `Clock()` method that handles instruction fetching, decoding, and execution. It includes an optional `InstructionCache` which, according to benchmarks, provides negligible or even negative performance benefits in some cases.

### Bottlenecks

1. **Function Pointer Overhead**: Every instruction execution involves at least two function pointer calls (`AddrMode` and `Operate`).
2. **Cache Overhead**: The `InstructionCache` adds branching and memory access overhead to every fetch cycle, which outweighs the cost of a simple array lookup in `c.lookup[opcode]`.
3. **Cycle-by-Cycle Overhead**: The `Clock()` method is designed for cycle-accuracy but adds significant overhead when running long sequences of instructions where cycle-accuracy isn't strictly required for every single clock tick.

## Proposed Improvements

### 1. Replace Function Pointers with Switch/Case Dispatch

Instead of storing function pointers in the `Instruction` struct, use a large switch statement or a code generation approach to inline instruction logic.

**Action:**

- Create a `Step()` method that executes a full instruction at once.
- Use a switch statement on `c.opcode` to dispatch to specific instruction logic.
- This allows the compiler to better optimize the dispatch and potentially inline small instructions.

### 2. Remove or Redesign Instruction Cache

The current `InstructionCache` is indexed by `PC & 0xFF`. Since the `lookup` table is already a 256-entry array indexed by `opcode`, the cache is essentially a redundant lookup that adds more work than it saves.

**Action:**

- Remove the `InstructionCache` entirely as it currently exists.
- If caching is desired, consider a "Compiled Block" approach where sequences of instructions are translated into Go functions.

### 3. Fast-Path for Non-Cycle-Accurate Execution

Many users don't need to call `Clock()` for every single cycle.

**Action:**

- Implement `Execute(cycles uint64)` which runs the CPU for a requested number of cycles as fast as possible.
- This method can skip the `c.cycles == 0` check on every iteration and instead process whole instructions.

## Expected Impact

- 20-40% reduction in instruction dispatch overhead.
- Improved branch prediction by the host CPU.
- Significant speedup for bulk execution.
