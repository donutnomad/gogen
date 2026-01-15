// Package v2api 演示版本化包路径的情况
// 文件夹名: v2-api
// 包名: v2api
// 完整路径: github.com/donutnomad/gogen/pickgen/examples/v2-api
package v2api

import "time"

// Request API 请求模型
// @Pick(name=RequestBasic, fields=`[ID,Method,Path,Headers]`)
// @Omit(name=RequestPublic, fields=`[InternalTrace,RawBody]`)
type Request struct {
	ID            string            `json:"id"`
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	Headers       map[string]string `json:"headers"`
	Body          []byte            `json:"body"`
	RawBody       []byte            `json:"-"`
	InternalTrace string            `json:"-"`
	Timestamp     time.Time         `json:"timestamp"`
}

// Response API 响应模型
// @Pick(name=ResponseSummary, fields=`[ID,StatusCode,Duration]`)
type Response struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Error      error             `json:"-"`
}
