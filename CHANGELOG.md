# Changelog

All notable changes to skelc are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public version history starts at `v0.9.0`; versions from the former private repository are not part of the public release history.

## [Unreleased]

### Added

- LSP document formatting, keyword and type completion, and declaration hover details
- Hierarchical document symbols, workspace symbol search, and top-level declaration rename
- Best-effort domain, import, and top-level declaration indexing while a document has syntax errors
- Debounced workspace-wide semantic diagnostics over unsaved documents, including same-domain file merging and cross-domain validation

### Changed

- Generated Go modules now target `go.yorun.ai/vine v0.9.3` by default
- Parsing and compilation now analyze the complete transitive Skel import graph while generating code only for the target domain
- Compiler validation aborts now carry structured error codes, source positions, and wrapped causes through centralized API and CLI recovery boundaries
- Analyzer validation now reports errors explicitly instead of using panic/recover control flow; `check` and LSP collect up to 50 independent diagnostics per domain while suppressing errors that only depend on invalid declarations

### Fixed

- Resolve enum, data, and generic type references declared inside imported domains before code generation
- Resolve named types used by actor authentication info, enforce its implicit-data rules, and keep type parameters scoped to their declaring data
- Prevent analyzer panics when syntax recovery produces an incomplete nested permission expression
- Preserve formatter idempotence and relative indentation for multiline comments and triple-quoted strings
- Calculate parser diagnostic and LSP ranges correctly when non-ASCII characters precede a token on the same line

## [0.9.0] - 2026-07-21

Initial public release.

### Included

- Skel parsing, validation, formatting, and symbol inspection
- Go source and standalone Go module generation
- TypeScript type, package, and vRPC service client generation
- Public Skel contract extraction for cross-domain sharing
- Binary-aware sparse vRPC wire-schema generation for TypeScript clients
- Language Server Protocol support for syntax diagnostics, document symbols, definitions, and references
- VS Code syntax highlighting and LSP-powered language features
