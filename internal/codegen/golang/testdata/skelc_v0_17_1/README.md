# skelc v0.17.1 compatibility fixture

The files in `providerpub` were generated without manual edits from `provider.skel`
using the skelc `v0.17.1` source tag (commit `e93d83c6d991df6dbcb90b58975425dc1a82e2ef`).

Reproduce with that checkout and `GOWORK=off`:

```sh
go run -ldflags='-X go.yorun.ai/skelc/internal/cli.ldModuleVersion=v0.17.1' ./cmd/skelc gen go-module --skel-in /path/to/provider.skel --go-out /tmp/provider --go-module example.com/generated/current --go-pub-out /path/to/providerpub --go-pub-module example.com/generated/currentpub --go-vine-version v0.15.7
```

Do not regenerate with the current compiler. Tests copy this module into a temporary
directory and combine it with current generated consumers and the minimum Vine runtime.
