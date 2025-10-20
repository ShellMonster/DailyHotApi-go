package models

import "time"

// HotData 热榜数据项
// 统一的热榜数据结构,所有平台的数据都会转换成这个格式
// 向后兼容原项目的 ListItem 结构
type HotData struct {
	ID        string      `json:"id"`                  // 唯一标识
	Title     string      `json:"title"`               // 标题
	Desc      string      `json:"desc,omitempty"`      // 描述 (可选)
	Cover     string      `json:"cover,omitempty"`     // 封面图片 URL (可选)
	Author    string      `json:"author,omitempty"`    // 作者/发布者 (可选)
	Hot       interface{} `json:"hot,omitempty"`       // 热度值 (支持 number 或 null)
	Timestamp interface{} `json:"timestamp,omitempty"` // 发布时间 (支持 number 或 string)
	URL       string      `json:"url"`                 // 详情页链接 (必需)
	MobileURL string      `json:"mobileUrl,omitempty"` // 移动端链接 (可选)
}

// Response 统一响应结构
// 所有 API 都返回这个格式,保证与原项目完全兼容
type Response struct {
	Code        int                    `json:"code"`                  // 状态码: 200 成功, 其他失败
	Message     string                 `json:"message"`               // 提示信息
	Name        string                 `json:"name"`                  // 平台调用名称, 如 "bilibili"、"weibo"
	Title       string                 `json:"title"`                 // 平台名称,如 "哔哩哔哩"、"微博" (新增)
	Type        string                 `json:"type"`                  // 榜单类型,如 "热榜" (替代 subtitle)
	Description string                 `json:"description,omitempty"` // 平台描述 (新增)
	Params      map[string]interface{} `json:"params,omitempty"`      // 参数说明 (新增)
	Link        string                 `json:"link,omitempty"`        // 官方链接 (新增)
	UpdateTime  string                 `json:"updateTime"`            // 更新时间 (改为驼峰式)
	Total       int                    `json:"total"`                 // 数据总数
	FromCache   bool                   `json:"fromCache"`             // 是否来自缓存
	Data        []HotData              `json:"data"`                  // 热榜数据列表
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
}

// SuccessResponse 创建成功响应 (新签名,向后兼容原项目)
// 参数说明:
//   - name: 平台调用名称 (如 "bilibili")
//   - title: 平台显示名称 (如 "哔哩哔哩")
//   - typeStr: 榜单类型 (如 "热榜")
//   - description: 平台描述 (可选)
//   - link: 官方链接 (可选)
//   - params: 参数说明 (可选)
//   - data: 热榜数据列表
//   - fromCache: 是否来自缓存
func SuccessResponse(
	name string,
	title string,
	typeStr string,
	description string,
	link string,
	params map[string]interface{},
	data []HotData,
	fromCache bool,
) *Response {
	return &Response{
		Code:        200,
		Message:     "success",
		Name:        name,
		Title:       title,
		Type:        typeStr,
		Description: description,
		Link:        link,
		Params:      params,
		UpdateTime:  getCurrentTime(),
		Total:       len(data),
		Data:        data,
		FromCache:   fromCache,
	}
}

// SimpleSuccessResponse 简化版本,用于向后兼容旧的调用方式
func SimpleSuccessResponse(name string, typeStr string, data []HotData, fromCache bool) *Response {
	return SuccessResponse(name, name, typeStr, "", "", nil, data, fromCache)
}

// ErrorResponseObj 创建错误响应
func ErrorResponseObj(code int, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// getCurrentTime 获取当前时间字符串 (RFC3339 格式,与原项目兼容)
func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}
