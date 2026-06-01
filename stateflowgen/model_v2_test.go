package stateflowgen

import "testing"

func TestBuildStateFlowV2Model(t *testing.T) {
	model, err := BuildStateFlowV2Model(
		&StateFlowV2Config{Name: "Wallet"},
		walletV2Rules(),
	)
	if err != nil {
		t.Fatalf("BuildStateFlowV2Model() error = %v", err)
	}

	assertStringSlice(t, model.Statuses, []string{"initial", "creating", "rejected", "active", "inactive"})
	assertStringSlice(t, model.Stages, []string{
		"none",
		"waiting_approval",
		"update_profile_waiting_commit_approval",
		"disable_wallet_waiting_commit_approval",
	})

	update := model.Transitions[2]
	if update.From != "active" {
		t.Errorf("update.From = %v, want active", update.From)
	}
	if update.To != "active" {
		t.Errorf("update.To = %v, want active", update.To)
	}
	if update.Via != "update_profile_waiting_commit_approval" {
		t.Errorf("update.Via = %v, want update_profile_waiting_commit_approval", update.Via)
	}
	if update.Fallback != "active" {
		t.Errorf("update.Fallback = %v, want active", update.Fallback)
	}
}

func walletV2Rules() []*FlowRule {
	return []*FlowRule{
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
		{
			Source: StateRef{Phase: "active"},
			Targets: []TargetRef{{
				Self:             true,
				ApprovalOptional: true,
				Via:              "update_profile_waiting_commit_approval",
				Else:             "active",
			}},
		},
		{
			Source: StateRef{Phase: "active"},
			Targets: []TargetRef{{
				Phase:            "inactive",
				ApprovalOptional: true,
				Via:              "disable_wallet_waiting_commit_approval",
				Else:             "active",
			}},
		},
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %v, want %v; got %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %v, want %v; got %v", i, got[i], want[i], got)
		}
	}
}
