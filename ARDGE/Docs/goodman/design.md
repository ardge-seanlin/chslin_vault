# Design Philosophy & Development Roadmap

This document describes the design philosophy, development approach, and evolution of Goodman.

## Core Development Philosophy

```
MVP First → Test Coverage → Feature Expansion → User Experience
```

Goodman follows an incremental development approach, prioritizing working software over comprehensive planning, while maintaining code quality through continuous testing and refactoring.

## Design Principles

| Principle | Practice |
|-----------|----------|
| **MVP First** | Ship core functionality first, refine details later |
| **Test Before Refactor** | Never refactor without test coverage |
| **Progressive Complexity** | Build simple features first, extend gradually |
| **Interface Abstraction** | Define contracts before implementations |
| **Single Responsibility** | Each PR addresses one concern |
| **Docs as Code** | Documentation updates accompany feature changes |

## Development Phases

### Phase 1: MVP Validation

**Goal**: Prove the concept works

```
Initial commit
    ↓
Core implementation (types, executor, runner)
    ↓
Examples and documentation
```

**Key Decisions**:
- YAML over JSON → Git-friendly diffs
- External tools (curl/grpcurl) over custom clients → Leverage mature tooling
- Table-driven tests → Reduce boilerplate

**Core Data Model**:
```
Collection
├── Environments[]
└── Suites[]
    └── Tests[]
        ├── Request
        ├── Assertions[]
        └── Table[]
```

### Phase 2: User Experience

**Goal**: Make the tool pleasant to use

**Problem**: Long test suites require waiting for all tests to complete before seeing results.

**Solution**: Streaming output with real-time feedback.

```
Before: Wait for all tests → Show results
After:  ✓ test-1 (5ms)    ← Immediate
        ✓ test-2 (3ms)    ← Immediate
        ✗ test-3 (4ms)    ← Immediate
```

**Enhancements**:
- Streaming reporter with flush mechanism
- Table-driven test aggregation display `(passed/total)`
- Colored output with status indicators

### Phase 3: Technical Debt Resolution

**Goal**: Establish solid foundation for future development

**Approach**: Test first, then refactor

```
Add unit tests (types, assertion, config, report, loader, executor)
    ↓
Extract hardcoded constants
    ↓
Refactor with confidence
```

**Rule**: No refactoring without tests

**Outcomes**:
- Comprehensive test coverage across all packages
- Constants extracted to `internal/constants/`
- Reduced coupling through interface abstractions

### Phase 4: Horizontal Expansion - API Specification

**Goal**: Enable test coverage validation against API specs

**Problem**: How do we know if all API endpoints are tested?

**Solution**: Parse API specifications and compare against test suites

```
                ┌─ proto parser ──┐
API Spec ───────┼─ protoc parser ─┼──→ Unified IR ──→ Validator
                └─ swagger parser ┘
```

**Architecture**:
1. Define unified Intermediate Representation (IR) in `pkg/spec`
2. Implement format-specific parsers converting to IR
3. Build validators operating on IR

**Supported Formats**:
| Format | Description |
|--------|-------------|
| `proto` | Native .proto files |
| `protoc` | protoc-gen-doc JSON output |
| `swagger` | Swagger/OpenAPI 2.0 |

### Phase 5: Vertical Expansion - gRPC Streaming

**Goal**: Support all gRPC communication patterns

**Approach**: Progressive complexity

```
Unary (1:1) → Server Stream (1:N) → Client Stream (N:1) → Bidirectional (N:N)
  Simple          Medium              Medium                Complex
```

Each step builds on the previous:

| Mode | Input | Output |
|------|-------|--------|
| `unary` | `payload` | `response` |
| `server_streaming` | `payload` | `messages[]` |
| `client_streaming` | `payloads[]` | `response` |
| `bidirectional` | `payloads[]` | `messages[]` |

**Additional Features**:
- `max_messages` for early stream termination
- gRPC reflection support
- Compression configuration

### Phase 6: Production Readiness

**Goal**: Enterprise-grade reliability

**Features**:

