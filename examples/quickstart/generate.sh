#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
output_root=${1:-"$script_dir/generated"}

cd "$repo_root"

# Build the CLI once; invoking it through `go run` would rebuild it every time.
# -buildvcs=false keeps the reported compiler version at v0.0.0-dev, as `go run`
# does, instead of the VCS pseudo-version that `go build` would otherwise stamp.
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT
GOWORK=off go build -buildvcs=false -o "$build_dir/skelc" ./cmd/skelc
skelc="$build_dir/skelc"

"$skelc" check \
  --skel-in "$script_dir/skel"

"$skelc" gen skel \
  --pub \
  --skel-in "$script_dir/skel" \
  --skel-out "$output_root/skel"

"$skelc" gen go-module \
  --skel-in "$script_dir/skel" \
  --go-out "$output_root/go" \
  --go-module example.com/yorun/quickstart

"$skelc" gen ts --api \
  --skel-in "$script_dir/skel" \
  --ts-out "$output_root/typescript" \
  --ts-as-module \
  --ts-module @yorun-example/quickstart

test -f "$output_root/skel/domain.skel"
test -f "$output_root/skel/types.skel"
test -f "$output_root/go/go.mod"
test -f "$output_root/typescript/package.json"

echo "Generated quickstart outputs in $output_root"
