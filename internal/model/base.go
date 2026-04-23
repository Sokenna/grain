package model

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageRequest 分页请求
type PageRequest struct {
	Page int `form:"page" json:"page" binding:"min=1"`
	Size int `form:"size" json:"size" binding:"min=1,max=100"`
}

// PageResponse 分页响应
type PageResponse struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	List  interface{} `json:"list"`
}

// GetOffset 计算偏移量
func (p *PageRequest) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Size <= 0 {
		p.Size = 10
	}
	return (p.Page - 1) * p.Size
}

// GetLimit 获取限制数量
func (p *PageRequest) GetLimit() int {
	if p.Size <= 0 {
		return 10
	}
	if p.Size > 100 {
		return 100
	}
	return p.Size
}