| Feature | Problem Solved |
|---------|----------------|
| Retry Policy | Network instability, transient failures |
| TLS Configuration | Secure connections, mTLS |
| Conditional Execution | Environment-specific tests |
| Fail Fast | CI/CD efficiency |
| Shuffle | Detect test order dependencies |

**Retry with Exponential Backoff**:
```yaml
request:
  retries: 3
  retry_delay: 100ms
  retry_backoff: exponential
  retry_on: [502, 503, 504]
```

**Conditional Execution**:
```yaml
tests:
  - name: prod-only-test
    when: "{{env}} == 'prod'"
```

### Phase 7: Developer Experience

**Goal**: Lower the barrier to writing tests

**Problem**: YAML errors only discovered at runtime

**Solution**: JSON Schema validation + lint command

```
JSON Schema Generator
    ↓
Lint Command (pre-commit, CI)
    ↓
IDE Integration (autocomplete, validation)
```

**Features**:
- Schema generation from Go types via reflection
- Line number mapping for error messages
- Typo suggestions for unknown fields
- Claude Code skill integration for AI assistance

## Version Milestones

| Version | Key Features |
|---------|--------------|
| **v0.1.0** | Core framework, HTTP/gRPC execution, streaming output |
| **v0.2.0** | Unit tests, hook captures, global headers |
| **v0.3.0** | API spec parsers, validators, proto/swagger support |
| **v0.4.0** | gRPC streaming (server/client/bidirectional), reflection |
| **v0.5.0** | Retry policy, conditional execution, JSON Schema lint, Claude Code skill |

## Architecture Evolution

### Initial Architecture (v0.1)
```
CLI → Loader → Runner → Executor → Reporter
                           ↓
                      Assertion
```

### Current Architecture (v0.5)
```
┌─────────────────────────────────────────────────────────────────┐
│                            CLI                                   │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ ┌───────┐        │
│  │  run   │ │  list  │ │  lint  │ │ validate │ │ skill │        │
│  └────────┘ └────────┘ └────────┘ └──────────┘ └───────┘        │
└─────────────────────────────────────────────────────────────────┘
        │           │           │           │
        ▼           │           ▼           ▼
┌──────────────┐    │    ┌──────────┐ ┌──────────────┐
│    Loader    │    │    │  Schema  │ │    Parser    │
│  Collection  │    │    │Validator │ │ proto/swagger│
│    Suite     │    │    └──────────┘ └──────────────┘
│ Environment  │    │                        │
└──────────────┘    │                        ▼
        │           │                 ┌──────────────┐
        ▼           │                 │   spec.IR    │
┌──────────────┐    │                 └──────────────┘
│    Runner    │    │                        │
│  • Retry     │    │                        ▼
│  • Condition │    │                 ┌──────────────┐
│  • Shuffle   │    │                 │  Validator   │
└──────────────┘    │                 │  • Coverage  │
        │           │                 │  • Naming    │
        ▼           │                 └──────────────┘
┌──────────────┐    │
│   Executor   │    │
│  • HTTP      │    │
│  • gRPC      │    │
│  • Streaming │    │
└──────────────┘    │
        │           │
        ▼           │
┌──────────────┐    │
│  Assertion   │◄───┘
│  • JSONPath  │
│  • Operators │
└──────────────┘
        │
        ▼
┌──────────────┐
│   Reporter   │
│  • CLI       │
│  • JSON      │
└──────────────┘
```

## Future Directions

Potential areas for future development:

1. **Native HTTP Client** - Replace curl dependency for better performance
2. **Native gRPC Client** - Replace grpcurl for streaming improvements
3. **OpenAPI 3.0 Support** - Extend parser capabilities
4. **Parallel Test Execution** - Run tests concurrently within suites
5. **Test Generation** - Generate test stubs from API specs
6. **Plugin System** - Custom executors and reporters

## Lessons Learned

1. **Start Simple**: External tools (curl/grpcurl) reduced initial complexity
2. **Test Early**: Test coverage enabled confident refactoring
3. **Iterate Quickly**: Small PRs with single concerns are easier to review
4. **Document Continuously**: Docs alongside code prevent documentation debt
5. **Listen to Users**: Streaming output came from real usage feedback
6. **Abstract at the Right Time**: Interfaces emerged from concrete implementations
