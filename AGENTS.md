# Skelc Agent Guidelines

## Working in the Repository

- Read the root `README.md` and the applicable documentation under `yorun-ai/vine-doc`'s `content/skelc` before changing Skel syntax, CLI behavior, or generated output.
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
- `internal/loader` discovers and loads Skel source files. `internal/parser` coordinates imports and owns its `grammar`, `analyzer`, and `hasher` stages: grammar parses syntax, analyzer directly builds and validates public semantic model objects, and hasher derives compatibility hashes. The public `model` package contains only parser-independent semantic data and source positions.
- `internal/codegen/{golang,skeleton,typescript}` own generated Go, Skel, and TypeScript output. The `internal/codegen` root package provides shared rendering and generated-text helpers without depending on a target generator.
- `internal/formatter` owns pure Skel source formatting. The CLI owns in-place formatting and must validate all applicable inputs before writing files so a failed operation does not leave a partially updated source tree.
- Keep implementation packages under `internal` unless they form part of the supported programmatic API. The root `skelc` facade exposes parsing and generation, while `model` exposes parser-independent semantic data required by custom generators. Do not import parser grammar or internal implementation packages from `model`.

## Language and Compatibility

- Treat the Skel grammar, accepted legacy syntax, diagnostics, CLI flags, exit codes, JSON/JSONL fields, generated filenames, generated APIs, and generated module metadata as public compatibility boundaries.
- When changing Skel syntax, update the grammar, semantic model, formatter, generators, tests, and the applicable `vine-doc/content/skelc/language/syntax.md` and `vine-doc/content/skelc/reference/cli.md` pages.
- When changing generated code, update every affected language backend and golden or structural tests. Confirm that generated Go code remains compatible with the declared Vine version.
- Keep deterministic behavior: input discovery, symbols, imports, dependencies, diagnostics, and generated files must have stable ordering.
- Do not add silent recovery for invalid contracts. Diagnostics should identify the relevant source path and location whenever available.

## Generated and Packaged Artifacts

- Modify generator templates under the relevant `internal/codegen/{golang,skeleton,typescript}` package rather than patching expected generated output behavior elsewhere.
- Treat `tool/vscode-skel/dist/*.vsix` as release artifacts. Change the extension sources and manifest under `tool/vscode-skel`, validate them, and rebuild the package only when the task explicitly includes updating the release artifact.
- Do not commit temporary generated projects, test output, coverage files, editor settings, dependency directories, or local workspace files.

## Documentation

- Keep `README.md` and `README.zh-CN.md` synchronized, including language-switch links, commands, compatibility notes, and license information.
- `vine-doc/content/skelc/language/syntax.md` is the detailed Skel language reference; `vine-doc/content/skelc/reference/cli.md` is the detailed CLI reference. Keep their English translations under `vine-doc/i18n/en/docusaurus-plugin-content-docs-skelc/current` synchronized.
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
- Run `npm run check` in `tool/vscode-skel` after changing the VS Code extension, grammar, language configuration, or theme.
- Run `pnpm build` in `vine-doc` after changing skelc user-facing documentation there.
- For CLI, syntax, or generator changes, exercise at least one representative `skelc check` or `skelc gen` flow in addition to automated tests.
