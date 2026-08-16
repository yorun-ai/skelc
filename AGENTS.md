# Skelc Agent Guidelines

## Working in the Repository

- Read the root `README.md` and the applicable documentation under `yorun-ai/skel-site`'s `docs` before changing Skel syntax, CLI behavior, or generated output.
- Keep changes within the parser and generator boundaries described below. Update documentation when a change alters those boundaries or user-visible behavior.
- Preserve existing user changes in the worktree and keep unrelated refactoring out of focused changes.

## Go Version and Syntax

- Target Go 1.26 syntax. Prefer `new` with a composite literal when creating a pointer, for example: `option := new(SomeOption{Field: value})`.
- Use `kind` when `type` would otherwise be the natural local variable name.
- Prefix unexported package-local production type declarations with `_`, such as `_Parser` and `_Option`. This applies only to types; do not prefix unexported constants, variables, functions, or methods with `_`. Test fixture types may use descriptive lowercase names.
- Use `Rpc`, not `RPC`, in identifiers and generated Go APIs.

## Architecture Boundaries

- `cmd/skelc` is the executable entry point; keep it thin and delegate CLI behavior to `internal/cli`.
- `internal/cli` owns command definitions, flag-specific validation, terminal output, and exit codes. Generation commands call the root `skelc` API; input normalization, target-option normalization, and output-directory lifecycle must not be duplicated in CLI code.
- `internal/loader` discovers and loads Skel source files. `internal/parser` owns strict syntax parsing, parser-library error normalization, recoverable source segmentation, and its `grammar` subpackage, which contains the Participle grammar and syntax-tree representation. `internal/analyzer` builds and validates semantic model objects from that syntax tree, while `internal/hasher` derives compatibility hashes. `internal/compiler` coordinates loading, recovery policy, diagnostics, import resolution, semantic analysis, hashes, and incremental workspace analysis. `internal/model` owns the parser-independent semantic model implementation; the public `model` package is its documented facade for custom generators. `internal/schema` owns schema wire types, projection, validation, diffing, and source/Git baseline coordination; the public `schema` package is its documented facade. Public facades must not expose unrelated internal packages.
- `internal/codegen/{golang,skeleton,typescript}` own generated Go, Skel, and TypeScript output. `internal/codegen/golang/vineschema` adapts the canonical `internal/schema` projection to Vine's runtime registry schema and adds only runtime metadata such as hashes and generated services. `internal/codegen/common` provides public-contract projection, rendering, validation, and generated-text helpers without depending on a target generator. `internal/codegen/output` owns managed multi-target output transactions.
- `internal/formatter` owns pure Skel source formatting. The CLI owns in-place formatting and must validate all applicable inputs before writing files so a failed operation does not leave a partially updated source tree.
- Keep implementation packages under `internal` unless they form part of the supported programmatic API. The root `skelc` facade exposes parsing and generation, while `model` exposes parser-independent semantic data required by custom generators. Keep public facade packages limited to aliases, constants, and narrowly scoped function forwarding to their matching implementation package.

## Language and Compatibility

- Treat the Skel grammar, accepted legacy syntax, diagnostics, CLI flags, exit codes, JSON/JSONL fields, generated filenames, generated APIs, and generated module metadata as public compatibility boundaries.
- When changing Skel syntax, update the grammar, semantic model, formatter, generators, tests, and the applicable `skel-site/docs/language/syntax.md` and `skel-site/docs/reference/cli.md` pages.
- When changing generated code, update every affected language backend and golden or structural tests. Confirm that generated Go code remains compatible with the declared Vine version.
- Keep deterministic behavior: input discovery, symbols, imports, dependencies, diagnostics, and generated files must have stable ordering.
- Do not add silent recovery for invalid contracts. Diagnostics should identify the relevant source path and location whenever available.

## Generated Artifacts

- Modify generator templates under the relevant `internal/codegen/{golang,skeleton,typescript}` package rather than patching expected generated output behavior elsewhere.
- Editor integrations live in the independent `yorun-ai/skel-editor-support` repository. Keep editor client code and Marketplace packaging out of skelc; coordinate LSP compatibility across the two repositories.
- Do not commit temporary generated projects, test output, coverage files, editor settings, dependency directories, or local workspace files.

## Documentation

- Keep `README.md` and `README.zh-CN.md` synchronized, including language-switch links, commands, compatibility notes, and license information.
- `skel-site/docs/language/syntax.md` is the detailed English Skel language
  reference; `skel-site/docs/reference/cli.md` is the detailed English CLI
  reference. Keep their Simplified Chinese translations under
  `skel-site/i18n/zh-CN/docusaurus-plugin-content-docs/current` synchronized.
- Keep examples executable against the current CLI and syntax. Avoid documenting planned commands or unsupported flags.

## Tests

- Keep implementation tests paired with their source files. Shared setup may live in a narrowly scoped test helper file.
- Use `t.TempDir` for filesystem tests and `t.Cleanup` to restore modified globals or environment variables.
- Do not write test output into repository source directories.
- Add parser and formatter coverage for whitespace, comments, source locations, invalid input, and round trips when relevant.
- Add generator coverage for deterministic output and all affected declaration kinds when changing templates or rendering behavior.

## Validation

- Run `gofmt` on changed Go files and run `git diff --check`.
- Run targeted package tests while iterating, then run `GOWORK=off go test ./...` for repository-wide Go changes so an enclosing workspace cannot replace published dependencies.
- Run `GOWORK=off go vet ./...` after changes involving exported APIs, reflection, filesystem safety, or CLI/runtime wiring.
- Run `pnpm build` in `skel-site` after changing skelc user-facing documentation there.
- For CLI, syntax, or generator changes, exercise at least one representative `skelc check` or `skelc gen` flow in addition to automated tests.
