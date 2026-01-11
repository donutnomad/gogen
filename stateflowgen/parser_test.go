package stateflowgen

import (
	"testing"
)

func TestParseStateFlowConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *StateFlowConfig
		wantErr bool
	}{
		{
			name:  "basic config",
			input: `@StateFlow(name="Server")`,
			want:  &StateFlowConfig{Name: "Server"},
		},
		{
			name:  "config with output",
			input: `@StateFlow(name="Server", output="server_state.go")`,
			want:  &StateFlowConfig{Name: "Server", Output: "server_state.go"},
		},
		{
			name:  "backtick format",
			input: "@StateFlow(name=`Server`)",
			want:  &StateFlowConfig{Name: "Server"},
		},
		{
			name:  "no quotes",
			input: `@StateFlow(name=Server)`,
			want:  &StateFlowConfig{Name: "Server"},
		},
		{
			name:  "empty params - name empty",
			input: `@StateFlow()`,
			want:  &StateFlowConfig{Name: ""},
		},
		{
			name:  "no params - name empty",
			input: `@StateFlow`,
			want:  &StateFlowConfig{Name: ""},
		},
		{
			name:    "invalid format - no @StateFlow",
			input:   `@SomeOtherAnnotation`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStateFlowConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStateFlowConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Output != tt.want.Output {
				t.Errorf("Output = %v, want %v", got.Output, tt.want.Output)
			}
		})
	}
}

func TestParseFlowRule_Basic(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *FlowRule
		wantErr bool
	}{
		{
			name:  "basic flow",
			input: `@Flow: Init => [ Provisioning ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Init"},
				Targets: []TargetRef{
					{Phase: "Provisioning"},
				},
			},
		},
		{
			name:  "single node declaration",
			input: `@Flow: Init`,
			want: &FlowRule{
				Source:  StateRef{Phase: "Init"},
				Targets: nil,
			},
		},
		{
			name:  "with status",
			input: `@Flow: Ready(Enabled) => [ (Disabled) ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Status: "Disabled"},
				},
			},
		},
		{
			name:  "wildcard status",
			input: `@Flow: Ready(*) => [ Deleted ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Wildcard: true},
				Targets: []TargetRef{
					{Phase: "Deleted"},
				},
			},
		},
		{
			name:  "multiple targets",
			input: `@Flow: Provisioning => [ Ready(Enabled), Failed ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Provisioning"},
				Targets: []TargetRef{
					{Phase: "Ready", Status: "Enabled"},
					{Phase: "Failed"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlowRule(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlowRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check source
			if got.Source.Phase != tt.want.Source.Phase {
				t.Errorf("Source.Phase = %v, want %v", got.Source.Phase, tt.want.Source.Phase)
			}
			if got.Source.Status != tt.want.Source.Status {
				t.Errorf("Source.Status = %v, want %v", got.Source.Status, tt.want.Source.Status)
			}
			if got.Source.Wildcard != tt.want.Source.Wildcard {
				t.Errorf("Source.Wildcard = %v, want %v", got.Source.Wildcard, tt.want.Source.Wildcard)
			}

			// Check targets count
			if len(got.Targets) != len(tt.want.Targets) {
				t.Errorf("len(Targets) = %v, want %v", len(got.Targets), len(tt.want.Targets))
				return
			}

			for i, target := range got.Targets {
				wantTarget := tt.want.Targets[i]
				if target.Phase != wantTarget.Phase {
					t.Errorf("Targets[%d].Phase = %v, want %v", i, target.Phase, wantTarget.Phase)
				}
				if target.Status != wantTarget.Status {
					t.Errorf("Targets[%d].Status = %v, want %v", i, target.Status, wantTarget.Status)
				}
			}
		})
	}
}

func TestParseFlowRule_Approval(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *FlowRule
		wantErr bool
	}{
		{
			name:  "required approval",
			input: `@Flow: Ready(Enabled) => [ (Disabled)! ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Status: "Disabled", ApprovalRequired: true},
				},
			},
		},
		{
			name:  "optional approval",
			input: `@Flow: Ready(Enabled) => [ (=)? ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Self: true, ApprovalOptional: true},
				},
			},
		},
		{
			name:  "self transition",
			input: `@Flow: Ready(Enabled) => [ (=)! ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Self: true, ApprovalRequired: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlowRule(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlowRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got.Targets) != len(tt.want.Targets) {
				t.Errorf("len(Targets) = %v, want %v", len(got.Targets), len(tt.want.Targets))
				return
			}

			for i, target := range got.Targets {
				wantTarget := tt.want.Targets[i]
				if target.ApprovalRequired != wantTarget.ApprovalRequired {
					t.Errorf("Targets[%d].ApprovalRequired = %v, want %v", i, target.ApprovalRequired, wantTarget.ApprovalRequired)
				}
				if target.ApprovalOptional != wantTarget.ApprovalOptional {
					t.Errorf("Targets[%d].ApprovalOptional = %v, want %v", i, target.ApprovalOptional, wantTarget.ApprovalOptional)
				}
				if target.Self != wantTarget.Self {
					t.Errorf("Targets[%d].Self = %v, want %v", i, target.Self, wantTarget.Self)
				}
			}
		})
	}
}

func TestParseFlowRule_ViaAndElse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *FlowRule
		wantErr bool
	}{
		{
			name:  "via intermediate state",
			input: `@Flow: Ready(Enabled) => [ Created! via Updating ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Phase: "Created", ApprovalRequired: true, Via: "Updating"},
				},
			},
		},
		{
			name:  "via with else",
			input: `@Flow: Ready(Enabled) => [ Created! via Updating else Failed ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Phase: "Created", ApprovalRequired: true, Via: "Updating", Else: "Failed"},
				},
			},
		},
		{
			name:  "via with status",
			input: `@Flow: Ready(Enabled) => [ Created! via Updating(Pending) ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Phase: "Created", ApprovalRequired: true, Via: "Updating", ViaStatus: "Pending"},
				},
			},
		},
		{
			name:  "self transition with via",
			input: `@Flow: Ready(Enabled) => [ (=)! via Updating ]`,
			want: &FlowRule{
				Source: StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{
					{Self: true, ApprovalRequired: true, Via: "Updating"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlowRule(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlowRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got.Targets) != len(tt.want.Targets) {
				t.Errorf("len(Targets) = %v, want %v", len(got.Targets), len(tt.want.Targets))
				return
			}

			for i, target := range got.Targets {
				wantTarget := tt.want.Targets[i]
				if target.Via != wantTarget.Via {
					t.Errorf("Targets[%d].Via = %v, want %v", i, target.Via, wantTarget.Via)
				}
				if target.ViaStatus != wantTarget.ViaStatus {
					t.Errorf("Targets[%d].ViaStatus = %v, want %v", i, target.ViaStatus, wantTarget.ViaStatus)
				}
				if target.Else != wantTarget.Else {
					t.Errorf("Targets[%d].Else = %v, want %v", i, target.Else, wantTarget.Else)
				}
				if target.Self != wantTarget.Self {
					t.Errorf("Targets[%d].Self = %v, want %v", i, target.Self, wantTarget.Self)
				}
			}
		})
	}
}

