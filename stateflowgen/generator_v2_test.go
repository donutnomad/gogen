package stateflowgen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donutnomad/gogen/plugin"
)

func TestStateFlowV2Generator_Run(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "types.go")
	source := `package wallet

// @StateFlowV2(name="Wallet", output=types_state.go)
// @Flow: initial => [ creating? via waiting_approval else rejected ]
// @Flow: creating => [ active ]
// @Flow: active => [ (=)? via update_profile_waiting_commit_approval else active ]
// @Flow: active => [ inactive? via disable_wallet_waiting_commit_approval else active ]
const _ = ""
`
	if err := os.WriteFile(sourceFile, []byte(source), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry := plugin.NewRegistry()
	if err := registry.Register(NewStateFlowV2Generator()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := plugin.Run(context.Background(), registry, "", tmpDir); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output, err := os.ReadFile(filepath.Join(tmpDir, "types_state.go"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(output), "UpdateProfileWaitingCommitApproval: WalletVia(\"update_profile_waiting_commit_approval\")") {
		t.Fatalf("generated file missing transient via:\n%s", string(output))
	}
}
