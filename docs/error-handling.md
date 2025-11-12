# Error Handling Guide

This guide covers error handling strategies and patterns for the sixty502 emulator.

## Error Types

The emulator defines several error types through the [`ErrorType`](errors.go:8) enum:

```go
type ErrorType uint8

const (
    ErrorIllegalOpcode  // Illegal/unofficial opcode encountered
    ErrorInvalidState   // CPU in invalid state
    ErrorBusError       // Bus access error
)
```

## CPUError Structure

All CPU errors are represented by the [`CPUError`](errors.go:20) struct:

```go
type CPUError struct {
    Type    ErrorType  // Category of error
    Opcode  uint8      // Opcode that caused error
    PC      uint16     // Program counter at error
    Message string     // Human-readable description
}
```

**Example**:

```go
if err := cpu.Clock(); err != nil {
    if cpuErr, ok := err.(*cpu6502.CPUError); ok {
        fmt.Printf("Error at $%04X: %s (opcode $%02X)\n",
            cpuErr.PC, cpuErr.Message, cpuErr.Opcode)
    }
}
```

## Error Handlers

Error handlers implement the [`ErrorHandler`](errors.go:34) interface:

```go
type ErrorHandler interface {
    HandleError(err *CPUError) error
}
```

### Built-in Handlers

#### LoggingErrorHandler

Logs errors but continues execution.

```go
type LoggingErrorHandler struct {
    Logger *log.Logger
}
```

**Behavior**:

- Logs error details
- Returns `nil` (continues execution)
- Default handler

**Example**:

```go
handler := &cpu6502.LoggingErrorHandler{
    Logger: log.New(os.Stdout, "CPU: ", log.LstdFlags),
}
cpu := cpu6502.NewCPUWithErrorHandler(bus, handler)
```

#### StrictErrorHandler

Halts execution on any error.

```go
type StrictErrorHandler struct{}
```

**Behavior**:

- Returns error immediately
- Halts execution
- Used in strict mode

**Example**:

```go
handler := &cpu6502.StrictErrorHandler{}
cpu := cpu6502.NewCPUWithErrorHandler(bus, handler)
```

## Custom Error Handlers

### Basic Custom Handler

```go
type MyErrorHandler struct {
    errorCount int
}

func (h *MyErrorHandler) HandleError(err *cpu6502.CPUError) error {
    h.errorCount++
    
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        log.Printf("Illegal opcode $%02X at $%04X", err.Opcode, err.PC)
        return nil // Continue
    case cpu6502.ErrorInvalidState:
        return err // Halt
    default:
        return err // Halt
    }
}
```

### Selective Error Handler

Handle different error types differently:

```go
type SelectiveErrorHandler struct {
    allowIllegalOpcodes bool
    logger              *log.Logger
}

func (h *SelectiveErrorHandler) HandleError(err *cpu6502.CPUError) error {
    // Log all errors
    h.logger.Printf("CPU Error: %v", err)
    
    // Handle based on type
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        if h.allowIllegalOpcodes {
            return nil // Continue
        }
        return err // Halt
        
    case cpu6502.ErrorInvalidState:
        return err // Always halt
        
    case cpu6502.ErrorBusError:
        return err // Always halt
        
    default:
        return err // Halt on unknown errors
    }
}
```

### Error Counting Handler

Track error statistics:

```go
type StatisticsErrorHandler struct {
    illegalOpcodes map[uint8]int
    totalErrors    int
    maxErrors      int
}

func NewStatisticsErrorHandler(maxErrors int) *StatisticsErrorHandler {
    return &StatisticsErrorHandler{
        illegalOpcodes: make(map[uint8]int),
        maxErrors:      maxErrors,
    }
}

func (h *StatisticsErrorHandler) HandleError(err *cpu6502.CPUError) error {
    h.totalErrors++
    
    if err.Type == cpu6502.ErrorIllegalOpcode {
        h.illegalOpcodes[err.Opcode]++
    }
    
    // Halt if too many errors
    if h.totalErrors >= h.maxErrors {
        return fmt.Errorf("too many errors (%d)", h.totalErrors)
    }
    
    return nil // Continue
}

func (h *StatisticsErrorHandler) PrintStatistics() {
    fmt.Printf("Total errors: %d\n", h.totalErrors)
    fmt.Println("Illegal opcodes used:")
    for opcode, count := range h.illegalOpcodes {
        fmt.Printf("  $%02X: %d times\n", opcode, count)
    }
}
```

### Recovery Handler

Attempt to recover from errors:

