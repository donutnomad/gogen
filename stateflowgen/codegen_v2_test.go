package stateflowgen

import (
	"strings"
	"testing"
)

func TestStateFlowV2CodeGenerator_GeneratesTransientStageAPI(t *testing.T) {
	model := mustBuildWalletV2Model(t)

	gen, err := NewCodeGeneratorV2(model, "wallet").Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	output := gen.String()

	wantFragments := []string{
		"type WalletState struct",
		"Status WalletStatus `json:\"status\"`",
		"Via WalletVia `json:\"via\"`",
		"Transition *WalletTransition `json:\"transition,omitempty\"`",
		"type WalletTransition struct",
		"From WalletStatus `json:\"from\"`",
		"To WalletStatus `json:\"to\"`",
		"Fallback WalletStatus `json:\"fallback\"`",
		"type WalletStateColumns struct",
		"Via WalletVia `gorm:\"column:via\" json:\"via\"`",
		"TransitionFrom *string `gorm:\"column:transition_from\" json:\"transition_from\"`",
		"TransitionTo *string `gorm:\"column:transition_to\" json:\"transition_to\"`",
		"TransitionFallback *string `gorm:\"column:transition_fallback\" json:\"transition_fallback\"`",
		"func (s WalletState) TransitionTo(to WalletStatus, withApproval bool) (WalletState, error)",
		"func (s WalletState) Commit() (WalletState, error)",
		"func (s WalletState) Reject() (WalletState, error)",
		"func (s WalletState) IsTransitionPending() bool",
		"func (s WalletStatus) String() string",
		"func NewWalletStatus(raw string) WalletStatus",
		"type WalletStatus string",
		"type WalletVia string",
		"WalletStatusEnums = struct",
		"Initial: WalletStatus(\"initial\")",
		"WalletViaEnums = struct",
		"UpdateProfileWaitingCommitApproval: WalletVia(\"update_profile_waiting_commit_approval\")",
		"case WalletStatusEnums.Initial:",
		"case WalletStatusEnums.Active:",
		"case WalletStatusEnums.Active:",
		"return WalletState{Status: WalletStatusEnums.Initial, Via: WalletViaEnums.None}",
		"Via: WalletViaEnums.UpdateProfileWaitingCommitApproval",
	}

	for _, fragment := range wantFragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("generated output missing %q\n%s", fragment, output)
		}
	}

	assertOrder(t, output, "type WalletState struct", "type WalletStatus string")
	assertOrder(t, output, "func (s WalletState) Next()", "type WalletStatus string")

	for _, unexpected := range []string{
		"const (",
		"WalletStatusInitial",
		"WalletViaNone WalletVia",
		"WalletStage",
		"WalletPendingTransition",
		"`json:\"pending",
		"column:pending",
		"gorm.io/datatypes",
		"case WalletStatus(\"initial\")",
	} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("generated output contains unexpected %q\n%s", unexpected, output)
		}
	}
}

func TestStateFlowV2CodeGenerator_GeneratesNumericStatusValues(t *testing.T) {
	model, err := BuildStateFlowV2Model(&StateFlowV2Config{
		Name:       "Wallet",
		StatusType: "int32",
		StatusValues: map[string]string{
			"initial":  "0",
			"creating": "100",
			"active":   "101",
			"rejected": "300",
		},
	}, []*FlowRule{
		{
			Source: StateRef{Phase: "initial"},
			Targets: []TargetRef{{
				Phase:            "creating",
				ApprovalOptional: true,
				Via:              "waiting_approval",
				Else:             "rejected",
			}},
		},
		{
			Source:  StateRef{Phase: "creating"},
			Targets: []TargetRef{{Phase: "active"}},
		},
	})
	if err != nil {
		t.Fatalf("BuildStateFlowV2Model() error = %v", err)
	}

	gen, err := NewCodeGeneratorV2(model, "wallet").Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	output := gen.String()

	for _, fragment := range []string{
		"type WalletStatus int32",
		"Initial: WalletStatus(0)",
		"Creating: WalletStatus(100)",
		"Active: WalletStatus(101)",
		"Rejected: WalletStatus(300)",
		"Via: WalletViaEnums.WaitingApproval",
		"func NewWalletStatus(raw string) WalletStatus",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("generated output missing %q\n%s", fragment, output)
		}
	}
}

func assertOrder(t *testing.T, output, before, after string) {
	t.Helper()
	beforeIdx := strings.Index(output, before)
	afterIdx := strings.Index(output, after)
	if beforeIdx == -1 || afterIdx == -1 {
		t.Fatalf("order fragments missing: before=%q after=%q\n%s", before, after, output)
	}
	if beforeIdx > afterIdx {
		t.Fatalf("expected %q before %q\n%s", before, after, output)
	}
}

func mustBuildWalletV2Model(t *testing.T) *StateFlowV2Model {
	t.Helper()

	model, err := BuildStateFlowV2Model(&StateFlowV2Config{Name: "Wallet"}, walletV2Rules())
	if err != nil {
		t.Fatalf("BuildStateFlowV2Model() error = %v", err)
	}
	return model
}
