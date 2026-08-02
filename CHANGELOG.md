# Changelog

All notable changes to skelc are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public version history starts at `v0.9.0`; versions from the former private repository are not part of the public release history.

## [Unreleased]

### Added

- Public diagnostic types and stable code constants through
  `go.yorun.ai/skelc/diagnostic` and the root API
- `format --check` and machine-readable format results through
  `--output-format json`

### Changed

- Compiler validation and compatibility-hash failures now propagate explicit
  errors instead of relying on panic-based checks
- Formatter lexer failures now propagate explicitly to CLI, LSP, and generated
  Skel callers

### Fixed

- Semantic diagnostic codes and naming suggestions are now assigned from
  structured analyzer metadata instead of parsed from human-readable messages
- `format` stages all changed files before committing them and restores files
  already replaced if a later write fails
- Formatter transactions preserve ownership, modes, supported extended
  metadata, and synchronize parent directories before reporting success
- Multiline block comments retain their relative indentation when rebased
- Multi-target generation stages every output before committing and restores
  all affected targets when a commit fails
- `@deprecated` completion inserts its required reason and uses a snippet
  placeholder when the language client supports snippets

## [0.10.3] - 2026-07-30

### Fixed

- Language-server semantic analysis now matches `check`: it merges same-domain
  files only within one source directory, leaves imports unresolved, and keeps
  same-named domain instances in separate directories independent
- Decorator completion now filters and replaces prefixes after `@` correctly
  and only suggests decorators accepted for the following declaration, block,
  field, or argument; decorators already present on the same target are omitted

## [0.10.2] - 2026-07-30

### Added

- Context-aware completion for actor transports and config lifecycle values
- Remote workspace URI preservation and dynamic workspace-folder indexing in the language server

## [0.10.1] - 2026-07-29

### Added

- `@deprecated("reason")` metadata for declarations and their deprecatable
  elements, including generated Vine schemas, Go docs, TypeScript JSDoc,
  public Skel output, and LSP presentation
- UUID map keys in contracts, generated Go maps, TypeScript records, and public Skel output

### Changed

- Generated Go code now requires at least `go.yorun.ai/vine v0.10.1`, which is
  also the default dependency version for generated Go modules
- Documentation links now target the public Skel site and editor-support repository

## [0.10.0] - 2026-07-26

### Added

- `@sensitive` metadata across Skel contracts, generated schemas and code, and
  public contract projection

### Fixed

- Release binaries are built from the checked-out release tag

## [0.9.5] - 2026-07-23

### Added

- Complete transitive Skel import-graph analysis

### Fixed

- Normalize imported enum, data, generic, and named-type references before code generation

## [0.9.4] - 2026-07-22

### Fixed

- Harden formatter and diagnostic edge cases
- Suggest corrected names for invalid identifiers

## [0.9.3] - 2026-07-22

### Added

- LSP document formatting, keyword and type completion, and declaration hover details
- Hierarchical document symbols, workspace symbol search, and top-level declaration rename
- Best-effort domain, import, and top-level declaration indexing while a document has syntax errors
- Debounced workspace-wide semantic diagnostics over unsaved documents, including same-domain file merging and cross-domain validation
- Collection of multiple compiler diagnostics in a single run

### Changed

- Compiler validation aborts now carry structured error codes, source positions, and wrapped causes through centralized API and CLI recovery boundaries
- Analyzer validation now reports errors explicitly instead of using panic/recover control flow; `check` and LSP collect up to 50 independent diagnostics per domain while suppressing errors that only depend on invalid declarations

### Fixed

- Prevent analyzer panics when syntax recovery produces an incomplete nested permission expression
- Preserve formatter idempotence and relative indentation for multiline comments and triple-quoted strings
- Calculate parser diagnostic and LSP ranges correctly when non-ASCII characters precede a token on the same line

## [0.9.2] - 2026-07-22

### Added

- Automated publication of release binaries

### Changed

- Documentation now links to the independent editor-support repository

## [0.9.1] - 2026-07-21

### Added

- Language Server Protocol support for syntax diagnostics, document symbols,
  definitions, and references

## [0.9.0] - 2026-07-21

Initial public release.

### Included

- Skel parsing, validation, formatting, and symbol inspection
- Go source and standalone Go module generation
- TypeScript type, package, and vRPC service client generation
- Public Skel contract extraction for cross-domain sharing
- Binary-aware sparse vRPC wire-schema generation for TypeScript clients
