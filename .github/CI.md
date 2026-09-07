# CI maintenance

`ci.yml` runs three independent jobs on pull requests and pushes to main:

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

PR updates cancel older runs for that PR. Main runs are grouped by commit and
are not cancelled by later pushes, preserving validation of release candidates.
Each check has a 15-minute timeout; the final gate has a 5-minute timeout.

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
