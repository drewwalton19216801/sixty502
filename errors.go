package cpu6502

import (
	"fmt"
	"log"
)

// ErrorType categorizes CPU errors
type ErrorType uint8

const (
	// ErrorIllegalOpcode indicates an illegal/unofficial opcode was encountered
	ErrorIllegalOpcode ErrorType = iota
	// ErrorInvalidState indicates the CPU is in an invalid state
	ErrorInvalidState
	// ErrorBusError indicates a bus access error occurred
	ErrorBusError
)

// CPUError represents an error during CPU execution
type CPUError struct {
	Type    ErrorType
	Opcode  uint8
	PC      uint16
	Message string
}

// Error implements the error interface for CPUError
func (e *CPUError) Error() string {
	return fmt.Sprintf("CPU error at $%04X: %s (opcode $%02X)", e.PC, e.Message, e.Opcode)
}

// ErrorHandler defines how the CPU handles errors
type ErrorHandler interface {
	HandleError(err *CPUError) error
}

// StrictErrorHandler halts execution on any error
type StrictErrorHandler struct{}

// HandleError returns the error, halting execution
func (h *StrictErrorHandler) HandleError(err *CPUError) error {
	return err
}

// LoggingErrorHandler logs errors but continues execution
type LoggingErrorHandler struct {
	Logger *log.Logger
}

// HandleError logs the error and continues execution
func (h *LoggingErrorHandler) HandleError(err *CPUError) error {
	if h.Logger != nil {
		h.Logger.Printf("CPU error: %v", err)
	}
	return nil // Continue execution
}
