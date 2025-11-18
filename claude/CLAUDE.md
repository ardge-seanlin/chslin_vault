## Language Rules

### Response Language
- **MANDATORY**: Always respond in Traditional Chinese (繁體中文), regardless of the language used in the question
- **TERMINOLOGY**: Use Taiwan-style terminology and proper nouns:
  - 「軟體」not「软件」
  - 「程式」not「程序」
  - 「伺服器」not「服务器」
  - 「網路」not「网络」
  - 「資料庫」not「数据库」
  - 「檔案」not「文件」

### Prohibited Language
- **STRICTLY FORBIDDEN**: Simplified Chinese (簡體中文) characters
- **STRICTLY FORBIDDEN**: China Mainland terminology (e.g.,「鼠标、软件、硬件、文件夹」)

### Code Comments
- **MANDATORY**: All code comments MUST be written in English only
- **PROHIBITED**: Chinese or any other non-English language in code comments
- Example:
  - Good: `// Initialize database connection`
  - Bad: `// 初始化資料庫連線`

### Special Comment Markers

#### 1. TODO
- **PURPOSE**: Mark incomplete features or planned improvements
- **URGENCY**: Low - for future work
- **FORMAT**: `// TODO: <clear description>`

#### 2. FIXME
- **PURPOSE**: Mark known bugs or incorrect implementations
- **URGENCY**: Medium to High
- **FORMAT**: `// FIXME: <clear description>`

#### 3. XXX
- **PURPOSE**: Strong warning for serious issues, hacky code, or workarounds
- **URGENCY**: Medium to High
- **FORMAT**: `// XXX: <clear warning and explanation>`

**Best Practices:**
- Always provide clear, actionable descriptions
- Prefer fixing issues immediately over adding markers when possible

## Go Coding Standards

Follow official Go style guidelines and community best practices.

**References:**
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### 1. Naming Conventions

**General Rules:**
- Use **MixedCaps** or **mixedCaps**, not snake_case
- Acronyms should be all caps: `HTTP`, `URL`, `ID`, `API`
- Be concise but clear

**Packages:** Short, lowercase, single-word names (e.g., `user`, `http`, `auth`)

**Functions:** MixedCaps for exported, mixedCaps for unexported

**Variables:** Short names in small scopes (`i`, `n`, `err`), descriptive in larger scopes

**Constants:** MixedCaps, not ALL_CAPS (e.g., `MaxRetries`)

**Interfaces:** Single-method interfaces end in `-er` (e.g., `Reader`, `Writer`)

### 2. Code Organization

**File Structure:**
- One main concept per file
- Order: types → constructors → methods → helpers

**Import Groups:** Separate into three groups with blank lines:
```go
import (
    // Standard library
    "context"
    "fmt"

    // External dependencies
    "github.com/gin-gonic/gin"

    // Internal packages
    "yourproject/pkg/auth"
)
```

### 3. Function Guidelines

- Keep functions under 50 lines when possible
- Limit to 3-4 parameters; use structs for more
- Return errors as the last return value
- Context should always be the first parameter

### 4. Methods and Receivers

**Receiver Naming:**
- Use consistent, short receiver names (1-2 characters)
- Same receiver name across all methods of a type

**Pointer vs Value:**
- Use pointer receivers when method needs to modify receiver or struct is large
- Use value receivers for small, immutable types
- Be consistent: if any method uses pointer, all should

**Context:**
- Context should always be first parameter
- Never store Context in a struct
- Pass Context explicitly through the call chain

### 5. Common Go Idioms

**Early Return:**
```go
func ProcessData(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }
    // main logic
    return nil
}
```

**Defer for Cleanup:**
```go
func ReadFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    return io.ReadAll(f)
}
```

**Accept Interfaces, Return Structs:**
```go
func ProcessData(r io.Reader) (*Result, error) {
    // ...
}
```

## Go Error Handling Rules

