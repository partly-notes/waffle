# Logging Package

This package provides structured logging capabilities for Waffle with support for multiple log levels, file and console output, and contextual error handling with troubleshooting guidance.

## Features

- **Structured Logging**: Uses Go's `log/slog` for structured, leveled logging
- **Multiple Outputs**: Supports both file and console logging simultaneously
- **Log Levels**: DEBUG, INFO, WARNING, ERROR
- **File Rotation**: Logs are stored in `~/.waffle/logs/` with daily rotation
- **Error Context**: Rich error types with troubleshooting guidance
- **Context Integration**: Correlation IDs, session IDs, and workload IDs
- **JSON Support**: Optional JSON formatting for machine-readable logs

## Quick Start

### Initialize Global Logger

```go
import "github.com/waffle/waffle/internal/logging"

func main() {
    // Initialize with default config
    if err := logging.InitGlobalLogger(nil); err != nil {
        log.Fatal(err)
    }
    defer logging.CloseGlobalLogger()

    // Use the global logger
    logger := logging.GetLogger()
    logger.Info("application started")
}
```

### Custom Configuration

```go
config := &logging.Config{
    Level:      logging.LevelDebug,
    LogDir:     "/custom/log/path",
    MaxSizeMB:  100,
    EnableFile: true,
    EnableJSON: false,
}

if err := logging.InitGlobalLogger(config); err != nil {
    log.Fatal(err)
}
```

## Usage Examples

### Basic Logging

```go
logger := logging.GetLogger()

// Simple messages
logger.Debug("debug message")
logger.Info("info message")
logger.Warn("warning message")
logger.Error("error message")

// With structured attributes
logger.Info("user logged in",
    "user_id", "12345",
    "ip_address", "192.168.1.1",
)
```

### Context-Aware Logging

```go
ctx := context.Background()
logger := logging.GetLogger()

// Add context values
ctx = logging.WithCorrelationID(ctx, "req-123")
ctx = logging.WithSessionID(ctx, "session-456")
ctx = logging.WithWorkloadID(ctx, "workload-789")

// Log with context
logger.InfoContext(ctx, "processing request",
    "operation", "create_workload",
)

// Enrich logger with context values
enrichedLogger := logging.EnrichContext(ctx, logger)
enrichedLogger.Info("this log includes correlation_id, session_id, and workload_id")
```

### Error Handling with Troubleshooting

```go
// Create error with context and troubleshooting guidance
err := logging.NewErrorWithContext(
    "parse terraform file",
    baseErr,
    logging.TroubleshootingTerraformParsing,
)

// Add additional context
err.WithContext("file", "main.tf")
err.WithContext("line", 42)

// Log the error
err.LogError(ctx, logger)

// Return the error (includes troubleshooting in error message)
return err
```

### Wrap and Log Errors

```go
// Wrap an error with troubleshooting guidance
if err := someOperation(); err != nil {
    return logging.WrapError(
        "create AWS workload",
        err,
        logging.TroubleshootingAWSCredentials,
    )
}

// Log and wrap in one step
if err := someOperation(); err != nil {
    return logging.LogAndWrapError(
        ctx,
        logger,
        "parse terraform plan",
        err,
        logging.TroubleshootingTerraformPlan,
    )
}
```

### Logger with Attributes

```go
// Create logger with persistent attributes
componentLogger := logger.With(
    "component", "iac_analyzer",
    "version", "1.0.0",
)

componentLogger.Info("analyzing files")
// Output: ... component=iac_analyzer version=1.0.0 msg="analyzing files"
```

### Logger with Groups

```go
// Create logger with grouped attributes
groupedLogger := logger.WithGroup("request")

groupedLogger.Info("processing",
    "method", "POST",
    "path", "/api/review",
)
// Output: ... request.method=POST request.path=/api/review msg="processing"
```

## Available Troubleshooting Messages

The package provides pre-defined troubleshooting messages for common error scenarios:

- `TroubleshootingDirectoryAccess` - Directory access issues
- `TroubleshootingTerraformParsing` - Terraform syntax errors
- `TroubleshootingTerraformPlan` - Terraform plan file issues
- `TroubleshootingAWSCredentials` - AWS credential problems
- `TroubleshootingBedrockAccess` - Bedrock model access issues
- `TroubleshootingWAFRAccess` - Well-Architected Tool access problems
- `TroubleshootingRateLimit` - API rate limiting
- `TroubleshootingSessionNotFound` - Session management issues
- `TroubleshootingDiskSpace` - Disk space problems

## Log File Management

### Location

Logs are stored in `~/.waffle/logs/` by default.

### File Naming

Log files are named with the format: `waffle-YYYY-MM-DD.log`

### Rotation

A new log file is created each day. Old log files are not automatically deleted.

### Manual Cleanup

```bash
# Remove logs older than 30 days
find ~/.waffle/logs/ -name "waffle-*.log" -mtime +30 -delete
```

## Configuration Options

### LogLevel

- `LevelDebug` - Most verbose, includes source file locations
- `LevelInfo` - Standard operational messages
- `LevelWarning` - Warning messages
- `LevelError` - Error messages only

### EnableFile

When `true`, logs are written to files in `LogDir`. Default: `true`

### EnableJSON

When `true`, logs are formatted as JSON. Useful for log aggregation systems. Default: `false`

### MaxSizeMB

Maximum size of a single log file in megabytes. Default: `100`

## Best Practices

1. **Initialize Early**: Initialize the global logger at application startup
2. **Use Context**: Pass context through your application for correlation
3. **Structured Attributes**: Use key-value pairs instead of string formatting
4. **Error Wrapping**: Always wrap errors with context and troubleshooting
5. **Log Levels**: Use appropriate log levels (DEBUG for development, INFO for production)
6. **Cleanup**: Close the logger on application shutdown
7. **Sensitive Data**: Never log credentials, keys, or PII

## Example: Complete Application Setup

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/waffle/waffle/internal/logging"
)

func main() {
    // Initialize logger
    config := &logging.Config{
        Level:      logging.LevelInfo,
        EnableFile: true,
        EnableJSON: false,
    }

    if err := logging.InitGlobalLogger(config); err != nil {
        log.Fatalf("failed to initialize logger: %v", err)
    }
    defer logging.CloseGlobalLogger()

    logger := logging.GetLogger()
    logger.Info("application started")

    // Setup context with correlation ID
    ctx := context.Background()
    ctx = logging.WithCorrelationID(ctx, "app-startup")

    // Run application
    if err := run(ctx); err != nil {
        logger.ErrorContext(ctx, "application failed", "error", err)
        os.Exit(1)
    }

    logger.Info("application stopped")
}

func run(ctx context.Context) error {
    logger := logging.FromContext(ctx)
    
    // Your application logic here
    logger.InfoContext(ctx, "running application")
    
    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    logger.InfoContext(ctx, "shutdown signal received")
    
    return nil
}
```

## Testing

The logging package includes comprehensive tests. Run them with:

```bash
go test ./internal/logging/... -v
```

## Performance Considerations

- File I/O is buffered for performance
- Multiple handlers (file + console) have minimal overhead
- JSON formatting is slightly slower than text formatting
- Debug level logging includes source file information (slower)

## Thread Safety

All logger operations are thread-safe. The underlying `slog` package handles concurrent access.
