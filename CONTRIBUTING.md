# Contributing to skelc

Thank you for contributing to skelc. This guide describes the development workflow and the checks expected before a change is submitted.

## Before You Start

- For substantial Skel syntax, generated API, CLI, or architecture changes, open an issue or discussion before investing in an implementation.
- Read [AGENTS.md](AGENTS.md) for repository-wide coding, compatibility, documentation, and testing rules.
- Read the [Skel language reference](https://github.com/yorun-ai/vine-doc/blob/main/content/skelc/language/syntax.md) and the [CLI reference](https://github.com/yorun-ai/vine-doc/blob/main/content/skelc/reference/cli.md) when changing behavior covered by either document.
- Keep each pull request focused on one coherent change. Separate unrelated refactoring, formatting, and behavior changes.

## Prerequisites

The Go module targets Go 1.26.

Download Go dependencies and run the baseline test suite with:

```bash
GOWORK=off go mod download
GOWORK=off go test ./...
```

## Repository Boundaries

The source-processing pipeline is intentionally separated:

1. `internal/loader` discovers and loads source files.
2. `internal/parser` coordinates imports and parses grammar.
3. `internal/parser/{grammar,analyzer,hasher}` parse syntax, build and validate semantic state, and derive compatibility hashes; the public `model` package contains parser-independent semantic data.
4. `internal/codegen/{golang,skeleton,typescript}` renders Go, public Skel, and TypeScript output.
5. The root `skelc` API normalizes inputs and target options and manages output-directory lifecycle. `internal/cli` maps flags to that API and exposes stable terminal output and exit codes.

Keep the executable under `cmd/skelc` thin. Implementation packages remain under `internal`; avoid exposing their types through the CLI package.

## Compatibility

Treat the following as user-facing compatibility boundaries:

- accepted Skel syntax and migrations
- diagnostics and source locations
- CLI command names, flags, output, and exit codes
- JSON and JSONL fields consumed by tools
- generated filenames, APIs, imports, and module metadata
- ordering and formatting of generated output

Changes to one boundary often require coordinated parser, formatter, generator, test, and documentation updates. Describe compatibility impact and required regeneration or manual migration steps in the pull request.

## Go Changes

- Format changed Go files with `gofmt`.
- Follow the naming and implementation rules in [AGENTS.md](AGENTS.md).
- Keep implementation tests paired with their source files.
- Use `t.TempDir` for generated fixtures and avoid writing test output into the repository.
- Preserve deterministic ordering for inputs, symbols, dependencies, diagnostics, and generated files.

While iterating, run the narrowest relevant package tests:

```bash
go test ./internal/parser/...
go test ./internal/codegen/...
go test ./internal/cli
```

Before submitting a repository-wide Go change, run:

```bash
GOWORK=off go test ./...
```

Also run `GOWORK=off go vet ./...` after changes involving exported APIs, reflection, filesystem safety, or CLI/runtime wiring.

## Language and Generator Changes

For Skel syntax changes:

- add valid and invalid parser coverage
- update formatting behavior where applicable
- verify source-aware diagnostics
- update every affected generator
- update the applicable language and CLI references in `vine-doc/content/skelc`

For generated output changes, modify the templates and generator implementation under the relevant `internal/codegen/{golang,skeleton,typescript}` package, inspect representative generated output, and test its deterministic shape. Generated Go modules must declare a compatible Vine version.

The generator cleans output directories by default. Use temporary directories in manual tests and verify the target before running a generation command against an existing directory.

## Documentation

Keep [README.md](README.md) and [README.zh-CN.md](README.zh-CN.md) synchronized. Ensure commands and Skel examples remain valid against the current implementation.

The detailed references are maintained in
[`yorun-ai/vine-doc`](https://github.com/yorun-ai/vine-doc) repository:

- Skel language: `content/skelc/language/syntax.md`
- CLI behavior: `content/skelc/reference/cli.md`

After documentation changes, build both locales from `vine-doc`:

```bash
pnpm install
pnpm build
```

Update both when a change affects both language semantics and command behavior.

## VS Code Extension

The extension is maintained in the independent
[`yorun-ai/vscode-skel`](https://github.com/yorun-ai/vscode-skel) repository.
Changes to `skelc lsp` capabilities should document their compatibility impact
and be validated with the extension client, but extension source and packaged
`.vsix` files do not belong in this repository.

## Runnable Example

The repository includes a complete contract under `examples/quickstart`. Exercise validation and all generators without writing output into the source tree:

```bash
./examples/quickstart/generate.sh /tmp/skelc-quickstart
```

The default output directory under the example is ignored by Git. CI passes an explicit temporary directory.

## Pull Request Checklist

Before submitting a pull request, confirm that:

- The change is focused and follows the repository rules.
- Changed Go files are formatted and `git diff --check` passes.
- Relevant targeted tests and `go test ./...` pass.
- `go vet ./...` has been run when applicable.
- CLI or generator changes were exercised with a representative input.
- Generated output is deterministic and its diff was reviewed.
- English and Chinese root READMEs and detailed references are updated where applicable.
- Compatibility impact and migration or regeneration requirements are documented.
- No credentials, local paths, editor files, dependency directories, or temporary generated output are included.

## License

Unless explicitly stated otherwise, any contribution intentionally submitted for inclusion in skelc is licensed under the terms and conditions of the [Apache License 2.0](LICENSE), in accordance with Section 5 of the license.

By submitting a contribution, you represent that you have the right to submit it under these terms.