**References:**
- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)
- [Uber Go Style Guide - Error Handling](https://github.com/uber-go/guide/blob/master/style.md#error-handling)

### Core Principles

1. **Always Check Errors**: Never ignore errors with `_` unless you have a very good reason

2. **Error Wrapping**: Use `%w` to maintain error chain for `errors.Is()` and `errors.As()`
   ```go
   return fmt.Errorf("process data failed: %w", err)
   ```

3. **Error Messages**: Use lowercase, no trailing punctuation, provide context

4. **Sentinel Errors**: Define package-level error variables for expected errors
   ```go
   var (
       ErrNotFound     = errors.New("resource not found")
       ErrUnauthorized = errors.New("unauthorized access")
   )
   ```

5. **Custom Error Types**: For errors needing additional context, implement `Error()` and `Unwrap()`

6. **When to Panic**: ONLY for unrecoverable programming errors or initialization failures

### Common Patterns

**Early Return:**
```go
func ProcessRequest(req *Request) error {
    if req == nil {
        return errors.New("request is nil")
    }
    if err := validateRequest(req); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}
```

**Retry with Context:**
```go
func DoWithRetry(ctx context.Context, maxRetries int, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }
        // Exponential backoff with jitter
        if attempt < maxRetries-1 {
            backoff := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
    return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
```

**Timeout with Context:**
```go
func FetchWithTimeout(userID string) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return FetchUser(ctx, userID)
}
```

## Go Testing Rules

### Best Practices
- Tests MUST be clear, meaningful, and maintainable
- Use table-driven tests for multiple similar cases
- Each test should focus on a single behavior

### Test Organization

**Unit Tests:**
- Test single function/method in isolation
- No external dependencies
- Fast execution (milliseconds)
- File naming: `*_test.go`

**Integration Tests:**
- Test interaction between components
- May use external dependencies
- Use build tags: `// +build integration`
- Run separately: `go test -tags=integration ./...`

### Table-Driven Tests

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid", "user@example.com", false},
        {"invalid", "not-an-email", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Mocking Philosophy

**Hand-Written Mocks First:**
- Go's interface design makes manual mocking straightforward
- Use mocking tools only for large interfaces (10+ methods)

**Simple Mock Example:**
```go
type mockUserRepository struct {
    users map[string]*User
    err   error
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.users[id], nil
}
```

### Coverage Philosophy
- **DO NOT** write tests solely to increase coverage percentage
- **ONLY** test meaningful scenarios that verify actual behavior
- Focus on business logic, edge cases, and error handling
- Coverage metrics are indicators, not goals

### Validation Workflow
- **WITH Makefile**: Execute `make lint` and `make test`
- **WITHOUT Makefile**: Execute `golangci-lint run` and `go test -v ./...`
- **REQUIREMENT**: Both must pass before considering work complete

## Code Refactoring Rules

### Core Principles
- **PROHIBITED**: Refactoring without tests
- **MANDATORY**: Write tests before every refactoring
- **PHILOSOPHY**: Small steps, frequent commits, always working code

### When to Refactor (Code Smells)

1. **Duplicated Code** - Same logic in multiple places
2. **Long Function** - Function > 50 lines or does multiple things
3. **Long Parameter List** - More than 3-4 parameters
4. **Large Struct** - Too many fields/methods
5. **Deep Nesting** - More than 3 levels of if/for
6. **Feature Envy** - Function uses more data from another struct

### Test-Driven Refactoring (TDR) Workflow

**Step 1: Write Tests First**
- Write comprehensive tests for code to be refactored
- Tests MUST cover existing functionality and edge cases
- Execute tests to ensure all pass (baseline)

**Step 2: Refactor in Small Steps**
- Make ONE improvement at a time
- Run tests after each step
- Commit after each successful refactoring

**Step 3: Verify and Commit**
- Execute `make lint && make test`
- Ensure all tests still pass
- Commit with `refactor:` type

### Common Refactoring Techniques

| Technique | When to Use |
|-----------|-------------|
| **Extract Function** | Function > 50 lines or does multiple things |
| **Extract Interface** | Need polymorphism or testability |
| **Introduce Parameter Object** | More than 3-4 parameters |
| **Replace Magic Number with Constant** | Hardcoded numbers without context |

### Scope Control

**Golden Rule**: One code smell, one PR

- Focus on ONE specific issue per session
- Avoid "While I'm here, let me also fix..." (scope creep)
- Create separate branches for other problems

### SOLID Principles Reference
- **S (Single Responsibility)**: Each function/struct does only one thing
- **O (Open/Closed)**: Extensible without modifying existing code
- **L (Liskov Substitution)**: Subtypes substitutable for parent types
- **I (Interface Segregation)**: Small, focused interfaces
- **D (Dependency Inversion)**: Depend on abstractions, not concrete implementations

### When to Stop Refactoring
- Code smell is resolved
- Code is clear and maintainable
- Further changes don't add practical value
- **YAGNI**: You Aren't Gonna Need It

## Git Operation Rules

### 1. Git Flow Branch Types

**Long-lived branches:**
- `main` - Production environment
- `develop` - Development integration
- `support/*` - Supporting multiple versions

**Temporary branches:**
- `feature/*` - New feature development
- `release/*` - Release preparation
- `hotfix/*` - Production emergency fixes

**Branch Naming:** Use lowercase with hyphens: `<type>/<descriptive-name>`

Examples: `feature/user-authentication`, `fix/database-connection-leak`

### 2. Commit Message Standards (Conventional Commits)

#### Structure
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Commit Types
- `feat:` - New feature (MINOR in SemVer)
- `fix:` - Bug fix (PATCH in SemVer)
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, no code change)
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `test:` - Adding or updating tests
- `build:` - Build system or dependency changes
- `ci:` - CI/CD configuration changes
- `chore:` - Other changes

#### Breaking Changes
- Add `!` after type/scope: `feat(api)!: breaking change`
- Or use footer: `BREAKING CHANGE: description` (MAJOR in SemVer)

#### Commit Message Examples

**Example 1: Test**
```
test: add comprehensive tests for binfile and generator packages

Add extensive unit and integration tests for core functionality:

pkg/binfile:
- Unit tests for Writer and Reader (28 tests)
- Integration tests for write-read cycles (10 tests)

pkg/manifest/generator:
- Unit tests for template processing (14 tests)
- HTTP download tests with retry mechanism (9 tests)

All tests pass with race detection enabled.
```

**Example 2: Feature**
```
feat(api): add user authentication and authorization system

Implement JWT-based authentication with role-based access control:

pkg/auth:
- JWT token generation and validation
- Token refresh mechanism with sliding expiration

pkg/middleware:
- Authentication middleware for protected routes
- Role-based authorization middleware

Database migrations included for user and role tables.
```

**Example 3: Fix**
```
fix(database): resolve connection pool exhaustion under high load

Fix goroutine leak causing connection pool exhaustion:

Root Cause:
- Database connections not properly released in error paths
- Context cancellation not properly handled

Solution:
- Add deferred connection close in all query functions
- Implement proper context cancellation handling
- Add connection pool metrics for monitoring

Testing:
- Add stress tests simulating 1000 concurrent requests
- Verify no connection leaks under error conditions
```

**Example 4: Performance**
```
perf(cache): optimize Redis operations for frequently accessed data

Implement caching layer to reduce database load:

Changes:
- Add Redis-based caching for user profile data
- Implement cache-aside pattern with TTL of 5 minutes
- Add cache invalidation on profile updates

Performance Impact:
- User profile API response time: 250ms → 15ms (94% improvement)
- Database query reduction: 85% for cached endpoints
- Redis hit rate: 92% after warmup
```

### 3. File Status Management
- **PROHIBITED**: `git add`, `git reset`, `git restore`, `git stash` unless explicitly requested
- **CONDITION**: Do not execute operations affecting file staging status unless requested

### 4. Pre-Commit Workflow
When specific files are designated for commit:
1. Execute `git diff --staged <specified files>` to confirm changes
2. Provide gitflow branch name suggestions based on changes
3. Wait for confirmation before proceeding

### 5. Commit Execution
- **MANDATORY**: Use `git commit <specified files>` format
- **PURPOSE**: Ensure commit scope is precisely limited to specified files
- **PROHIBITED**: Using `git commit` without specifying files

### 6. Commit Message Requirements
- **LANGUAGE**: All commit messages MUST be written in English
- **TITLE LENGTH**: Subject line MUST be 72 characters or less
- **CONTENT BASIS**: Base suggestions ONLY on actual changes from `git diff --staged`
- **FORMAT**: Strictly follow Conventional Commits specification
- **SIGNATURES**: Do NOT add automatic signatures (e.g., "Generated with Claude Code", "Co-Authored-By") to commit messages

### 7. File Movement and Renaming
- **MANDATORY**: Use `git mv` instead of regular `mv` for git-managed files
- **PURPOSE**: Preserve git history and ensure proper tracking
