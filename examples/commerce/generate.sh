#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
output_root=${1:-"$script_dir/generated"}
identity_skel="$script_dir/identity/skel"
order_skel="$script_dir/order/skel"
identity_public="$output_root/public/identity"
identity_go_public="$output_root/go/identitypub"

cd "$repo_root"

# Build the CLI once; invoking it through `go run` would rebuild it every time.
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT
GOWORK=off go build -o "$build_dir/skelc" ./cmd/skelc
skelc="$build_dir/skelc"

"$skelc" check \
  --skel-in "$identity_skel"

"$skelc" gen skel \
  --pub \
  --skel-in "$identity_skel" \
  --skel-out "$identity_public"

"$skelc" gen go-module \
  --skel-in "$identity_skel" \
  --go-out "$output_root/go/identity" \
  --go-module example.com/yorun/commerce/identity \
  --go-pub-out "$identity_go_public" \
  --go-pub-module example.com/yorun/commerce/identitypub

"$skelc" gen ts --api \
  --skel-in "$identity_skel" \
  --ts-out "$output_root/typescript/identity" \
  --ts-as-module \
  --ts-module @yorun-example/commerce-identity

"$skelc" check \
  --skel-in "$order_skel"

"$skelc" gen skel \
  --pub \
  --skel-in "$order_skel" \
  --skel-import "identity.user=$identity_skel" \
  --skel-out "$output_root/public/order"

"$skelc" gen go-module \
  --skel-in "$order_skel" \
  --skel-import "identity.user=$identity_skel" \
  --go-import identity.user=example.com/yorun/commerce/identitypub \
  --go-out "$output_root/go/order" \
  --go-module example.com/yorun/commerce/order

"$skelc" gen ts --api \
  --skel-in "$order_skel" \
  --skel-import "identity.user=$identity_skel" \
  --ts-import identity.user=@yorun-example/commerce-identity \
  --ts-out "$output_root/typescript/order" \
  --ts-as-module \
  --ts-module @yorun-example/commerce-order

test -f "$identity_public/domain.skel"
test -f "$output_root/public/order/event.skel"
test -f "$output_root/go/identity/go.mod"
test -f "$identity_go_public/go.mod"
test -f "$output_root/go/order/go.mod"
test -f "$output_root/go/order/config.go"
test -f "$output_root/go/order/event.go"
test -f "$output_root/go/order/resource.go"
test -f "$output_root/go/order/task.go"
test -f "$output_root/go/order/web.go"
test -f "$output_root/typescript/identity/package.json"
test -f "$output_root/typescript/order/package.json"

echo "Generated commerce example outputs in $output_root"
