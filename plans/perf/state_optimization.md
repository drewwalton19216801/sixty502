# Performance Improvement Plan: State Management and Flags

## Current State

The CPU flags are managed via a `Flags` type (likely a bitmask) and helper methods `setFlag` and `getFlag`.

### Bottlenecks

1. **Bit Manipulation**: Setting and getting flags involves bitwise AND/OR/NOT operations. While fast, doing this for every instruction (especially Zero and Negative flags) adds up.
2. **Conditional Branching**: The 6502's `P` register is often reconstructed or decomposed.
3. **Decimal Mode**: The `D` flag triggers complex BCD arithmetic in `ADC` and `SBC`.

## Proposed Improvements

### 1. Exploded Flags

Instead of keeping flags in a single `uint8`, store them as individual boolean or integer variables in the `CPU` struct.

**Action:**

- Define `c.flagN`, `c.flagZ`, `c.flagC`, etc., as separate fields.
- Only pack them into the `P` register when an instruction explicitly reads it (PHP, RTI) or when an interrupt occurs.
- This makes `if c.flagZ` much faster than `if c.getFlag(Z)`.

### 2. Lazy Zero and Negative Flag Calculation

The Zero (Z) and Negative (N) flags are updated by almost every instruction.

**Action:**

- Instead of calculating `Z = (result == 0)` and `N = (result & 0x80) != 0` immediately, store the `lastResult`.
- `getFlag(Z)` then becomes `return lastResult == 0`.
- `getFlag(N)` then becomes `return (lastResult & 0x80) != 0`.
- This avoids redundant work if the flags are overwritten before being read.

### 3. Optimized BCD Arithmetic

Decimal mode is rarely used but checked in every `ADC`/`SBC`.

**Action:**

- Use a specialized "Fast ADC" and "Fast SBC" function that is chosen at dispatch time if the `D` flag is set, or use a simple `if c.flagD` check inside the instruction.
- Ensure the non-decimal path is the "hot" path for the branch predictor.

## Expected Impact

- 10-15% improvement in arithmetic and logic instructions.
- Cleaner code in the instruction implementations.
