package cpu6502

import (
	"testing"
)

// --- Error Handling Tests ---

// TestErrorHandling_IllegalOpcode tests that illegal opcodes are properly detected and handled
func TestErrorHandling_IllegalOpcode(t *testing.T) {
	t.Run("StrictMode_HaltsOnIllegalOpcode", func(t *testing.T) {
		bus := NewMockBus()
		cpu := NewCPUWithErrorHandler(bus, &StrictErrorHandler{})

		// Set up vectors
		bus.Write(0xFFFC, 0x00)
		bus.Write(0xFFFD, 0x80)

		// Load an illegal opcode
		illegalOpcode := uint8(0x02) // KIL/JAM - illegal opcode
		bus.Write(0x8000, illegalOpcode)
		bus.Write(0x8001, 0x00) // BRK after

		cpu.PC = 0x8000
		cpu.Cycles = 0

		// Execute - should return error
		err := cpu.Clock()
		if err == nil {
			t.Error("Expected error from illegal opcode in strict mode, got nil")
		}

		// Verify error details
		cpuErr, ok := err.(*CPUError)
		if !ok {
			t.Errorf("Expected *CPUError, got %T", err)
		} else {
			if cpuErr.Type != ErrorIllegalOpcode {
				t.Errorf("Expected ErrorIllegalOpcode, got %v", cpuErr.Type)
			}
			if cpuErr.Opcode != illegalOpcode {
				t.Errorf("Expected opcode 0x%02X, got 0x%02X", illegalOpcode, cpuErr.Opcode)
			}
			if cpuErr.PC != 0x8000 {
				t.Errorf("Expected PC 0x8000, got 0x%04X", cpuErr.PC)
			}
		}

		// Verify lastError is set
		if cpu.LastError() == nil {
			t.Error("Expected LastError to be set")
		}
	})

	t.Run("LoggingMode_ContinuesOnIllegalOpcode", func(t *testing.T) {
		bus := NewMockBus()
		cpu := NewCPUWithErrorHandler(bus, &LoggingErrorHandler{Logger: nil})

		// Set up vectors
		bus.Write(0xFFFC, 0x00)
		bus.Write(0xFFFD, 0x80)

		// Load an illegal opcode followed by a valid instruction
		illegalOpcode := uint8(0x02) // KIL/JAM - illegal opcode
		bus.Write(0x8000, illegalOpcode)
		bus.Write(0x8001, 0xEA) // NOP
		bus.Write(0x8002, 0x00) // BRK

		cpu.PC = 0x8000
		cpu.Cycles = 0

		// Execute illegal opcode - should NOT return error (logging mode)
		err := cpu.Clock()
		if err != nil {
			t.Errorf("Expected no error in logging mode, got: %v", err)
		}

		// Verify lastError is still set
		if cpu.LastError() == nil {
			t.Error("Expected LastError to be set even in logging mode")
		}

		// Should be able to continue execution
		cpu.Cycles = 0
		err = cpu.Clock()
		if err != nil {
			t.Errorf("Expected to continue execution after illegal opcode, got error: %v", err)
		}
	})

	t.Run("DefaultMode_UsesLoggingHandler", func(t *testing.T) {
		bus := NewMockBus()
		cpu := NewCPU(bus) // Uses default logging handler

		// Set up vectors
		bus.Write(0xFFFC, 0x00)
		bus.Write(0xFFFD, 0x80)

		// Load an illegal opcode
		illegalOpcode := uint8(0x12) // Another illegal opcode
		bus.Write(0x8000, illegalOpcode)

		cpu.PC = 0x8000
		cpu.Cycles = 0

		// Execute - should NOT return error (default is logging mode)
		err := cpu.Clock()
		if err != nil {
			t.Errorf("Expected no error with default handler, got: %v", err)
		}

		// Verify lastError is set
		if cpu.LastError() == nil {
			t.Error("Expected LastError to be set")
		}
	})
}

// TestErrorHandling_MultipleErrors tests handling of multiple errors
func TestErrorHandling_MultipleErrors(t *testing.T) {
	bus := NewMockBus()
	cpu := NewCPUWithErrorHandler(bus, &LoggingErrorHandler{Logger: nil})

	// Set up vectors
	bus.Write(0xFFFC, 0x00)
	bus.Write(0xFFFD, 0x80)

	// Load multiple illegal opcodes
	bus.Write(0x8000, 0x02) // Illegal
	bus.Write(0x8001, 0x12) // Illegal
	bus.Write(0x8002, 0x00) // BRK

	cpu.PC = 0x8000
	cpu.Cycles = 0

	// Execute first illegal opcode
	err := cpu.Clock()
	if err != nil {
		t.Errorf("Expected no error in logging mode, got: %v", err)
	}

	firstError := cpu.LastError()
	if firstError == nil {
		t.Fatal("Expected first error to be set")
	}
	if firstError.Opcode != 0x02 {
		t.Errorf("Expected first error opcode 0x02, got 0x%02X", firstError.Opcode)
	}

	// Execute second illegal opcode
	cpu.Cycles = 0
	err = cpu.Clock()
	if err != nil {
		t.Errorf("Expected no error in logging mode, got: %v", err)
	}

	secondError := cpu.LastError()
	if secondError == nil {
		t.Fatal("Expected second error to be set")
	}
	if secondError.Opcode != 0x12 {
		t.Errorf("Expected second error opcode 0x12, got 0x%02X", secondError.Opcode)
	}

	// Verify lastError was updated
	if firstError == secondError {
		t.Error("Expected lastError to be updated with new error")
	}
}

// TestErrorHandling_ErrorMessage tests the error message formatting
func TestErrorHandling_ErrorMessage(t *testing.T) {
	err := &CPUError{
		Type:    ErrorIllegalOpcode,
		Opcode:  0x02,
		PC:      0x8000,
		Message: "illegal opcode $02",
	}

	expectedMsg := "CPU error at $8000: illegal opcode $02 (opcode $02)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, err.Error())
	}
}
