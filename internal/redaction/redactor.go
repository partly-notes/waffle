package redaction

import (
	"log/slog"
	"regexp"
	"strings"
)

// RedactionRule defines a pattern to match and its replacement
type RedactionRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
}

// Redactor handles sensitive data redaction
type Redactor struct {
	rules []RedactionRule
}

// NewRedactor creates a new Redactor with default rules
func NewRedactor() *Redactor {
	return &Redactor{
		rules: DefaultRedactionRules(),
	}
}

// NewRedactorWithRules creates a new Redactor with custom rules
func NewRedactorWithRules(rules []RedactionRule) *Redactor {
	return &Redactor{
		rules: rules,
	}
}

// DefaultRedactionRules returns the default set of redaction rules
func DefaultRedactionRules() []RedactionRule {
	return []RedactionRule{
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
			Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*([:=])\s*["']?([^"'\s]+)["']?`),
			Replacement: `$1 $2 "[REDACTED]"`,
		},
		{
			Name:        "API Key",
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|token)\s*([:=])\s*["']?([^"'\s]+)["']?`),
			Replacement: `$1 $2 "[REDACTED]"`,
		},
		{
			Name:        "Secret Field",
			Pattern:     regexp.MustCompile(`(?i)(secret|secret_key|secret_access_key)\s*([:=])\s*["']?([^"'\s]+)["']?`),
			Replacement: `$1 $2 "[REDACTED]"`,
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
}

// Redact applies all redaction rules to the content
// Returns the redacted content and a list of findings (rule names that matched)
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

// RedactProperties recursively redacts sensitive data from a properties map
// Returns the redacted properties and a list of findings
func (r *Redactor) RedactProperties(properties map[string]interface{}) (map[string]interface{}, []string) {
	if properties == nil {
		return nil, nil
	}

	redacted := make(map[string]interface{})
	allFindings := []string{}

	for key, value := range properties {
		// Check if the key itself indicates sensitive data
		if r.isSensitiveKey(key) {
			// Redact the value based on key name
			redacted[key] = "[REDACTED]"
			finding := r.getFindingForKey(key)
			if !contains(allFindings, finding) {
				allFindings = append(allFindings, finding)
			}
		} else {
			// Otherwise, recursively redact the value
			redactedValue, findings := r.redactValue(value)
			redacted[key] = redactedValue
			
			// Collect unique findings
			for _, finding := range findings {
				if !contains(allFindings, finding) {
					allFindings = append(allFindings, finding)
				}
			}
		}
	}

	return redacted, allFindings
}

// isSensitiveKey checks if a key name indicates sensitive data
func (r *Redactor) isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"secret", "secret_key", "secret_access_key",
		"api_key", "apikey", "api-key",
		"token", "access_token", "auth_token",
		"private_key", "privatekey",
	}
	
	for _, sensitive := range sensitiveKeys {
		if lowerKey == sensitive || strings.Contains(lowerKey, sensitive) {
			return true
		}
	}
	
	return false
}

// getFindingForKey returns the appropriate finding name for a sensitive key
func (r *Redactor) getFindingForKey(key string) string {
	lowerKey := strings.ToLower(key)
	
	if strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "passwd") || strings.Contains(lowerKey, "pwd") {
		return "Password Field"
	}
	if strings.Contains(lowerKey, "secret") {
		return "Secret Field"
	}
	if strings.Contains(lowerKey, "api") && strings.Contains(lowerKey, "key") {
		return "API Key"
	}
	if strings.Contains(lowerKey, "token") {
		return "API Key"
	}
	if strings.Contains(lowerKey, "private") && strings.Contains(lowerKey, "key") {
		return "Secret Field"
	}
	
	return "Secret Field"
}

// redactValue recursively redacts a value of any type
func (r *Redactor) redactValue(value interface{}) (interface{}, []string) {
	findings := []string{}

	switch v := value.(type) {
	case string:
		// Redact string values
		redacted, stringFindings := r.Redact(v)
		findings = append(findings, stringFindings...)
		return redacted, findings

	case map[string]interface{}:
		// Recursively redact nested maps
		redactedMap, mapFindings := r.RedactProperties(v)
		findings = append(findings, mapFindings...)
		return redactedMap, findings

	case []interface{}:
		// Recursively redact arrays
		redactedArray := make([]interface{}, len(v))
		for i, item := range v {
			redactedItem, itemFindings := r.redactValue(item)
			redactedArray[i] = redactedItem
			findings = append(findings, itemFindings...)
		}
		return redactedArray, findings

	default:
		// Return other types as-is (numbers, booleans, etc.)
		return value, findings
	}
}

// LogRedactionFindings logs redaction findings with appropriate context
func LogRedactionFindings(findings []string, context string) {
	if len(findings) > 0 {
		slog.Warn("sensitive data redacted",
			"context", context,
			"findings", findings,
			"count", len(findings),
		)
	}
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
