package example

import "context"

// UserResponse 用户响应
type UserResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name string `json:"name"`
}

// @TAG(用户管理)
// @SECURITY(Bearer)
type IUserAPI interface {
	// 获取用户
	// @GET(/api/v1/user/{id})
	GetUser(ctx context.Context, id int64) (UserResponse, error)

	// 创建用户
	// @POST(/api/v1/user)
	CreateUser(ctx context.Context, req CreateUserReq) (UserResponse, error)

	// 删除用户
	// @DELETE(/api/v1/user/{id})
	DeleteUser(ctx context.Context, id int64) error
}
