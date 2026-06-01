package stateflowgen

import "testing"

func TestParseStateFlowV2Annotations(t *testing.T) {
	input := `
// @StateFlowV2(name="Wallet", output=types_state.go, statusType=int32, statusValues="initial=0,creating=100,active=101,rejected=300")
// @Flow: initial => [ creating? via waiting_approval else rejected ]
// @Flow: active => [ (=)? via update_profile_waiting_commit_approval else active ]
// @Flow: active => [ inactive? via disable_wallet_waiting_commit_approval else active ]
`

	config, rules, err := ParseFlowV2Annotations(input)
	if err != nil {
		t.Fatalf("ParseFlowV2Annotations() error = %v", err)
	}

	if config == nil {
		t.Fatal("config is nil")
	}
	if config.Name != "Wallet" {
		t.Errorf("config.Name = %v, want Wallet", config.Name)
	}
	if config.Output != "types_state.go" {
		t.Errorf("config.Output = %v, want types_state.go", config.Output)
	}
	if config.StatusType != "int32" {
		t.Errorf("config.StatusType = %v, want int32", config.StatusType)
	}
	if config.StatusValues["creating"] != "100" {
		t.Errorf("config.StatusValues[creating] = %v, want 100", config.StatusValues["creating"])
	}
	if len(rules) != 3 {
		t.Fatalf("len(rules) = %v, want 3", len(rules))
	}

	create := rules[0].Targets[0]
	if create.Phase != "creating" {
		t.Errorf("create.Phase = %v, want creating", create.Phase)
	}
	if create.Via != "waiting_approval" {
		t.Errorf("create.Via = %v, want waiting_approval", create.Via)
	}
	if create.Else != "rejected" {
		t.Errorf("create.Else = %v, want rejected", create.Else)
	}

	update := rules[1].Targets[0]
	if !update.Self {
		t.Error("update.Self = false, want true")
	}
}
