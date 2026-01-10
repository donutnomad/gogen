package testdata

import "context"

// @Mock
type GenericRepository[T any] interface {
	Get(ctx context.Context, id int64) (T, error)
	Save(ctx context.Context, entity T) error
	List(ctx context.Context, opts ...ListOption) ([]T, error)
}

// @Mock
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string)
}

type ListOption struct {
	Limit  int
	Offset int
}
