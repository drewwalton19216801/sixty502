# Errors Module (errors.go)

## Purpose

Centralize all error handling types, error definitions, and error handler implementations. This provides a clean separation of error handling concerns from core CPU logic.

## Contents

### 1. Error Types (Lines 33-40)

```go
type ErrorType uint8

const (
    ErrorIllegalOpcode ErrorType = iota
    ErrorInvalidState
    ErrorBusError
)
```

### 2. CPU Error Structure (Lines 42-52)

```go
type CPUError struct {
    Type    ErrorType
    Opcode  uint8
    PC      uint16
    Message string
}

func (e *CPUError) Error() string
```

### 3. Error Handler Interface (Lines 54-57)

```go
type ErrorHandler interface {
    HandleError(err *CPUError) error
}
```

### 4. Strict Error Handler (Lines 59-64)

```go
type StrictErrorHandler struct{}

func (h *StrictErrorHandler) HandleError(err *CPUError) error
```

### 5. Logging Error Handler (Lines 66-76)

```go
type LoggingErrorHandler struct {
    Logger *log.Logger
}

func (h *LoggingErrorHandler) HandleError(err *CPUError) error
```

## Dependencies

- `fmt` package (for error formatting)
- `log` package (for LoggingErrorHandler)

## Exports

- `ErrorType` - Error category enumeration
- `ErrorIllegalOpcode`, `ErrorInvalidState`, `ErrorBusError` - Error constants
- `CPUError` - Main error structure
- `ErrorHandler` - Interface for custom error handling
- `StrictErrorHandler` - Halts on any error
- `LoggingErrorHandler` - Logs errors but continues

## File Size Estimate

~80-100 lines (including documentation)

## Migration Notes

### Step 1: Create File

```go
package cpu6502

import (
    "fmt"
    "log"
)

// Error handling for the 6502 CPU emulator
```

### Step 2: Copy Error Definitions

- Copy ErrorType and constants
- Copy CPUError struct and Error() method
- Copy ErrorHandler interface
- Copy both handler implementations

### Step 3: Update cpu.go

- Remove error definitions
- Keep error handling logic in CPU methods
- Verify error references still work

### Step 4: Verify

- Run `go build`
- Run error-related tests
- Verify error messages format correctly

## Testing Impact

- Error-related tests may need import updates
- Test behavior remains unchanged
- Error handler tests can be more focused

## Usage Examples

### Creating Custom Error Handler

```go
type CustomErrorHandler struct {
    errorCount int
}

func (h *CustomErrorHandler) HandleError(err *cpu6502.CPUError) error {
    h.errorCount++
    if h.errorCount > 10 {
        return err // Halt after 10 errors
    }
    log.Printf("Error #%d: %v", h.errorCount, err)
    return nil // Continue
}
```

### Using Error Handlers

```go
// Strict mode - halt on first error
cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&cpu6502.StrictErrorHandler{}).
    Build()

// Logging mode - log and continue
cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&cpu6502.LoggingErrorHandler{
        Logger: log.Default(),
    }).
    Build()
```

## Documentation Requirements

### Package Documentation

```go
// Package errors provides error handling types and handlers for the
// 6502 CPU emulator. It defines error categories, error structures,
// and built-in error handlers for different execution modes.
```

### Type Documentation

- Document each ErrorType constant
- Explain when each error type occurs
- Provide examples of error handling strategies

## Benefits

1. **Centralized Error Handling**: All error types in one place
2. **Easy Customization**: Simple to create custom error handlers
3. **Clear Error Categories**: Well-defined error types
4. **Flexible Strategies**: Choose between strict and lenient modes
5. **Better Testing**: Error handling can be tested independently

## Future Enhancements

- Add more error types (e.g., ErrorStackOverflow, ErrorStackUnderflow)
- Add error statistics tracking
- Add error recovery strategies
- Add error logging levels
