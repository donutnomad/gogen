package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFormatPreservesMajorVersionImportPath(t *testing.T) {
	src := []byte(`package test

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
)

var metadata = bind.MetaData{}
`)

	outFile := filepath.Join(t.TempDir(), "generated.go")
	if err := WriteFormat(outFile, src); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	output := string(out)
	if !strings.Contains(output, `"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"`) {
		t.Fatalf("formatted output missing bind/v2 import:\n%s", output)
	}
}

func TestPackageNameFromImportPathMajorVersion(t *testing.T) {
	got := packageNameFromImportPath("github.com/ethereum/go-ethereum/accounts/abi/bind/v2")
	if got != "bind" {
		t.Fatalf("package name = %q, want bind", got)
	}
}
