package cpu6502

import (
	"testing"
)

// setupCPUWithVariant creates a CPU instance with a specific variant for testing
func setupCPUWithVariant(variant CPUVariant) (*CPU, *MockBus) {
	bus := NewMockBus()
	cpu := NewCPUWithVariant(bus, variant)
	// Set reset/irq/nmi vectors for safety during tests
	bus.Write(0xFFFA, 0x00)
	bus.Write(0xFFFB, 0xF0) // NMI -> F000
	bus.Write(0xFFFC, 0x00)
	bus.Write(0xFFFD, 0xF1) // Reset -> F100
	bus.Write(0xFFFE, 0x00)
	bus.Write(0xFFFF, 0xF2) // IRQ/BRK -> F200
	return cpu, bus
}

// TestVariantProperties verifies the basic properties of each variant
func TestVariantProperties(t *testing.T) {
	tests := []struct {
		variant             CPUVariant
		name                string
		supportsDecimalMode bool
		hasIndirectJMPBug   bool
	}{
		{VariantNMOS6502, "NMOS 6502", true, true},
		{VariantCMOS65C02, "CMOS 65C02", true, false},
		{VariantRicoh2A03, "Ricoh 2A03 (NTSC)", false, true},
		{VariantRicoh2A07, "Ricoh 2A07 (PAL)", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.variant.String() != tt.name {
				t.Errorf("String() = %q, want %q", tt.variant.String(), tt.name)
			}
			if tt.variant.SupportsDecimalMode() != tt.supportsDecimalMode {
				t.Errorf("SupportsDecimalMode() = %v, want %v",
					tt.variant.SupportsDecimalMode(), tt.supportsDecimalMode)
			}
			if tt.variant.HasIndirectJMPBug() != tt.hasIndirectJMPBug {
				t.Errorf("HasIndirectJMPBug() = %v, want %v",
					tt.variant.HasIndirectJMPBug(), tt.hasIndirectJMPBug)
			}
		})
	}
}

// TestVariantConstructors verifies CPU construction with variants
func TestVariantConstructors(t *testing.T) {
	bus := NewMockBus()

	t.Run("NewCPU defaults to NMOS", func(t *testing.T) {
		cpu := NewCPU(bus)
		if cpu.Variant() != VariantNMOS6502 {
			t.Errorf("NewCPU() variant = %v, want %v", cpu.Variant(), VariantNMOS6502)
		}
	})

	t.Run("NewCPUWithVariant sets variant", func(t *testing.T) {
		variants := []CPUVariant{
			VariantNMOS6502,
			VariantCMOS65C02,
			VariantRicoh2A03,
			VariantRicoh2A07,
		}
		for _, variant := range variants {
			cpu := NewCPUWithVariant(bus, variant)
			if cpu.Variant() != variant {
				t.Errorf("NewCPUWithVariant(%v) variant = %v, want %v",
					variant, cpu.Variant(), variant)
			}
		}
	})

	t.Run("NewCPUWithErrorHandler defaults to NMOS", func(t *testing.T) {
		cpu := NewCPUWithErrorHandler(bus, &StrictErrorHandler{})
		if cpu.Variant() != VariantNMOS6502 {
			t.Errorf("NewCPUWithErrorHandler() variant = %v, want %v",
				cpu.Variant(), VariantNMOS6502)
		}
	})

	t.Run("NewCPUWithVariantAndErrorHandler sets both", func(t *testing.T) {
		handler := &StrictErrorHandler{}
		cpu := NewCPUWithVariantAndErrorHandler(bus, VariantCMOS65C02, handler)
		if cpu.Variant() != VariantCMOS65C02 {
			t.Errorf("variant = %v, want %v", cpu.Variant(), VariantCMOS65C02)
		}
	})
}

// TestIndirectJMPBug tests the page boundary bug behavior across variants
func TestIndirectJMPBug(t *testing.T) {
	variants := []struct {
		variant CPUVariant
		hasBug  bool
	}{
		{VariantNMOS6502, true},
		{VariantCMOS65C02, false},
		{VariantRicoh2A03, true},
		{VariantRicoh2A07, true},
	}

	for _, tt := range variants {
		t.Run(tt.variant.String(), func(t *testing.T) {
			cpu, bus := setupCPUWithVariant(tt.variant)

			// Set up JMP ($10FF) where low byte is $FF
			indirectAddr := uint16(0x10FF)
			targetLow := uint8(0x34)
			targetHigh := uint8(0x12)

			// Program: JMP ($10FF)
			program := []uint8{
				0x6C,                     // JMP Indirect
				uint8(indirectAddr),      // Low byte ($FF)
				uint8(indirectAddr >> 8), // High byte ($10)
				0x00,                     // BRK
			}

			baseAddr := uint16(0x8000)
			bus.load(baseAddr, program)

			// Set up pointer bytes
			bus.Write(indirectAddr, targetLow)         // $10FF = $34
			bus.Write(indirectAddr&0xFF00, targetHigh) // $1000 = $12 (bug location)
			bus.Write(indirectAddr+1, 0xEE)            // $1100 = $EE (correct location)

			cpu.PC = baseAddr
			cpu.SetCycles(0)

			runCycles(cpu, 5)

			var expectedPC uint16
			if tt.hasBug {
				// Bug: reads high byte from $1000 instead of $1100
				expectedPC = uint16(targetHigh)<<8 | uint16(targetLow) // $1234
			} else {
				// Fixed: reads high byte from $1100
				expectedPC = uint16(0xEE)<<8 | uint16(targetLow) // $EE34
			}

			if cpu.PC != expectedPC {
				t.Errorf("%s: JMP bug test failed. Expected PC=$%04X, got PC=$%04X",
					tt.variant.String(), expectedPC, cpu.PC)
			}
		})
	}
}

