package testdata

import (
	"context"
	"io"
)

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

// Reader is a basic reader interface
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Writer is a basic writer interface
type Writer interface {
	Write(p []byte) (n int, err error)
}

// @Mock
type ReadWriter interface {
	Reader
	Writer
	Close() error
}

// @Mock
type IOReadWriter interface {
	io.Reader
	io.Writer
	Sync() error
}

// @Mock
// NamedReturnsInterface 用于测试命名返回值的解析
type NamedReturnsInterface interface {
	// 有命名返回值
	GetData(key string) (data []byte, found bool, err error)
	// 全部未命名返回值
	Process(input string) (string, error)
}

// @Mock
// SameTypeReturns 用于测试连续相同类型的返回值合并
type SameTypeReturns interface {
	// 连续相同类型返回值应该合并: (publicKey, rawPublicKey types.PubKey, refID RefID, err error)
	CreateWallet(ctx context.Context) (publicKey, rawPublicKey string, refID string, err error)
}
