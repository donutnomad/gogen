package stateflowgen

import (
	"testing"
)

func TestBuildModel_Basic(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Init"},
			Targets: []TargetRef{{Phase: "Provisioning"}},
		},
		{
			Source:  StateRef{Phase: "Provisioning"},
			Targets: []TargetRef{{Phase: "Ready"}, {Phase: "Failed"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if model.Name != "Server" {
		t.Errorf("Name = %v, want Server", model.Name)
	}

	// Check phases
	expectedPhases := []string{"Init", "Provisioning", "Ready", "Failed"}
	if len(model.Phases) != len(expectedPhases) {
		t.Errorf("len(Phases) = %v, want %v", len(model.Phases), len(expectedPhases))
	}
	for i, phase := range expectedPhases {
		if model.Phases[i] != phase {
			t.Errorf("Phases[%d] = %v, want %v", i, model.Phases[i], phase)
		}
	}

	// Check HasStatus
	if model.HasStatus {
		t.Error("HasStatus = true, want false")
	}

	// Check HasApproval
	if model.HasApproval {
		t.Error("HasApproval = true, want false")
	}

	// Check InitStage
	if model.InitStage.Phase != "Init" {
		t.Errorf("InitStage.Phase = %v, want Init", model.InitStage.Phase)
	}
}

func TestBuildModel_WithStatus(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Status: "Disabled"}},
		},
		{
			Source:  StateRef{Phase: "Ready", Status: "Disabled"},
			Targets: []TargetRef{{Status: "Enabled"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if !model.HasStatus {
		t.Error("HasStatus = false, want true")
	}

	// Check PhaseStatus
	statuses := model.PhaseStatus["Ready"]
	if len(statuses) != 2 {
		t.Errorf("len(PhaseStatus[Ready]) = %v, want 2", len(statuses))
	}
}

func TestBuildModel_WithApproval(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Status: "Disabled", ApprovalRequired: true, Via: "Updating"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if !model.HasApproval {
		t.Error("HasApproval = false, want true")
	}
}

func TestBuildModel_WildcardExpansion(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Status: "Disabled"}},
		},
		{
			Source:  StateRef{Phase: "Ready", Status: "Disabled"},
			Targets: []TargetRef{{Status: "Enabled"}},
		},
		{
			Source:  StateRef{Phase: "Ready", Wildcard: true},
			Targets: []TargetRef{{Phase: "Deleted"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	// 通配符应该展开为 Ready(Enabled) -> Deleted 和 Ready(Disabled) -> Deleted
	deletedTransitions := 0
	for _, trans := range model.Transitions {
		if trans.To.Phase == "Deleted" {
			deletedTransitions++
		}
	}
	if deletedTransitions != 2 {
		t.Errorf("deletedTransitions = %v, want 2", deletedTransitions)
	}
}

func TestBuildModel_WildcardExcludesSelfTransition(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Status: "Disabled"}},
		},
		{
			Source:  StateRef{Phase: "Ready", Status: "Disabled"},
			Targets: []TargetRef{{Status: "Enabled"}},
		},
		{
			// 通配符规则：Ready(*) => [ (Disabled) ]
			// 应该只展开为 Ready(Enabled) => Ready(Disabled)
			// 不应该包含 Ready(Disabled) => Ready(Disabled)
			Source:  StateRef{Phase: "Ready", Wildcard: true},
			Targets: []TargetRef{{Status: "Disabled"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	// 检查通配符展开不包含自我流转
	for _, trans := range model.Transitions {
		if trans.From.Phase == "Ready" && trans.From.Status == "Disabled" &&
			trans.To.Phase == "Ready" && trans.To.Status == "Disabled" {
			t.Error("Wildcard expansion should not include self-transition: Ready(Disabled) => Ready(Disabled)")
		}
	}
}

func TestBuildModel_ViaState(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source: StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{
				{Phase: "Created", ApprovalRequired: true, Via: "Updating"},
			},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	// Check ViaPhases
	if len(model.ViaPhases) != 1 || model.ViaPhases[0] != "Updating" {
		t.Errorf("ViaPhases = %v, want [Updating]", model.ViaPhases)
	}

	// Check transition has via
	if len(model.Transitions) != 1 {
		t.Fatalf("len(Transitions) = %v, want 1", len(model.Transitions))
	}
	trans := model.Transitions[0]
	if trans.Via.Phase != "Updating" {
		t.Errorf("Via.Phase = %v, want Updating", trans.Via.Phase)
	}
}

func TestBuildModel_ElseState(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Init"},
			Targets: []TargetRef{{Phase: "Ready", Status: "Enabled"}},
		},
		{
			Source: StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{
				{Phase: "Created", ApprovalRequired: true, Via: "Updating", Else: "Failed"},
			},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	// Find the transition with else
	var foundTrans *Transition
	for i := range model.Transitions {
		if model.Transitions[i].Fallback.Phase == "Failed" {
			foundTrans = &model.Transitions[i]
			break
		}
	}

	if foundTrans == nil {
		t.Fatal("Expected transition with Fallback.Phase=Failed not found")
	}
	if foundTrans.Fallback.Phase != "Failed" {
		t.Errorf("Fallback.Phase = %v, want Failed", foundTrans.Fallback.Phase)
	}
}

func TestBuildModel_DefaultFallback(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source: StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{
				{Status: "Disabled", ApprovalRequired: true, Via: "Updating"},
			},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if len(model.Transitions) != 1 {
		t.Fatalf("len(Transitions) = %v, want 1", len(model.Transitions))
	}
	trans := model.Transitions[0]
	// 默认回退到源状态
	if trans.Fallback.Phase != "Ready" || trans.Fallback.Status != "Enabled" {
		t.Errorf("Fallback = %v, want Ready(Enabled)", trans.Fallback)
	}
}

func TestBuildModel_SingleNode(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Init"},
			Targets: nil, // 单节点声明，无目标
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if len(model.Phases) != 1 || model.Phases[0] != "Init" {
		t.Errorf("Phases = %v, want [Init]", model.Phases)
	}
	if len(model.Transitions) != 0 {
		t.Errorf("len(Transitions) = %v, want 0", len(model.Transitions))
	}
}

func TestBuildModel_IsolatedNodeError(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Init"},
			Targets: []TargetRef{{Phase: "Provisioning"}},
		},
		{
			// Ready 是孤立的子图 - 与主图（Init -> Provisioning）不连通
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Status: "Disabled"}},
		},
		{
			Source:  StateRef{Phase: "Ready", Status: "Disabled"},
			Targets: []TargetRef{{Status: "Enabled"}},
		},
	}

	_, err := BuildModel(config, rules)
	if err == nil {
		t.Error("BuildModel() expected error for disconnected subgraph, got nil")
	}
}

