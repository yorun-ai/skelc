package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/testutil"
)

func TestOpenServicePublicServer(t *testing.T) {
	testutil.RequireToolchain(t)
	root := t.TempDir()
	input := filepath.Join(root, "service.skel")
	source := `domain demo.storage
 data Item { value: string }
 open service StorageService {
   method get { input { key: string } output Item }
 }
 pub service LookupService { method ping {} }
 `
	writeFileForTest(t, input, source)
	skelOut := filepath.Join(root, "skel")
	if _, err := skelc.CompileSkeleton(skelc.Input{SkelIn: input, Strict: true}, skelc.SkeletonOption{Out: skelOut, PubOnly: true}); err != nil {
		t.Fatal(err)
	}
	pub := filepath.Join(root, "storagepub")
	regular := filepath.Join(root, "storage")
	if _, err := skelc.CompileGolang(skelc.Input{SkelIn: input, Strict: true}, skelc.GolangOption{CompilerVersion: "v0.19.0", Out: regular, Module: "example.com/storage", PubOut: pub, PubModule: "example.com/storagepub", AsModule: true}); err != nil {
		t.Fatal(err)
	}
	service, err := os.ReadFile(filepath.Join(pub, "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(service), "type StorageServiceServer interface") || strings.Contains(string(service), "type LookupServiceServer interface") {
		t.Fatalf("wrong public servers: %s", service)
	}
	testSource := `package storage_test
import (
 "reflect"
 "testing"
 storage "example.com/storage"
 pub "example.com/storagepub"
 "go.yorun.ai/vine/core/ex"
)
type implementation struct { pub.DefaultStorageServiceServer }
func (*implementation) Get(key string) pub.Item { return pub.Item{Value:key} }
type errorImplementation struct { pub.DefaultStorageServiceServerER }
func (*errorImplementation) Get(key string) (pub.Item, ex.Error) { return pub.Item{Value:key}, nil }
var _ pub.StorageServiceServer = (*implementation)(nil)
var _ storage.StorageServiceServer = (*implementation)(nil)
var _ pub.StorageServiceServerER = (*errorImplementation)(nil)
var _ storage.StorageServiceServerER = (*errorImplementation)(nil)
func TestPublicServer(t *testing.T) {
 if reflect.TypeFor[storage.StorageServiceServer]() != reflect.TypeFor[pub.StorageServiceServer]() { t.Fatal("server types differ") }
 var server storage.StorageServiceServer = &implementation{}
 if server.Get("ok").Value != "ok" { t.Fatal("wrong result") }
}
`
	writeFileForTest(t, filepath.Join(regular, "open_test.go"), testSource)

	for _, pubOnly := range []bool{false, true} {
		out := filepath.Join(root, "full")
		if pubOnly {
			out = filepath.Join(root, "public-only")
		}
		if _, err := skelc.CompileGolang(skelc.Input{SkelIn: skelOut, Strict: true}, skelc.GolangOption{CompilerVersion: "v0.19.0", Out: out, Module: "example.com/standalone", AsModule: true, PubOnly: pubOnly}); err != nil {
			t.Fatal(err)
		}
		testutil.Go(t, out, "build", "-mod=mod", "./...")
	}
	testutil.Go(t, pub, "build", "-mod=mod", "./...")
	testutil.Go(t, regular, "mod", "edit", "-replace=example.com/storagepub="+pub)
	testutil.Go(t, regular, "test", "-mod=mod", "./...")
}
