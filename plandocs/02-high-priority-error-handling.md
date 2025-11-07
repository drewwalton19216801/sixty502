# High Priority: Add Error Handling to Clock()

## Problem Statement

The current [`Clock()`](../cpu.go:1138) method has no error handling:

- Illegal opcodes only log errors via [`XXX()`](../cpu.go:767)
- No way for calling code to detect execution errors
- Silent failures make debugging difficult
- No mechanism to halt on invalid states

## Proposed Solution

Add error returns to critical methods and provide configurable error handling behavior.

### Implementation Steps

#### Step 1: Define Error Types

```go
// CPUError represents an error during CPU execution
type CPUError struct {
    Type    ErrorType
    Opcode  uint8
    PC      uint16
    Message string
}

func (e *CPUError) Error() string {
    return fmt.Sprintf("CPU error at $%04X: %s (opcode $%02X)", e.PC, e.Message, e.Opcode)
}

// ErrorType categorizes CPU errors
type ErrorType uint8

const (
    ErrorIllegalOpcode ErrorType = iota
    ErrorInvalidState
    ErrorBusError
)
```

#### Step 2: Add Error Handling Configuration

```go
// ErrorHandler defines how the CPU handles errors
type ErrorHandler interface {
    HandleError(err *CPUError) error
}

// StrictErrorHandler halts execution on any error
type StrictErrorHandler struct{}

func (h *StrictErrorHandler) HandleError(err *CPUError) error {
    return err
}

// LoggingErrorHandler logs errors but continues execution
type LoggingErrorHandler struct {
    Logger *log.Logger
}

func (h *LoggingErrorHandler) HandleError(err *CPUError) error {
    if h.Logger != nil {
        h.Logger.Printf("CPU error: %v", err)
    }
    return nil // Continue execution
}
```

#### Step 3: Update CPU Struct

```go
type CPU struct {
    // ... existing fields
    
    // NEW: Error handling
    errorHandler ErrorHandler
    lastError    *CPUError
}
```

#### Step 4: Update Clock() Method

```go
// Clock executes one clock cycle of the CPU
// Returns an error if an unrecoverable error occurs
func (c *CPU) Clock() error {
    if c.Cycles == 0 {
        c.opcode = c.read(c.PC)
        c.PC++

        c.setFlag(U, true)
        c.currentInstruction = &c.lookup[c.opcode]

        // Check for illegal opcodes
        if c.currentInstruction.Illegal {
            err := &CPUError{
                Type:    ErrorIllegalOpcode,
                Opcode:  c.opcode,
                PC:      c.PC - 1,
                Message: fmt.Sprintf("illegal opcode $%02X", c.opcode),
            }
            c.lastError = err
            
            if handlerErr := c.errorHandler.HandleError(err); handlerErr != nil {
                return handlerErr
            }
        }

        baseCycles := c.currentInstruction.Cycles
        addrModeCycles := c.currentInstruction.AddrMode(c)
        opCycles := c.currentInstruction.Operate(c)
        c.Cycles = baseCycles + addrModeCycles + opCycles
        c.setFlag(U, true)
    }

    c.Cycles--
    c.totalCycles++
    return nil
}
```

## Usage Examples

### Strict Mode (Halt on Error)

```go
cpu := NewCPUWithErrorHandler(bus, &StrictErrorHandler{})
for {
    if err := cpu.Clock(); err != nil {
        fmt.Printf("CPU halted: %v\n", err)
        break
    }
}
```

### Logging Mode (Continue on Error)

```go
logger := log.New(os.Stderr, "CPU: ", log.LstdFlags)
cpu := NewCPUWithErrorHandler(bus, &LoggingErrorHandler{Logger: logger})
for i := 0; i < 1000; i++ {
    cpu.Clock() // Errors logged but execution continues
}
```

## Testing Strategy

1. Test illegal opcode detection
2. Test error handler invocation
3. Test strict vs logging modes
4. Verify backward compatibility

## Success Criteria

- [ ] Error types defined
- [ ] Error handlers implemented
- [ ] Clock() returns errors
- [ ] All tests pass
- [ ] Backward compatible (default logging behavior)
