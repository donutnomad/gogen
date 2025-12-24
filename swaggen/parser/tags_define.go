package parsers

type Tag struct {
	Value string `sg:"required,delimiter=,"`
}

func (t Tag) Name() string    { return "TAG" }
func (t Tag) Mode() ParseMode { return ModeNamed }

type Security struct {
	Value   string   `sg:"required"`
	Exclude []string `sg:"delimiter=,"` // 支持,分割和默认支持空格分割
	Include []string `sg:"delimiter=,"`
}

func (s Security) Name() string    { return "SECURITY" }
func (s Security) Mode() ParseMode { return ModeNamed }

type Header struct {
	Value       string `sg:"required"`
	Required    bool   `sg:"required"`
	Description string
}

func (s Header) Name() string    { return "HEADER" }
func (s Header) Mode() ParseMode { return ModeNamed }

/////////////////////////////// 响应 /////////////////////////////////////

type JSON struct {
}

func (s JSON) Name() string    { return "JSON" }
func (s JSON) Mode() ParseMode { return ModeNamed }

type MIME struct {
	// Alias	MIME Type
	//json	application/json
	//xml	text/xml
	//plain	text/plain
	//html	text/html
	//mpfd	multipart/form-data
	//x-www-form-urlencoded	application/x-www-form-urlencoded
	//json-api	application/vnd.api+json
	//json-stream	application/x-json-stream
	//octet-stream	application/octet-stream
	//png	image/png
	//jpeg	image/jpeg
	//gif	image/gif
	//event-stream	text/event-stream
	Value string `sg:"required"`
}

func (s MIME) Name() string    { return "MIME" }
func (s MIME) Mode() ParseMode { return ModeNamed }

/////////////////////////////// 请求 /////////////////////////////////////

type FormReq struct{}

func (s FormReq) Name() string    { return "FORM-REQ" }
func (s FormReq) Mode() ParseMode { return ModeNamed }

type JsonReq struct{}

func (s JsonReq) Name() string    { return "JSON-REQ" }
func (s JsonReq) Mode() ParseMode { return ModeNamed }

type MimeReq struct {
	Value string `sg:"required"`
}

func (s MimeReq) Name() string    { return "MIME-REQ" }
func (s MimeReq) Mode() ParseMode { return ModeNamed }

/////////////////////////////// GIN-Handler /////////////////////////////////////

type MiddleWare struct {
	Value []string `sg:"required"`
}

func (s MiddleWare) Name() string    { return "MID" }
func (s MiddleWare) Mode() ParseMode { return ModeNamed }

/////////////////////// GET|PUT|POST|PATCH|DELETE ///////////////////////

type GET struct {
	Value string `sg:"required,delimiter=,"`
}

func (s GET) Name() string    { return "GET" }
func (s GET) Mode() ParseMode { return ModeNamed }

type POST struct {
	Value string `sg:"required"`
}

func (s POST) Name() string    { return "POST" }
func (s POST) Mode() ParseMode { return ModeNamed }

type PUT struct {
	Value string `sg:"required"`
}

func (s PUT) Name() string    { return "PUT" }
func (s PUT) Mode() ParseMode { return ModeNamed }

type PATCH struct {
	Value string `sg:"required"`
}

func (s PATCH) Name() string    { return "PATCH" }
func (s PATCH) Mode() ParseMode { return ModeNamed }

type DELETE struct {
	Value string `sg:"required"`
}

func (s DELETE) Name() string    { return "DELETE" }
func (s DELETE) Mode() ParseMode { return ModeNamed }

/////////////////////// 参数注释标签 ///////////////////////

// FORM 表单参数标签
type FORM struct{}

func (s FORM) Name() string    { return "FORM" }
func (s FORM) Mode() ParseMode { return ModeNamed }

// BODY 请求体参数标签
type BODY struct{}

func (s BODY) Name() string    { return "BODY" }
func (s BODY) Mode() ParseMode { return ModeNamed }

// PARAM 路径参数标签
type PARAM struct {
	Value string // 可选的别名
}

func (s PARAM) Name() string    { return "PARAM" }
func (s PARAM) Mode() ParseMode { return ModeNamed }

// QUERY 查询参数标签
type QUERY struct{}

func (s QUERY) Name() string    { return "QUERY" }
func (s QUERY) Mode() ParseMode { return ModeNamed }

/////////////////////// 控制标签 ///////////////////////

type Removed struct{}

func (s Removed) Name() string    { return "Removed" }
func (s Removed) Mode() ParseMode { return ModeNamed }

type ExcludeFromBindAll struct {
}

func (s ExcludeFromBindAll) Name() string    { return "ExcludeFromBindAll" }
func (s ExcludeFromBindAll) Mode() ParseMode { return ModeNamed }

type Raw struct {
	Value string `sg:"required"`
}

func (s Raw) Name() string    { return "Raw" }
func (s Raw) Mode() ParseMode { return ModeNamed }

type Prefix struct {
	Value string `sg:"required"`
}

func (s Prefix) Name() string    { return "PREFIX" }
func (s Prefix) Mode() ParseMode { return ModeNamed }
