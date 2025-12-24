package cpu6502

import "log"

// CPUConfig holds configuration options for CPU creation.
//
// This structure provides fine-grained control over CPU behavior,
// including variant selection, error handling, and performance options.
//
// # Configuration Options
//
// Variant: Selects the CPU variant (NMOS 6502, CMOS 65C02, Ricoh 2A03)
//   - Affects instruction behavior and bug emulation
//   - Default: VariantNMOS6502
//
// ErrorHandler: Defines how errors are handled during execution
//   - LoggingErrorHandler: Logs errors but continues execution
//   - StrictErrorHandler: Halts execution on any error
//   - Custom handlers can be implemented
//
// StrictMode: Halts execution on illegal opcodes
//   - Overrides ErrorHandler for illegal opcode errors
//   - Useful for debugging or strict compatibility testing
//
// EnableDecimalMode: Controls decimal mode support
//   - Some variants (Ricoh 2A03) don't support decimal mode
//   - Can be disabled for performance or compatibility
//
// Example usage:
//
//	config := cpu6502.DefaultConfig()
//	config.Variant = cpu6502.VariantCMOS65C02
//	config.StrictMode = true
//	cpu := cpu6502.NewCPUWithConfig(bus, config)
type CPUConfig struct {
	// Variant specifies the CPU variant (NMOS, CMOS, Ricoh)
	Variant CPUVariant

	// ErrorHandler defines how errors are handled
	ErrorHandler ErrorHandler

	// StrictMode halts execution on illegal opcodes
	StrictMode bool

	// EnableDecimalMode allows disabling decimal mode even on variants that support it
	EnableDecimalMode bool
}

// DefaultConfig returns a configuration with sensible defaults.
//
// Default configuration:
//   - Variant: NMOS 6502 (original)
//   - ErrorHandler: LoggingErrorHandler (logs to default logger)
//   - StrictMode: false (continues on errors)
//   - EnableDecimalMode: true
//
// This configuration is suitable for most emulation scenarios and
// provides a good balance between accuracy and performance.
func DefaultConfig() CPUConfig {
	return CPUConfig{
		Variant:           VariantNMOS6502,
		ErrorHandler:      &LoggingErrorHandler{Logger: log.Default()},
		StrictMode:        false,
		EnableDecimalMode: true,
	}
}

// CPUBuilder provides a fluent interface for CPU configuration.
//
// The builder pattern allows for readable, chainable configuration:
//
//	cpu := cpu6502.NewBuilder(bus).
//	    WithVariant(cpu6502.VariantCMOS65C02).
//	    WithStrictMode().
//	    DisableDecimalMode().
//	    Build()
//
// This is more convenient than manually creating a CPUConfig struct
// when you only need to change a few settings from the defaults.
type CPUBuilder struct {
	bus    Bus
	config CPUConfig
}

// NewBuilder creates a new CPU builder with default configuration.
//
// Parameters:
//   - bus: The memory bus interface for the CPU
//
// Returns a builder that can be configured using method chaining.
//
// Example:
//
//	builder := cpu6502.NewBuilder(bus)
//	cpu := builder.WithVariant(cpu6502.VariantCMOS65C02).Build()
func NewBuilder(bus Bus) *CPUBuilder {
	return &CPUBuilder{
		bus:    bus,
		config: DefaultConfig(),
	}
}

// WithVariant sets the CPU variant.
//
// Parameters:
//   - variant: The CPU variant to emulate
//
// Returns the builder for method chaining.
//
// Example:
//
//	builder.WithVariant(cpu6502.VariantCMOS65C02)
func (b *CPUBuilder) WithVariant(variant CPUVariant) *CPUBuilder {
	b.config.Variant = variant
	return b
}

// WithStrictMode enables strict mode.
//
// In strict mode, the CPU halts execution when encountering illegal
// opcodes instead of continuing with default behavior.
//
// Returns the builder for method chaining.
//
// Example:
//
//	builder.WithStrictMode()
func (b *CPUBuilder) WithStrictMode() *CPUBuilder {
	b.config.StrictMode = true
	return b
}

// WithErrorHandler sets a custom error handler.
//
// Parameters:
//   - handler: The error handler to use
//
// Returns the builder for method chaining.
//
// Example:
//
//	handler := &MyCustomErrorHandler{}
//	builder.WithErrorHandler(handler)
func (b *CPUBuilder) WithErrorHandler(handler ErrorHandler) *CPUBuilder {
	b.config.ErrorHandler = handler
	return b
}

