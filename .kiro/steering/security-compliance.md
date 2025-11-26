# Security and Compliance Guidelines for Waffle

## Sensitive Data Handling

### Data Classification

Classify data into categories:

1. **Highly Sensitive**: AWS credentials, private keys, passwords
2. **Sensitive**: API keys, tokens, connection strings
3. **Personal**: Email addresses, IP addresses, names
4. **Public**: Resource names, configurations (non-sensitive)

### Redaction Patterns

```go
type RedactionRule struct {
    Name        string
    Pattern     *regexp.Regexp
    Replacement string
}

var RedactionRules = []RedactionRule{
    {
        Name:        "AWS Access Key",
        Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
        Replacement: "AKIA****************",
    },
    {
        Name:        "AWS Secret Key",
        Pattern:     regexp.MustCompile(`[A-Za-z0-9/+=]{40}`),
        Replacement: "[REDACTED_SECRET]",
    },
    {
        Name:        "Password Field",
        Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?([^"'\s]+)["']?`),
        Replacement: `$1: "[REDACTED]"`,
    },
    {
        Name:        "API Key",
        Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|token)\s*[:=]\s*["']?([^"'\s]+)["']?`),
        Replacement: `$1: "[REDACTED]"`,
    },
    {
        Name:        "Email Address",
        Pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
        Replacement: "[EMAIL]",
    },
    {
        Name:        "Private IP",
        Pattern:     regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2[0-9]|3[0-1])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`),
        Replacement: "[PRIVATE_IP]",
    },
}
```

### Redaction Implementation

```go
type Redactor struct {
    rules []RedactionRule
}

func NewRedactor() *Redactor {
    return &Redactor{
        rules: RedactionRules,
    }
}

func (r *Redactor) Redact(content string) (string, []string) {
    redacted := content
    findings := []string{}
    
    for _, rule := range r.rules {
        if rule.Pattern.MatchString(redacted) {
            findings = append(findings, rule.Name)
            redacted = rule.Pattern.ReplaceAllString(redacted, rule.Replacement)
        }
    }
    
    return redacted, findings
}

func (r *Redactor) RedactIaCFile(file *IaCFile) (*IaCFile, error) {
    redactedContent, findings := r.Redact(file.Content)
    
    if len(findings) > 0 {
        slog.Warn("sensitive data redacted from IaC file",
            "file", file.Path,
            "findings", findings,
        )
    }
    
    return &IaCFile{
        Path:    file.Path,
        Content: redactedContent,
    }, nil
}
```

## Credential Management

### Never Store Credentials in Code

```go
// BAD: Hardcoded credentials
const awsAccessKey = "AKIAIOSFODNN7EXAMPLE"

// GOOD: Load from environment or AWS credential chain
cfg, err := config.LoadDefaultConfig(ctx)
```

### Use AWS Credential Chain

```go
func LoadAWSConfig(ctx context.Context, profile string) (aws.Config, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithRegion("us-east-1"),
    }
    
    if profile != "" {
        opts = append(opts, config.WithSharedConfigProfile(profile))
    }
    
    // Credential chain order:
    // 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
    // 2. Shared credentials file (~/.aws/credentials)
    // 3. IAM role (if running on EC2/ECS/Lambda)
    cfg, err := config.LoadDefaultConfig(ctx, opts...)
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    return cfg, nil
}
```

### Secure Storage for Session Data

```go
func (s *SessionStore) SaveSession(session *ReviewSession) error {
    // Redact sensitive data before saving
    redactedSession := s.redactSession(session)
    
    sessionPath := s.sessionPath(session.SessionID)
    
    // Create file with restricted permissions (owner read/write only)
    f, err := os.OpenFile(sessionPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("failed to create session file: %w", err)
    }
    defer f.Close()
    
    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(redactedSession); err != nil {
        return fmt.Errorf("failed to encode session: %w", err)
    }
    
    return nil
}

