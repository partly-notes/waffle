# Dependency Management Guidelines

## Always Use Context7 for Latest Versions

When adding dependencies to the project, always use Context7 MCP to look up the latest versions and best practices for the library.

### Why Use Context7

1. **Latest Versions**: Context7 provides up-to-date documentation and version information
2. **Best Practices**: Get current usage patterns and recommendations
3. **Breaking Changes**: Understand API changes between versions
4. **Security**: Ensure you're using versions without known vulnerabilities

### Workflow for Adding Dependencies

#### For Go Dependencies

1. **Look up the library first**:
   - Use Context7 to resolve the library ID
   - Get documentation for the latest version
   - Check for any breaking changes or migration guides

2. **Add the dependency**:
   ```bash
   go get <package>@latest
   ```

3. **Verify the version**:
   - Check `go.mod` for the added version
   - Ensure it matches the latest stable release

#### Example: Adding AWS SDK v2

```go
// Before adding, use Context7 to check:
// - Latest version of github.com/aws/aws-sdk-go-v2
// - Current best practices for initialization
// - Any recent API changes

// Then add:
// go get github.com/aws/aws-sdk-go-v2@latest
// go get github.com/aws/aws-sdk-go-v2/config@latest
// go get github.com/aws/aws-sdk-go-v2/service/wellarchitected@latest
```

### Context7 Usage Pattern

1. **Resolve Library ID**:
   ```
   Use Context7 resolve-library-id to find the correct library identifier
   Example: "aws-sdk-go-v2" → "/aws/aws-sdk-go-v2"
   ```

2. **Get Documentation**:
   ```
   Use Context7 get-library-docs with the resolved ID
   Mode: 'code' for API references
   Mode: 'info' for conceptual guides
   ```

3. **Check Multiple Pages**:
   ```
   If initial context isn't sufficient, request page=2, page=3, etc.
   ```

### Common Go Libraries to Check

- **AWS SDK v2**: `/aws/aws-sdk-go-v2`
- **Viper**: `/spf13/viper`
- **Cobra**: `/spf13/cobra`
- **Testify**: `/stretchr/testify`
- **Zap/Slog**: Standard library or `/uber-go/zap`
- **Gopter**: `/leanovate/gopter`

### Version Pinning Strategy

1. **Development**: Use `@latest` to get the newest features
2. **Production**: Pin to specific versions after testing
3. **Security Updates**: Regularly check for updates using Context7
4. **Breaking Changes**: Review Context7 docs before major version upgrades

### Before Committing

- [ ] Verified latest version using Context7
- [ ] Checked for breaking changes
- [ ] Updated import statements if needed
- [ ] Ran `go mod tidy` to clean up
- [ ] Tested that code compiles and tests pass

### Keeping Dependencies Updated

Run periodic checks:
```bash
# Check for outdated dependencies
go list -u -m all

# Update all dependencies
go get -u ./...
go mod tidy
```

Use Context7 to review changelogs and migration guides for major updates.

## Exception Cases

Only skip Context7 lookup when:
- Adding a well-known standard library package
- Re-adding a dependency that was just removed
- Following explicit version requirements from documentation

In all other cases, use Context7 to ensure you're using the latest stable version with current best practices.
