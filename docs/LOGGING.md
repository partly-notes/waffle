# Waffle Logging System

## Overview

Waffle uses a comprehensive structured logging system built on Go's `log/slog` package. The logging system provides:

- **Structured Logging**: Key-value pairs for easy parsing and analysis
- **Multiple Log Levels**: DEBUG, INFO, WARNING, ERROR
- **Multiple Outputs**: Console (stderr) and file logging simultaneously
- **File Management**: Daily log files stored in `.waffle/logs/` (current directory by default)
- **Error Context**: Rich error types with troubleshooting guidance
- **Context Integration**: Correlation IDs, session IDs, and workload IDs

## Quick Start

### Configuration

By default, Waffle writes logs to `.waffle/logs/` in the **current working directory** where you run the command. This keeps logs with your IaC project.

To use a global log directory instead, create a configuration file at `~/.waffle/config.yaml`:

```yaml
storage:
  # Use global log directory (optional)
  log_dir: ~/.waffle/logs
  
  # Sessions are global by default
  session_dir: ~/.waffle/sessions
  retention_days: 90

logging:
  level: INFO
  format: text
```

**Note**: 
- Without a config file, logs are written to `.waffle/logs/` in your current directory
- Sessions are stored globally in `~/.waffle/sessions/` by default (shared across projects)
- Logs are per-project by default (keeps logs with your IaC)

### Environment Variable

Set the log level using the `WAFFLE_LOG_LEVEL` environment variable:

```bash
export WAFFLE_LOG_LEVEL=DEBUG
waffle review --workload-id my-app
```

Available levels: `DEBUG`, `INFO`, `WARNING`, `ERROR` (default: `INFO`)

### Log Files

Logs are automatically written to:
- **Default Location**: `.waffle/logs/` (in your current working directory)
- **Format**: `waffle-YYYY-MM-DD.log`
- **Rotation**: New file created daily
- **Permissions**: `0644` (readable by owner and group)

**Example**: If you run `waffle review` from `/home/user/my-terraform-project/`, logs will be in `/home/user/my-terraform-project/.waffle/logs/`

**Note**: You can configure a global log directory (like `~/.waffle/logs/`) in `~/.waffle/config.yaml` under `storage.log_dir`.

### Viewing Logs

```bash
# View today's log (from your project directory)
tail -f .waffle/logs/waffle-$(date +%Y-%m-%d).log

# View all logs
cat .waffle/logs/waffle-*.log

# Search for errors
grep "level=ERROR" .waffle/logs/waffle-*.log

# If using global log directory (via config file)
tail -f ~/.waffle/logs/waffle-$(date +%Y-%m-%d).log
```

## Log Format

### Text Format (Default)

```
time=2025-11-26T14:26:02.174+01:00 level=INFO msg="waffle started" version=dev commit=none date=unknown
```

### JSON Format (Optional)

```json
{"time":"2025-11-26T14:26:02.174+01:00","level":"INFO","msg":"waffle started","version":"dev","commit":"none","date":"unknown"}
```

## Log Levels

### DEBUG
- Most verbose level
- Includes source file locations
- Use for development and troubleshooting
- Example: `WAFFLE_LOG_LEVEL=DEBUG waffle review --workload-id my-app`

### INFO (Default)
- Standard operational messages
- Progress updates
- Successful operations
- Recommended for production use

### WARNING
- Non-critical issues
- Degraded functionality
- Retryable errors
- Operations that succeeded with caveats

### ERROR
- Critical failures
- Operations that failed
- Includes troubleshooting guidance
- Requires user attention

## Error Messages with Troubleshooting

When errors occur, Waffle provides detailed troubleshooting guidance:

### Example Error Output

```
Error: parse terraform file failed: syntax error at line 42

Troubleshooting:
Possible causes:
1. Invalid Terraform HCL syntax
2. Unsupported Terraform version
3. Missing required Terraform files

Solutions:
- Validate Terraform syntax: terraform validate
- Check Terraform version: terraform version
- Ensure all .tf files are present in the directory
- Review the Terraform documentation for syntax errors
```

### Common Error Categories

1. **Directory Access Errors**
   - Invalid directory paths
   - Permission issues
   - Missing IaC files

2. **Terraform Parsing Errors**
   - Syntax errors in HCL files
   - Invalid Terraform plan files
   - Unsupported Terraform versions

3. **AWS Credential Errors**
   - Missing or expired credentials
   - Insufficient IAM permissions
   - Invalid AWS profiles

4. **Bedrock Access Errors**
   - Model not enabled in region
   - Missing model access permissions
   - API throttling

5. **WAFR API Errors**
   - Workload not found
   - Insufficient permissions
   - Rate limiting

## Log File Management

### Automatic Cleanup

Waffle does not automatically delete old log files. To clean up old logs:

```bash
# Remove logs older than 30 days (from current directory)
find .waffle/logs/ -name "waffle-*.log" -mtime +30 -delete

# Remove logs older than 7 days
find .waffle/logs/ -name "waffle-*.log" -mtime +7 -delete

# If using global log directory
find ~/.waffle/logs/ -name "waffle-*.log" -mtime +30 -delete
```

