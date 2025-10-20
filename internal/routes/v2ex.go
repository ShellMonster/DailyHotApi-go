package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// V2exHandler V2EX处理器
type V2exHandler struct {
	fetcher *service.Fetcher
}

// NewV2exHandler 创建V2EX处理器
func NewV2exHandler(fetcher *service.Fetcher) *V2exHandler {
	return &V2exHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *V2exHandler) GetPath() string {
	return "/v2ex"
}

// Handle 处理请求
func (h *V2exHandler) Handle(c *fiber.Ctx) error {
	// 支持不同类型: hot-最热, latest-最新
	topicType := c.Query("type", "hot")
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchV2exHot(c.Context(), topicType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 准备类型映射表 - 用于前端显示支持的类型
	typeMap := map[string]string{
		"hot":    "最热主题",
		"latest": "最新主题",
	}

	// 获取当前类型名称
	typeName := typeMap[topicType]
	if typeName == "" {
		typeName = "最热主题"
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"v2ex",                  // name: 平台调用名称
		"V2EX",                  // title: 平台显示名称
		typeName,                // type: 当前类型
		"V2EX 最有趣的社区",           // description: 平台描述
		"https://www.v2ex.com/", // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchV2exHot 从V2EX API 获取数据
func (h *V2exHandler) fetchV2exHot(ctx context.Context, topicType string) ([]models.HotData, error) {
	var apiURL string
	if topicType == "hot" {
		apiURL = "https://r.jina.ai/https://www.v2ex.com/api/topics/hot.json"
	} else {
		apiURL = "https://r.jina.ai/https://www.v2ex.com/api/topics/latest.json"
	}

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":          "text/plain; charset=utf-8",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求V2EX API 失败: %w", err)
	}

	items, err := parseJinaJSONArray(body)
	if err != nil {
		return nil, fmt.Errorf("解析V2EX响应失败: %w", err)
	}

	return h.transformData(items), nil
}

// transformData 转换数据格式
func (h *V2exHandler) transformData(items []V2exItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 构建时间戳(如果存在则转换为毫秒)
		var timestamp interface{} // 保持原始类型
		if item.Created > 0 {
			timestamp = item.Created * 1000 // 转换为毫秒
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Desc:      item.Content,
			Author:    item.Member.Username,
			Hot:       int64(item.Replies),
			Timestamp: timestamp, // 包含时间戳(如果 API 返回了)
			URL:       item.URL,
			MobileURL: item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// V2exItem 主题项
type V2exItem struct {
	ID      int64      `json:"id"`
	Title   string     `json:"title"`
	Content string     `json:"content"`
	URL     string     `json:"url"`
	Replies int        `json:"replies"`
	Created int64      `json:"created"` // 创建时间戳(秒级)
	Member  V2exMember `json:"member"`
}

// V2exMember 成员信息
type V2exMember struct {
	Username string `json:"username"`
}

func parseJinaJSONArray(raw []byte) ([]V2exItem, error) {
	text := string(raw)
	start := strings.Index(text, "[{")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("未找到数据段")
	}
	data := text[start : end+1]

	var builder strings.Builder
	builder.Grow(len(data) * 2)
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		default:
			builder.WriteByte(data[i])
		}
	}
	clean := builder.String()

	var items []V2exItem
	if err := json.Unmarshal([]byte(clean), &items); err != nil {
		return nil, err
	}
	return items, nil
}
