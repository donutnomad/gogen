//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/crud.tmpl

package template_inheritance

import "context"

// @Define(name=CRUD, table="customers", softDelete="true")
type Customer struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt string
	UpdatedAt string
	DeletedAt *string
}

// @Define(name=Op, action="create")
func (c *Customer) Create(ctx context.Context) error {
	return nil
}

// @Define(name=Op, action="read")
func (c *Customer) Read(ctx context.Context, id int64) error {
	return nil
}

// @Define(name=Op, action="update")
func (c *Customer) Update(ctx context.Context) error {
	return nil
}

// @Define(name=Op, action="delete")
func (c *Customer) Delete(ctx context.Context, id int64) error {
	return nil
}