// TestDecimalModeSupport tests decimal mode behavior across variants
func TestDecimalModeSupport(t *testing.T) {
	variants := []struct {
		variant  CPUVariant
		supports bool
	}{
		{VariantNMOS6502, true},
		{VariantCMOS65C02, true},
		{VariantRicoh2A03, false},
		{VariantRicoh2A07, false},
	}

	for _, tt := range variants {
		t.Run(tt.variant.String()+" ADC", func(t *testing.T) {
			cpu, bus := setupCPUWithVariant(tt.variant)

			// Test: 09 + 01 in decimal mode should give 10 (BCD)
			// or 0A (binary) if decimal mode is disabled
			cpu.A = 0x09
			cpu.setFlag(D, true) // Enable decimal mode
			cpu.setFlag(C, false)

			program := []uint8{0x69, 0x01, 0x00} // ADC #$01, BRK
			baseAddr := uint16(0x8000)
			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.SetCycles(0)

			runCycles(cpu, 2)

			var expectedA uint8
			if tt.supports {
				expectedA = 0x10 // BCD result
			} else {
				expectedA = 0x0A // Binary result (decimal mode ignored)
			}

			if cpu.A != expectedA {
				t.Errorf("%s: ADC decimal test failed. Expected A=$%02X, got A=$%02X",
					tt.variant.String(), expectedA, cpu.A)
			}
		})

		t.Run(tt.variant.String()+" SBC", func(t *testing.T) {
			cpu, bus := setupCPUWithVariant(tt.variant)

			// Test: 10 - 01 in decimal mode should give 09 (BCD)
			// or 0F (binary) if decimal mode is disabled
			cpu.A = 0x10
			cpu.setFlag(D, true) // Enable decimal mode
			cpu.setFlag(C, true) // No borrow

			program := []uint8{0xE9, 0x01, 0x00} // SBC #$01, BRK
			baseAddr := uint16(0x8000)
			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.SetCycles(0)

			runCycles(cpu, 2)

			var expectedA uint8
			if tt.supports {
				expectedA = 0x09 // BCD result
			} else {
				expectedA = 0x0F // Binary result (decimal mode ignored)
			}

			if cpu.A != expectedA {
				t.Errorf("%s: SBC decimal test failed. Expected A=$%02X, got A=$%02X",
					tt.variant.String(), expectedA, cpu.A)
			}
		})
	}
}

// TestDecimalModeADCEdgeCases tests edge cases in decimal mode ADC
func TestDecimalModeADCEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		a         uint8
		operand   uint8
		carryIn   bool
		expectedA uint8
		expectedC bool
		expectedZ bool
	}{
		{"99+1 C=0", 0x99, 0x01, false, 0x00, true, true},
		{"99+0 C=1", 0x99, 0x00, true, 0x00, true, true},
		{"50+50 C=0", 0x50, 0x50, false, 0x00, true, true},
		{"09+1 C=0", 0x09, 0x01, false, 0x10, false, false},
		{"09+1 C=1", 0x09, 0x01, true, 0x11, false, false},
		{"00+00 C=0", 0x00, 0x00, false, 0x00, false, true},
		{"00+00 C=1", 0x00, 0x00, true, 0x01, false, false},
	}

	// Test only variants that support decimal mode
	variants := []CPUVariant{VariantNMOS6502, VariantCMOS65C02}

	for _, variant := range variants {
		for _, tt := range tests {
			t.Run(variant.String()+" "+tt.name, func(t *testing.T) {
				cpu, bus := setupCPUWithVariant(variant)

				cpu.A = tt.a
				cpu.setFlag(D, true)
				cpu.setFlag(C, tt.carryIn)

				program := []uint8{0x69, tt.operand, 0x00} // ADC #operand, BRK
				baseAddr := uint16(0x8000)
				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.SetCycles(0)

				runCycles(cpu, 2)

				if cpu.A != tt.expectedA {
					t.Errorf("Expected A=$%02X, got A=$%02X", tt.expectedA, cpu.A)
				}
				if cpu.getFlag(C) != tt.expectedC {
					t.Errorf("Expected C=%v, got C=%v", tt.expectedC, cpu.getFlag(C))
				}
				if cpu.getFlag(Z) != tt.expectedZ {
					t.Errorf("Expected Z=%v, got Z=%v", tt.expectedZ, cpu.getFlag(Z))
				}
			})
		}
	}
}

