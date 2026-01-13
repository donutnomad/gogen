//go:generate gotoolkit gen .

//go:gogen: plugin:templategen -template ./templates/getters.tmpl

package basic

import (
	"io"
	"time"
)

// @Define(name=Config, description="带有配置的服务")
type Service struct {
	reader  io.Reader
	timeout time.Duration
	name    string
}

// @Define(name=Config, description="带有重试功能的服务")
type RetryService struct {
	maxRetries int
	retryDelay time.Duration
}
