//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/wrapper.tmpl

package import_alias

import "context"

// @Import(alias=xerrors, path="golang.org/x/exp/errors")
// @Import(alias=uuid, path="github.com/google/uuid")

// @Define(name=Types, id=uuid.UUID, ctx=context.Context)
// @Define(name=Errors, errType=xerrors.Frame)
type Entity struct {
	id  string
	ctx context.Context
}

// @Define(name=Method, operation="create", returnErr=xerrors.Frame)
func (e *Entity) Create(ctx context.Context) error {
	return nil
}

// @Define(name=Method, operation="delete", returnErr=xerrors.Frame)
func (e *Entity) Delete(ctx context.Context, id string) error {
	return nil
}
