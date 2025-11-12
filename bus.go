package cpu6502

// Bus defines the interface for memory access.
//
// Implementations of this interface provide the CPU with access to
// memory and memory-mapped I/O. The interface is intentionally simple
// to allow for flexible implementations.
//
// # Memory Mapping Patterns
//
// The Bus interface supports various memory mapping strategies:
//
//   - Simple RAM: Direct array access for the full 64KB address space
//   - Banked Memory: Multiple memory banks switched via control registers
//   - Memory-Mapped I/O: Special addresses that trigger hardware behavior
//   - ROM/RAM Combinations: Read-only regions mixed with writable RAM
//
// # Implementation Guidelines
//
// Implementations should:
//   - Handle all 64KB addresses (0x0000-0xFFFF)
//   - Return consistent values for reads (no side effects unless intended)
//   - Complete operations quickly (the CPU expects cycle-accurate timing)
//   - Consider thread safety if used in concurrent contexts
//
// # Example Implementation
//
// Simple 64KB RAM:
//
//	type SimpleBus struct {
//	    ram [65536]uint8
//	}
//
//	func (b *SimpleBus) Read(addr uint16) uint8 {
//	    return b.ram[addr]
//	}
//
//	func (b *SimpleBus) Write(addr uint16, data uint8) {
//	    b.ram[addr] = data
//	}
//
// Memory-mapped I/O example:
//
//	type IOBus struct {
//	    ram    [65536]uint8
//	    output io.Writer
//	}
//
//	func (b *IOBus) Write(addr uint16, data uint8) {
//	    if addr == 0x6000 {
//	        // Write to output device
//	        b.output.Write([]byte{data})
//	    } else {
//	        b.ram[addr] = data
//	    }
//	}
type Bus interface {
	// Read returns the byte at the specified address.
	//
	// The address space is 16-bit (0x0000-0xFFFF). Implementations
	// must handle all possible addresses, even if they map to the
	// same physical memory or return constant values.
	//
	// For memory-mapped I/O, reads may have side effects (e.g.,
	// clearing interrupt flags, advancing hardware state).
	Read(addr uint16) uint8

	// Write stores a byte at the specified address.
	//
	// The address space is 16-bit (0x0000-0xFFFF). Implementations
	// may ignore writes to read-only regions (ROM) or trigger
	// hardware behavior for memory-mapped I/O addresses.
	//
	// For memory-mapped I/O, writes may trigger immediate hardware
	// actions (e.g., sending data to a device, updating graphics).
	Write(addr uint16, data uint8)
}