```go
type RecoveryErrorHandler struct {
    cpu    *cpu6502.CPU
    logger *log.Logger
}

func (h *RecoveryErrorHandler) HandleError(err *cpu6502.CPUError) error {
    h.logger.Printf("Error at $%04X: %s", err.PC, err.Message)
    
    switch err.Type {
    case cpu6502.ErrorIllegalOpcode:
        // Skip illegal opcode and continue
        h.logger.Printf("Skipping illegal opcode $%02X", err.Opcode)
        h.cpu.PC++ // Skip opcode
        return nil
        
    case cpu6502.ErrorInvalidState:
        // Try to reset CPU
        h.logger.Println("Attempting CPU reset")
        h.cpu.Reset()
        return nil
        
    default:
        return err // Can't recover
    }
}
```

## Error Handling Patterns

### Pattern 1: Fail Fast

Halt immediately on any error:

```go
cpu := cpu6502.NewBuilder(bus).
    WithStrictMode().
    Build()

for {
    if err := cpu.Clock(); err != nil {
        log.Fatalf("CPU error: %v", err)
    }
}
```

### Pattern 2: Log and Continue

Log errors but keep running:

```go
cpu := cpu6502.NewCPU(bus) // Uses LoggingErrorHandler

for {
    if err := cpu.Clock(); err != nil {
        // Error was logged, but execution continues
        // This shouldn't happen with LoggingErrorHandler
        log.Printf("Unexpected halt: %v", err)
        break
    }
}
```

### Pattern 3: Conditional Handling

Handle errors based on context:

```go
type ContextualErrorHandler struct {
    debugMode bool
}

func (h *ContextualErrorHandler) HandleError(err *cpu6502.CPUError) error {
    if h.debugMode {
        // In debug mode, halt on all errors
        return err
    }
    
    // In production, only halt on critical errors
    if err.Type == cpu6502.ErrorInvalidState {
        return err
    }
    
    return nil // Continue on non-critical errors
}
```

### Pattern 4: Error Recovery with Retry

```go
func runWithRetry(cpu *cpu6502.CPU, maxRetries int) error {
    retries := 0
    
    for {
        err := cpu.Clock()
        if err == nil {
            continue
        }
        
        cpuErr, ok := err.(*cpu6502.CPUError)
        if !ok {
            return err // Unknown error type
        }
        
        // Try to recover
        if retries < maxRetries {
            retries++
            log.Printf("Error at $%04X, retry %d/%d",
                cpuErr.PC, retries, maxRetries)
            cpu.Reset()
            continue
        }
        
        return fmt.Errorf("max retries exceeded: %w", err)
    }
}
```

### Pattern 5: Error Aggregation

Collect errors for batch processing:

```go
type ErrorCollector struct {
    errors []*cpu6502.CPUError
    maxErrors int
}

func (h *ErrorCollector) HandleError(err *cpu6502.CPUError) error {
    h.errors = append(h.errors, err)
    
    if len(h.errors) >= h.maxErrors {
        return fmt.Errorf("collected %d errors", len(h.errors))
    }
    
    return nil // Continue
}

func (h *ErrorCollector) GetErrors() []*cpu6502.CPUError {
    return h.errors
}
```

## Checking for Errors

### Using LastError

Access the last error without halting:

```go
for i := 0; i < 1000; i++ {
    cpu.Clock()
    
    if lastErr := cpu.LastError(); lastErr != nil {
        fmt.Printf("Error occurred: %v\n", lastErr)
        // Error was handled by error handler
        // Execution may have continued
    }
}
```

### Checking Illegal Opcodes

Proactively check for illegal opcodes:

```go
opcode := bus.Read(cpu.PC)
if cpu.IsIllegalOpcode(opcode) {
    fmt.Printf("Warning: illegal opcode $%02X at $%04X\n",
        opcode, cpu.PC)
}
```

## Error Handling Best Practices

### 1. Choose Appropriate Handler

Match handler to use case:

```go
// Development: Strict mode
devCPU := cpu6502.NewBuilder(bus).
    WithStrictMode().
    Build()

// Production: Lenient mode
prodCPU := cpu6502.NewBuilder(bus).
    WithErrorHandler(&cpu6502.LoggingErrorHandler{}).
    Build()

// Testing: Custom handler
testCPU := cpu6502.NewBuilder(bus).
    WithErrorHandler(&TestErrorHandler{t: t}).
    Build()
```

### 2. Log Error Context

Include useful context in error logs:

```go
func (h *MyErrorHandler) HandleError(err *cpu6502.CPUError) error {
    state := h.cpu.GetStateSnapshot()
    log.Printf("Error: %v\nState: %s\n", err, state)
    return nil
}
```

### 3. Implement Graceful Degradation