func TestBuildModel_SelfTransition(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}
	rules := []*FlowRule{
		{
			Source:  StateRef{Phase: "Ready", Status: "Enabled"},
			Targets: []TargetRef{{Self: true, ApprovalOptional: true, Via: "Updating"}},
		},
	}

	model, err := BuildModel(config, rules)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	if len(model.Transitions) != 1 {
		t.Fatalf("len(Transitions) = %v, want 1", len(model.Transitions))
	}
	trans := model.Transitions[0]
	if !trans.From.Equal(trans.To) {
		t.Errorf("Self transition should have From == To, got From=%v, To=%v", trans.From, trans.To)
	}
}

func TestBuildModel_ApprovalRequiresVia(t *testing.T) {
	config := &StateFlowConfig{Name: "Server"}

	// 测试 ! 标记没有 via 应该报错
	t.Run("required approval without via", func(t *testing.T) {
		rules := []*FlowRule{
			{
				Source:  StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{{Status: "Disabled", ApprovalRequired: true}},
			},
		}

		_, err := BuildModel(config, rules)
		if err == nil {
			t.Error("BuildModel() expected error for approval without via, got nil")
		}
	})

	// 测试 ? 标记没有 via 应该报错
	t.Run("optional approval without via", func(t *testing.T) {
		rules := []*FlowRule{
			{
				Source:  StateRef{Phase: "Ready", Status: "Enabled"},
				Targets: []TargetRef{{Status: "Disabled", ApprovalOptional: true}},
			},
		}

		_, err := BuildModel(config, rules)
		if err == nil {
			t.Error("BuildModel() expected error for approval without via, got nil")
		}
	})
}

func TestStage_Equal(t *testing.T) {
	tests := []struct {
		name string
		a, b Stage
		want bool
	}{
		{
			name: "equal without status",
			a:    Stage{Phase: "Init"},
			b:    Stage{Phase: "Init"},
			want: true,
		},
		{
			name: "equal with status",
			a:    Stage{Phase: "Ready", Status: "Enabled"},
			b:    Stage{Phase: "Ready", Status: "Enabled"},
			want: true,
		},
		{
			name: "different phase",
			a:    Stage{Phase: "Init"},
			b:    Stage{Phase: "Ready"},
			want: false,
		},
		{
			name: "different status",
			a:    Stage{Phase: "Ready", Status: "Enabled"},
			b:    Stage{Phase: "Ready", Status: "Disabled"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllStages(t *testing.T) {
	model := &StateModel{
		Phases:      []string{"Init", "Ready"},
		PhaseStatus: map[string][]string{"Ready": {"Enabled", "Disabled"}},
	}

	stages := model.GetAllStages()

	// 应该有 3 个阶段：Init, Ready(Enabled), Ready(Disabled)
	if len(stages) != 3 {
		t.Errorf("len(stages) = %v, want 3", len(stages))
	}
}

func TestGetTransitionsFrom(t *testing.T) {
	model := &StateModel{
		Transitions: []Transition{
			{From: Stage{Phase: "Init"}, To: Stage{Phase: "Ready"}},
			{From: Stage{Phase: "Ready"}, To: Stage{Phase: "Done"}},
			{From: Stage{Phase: "Init"}, To: Stage{Phase: "Failed"}},
		},
	}

	transitions := model.GetTransitionsFrom(Stage{Phase: "Init"})
	if len(transitions) != 2 {
		t.Errorf("len(transitions) = %v, want 2", len(transitions))
	}
}