func (s *SessionStore) redactSession(session *ReviewSession) *ReviewSession {
    redacted := *session
    
    // Redact IaC content
    if redacted.WorkloadModel != nil {
        for i := range redacted.WorkloadModel.Resources {
            redacted.WorkloadModel.Resources[i].Properties = s.redactProperties(
                redacted.WorkloadModel.Resources[i].Properties,
            )
        }
    }
    
    return &redacted
}
```

## Audit Logging

### Log All Security-Relevant Events

```go
type AuditLogger struct {
    logger *slog.Logger
}

func NewAuditLogger(logPath string) (*AuditLogger, error) {
    f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
    if err != nil {
        return nil, fmt.Errorf("failed to open audit log: %w", err)
    }
    
    logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    
    return &AuditLogger{logger: logger}, nil
}

func (a *AuditLogger) LogReviewInitiated(ctx context.Context, workloadID, repoLocation string, user string) {
    a.logger.InfoContext(ctx, "review_initiated",
        "event_type", "review_initiated",
        "workload_id", workloadID,
        "repo_location", repoLocation,
        "user", user,
        "timestamp", time.Now().UTC(),
    )
}

func (a *AuditLogger) LogDataAccess(ctx context.Context, sessionID, dataType string, user string) {
    a.logger.InfoContext(ctx, "data_access",
        "event_type", "data_access",
        "session_id", sessionID,
        "data_type", dataType,
        "user", user,
        "timestamp", time.Now().UTC(),
    )
}

func (a *AuditLogger) LogSensitiveDataRedacted(ctx context.Context, file string, findings []string) {
    a.logger.WarnContext(ctx, "sensitive_data_redacted",
        "event_type", "sensitive_data_redacted",
        "file", file,
        "findings", findings,
        "timestamp", time.Now().UTC(),
    )
}
```

### Never Log Sensitive Data

```go
// BAD: Logging sensitive data
slog.Info("processing file", "content", fileContent)

// GOOD: Log metadata only
slog.Info("processing file",
    "path", file.Path,
    "size", len(file.Content),
    "hash", sha256Hash(file.Content),
)
```

## Data Retention and Deletion

### Implement Automatic Cleanup

```go
type SessionCleaner struct {
    store         *SessionStore
    retentionDays int
}

func NewSessionCleaner(store *SessionStore, retentionDays int) *SessionCleaner {
    return &SessionCleaner{
        store:         store,
        retentionDays: retentionDays,
    }
}

func (c *SessionCleaner) CleanupExpiredSessions(ctx context.Context) error {
    cutoffDate := time.Now().AddDate(0, 0, -c.retentionDays)
    
    sessions, err := c.store.ListAllSessions()
    if err != nil {
        return fmt.Errorf("failed to list sessions: %w", err)
    }
    
    for _, session := range sessions {
        if session.CreatedAt.Before(cutoffDate) {
            slog.InfoContext(ctx, "deleting expired session",
                "session_id", session.SessionID,
                "created_at", session.CreatedAt,
                "age_days", time.Since(session.CreatedAt).Hours()/24,
            )
            
            if err := c.store.DeleteSession(session.SessionID); err != nil {
                slog.ErrorContext(ctx, "failed to delete session",
                    "session_id", session.SessionID,
                    "error", err,
                )
                continue
            }
        }
    }
    
    return nil
}
```

### Support Data Deletion Requests

```go
func (s *SessionStore) DeleteSession(sessionID string) error {
    sessionPath := s.sessionPath(sessionID)
    
    // Securely delete file
    if err := secureDelete(sessionPath); err != nil {
        return fmt.Errorf("failed to delete session: %w", err)
    }
    
    slog.Info("session deleted", "session_id", sessionID)
    return nil
}

