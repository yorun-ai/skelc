# skelc

[![License](https://img.shields.io/github/license/yorun-ai/skelc)](LICENSE)
[![Version](https://img.shields.io/github/v/release/yorun-ai/skelc?label=version&cacheSeconds=300)](https://github.com/yorun-ai/skelc/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/yorun-ai/skelc)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/go.yorun.ai/skelc.svg)](https://pkg.go.dev/go.yorun.ai/skelc)
[![CI](https://github.com/yorun-ai/skelc/actions/workflows/ci.yml/badge.svg)](https://github.com/yorun-ai/skelc/actions/workflows/ci.yml)

**English** | [简体中文](README.zh-CN.md)

skelc is the compiler and command-line tool for the Skel contract language. Use `.skel` files to describe the domain data, services, events, and entry points of a Vine application, then generate Go server code, TypeScript clients, and public contracts for other domains.

When building a Vine application, skelc helps you:

- Keep server and client types and service contracts in one source of truth
- Catch syntax, naming, type, and cross-domain reference errors before generation
- Generate a Go module and TypeScript clients ready to add to a project
- Publish only domain boundaries marked `pub`
- Format and validate contracts
- Generate vRPC transport metadata for Binary parameters so applications can transfer binary data efficiently with CBOR

## Install

Go 1.26 or later is required:

```bash
go install go.yorun.ai/skelc/cmd/skelc@latest
skelc version
```

## Five-Minute Quick Start

Create `user.skel`:

```skel
domain demo.user

pub actor ClientActor {
    via client {}
}

pub data User {
    id: int
    name: string
}

pub service UserService {
    for ClientActor via client

    method getUser {
        input {
            userId: int
        }
        output User?
    }
}
```

This defines a `UserService` that can be called through a client. `pub` only allows a declaration to enter public generated output; it does not make a network endpoint anonymously accessible.

Validate and format the contract first:

```bash
skelc check --skel-in ./user.skel
skelc format --skel-in ./user.skel
```

Generate a standalone module for the Go server:

```bash
skelc gen go-module \
  --skel-in ./user.skel \
  --go-out ./generated/user-go \
  --go-module example.com/generated/user
```

Generate types and a service client for a TypeScript application:

```bash
skelc gen ts \
  --skel-in ./user.skel \
  --ts-out ./generated/user-ts
```

The TypeScript output contains:

```text
generated/user-ts/
├── data.ts       # data and enum types
├── spec.ts       # vRPC service descriptions
├── service.ts    # service client factories
└── index.ts      # public exports
```

After adding the generated directory to a TypeScript project, create the service with an already configured `VrpcClient`:

```ts
import { createUserService } from './generated/user-ts';

const userService = createUserService(client);
const user = await userService.getUser({ userId: 1001 });
```

Generation supports shared output directories. skelc records owned files in `.skelc-manifest.json`, preserves untracked files, and removes a stale generated file only when its content still matches the previous manifest.

A runnable version of this walkthrough lives in [`examples/quickstart`](examples/quickstart). From a repository checkout, validate the contract and generate every supported target with:

```bash
./examples/quickstart/generate.sh
```

## Common Workflows

### Organize a Domain as a Directory

As a contract grows, split the same domain across multiple files:

```text
skel/
├── domain.skel
├── actor.skel
├── data.skel
└── service.skel
```

`domain.skel` may contain only the domain declaration and an optional `@desc`. Every other file must also begin with the same domain declaration. Pass the whole directory to `--skel-in`:

```bash
skelc check --skel-in ./skel
```

### Generate a Public Contract

Extract declarations marked `pub` into shareable Skel:

```bash
skelc gen skel \
  --pub \
  --skel-in ./skel \
  --skel-out ./generated/public-skel
```

TypeScript generation also accepts `--pub` to emit only public data, enums, and eligible service clients.

### Reference Other Domains

After declaring an `import` in `.skel`, use repeatable `--skel-import domain=PATH` options to locate external contracts. When generating a Go module or TypeScript, map their language packages with `--go-import`, `--go-module-prefix`, or `--ts-import`. See the [CLI reference](https://yorun.ai/skelc/cli) for complete examples.

### Inspect and Format

```bash
skelc symbol list --skel-in ./skel
skelc symbol get demo.user.User --skel-in ./skel
skelc format --skel-in ./skel
```

`format` modifies files in place after validating all inputs. It applies one canonical style: four-space indentation, compact type and permission punctuation, one space after field and argument colons, compact empty blocks, and one blank line between top-level declarations. Declaration order and comment or string values are preserved. Tool integrations can request machine-readable diagnostics with the global `--log-format jsonl` option.

`check` recovers at declaration, block-member, closing-brace, and decorator boundaries and reports up to 50 independent syntax and semantic diagnostics per domain in one run. Invalid declarations are isolated so dependent errors do not cascade. JSONL diagnostics include a stable code, severity, exact range, related locations, and an optional fix suggestion.

`skelc lsp` provides recoverable syntax and workspace-wide semantic diagnostics, diagnostic quick fixes, editor formatting, keyword and type completion, declaration hover details, hierarchical document and workspace symbols, definition and reference navigation, and safe top-level declaration rename. Duplicate declarations include the first declaration as related information. Semantic analysis uses the current in-memory contents of every document, caches parsed syntax trees, and recalculates only changed domains and their reverse dependents. Superseded analysis is cancelled immediately.

## Programmatic API

Go programs can invoke generation through the root `go.yorun.ai/skelc` package without importing implementation packages:

```go
result, err := skelc.CompileGolang(
	skelc.Input{
		SkelIn: "./skel",
		SkelImports: map[string]string{
			"shared.types": "../shared/skel",
		},
	},
	skelc.GolangOption{
		Out:         "./generated/user-go",
		AsModule:    true,
		Module:      "example.com/generated/user",
		VineVersion: skelc.DefaultGolangVineVersion,
	},
)
if err != nil {
	return err
}
for _, diagnostic := range result.Diagnostics {
	log.Printf("%s [%s] %s", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
}
```

The API also provides `CompileTypeScript` and `CompileSkeleton`. Parser and loader warnings use the same structured diagnostic model instead of a separate string list. All public-contract generators consume one validated `internal/codegen/common` projection, preventing Go, Skel, and TypeScript visibility rules from drifting. Generation records owned files in `.skelc-manifest.json`, atomically replaces individual generated files, removes only unchanged stale generated files, and preserves every untracked file in a shared output directory.

Custom generators can call `skelc.Parse` and consume the returned `*model.Domain` through the parser-independent `go.yorun.ai/skelc/model` package. Parsed models already contain compatibility hashes calculated by skelc. Built-in generators accept the same parsed domain through `GenerateGolang`, `GenerateTypeScript`, and `GenerateSkeleton`, so several targets can share one parse result.

## skelc, Vine, and vRPC

skelc reads contracts and generates code; it is not the application runtime:

- Generated Go code uses runtime types and service infrastructure from `go.yorun.ai/vine`
- Generated TypeScript service clients use `@yorun-ai/vrpc`
- Runtime capabilities such as a CBOR codec are configured by the application when it creates a vRPC client; skelc does not bundle them into generated code

After upgrading skelc, regenerate the code and run type checks and tests in its consumers. Skel syntax, CLI behavior, diagnostic formats, and generated APIs are compatibility boundaries.

## Command Overview

```text
check          validate Skel definitions
format         format Skel definitions in place
lsp            run the Skel language server over stdio
symbol         list or inspect top-level symbols
gen skel       generate public Skel contracts
gen go         generate code inside an existing Go module
gen go-module  generate a standalone Go module
gen ts         generate TypeScript types and clients
version        show skelc and default Vine version information
```

Run `skelc --help` or `skelc <command> --help` for all options supported by the installed version.

## Documentation

- [Skel language reference](https://yorun.ai/skelc/syntax)
- [CLI reference](https://yorun.ai/skelc/cli)
- [TypeScript generation](https://yorun.ai/skelc/generation/typescript)
- [Changelog](CHANGELOG.md)
- [Documentation site source](https://github.com/yorun-ai/vine-doc)
- [Editor support and VS Code extension](https://github.com/yorun-ai/skel-editor-support)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and [AGENTS.md](AGENTS.md) for the repository layout and development conventions.

skelc follows [Semantic Versioning](https://semver.org/) and is open source under the [Apache License 2.0](LICENSE).