```go
type GracefulErrorHandler struct {
    fallbackMode bool
}

func (h *GracefulErrorHandler) HandleError(err *cpu6502.CPUError) error {
    if err.Type == cpu6502.ErrorIllegalOpcode {
        h.fallbackMode = true
        log.Println("Entering fallback mode")
        return nil
    }
    return err
}
```

### 4. Test Error Handling

```go
func TestErrorHandling(t *testing.T) {
    bus := &TestBus{}
    handler := &TestErrorHandler{t: t}
    cpu := cpu6502.NewCPUWithErrorHandler(bus, handler)
    
    // Trigger illegal opcode
    bus.Write(0x8000, 0x02) // Illegal NOP
    bus.Write(0xFFFC, 0x00)
    bus.Write(0xFFFD, 0x80)
    
    cpu.Reset()
    err := cpu.Clock()
    
    if err != nil {
        t.Errorf("Expected error to be handled, got: %v", err)
    }
    
    if !handler.errorHandled {
        t.Error("Error handler was not called")
    }
}
```

### 5. Document Error Behavior

```go
// MyEmulator uses lenient error handling for illegal opcodes
// commonly found in commercial games, but halts on invalid
// CPU state to prevent corruption.
type MyEmulator struct {
    cpu *cpu6502.CPU
}

func NewMyEmulator() *MyEmulator {
    handler := &SelectiveErrorHandler{
        allowIllegalOpcodes: true,
    }
    
    return &MyEmulator{
        cpu: cpu6502.NewCPUWithErrorHandler(bus, handler),
    }
}
```

## Common Error Scenarios

### Illegal Opcodes

**Cause**: Executing undocumented opcodes

**Solutions**:

1. Use lenient error handler
2. Implement illegal opcode support
3. Fix program to avoid illegal opcodes

```go
// Allow illegal opcodes
cpu := cpu6502.NewBuilder(bus).
    WithErrorHandler(&cpu6502.LoggingErrorHandler{}).
    Build()
```

### Invalid State

**Cause**: CPU registers in impossible state

**Solutions**:

1. Reset CPU
2. Restore from known good state
3. Debug program logic

```go
if err := cpu.Clock(); err != nil {
    if cpuErr, ok := err.(*cpu6502.CPUError); ok {
        if cpuErr.Type == cpu6502.ErrorInvalidState {
            log.Println("Resetting CPU")
            cpu.Reset()
        }
    }
}
```

### Bus Errors

**Cause**: Memory access issues

**Solutions**:

1. Verify bus implementation
2. Check address ranges
3. Implement proper memory mapping

```go
type SafeBus struct {
    ram [65536]uint8
}

func (b *SafeBus) Read(addr uint16) uint8 {
    // All addresses valid
    return b.ram[addr]
}

func (b *SafeBus) Write(addr uint16, data uint8) {
    // Check for ROM regions
    if addr >= 0x8000 {
        log.Printf("Warning: write to ROM at $%04X", addr)
        return
    }
    b.ram[addr] = data
}
```

## Debugging Errors

### Enable Detailed Logging

```go
type VerboseErrorHandler struct {
    logger *log.Logger
}

func (h *VerboseErrorHandler) HandleError(err *cpu6502.CPUError) error {
    h.logger.Printf("=== CPU Error ===")
    h.logger.Printf("Type: %v", err.Type)
    h.logger.Printf("Opcode: $%02X", err.Opcode)
    h.logger.Printf("PC: $%04X", err.PC)
    h.logger.Printf("Message: %s", err.Message)
    
    // Get CPU state
    state := cpu.GetStateSnapshot()
    h.logger.Printf("State: %s", state)
    
    // Disassemble around error
    disasm := cpu.Disassemble(err.PC-5, err.PC+5)
    h.logger.Println("Disassembly:")
    for addr, instr := range disasm {
        marker := " "
        if addr == err.PC {
            marker = ">"
        }
        h.logger.Printf("%s $%04X: %s", marker, addr, instr)
    }
    
    return nil
}
```

### Error Breakpoints

```go
type BreakpointErrorHandler struct {
    breakOnError bool
}

func (h *BreakpointErrorHandler) HandleError(err *cpu6502.CPUError) error {
    if h.breakOnError {
        fmt.Println("Error breakpoint hit")
        fmt.Printf("Error: %v\n", err)
        fmt.Print("Continue? (y/n): ")
        
        var response string
        fmt.Scanln(&response)
        
        if response != "y" {
            return err // Halt
        }
    }
    
    return nil // Continue
}
```

## See Also

- [API Reference](api-reference.md) - Error-related API methods
- [Configuration Guide](configuration.md) - Error handler configuration
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