func secureDelete(path string) error {
    // Overwrite file with random data before deletion
    f, err := os.OpenFile(path, os.O_WRONLY, 0600)
    if err != nil {
        return err
    }
    defer f.Close()
    
    stat, err := f.Stat()
    if err != nil {
        return err
    }
    
    // Overwrite with random data
    randomData := make([]byte, stat.Size())
    if _, err := rand.Read(randomData); err != nil {
        return err
    }
    
    if _, err := f.Write(randomData); err != nil {
        return err
    }
    
    if err := f.Sync(); err != nil {
        return err
    }
    
    // Delete file
    return os.Remove(path)
}
```

## Access Control

### Validate File Permissions

```go
func (s *SessionStore) ValidatePermissions() error {
    info, err := os.Stat(s.baseDir)
    if err != nil {
        return fmt.Errorf("failed to stat session directory: %w", err)
    }
    
    // Check directory permissions (should be 0700)
    if info.Mode().Perm() != 0700 {
        slog.Warn("insecure session directory permissions",
            "path", s.baseDir,
            "current", info.Mode().Perm(),
            "expected", 0700,
        )
        
        // Fix permissions
        if err := os.Chmod(s.baseDir, 0700); err != nil {
            return fmt.Errorf("failed to fix permissions: %w", err)
        }
    }
    
    return nil
}
```

### Implement User Context

```go
type UserContext struct {
    UserID    string
    Username  string
    Roles     []string
    SessionID string
}

func GetUserContext(ctx context.Context) (*UserContext, error) {
    user, ok := ctx.Value(userContextKey).(*UserContext)
    if !ok {
        return nil, errors.New("user context not found")
    }
    return user, nil
}

func WithUserContext(ctx context.Context, user *UserContext) context.Context {
    return context.WithValue(ctx, userContextKey, user)
}
```

## Compliance

### GDPR Compliance

```go
// Support data export for GDPR Article 15 (Right of Access)
func (s *SessionStore) ExportUserData(userID string) ([]byte, error) {
    sessions, err := s.ListSessionsByUser(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to list user sessions: %w", err)
    }
    
    export := struct {
        UserID    string           `json:"user_id"`
        ExportedAt time.Time       `json:"exported_at"`
        Sessions  []*ReviewSession `json:"sessions"`
    }{
        UserID:     userID,
        ExportedAt: time.Now().UTC(),
        Sessions:   sessions,
    }
    
    data, err := json.MarshalIndent(export, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal export: %w", err)
    }
    
    return data, nil
}

// Support data deletion for GDPR Article 17 (Right to Erasure)
func (s *SessionStore) DeleteUserData(userID string) error {
    sessions, err := s.ListSessionsByUser(userID)
    if err != nil {
        return fmt.Errorf("failed to list user sessions: %w", err)
    }
    
    for _, session := range sessions {
        if err := s.DeleteSession(session.SessionID); err != nil {
            return fmt.Errorf("failed to delete session %s: %w", session.SessionID, err)
        }
    }
    
    slog.Info("user data deleted", "user_id", userID, "sessions_deleted", len(sessions))
    return nil
}
```

### Data Minimization

```go
// Only store necessary data
type MinimalSessionData struct {
    SessionID    string    `json:"session_id"`
    WorkloadID   string    `json:"workload_id"`
    Status       string    `json:"status"`
    CreatedAt    time.Time `json:"created_at"`
    // Do NOT store: full IaC content, user details, credentials
}

func (s *SessionStore) SaveMinimalSession(session *ReviewSession) error {
    minimal := MinimalSessionData{
        SessionID:  session.SessionID,
        WorkloadID: session.WorkloadID,
        Status:     string(session.Status),
        CreatedAt:  session.CreatedAt,
    }
    
    // Store only minimal data
    return s.saveJSON(s.sessionPath(session.SessionID), minimal)
}
```

## Security Best Practices Summary

1. **Redact sensitive data**: Always redact credentials, keys, and PII before storage
2. **Use secure file permissions**: 0600 for files, 0700 for directories
3. **Never log sensitive data**: Log metadata only, never content
4. **Implement audit logging**: Track all security-relevant events
5. **Use AWS credential chain**: Never hardcode credentials
6. **Automatic cleanup**: Delete old data based on retention policy
7. **Support data deletion**: Implement secure deletion for compliance
8. **Validate permissions**: Check and fix file permissions on startup
9. **Data minimization**: Only store what's necessary
10. **Encrypt at rest**: Consider encrypting session files for additional security
