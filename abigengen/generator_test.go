package abigengen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/plugin"
)

func TestAbigenGeneratorUsesRelativeArtifactDefaults(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "bindings")
	artifactDir := filepath.Join(root, "artifacts")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(pkgDir, "generate.go")
	if err := os.WriteFile(src, []byte(`package bindings

//go:gogen @Abigen(abi=../artifacts/escrow_pack.json)
`), 0600); err != nil {
		t.Fatal(err)
	}

	artifact := filepath.Join(artifactDir, "escrow_pack.json")
	if err := os.WriteFile(artifact, []byte(`{
		"_format": "hh-sol-artifact-1",
		"contractName": "Escrow",
		"abi": [{"type":"function","name":"coordinator","inputs":[],"outputs":[{"type":"address"}],"stateMutability":"view"}],
		"bytecode": "0x60016002"
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	registry := plugin.NewRegistry()
	if err := registry.Register(NewAbigenGenerator()); err != nil {
		t.Fatal(err)
	}

	_, err := plugin.RunWithOptionsAndStats(context.Background(), &plugin.RunOptions{
		Registry: registry,
		Patterns: []string{pkgDir},
		Output:   "",
		Async:    false,
	})
	if err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(pkgDir, "escrow_pack.go"))
	if err != nil {
		t.Fatal(err)
	}
	output := string(generated)
	for _, want := range []string{
		"package bindings",
		"type Escrow struct",
		`Bin: "0x60016002"`,
		`github.com/ethereum/go-ethereum/accounts/abi/bind/v2`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated code missing %q", want)
		}
	}
}

func TestAbigenGeneratorOverridesOutputTypeAndPackage(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "bindings")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(pkgDir, "generate.go")
	if err := os.WriteFile(src, []byte(`package bindings

//go:gogen @Abigen(abi=escrow_pack.json,output=custom_escrow,name=EscrowPkg,pkg=chainpack)
`), 0600); err != nil {
		t.Fatal(err)
	}

	abiFile := filepath.Join(pkgDir, "escrow_pack.json")
	if err := os.WriteFile(abiFile, []byte(`[{"type":"function","name":"approve","inputs":[],"outputs":[]}]`), 0600); err != nil {
		t.Fatal(err)
	}

	registry := plugin.NewRegistry()
	if err := registry.Register(NewAbigenGenerator()); err != nil {
		t.Fatal(err)
	}

	_, err := plugin.RunWithOptionsAndStats(context.Background(), &plugin.RunOptions{
		Registry: registry,
		Patterns: []string{pkgDir},
		Output:   "",
		Async:    false,
	})
	if err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(pkgDir, "custom_escrow.go"))
	if err != nil {
		t.Fatal(err)
	}
	output := string(generated)
	for _, want := range []string{
		"package chainpack",
		"type EscrowPkg struct",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated code missing %q", want)
		}
	}
}