func TestParseFlowAnnotations(t *testing.T) {
	input := `
// =========================================================
// 服务器生命周期拓扑图
// =========================================================
//
// @StateFlow(name="Server")
// @Flow: Init           => [ Provisioning ]
// @Flow: Provisioning   => [ Ready(Enabled), Failed ]
// @Flow: Ready(Enabled) => [ (Disabled)! ]
// @Flow: Ready(Disabled)=> [ (Enabled) ]
// @Flow: Ready(*)       => [ Deleted! ]
// @Flow: Failed         => [ Deleted! ]
`

	config, rules, err := ParseFlowAnnotations(input)
	if err != nil {
		t.Fatalf("ParseFlowAnnotations() error = %v", err)
	}

	if config == nil {
		t.Fatal("config is nil")
	}
	if config.Name != "Server" {
		t.Errorf("config.Name = %v, want Server", config.Name)
	}

	if len(rules) != 6 {
		t.Errorf("len(rules) = %v, want 6", len(rules))
	}

	// Verify first rule
	if rules[0].Source.Phase != "Init" {
		t.Errorf("rules[0].Source.Phase = %v, want Init", rules[0].Source.Phase)
	}
	if len(rules[0].Targets) != 1 || rules[0].Targets[0].Phase != "Provisioning" {
		t.Errorf("rules[0].Targets = %v, want [Provisioning]", rules[0].Targets)
	}

	// Verify wildcard rule
	if !rules[4].Source.Wildcard {
		t.Errorf("rules[4].Source.Wildcard = false, want true")
	}
}

func TestParseFlowRule_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty target list",
			input: `@Flow: Init => []`,
		},
		{
			name:  "missing brackets",
			input: `@Flow: Init => Provisioning`,
		},
		{
			name:  "unmatched parenthesis",
			input: `@Flow: Init(Status => [ Provisioning ]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFlowRule(tt.input)
			if err == nil {
				t.Error("ParseFlowRule() expected error, got nil")
			}
		})
	}
}
