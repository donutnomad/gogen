//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/approval.tmpl -output $FILE_approval_gen.go

package approvegen

import (
	"context"
)

// ============ 模拟类型定义 ============

type SaveCmd struct {
	BusinessID uint64
}

type Transition struct {
	Data any
}

type ApprovalResult struct {
	Done   bool
	Result string
}

type VoteData[T, R any] struct {
	Show T
	Raw  R
}

// ============ UseCase 定义 ============

// @Define(name=Approval, module="APPROVALNODES", refType="approvalnode.RefType")
// @Define(name=Deps, voteRepo="vote.Repo", approvalService="vote.IDomainService")
type UseCase struct {
	approvalService any
}

// @Define(name=Method, event="SAVE", bodyType=Transition)
// @Define(name=Post, hasPost="true")
func (u *UseCase) Save(ctx context.Context, cmd *SaveCmd) error {
	return nil
}

// @Define(name=Method, event="DELETE", bodyType=Transition)
// @Define(name=Post, hasPost="false")
func (u *UseCase) Delete(ctx context.Context, id uint64) error {
	return nil
}