// TestDecimalModeSBCEdgeCases tests edge cases in decimal mode SBC
func TestDecimalModeSBCEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		a         uint8
		operand   uint8
		carryIn   bool
		expectedA uint8
		expectedC bool
		expectedZ bool
	}{
		{"00-1 C=1", 0x00, 0x01, true, 0x99, false, false},
		{"00-1 C=0", 0x00, 0x01, false, 0x98, false, false},
		{"10-1 C=1", 0x10, 0x01, true, 0x09, true, false},
		{"10-1 C=0", 0x10, 0x01, false, 0x08, true, false},
		{"00-0 C=1", 0x00, 0x00, true, 0x00, true, true},
	}

	// Test only variants that support decimal mode
	variants := []CPUVariant{VariantNMOS6502, VariantCMOS65C02}

	for _, variant := range variants {
		for _, tt := range tests {
			t.Run(variant.String()+" "+tt.name, func(t *testing.T) {
				cpu, bus := setupCPUWithVariant(variant)

				cpu.A = tt.a
				cpu.setFlag(D, true)
				cpu.setFlag(C, tt.carryIn)

				program := []uint8{0xE9, tt.operand, 0x00} // SBC #operand, BRK
				baseAddr := uint16(0x8000)
				bus.load(baseAddr, program)
				cpu.PC = baseAddr
				cpu.SetCycles(0)

				runCycles(cpu, 2)

				if cpu.A != tt.expectedA {
					t.Errorf("Expected A=$%02X, got A=$%02X", tt.expectedA, cpu.A)
				}
				if cpu.getFlag(C) != tt.expectedC {
					t.Errorf("Expected C=%v, got C=%v", tt.expectedC, cpu.getFlag(C))
				}
				if cpu.getFlag(Z) != tt.expectedZ {
					t.Errorf("Expected Z=%v, got Z=%v", tt.expectedZ, cpu.getFlag(Z))
				}
			})
		}
	}
}

// TestVariantNVFlagsInDecimalMode tests N/V flag behavior in decimal mode
func TestVariantNVFlagsInDecimalMode(t *testing.T) {
	// Both NMOS and CMOS set N/V based on binary intermediate result
	variants := []CPUVariant{VariantNMOS6502, VariantCMOS65C02}

	for _, variant := range variants {
		t.Run(variant.String()+" ADC N/V flags", func(t *testing.T) {
			cpu, bus := setupCPUWithVariant(variant)

			// Test: 0x70 + 0x10 = 0x80 (BCD)
			// Binary: 0x70 + 0x10 = 0x80 -> N=1, V=1 (pos+pos=neg overflow)
			cpu.A = 0x70
			cpu.setFlag(D, true)
			cpu.setFlag(C, false)

			program := []uint8{0x69, 0x10, 0x00} // ADC #$10, BRK
			baseAddr := uint16(0x8000)
			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.SetCycles(0)

			runCycles(cpu, 2)

			if cpu.A != 0x80 {
				t.Errorf("Expected A=$80, got A=$%02X", cpu.A)
			}
			if !cpu.getFlag(N) {
				t.Error("Expected N=1 (based on binary intermediate)")
			}
			if !cpu.getFlag(V) {
				t.Error("Expected V=1 (based on binary intermediate)")
			}
		})

		t.Run(variant.String()+" SBC N/V flags", func(t *testing.T) {
			cpu, bus := setupCPUWithVariant(variant)

			// Test: 0x50 - 0x50 = 0x00 (BCD)
			// Binary: 0x50 + 0xAF + 1 = 0x100 -> N=0, V=0
			cpu.A = 0x50
			cpu.setFlag(D, true)
			cpu.setFlag(C, true)

			program := []uint8{0xE9, 0x50, 0x00} // SBC #$50, BRK
			baseAddr := uint16(0x8000)
			bus.load(baseAddr, program)
			cpu.PC = baseAddr
			cpu.SetCycles(0)

			runCycles(cpu, 2)

			if cpu.A != 0x00 {
				t.Errorf("Expected A=$00, got A=$%02X", cpu.A)
			}
			if cpu.getFlag(N) {
				t.Error("Expected N=0 (based on binary intermediate)")
			}
			if cpu.getFlag(V) {
				t.Error("Expected V=0 (based on binary intermediate)")
			}
		})
	}
}
