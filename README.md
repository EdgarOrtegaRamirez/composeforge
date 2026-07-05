# ComposeForge

A comprehensive Docker Compose analysis and management toolkit. Parse, validate, analyze, diff, and merge Docker Compose files with ease.

## Features

- **Validate** — Check compose files for errors, security issues, and best practices
- **Analyze** — Dependency graphs, build order, network topology, resource analysis
- **Diff** — Compare two compose files and see exactly what changed
- **Merge** — Combine multiple compose files (base + overrides)

## Installation

```bash
go install github.com/EdgarOrtegaRamirez/composeforge/cmd/composeforge@latest
```

Or build from source:

```bash
git clone https://github.com/EdgarOrtegaRamirez/composeforge.git
cd composeforge
go build -o composeforge ./cmd/composeforge
```

## Quick Start

### Validate a compose file

```bash
composeforge validate docker-compose.yml
```

Output:
```
✓ docker-compose.yml: valid
═══
Total: 0 errors, 1 warnings across 1 file(s)
```

### Analyze a compose file

```bash
composeforge analyze docker-compose.yml
```

Output:
```
═══ ComposeForge Analysis ═══

Services: 4
Networks: 2
Volumes: 1
Secrets: 0
Exposed Ports: 3

── Build Order ──
  1. db
  2. redis
  3. api
  4. web

── Dependency Tree ──
web
└── api
    ├── db
    └── redis

── Network Graph ──
Network: default
  ├── api
  ├── db
  ├── redis
  └── web
Network: frontend
  ├── api
  └── web
```

### Diff two compose files

```bash
composeforge diff docker-compose.yml docker-compose.prod.yml
```

Output:
```
Modified: web
  ~ image: nginx:1.0 → nginx:2.0
  ~ ports: 8080:80 → 80:80

═══
Changes: 0 added, 0 removed, 1 modified
```

### Merge compose files

```bash
composeforge merge docker-compose.yml docker-compose.override.yml
```

Merges files with later files overriding earlier ones. Outputs to stdout or a file:

```bash
composeforge merge base.yml override.yml -o merged.yml
```

## Validation Rules

### Security Checks
- 🔴 **Privileged mode** — Container runs with full host access
- 🔴 **Dangerous capabilities** — SYS_ADMIN, NET_ADMIN, ALL capabilities
- 🔴 **Security profiles disabled** — apparmor:unconfined, seccomp:unconfined
- 🔴 **Sensitive host paths** — Docker socket, /proc, /sys, /dev mounts
- ⚠️ **Running as root** — No user specified or explicitly root
- ⚠️ **Secrets in environment** — API keys, passwords in env vars
- ⚠️ **Writable root filesystem** — Consider read_only: true

### Structure Checks
- Missing image and build configuration
- Invalid restart policies
- Missing health checks on restart:always services
- Non-existent service dependencies
- Non-existent network/volume references

### Best Practices
- No memory/CPU limits set
- Zero retries on health checks
- Non-standard Dockerfile naming

## Architecture

```
composeforge/
├── pkg/
│   ├── parser/       # YAML parsing and data models
│   ├── validator/    # Security and best practice validation
│   ├── analyzer/     # Dependency analysis and graph algorithms
│   ├── differ/       # Semantic diff between compose files
│   └── merger/       # Multi-file merge with override semantics
├── cmd/
│   └── composeforge/ # CLI entry point
└── tests/            # Comprehensive test suite
```

### Algorithms Used

- **Topological Sort (Kahn's Algorithm)** — Build order calculation from dependency DAG
- **DFS Cycle Detection** — Circular dependency detection
- **BFS Depth Calculation** — Service dependency depth levels
- **Semantic Diffing** — Field-by-field comparison with change classification
- **Deep Merge** — Environment, volumes, networks, and service config merging

## CLI Commands

| Command | Description |
|---------|-------------|
| `validate` | Validate compose files for errors and security issues |
| `analyze` | Analyze dependencies, networks, and resource usage |
| `diff` | Compare two compose files |
| `merge` | Merge multiple compose files |
| `version` | Print version information |

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (text, json) or output file |

## Development

```bash
# Run tests
go test ./tests/...

# Build
go build -o composeforge ./cmd/composeforge

# Vet
go vet ./...
```

## License

MIT
