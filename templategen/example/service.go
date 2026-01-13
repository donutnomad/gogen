//go:gogen: plugin:templategen -template ./templates/service.tmpl

package testdata

import "io"

// @Define(name=IO, reader=io.Reader, timeout="30s")
type MyService struct {
	reader io.Reader
	name   string
}

// @Define(name=Meta, permission="admin", audit="true")
func (s *MyService) Delete(id string) error {
	return nil
}

// @Define(name=Meta, permission="user", audit="false")
func (s *MyService) Get(id string) (any, error) {
	return nil, nil
}
