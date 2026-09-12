# CI maintenance

`ci.yml` classifies pull request changes, then runs three independent jobs when
the change affects code, dependencies, examples, or CI configuration. Markdown,
LICENSE, and issue-template-only changes skip those jobs while the required gate
still completes successfully:

| Job | Checks |
| --- | --- |
| Go static | Module metadata drift and full-repository `go vet` |
| Go race | Full-repository tests with the race detector |
| Examples | Generate both examples and compile their generated Go modules |

All Go commands use `GOWORK=off`, including commands in generated modules.
Race tests disable Go's automatic vet pass because the static job runs the full
vet check. `GORACE=atexit_sleep_ms=0` removes the race runtime's exit delay;
it does not disable race detection or change its failure exit code.

Keep `CI / Required Checks` as the single required gate. It runs even when
another job fails and requires all three jobs to succeed; failures,
cancellations, and skipped jobs must not pass. There are no path filters or
reduced package lists, so changes to templates and fixtures retain full coverage.

PR updates cancel older runs for that PR. There is no duplicate post-merge run
on `main`; the branch ruleset requires strict required checks and squash merging,
so the merged content has already passed this full suite. Each check has a
15-minute timeout; the final gate has a 5-minute timeout.

This split lets static and example failures surface without waiting for race
tests. It adds runner setup work, so compare actual run durations and runner
minutes before splitting further. The release workflow is separate and is not
changed by this CI layout.

Validate workflow edits with:

```bash
GOWORK=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/ci.yml
git diff --check
```

When editing a job's commands, also execute those commands locally and keep
generated example output outside the repository.