### Disk Space Monitoring

Check log directory size:

```bash
# Current directory logs
du -sh .waffle/logs/

# Global logs (if configured)
du -sh ~/.waffle/logs/
```

### Log Rotation

Logs are rotated daily. Each day creates a new file with the format `waffle-YYYY-MM-DD.log`.

## Integration with Application

### Context Values

Waffle automatically includes context values in logs:

- **correlation_id**: Unique identifier for request tracing
- **session_id**: Review session identifier
- **workload_id**: AWS workload identifier

Example log entry:
```
time=2025-11-26T14:26:02.174+01:00 level=INFO msg="evaluating question" correlation_id=req-123 session_id=abc-456 workload_id=my-app question_id=sec_data_1
```

### Progress Tracking

During review execution, Waffle logs progress:

```
level=INFO msg="step 1: analyzing IaC"
level=INFO msg="step 2: retrieving WAFR questions"
level=INFO msg="step 3: evaluating questions"
level=INFO msg="evaluating question" question_id=sec_data_1 progress="1/50"
level=INFO msg="step 4: submitting answers to AWS"
level=INFO msg="step 5: retrieving improvement plan"
level=INFO msg="step 6: creating milestone"
level=INFO msg="review execution completed" questions_evaluated=50 risks_identified=5
```

## Troubleshooting

### No Log Files Created

**Problem**: Log files are not being created

**Solutions**:
1. Check if logs are in your current directory: `ls -la .waffle/logs/`
2. Verify you're in the right directory where you ran waffle
3. Check disk space: `df -h .`
4. Check directory permissions: `ls -ld .waffle/logs/`
5. Check for errors in console output (stderr)
6. Try running with DEBUG level: `WAFFLE_LOG_LEVEL=DEBUG waffle review --workload-id test`
7. If using a config file, verify: `cat ~/.waffle/config.yaml` (check `storage.log_dir`)

### Log Files Too Large

**Problem**: Log files are consuming too much disk space

**Solutions**:
1. Reduce log level to INFO or WARNING
2. Clean up old log files (see Automatic Cleanup above)
3. Consider using log aggregation tools

### Cannot Read Log Files

**Problem**: Permission denied when reading log files

**Solutions**:
1. Check file permissions: `ls -l ~/.waffle/logs/waffle-*.log`
2. Ensure you're the owner: `stat ~/.waffle/logs/waffle-*.log`
3. Fix permissions if needed: `chmod 644 ~/.waffle/logs/waffle-*.log`

## Best Practices

1. **Use INFO level in production**: Provides good balance of detail and performance
2. **Use DEBUG level for troubleshooting**: Includes source locations and detailed traces
3. **Monitor log file sizes**: Set up alerts for large log directories
4. **Clean up old logs regularly**: Implement automated cleanup scripts
5. **Include context in operations**: Use correlation IDs for request tracing
6. **Review error logs**: Check for patterns in ERROR level logs
7. **Archive important logs**: Save logs before cleanup for compliance

## Advanced Usage

### Programmatic Access

For developers integrating with Waffle:

```go
import "github.com/waffle/waffle/internal/logging"

// Initialize logger
config := logging.DefaultConfig()
config.Level = logging.LevelDebug
if err := logging.InitGlobalLogger(config); err != nil {
    log.Fatal(err)
}
defer logging.CloseGlobalLogger()

// Get logger
logger := logging.GetLogger()

// Log with context
ctx := context.Background()
ctx = logging.WithCorrelationID(ctx, "req-123")
logger.InfoContext(ctx, "operation started")

// Handle errors with troubleshooting
if err := someOperation(); err != nil {
    return logging.LogAndWrapError(
        ctx,
        logger,
        "operation name",
        err,
        logging.TroubleshootingAWSCredentials,
    )
}
```

### Custom Log Directory

To use a custom log directory, set it before running Waffle:

```bash
# Not currently supported via CLI
# Requires code modification to Config
```

### JSON Output for Log Aggregation

For integration with log aggregation systems (ELK, Splunk, etc.), JSON format is recommended:

```go
config := logging.DefaultConfig()
config.EnableJSON = true
```

## Performance Considerations

- **File I/O**: Logs are buffered for performance
- **Multiple Handlers**: File and console logging have minimal overhead
- **JSON Formatting**: Slightly slower than text format
- **Debug Level**: Includes source file information (slower)

## Security Considerations

- **Sensitive Data**: Never logged (credentials, keys, PII are redacted)
- **File Permissions**: Log files are created with `0644` permissions
- **Directory Permissions**: Log directory is created with `0755` permissions
- **Audit Trail**: All operations are logged for compliance

## Support

For issues with logging:

1. Check this documentation
2. Review error messages and troubleshooting guidance
3. Enable DEBUG logging for detailed traces
4. Check GitHub issues: https://github.com/waffle/waffle/issues
5. Contact support with log excerpts (redact sensitive information)
