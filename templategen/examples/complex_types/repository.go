//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/repository.tmpl

package complex_types

import (
	"database/sql"
	"net/http"
	"sync"
)

// @Define(name=DB, db=sql.DB, mutex=sync.RWMutex)
// @Define(name=Meta, tableName="users", primaryKey="id")
type UserRepository struct {
	db    *sql.DB
	mutex sync.RWMutex
}

// @Define(name=API, request=http.Request, response=http.ResponseWriter)
// @Define(name=Meta, basePath="/api/v1/users", version="1.0")
type UserHandler struct {
	repo *UserRepository
}

// @Define(name=Config, method="GET", path="/list", auth="true")
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
}

// @Define(name=Config, method="POST", path="/create", auth="true")
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
}

// @Define(name=Config, method="GET", path="/{id}", auth="false")
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
}