// DisableDecimalMode disables decimal mode.
//
// This is useful for emulating systems that don't support decimal mode
// (like the Ricoh 2A03 in the NES) or for slight performance improvements.
//
// Returns the builder for method chaining.
//
// Example:
//
//	builder.DisableDecimalMode()
func (b *CPUBuilder) DisableDecimalMode() *CPUBuilder {
	b.config.EnableDecimalMode = false
	return b
}

// Build creates the configured CPU.
//
// Returns a new CPU instance with the configured settings.
//
// Example:
//
//	cpu := cpu6502.NewBuilder(bus).
//	    WithVariant(cpu6502.VariantCMOS65C02).
//	    WithStrictMode().
//	    Build()
func (b *CPUBuilder) Build() *CPU {
	return NewCPUWithConfig(b.bus, b.config)
}

// NewCPU creates a new 6502 CPU instance with default configuration.
//
// The CPU is initialized with:
//   - All registers cleared
//   - Stack pointer at $FD
//   - Status flags: U and I set
//   - NMOS 6502 variant
//   - Logging error handler
//
// The CPU must be reset before execution:
//
//	cpu := NewCPU(bus)
//	cpu.Reset() // Loads PC from reset vector at $FFFC/FD
//
// For custom configuration, use NewCPUWithConfig or the builder pattern.
//
// Parameters:
//   - bus: The memory bus interface
//
// Returns a new CPU instance ready to be reset and executed.
func NewCPU(bus Bus) *CPU {
	return NewCPUWithConfig(bus, DefaultConfig())
}

// NewCPUWithVariant creates a new CPU with specified variant.
//
// This is a convenience function for the common case of only needing
// to change the CPU variant from the default.
//
// Parameters:
//   - bus: The memory bus interface
//   - variant: The CPU variant to emulate
//
// Returns a new CPU instance with the specified variant.
//
// Example:
//
//	cpu := cpu6502.NewCPUWithVariant(bus, cpu6502.VariantCMOS65C02)
func NewCPUWithVariant(bus Bus, variant CPUVariant) *CPU {
	config := DefaultConfig()
	config.Variant = variant
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithErrorHandler creates a new CPU with custom error handler.
//
// This is a convenience function for the common case of only needing
// to change the error handler from the default.
//
// Parameters:
//   - bus: The memory bus interface
//   - handler: The error handler to use
//
// Returns a new CPU instance with the specified error handler.
//
// Example:
//
//	handler := &MyCustomErrorHandler{}
//	cpu := cpu6502.NewCPUWithErrorHandler(bus, handler)
func NewCPUWithErrorHandler(bus Bus, handler ErrorHandler) *CPU {
	config := DefaultConfig()
	config.ErrorHandler = handler
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithVariantAndErrorHandler creates a new CPU with specified variant and error handler.
//
// This is a convenience function for the common case of needing to change
// both the variant and error handler.
//
// Parameters:
//   - bus: The memory bus interface
//   - variant: The CPU variant to emulate
//   - handler: The error handler to use
//
// Returns a new CPU instance with the specified configuration.
//
// Example:
//
//	handler := &MyCustomErrorHandler{}
//	cpu := cpu6502.NewCPUWithVariantAndErrorHandler(bus,
//	    cpu6502.VariantCMOS65C02, handler)
func NewCPUWithVariantAndErrorHandler(bus Bus, variant CPUVariant, handler ErrorHandler) *CPU {
	config := DefaultConfig()
	config.Variant = variant
	config.ErrorHandler = handler
	return NewCPUWithConfig(bus, config)
}

// NewCPUWithConfig creates a new CPU with full configuration.
//
// This function provides complete control over CPU initialization.
// It applies the configuration and initializes the CPU's internal state.
//
// Parameters:
//   - bus: The memory bus interface
//   - config: The configuration to apply
//
// Returns a new CPU instance with the specified configuration.
//
// Example:
//
//	config := cpu6502.DefaultConfig()
//	config.Variant = cpu6502.VariantCMOS65C02
//	config.StrictMode = true
//	cpu := cpu6502.NewCPUWithConfig(bus, config)
func NewCPUWithConfig(bus Bus, config CPUConfig) *CPU {
	// Apply strict mode to error handler if requested
	errorHandler := config.ErrorHandler
	if config.StrictMode {
		errorHandler = &StrictErrorHandler{}
	}

	c := &CPU{
		bus:          bus,
		P:            U | I,
		SP:           0xFD,
		variant:      config.Variant,
		errorHandler: errorHandler,
	}

	c.buildLookupTable()

	// Apply decimal mode configuration
	if !config.EnableDecimalMode {
		c.P &^= D // Clear decimal flag
	}

	return c
}
