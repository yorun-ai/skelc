#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
output_root=${1:-"$script_dir/generated"}

cd "$repo_root"

GOWORK=off go run ./cmd/skelc check \
  --skel-in "$script_dir/skel"

GOWORK=off go run ./cmd/skelc gen skel \
  --pub \
  --skel-in "$script_dir/skel" \
  --skel-out "$output_root/skel"

GOWORK=off go run ./cmd/skelc gen go-module \
  --skel-in "$script_dir/skel" \
  --go-out "$output_root/go" \
  --go-module example.com/yorun/quickstart

GOWORK=off go run ./cmd/skelc gen ts \
  --skel-in "$script_dir/skel" \
  --ts-out "$output_root/typescript" \
  --ts-as-module \
  --ts-module @yorun-example/quickstart

test -f "$output_root/skel/domain.skel"
test -f "$output_root/skel/types.skel"
test -f "$output_root/go/go.mod"
test -f "$output_root/typescript/package.json"

echo "Generated quickstart outputs in $output_root"
