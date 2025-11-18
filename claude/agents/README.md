# Claude Code Custom Agents

This directory contains custom agent configurations for Claude Code.

## Available Agents

### Rust Senior Engineer (`rust-senior-engineer.yaml`)

A specialized agent for advanced Rust architecture, design patterns, performance optimization, and code quality.

**When to use:**
- Complex Rust system architecture design
- Code reviews for critical Rust components
- Performance optimization and profiling
- Architectural decision-making
- Safety and concurrency analysis
- Refactoring and code quality improvement

**Invocation:**
```bash
# Using slash command
/rust-senior-engineer <task description>

# Or invoke via Task tool with subagent_type: rust-senior-engineer
```

**Capabilities:**
- Code review and analysis
- Architecture design
- Performance analysis
- Refactoring guidance
- Testing strategy
- API design
- Error handling patterns
- Concurrency pattern review

**Model:** Claude Sonnet 4.5 (Latest)

**Languages:**
- User communication: Traditional Chinese (繁體中文)
- Code comments: English only

---

## How to Use Custom Agents

### 1. In Claude Code CLI

```bash
# List available agents
/agents list

# Use an agent for a task
/rust-senior-engineer Analyze the performance of this algorithm
```

### 2. Programmatically (Claude Code API)

```python
agent = load_agent("rust-senior-engineer")
result = agent.analyze(code, instructions)
```

### 3. Via Task Tool

```
Use the Task tool with:
- subagent_type: "rust-senior-engineer"
- description: Brief description
- prompt: Detailed task instructions
```

---

## Adding New Agents

1. Create a new `<agent-name>.yaml` file
2. Define the agent configuration with:
   - `name`: Display name
   - `description`: Purpose and capabilities
   - `model`: Model ID to use
   - `instructions`: Detailed system prompt
   - `tools`: Available tools
   - `capabilities`: Key capabilities

3. Optionally update this README

---

## Configuration Format

```yaml
name: Agent Display Name
description: Brief description of the agent

model: claude-sonnet-4-5-20250929
instructions: |
  Detailed instructions for the agent...

tools:
  - read
  - write
  - bash
  - grep

capabilities:
  - capability1: true
  - capability2: true

language: "Traditional Chinese (繁體中文)"
```

---

## Notes

- Agents inherit Claude Code's security model and tool restrictions
- All agents follow the user's global Claude.md instructions
- Agents can be task-specific or general purpose
- Configuration updates take effect immediately

Last updated: 2025-11-12
