package testdata

import "context"

// @Mock
type UserService interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int64) error
}

// @Mock(mock_name="CustomMockRepo")
type UserRepository interface {
	FindByID(id int64) (*User, error)
	Save(user *User) error
}

type User struct {
	ID   int64
	Name string
}
