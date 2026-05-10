# Contributing to BurpBridge

Thank you for your interest in contributing to BurpBridge!

## Code of Conduct

We are committed to providing a welcoming and inclusive experience for everyone. Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Bugs

1. Check if the issue already exists
2. Create a detailed issue with:
   - Clear title
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (Go version, OS, etc.)
   - Relevant logs/output

### Suggesting Features

1. Open an issue with the `feature` label
2. Describe the use case
3. Explain the proposed solution
4. Discuss potential alternatives

### Pull Requests

#### Workflow

1. **Fork** the repository
2. **Create** a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make** your changes
4. **Test** your changes
5. **Commit** using conventional commits:
   ```bash
   git commit -m "feat: add new feature"
   ```
6. **Push** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
7. **Submit** a Pull Request

#### Pull Request Guidelines

- Keep PRs focused and atomic
- Include tests for new functionality
- Update documentation as needed
- Ensure all tests pass
- Follow coding standards (see below)

## Coding Standards

### Go Formatting

All code must be formatted with `gofmt`:

```bash
# Format all Go files
gofmt -w ./...

# Check formatting without writing
gofmt -d ./
```

### No Unused Code

- Remove unused variables and functions
- Use `_` for unused function parameters when required by interfaces
- Build must complete without warnings

### Resource Management

Always ensure proper resource cleanup:

```go
// Good: defer closing
defer func() {
    if err := conn.Close(); err != nil {
        log.Printf("Error closing: %v", err)
    }
}()
```

```go
// Bad: no cleanup
conn, _ := net.Dial(...)
// ... code that might return early
```

### Error Handling

Handle errors explicitly:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to do thing: %w", err)
}

// Bad
_ = someFunction() // ignoring error
```

### Naming Conventions

- Use meaningful variable names
- Follow Go naming conventions (camelCase for variables, PascalCase for exported)
- Add comments for exported functions

### Logging

- Use consistent log format: `[Component] Message`
- Avoid logging sensitive data
- Use appropriate log levels

## Commit Message Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation |
| `style` | Code style (formatting) |
| `refactor` | Code refactoring |
| `test` | Tests |
| `chore` | Maintenance |

### Examples

```bash
# Feature
git commit -m "feat(engine): add UDP relay support"

# Bug fix
git commit -m "fix(mobile): close connection on error path"

# Documentation
git commit -m "docs: update API documentation"
```

## Testing Requirements

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package
go test ./pkg/engine/ -v
```

### Writing Tests

- Add tests for new functionality
- Follow existing test patterns
- Name test files: `*_test.go`

## Documentation

- Update README.md for user-facing changes
- Update DEVELOPMENT.md for process changes
- Add code comments for complex logic
- Document public API functions

## Review Process

1. Automated checks run (build, tests, lint)
2. At least one maintainer reviews
3. Address feedback promptly
4. Once approved, maintainer merges

## Getting Help

- Open an issue for bugs/features
- Join discussions in PRs
- Check existing documentation

## Recognition

Contributors will be acknowledged in the project (with permission).

---

*Thank you for contributing to BurpBridge!*