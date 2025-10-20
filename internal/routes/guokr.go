package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// GuokrHandler 果壳处理器
type GuokrHandler struct {
	fetcher *service.Fetcher
}

// NewGuokrHandler 创建果壳处理器
func NewGuokrHandler(fetcher *service.Fetcher) *GuokrHandler {
	return &GuokrHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *GuokrHandler) GetPath() string {
	return "/guokr"
}

// Handle 处理请求
func (h *GuokrHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchGuokr(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"guokr",                  // name: 平台调用名称
		"果壳",                     // title: 平台显示名称
		"热门文章",                   // type: 榜单类型
		"发现果壳平台科技热门文章",           // description: 平台描述
		"https://www.guokr.com/", // link: 官方链接
		nil,                      // params: 无参数映射
		data,                     // data: 热榜数据
		!noCache,                 // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchGuokr 从果壳 API 获取数据
func (h *GuokrHandler) fetchGuokr(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.guokr.com/beta/proxy/science_api/articles?limit=30"

	// 发起 HTTP 请求(需要特定 User-Agent)
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求果壳 API 失败: %w", err)
	}

	// 解析 JSON 响应(直接是数组)
	var items []GuokrItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("解析果壳响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(items), nil
}

// transformData 将果壳原始数据转换为统一格式
func (h *GuokrHandler) transformData(items []GuokrItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 时间戳转换
		timestamp := item.DateModified

		// 作者昵称
		author := ""
		if item.Author != nil {
			author = item.Author.Nickname
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Desc:      item.Summary,
			Cover:     item.SmallImage,
			Author:    author,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.guokr.com/article/%d", item.ID),
			MobileURL: fmt.Sprintf("https://m.guokr.com/article/%d", item.ID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是果壳 API 的响应结构体定义

// GuokrItem 单个文章项
type GuokrItem struct {
	ID           int64        `json:"id"`            // 文章 ID
	Title        string       `json:"title"`         // 标题
	Summary      string       `json:"summary"`       // 摘要
	SmallImage   string       `json:"small_image"`   // 封面图
	Author       *GuokrAuthor `json:"author"`        // 作者信息
	DateModified string       `json:"date_modified"` // 修改时间
}

// GuokrAuthor 作者信息
type GuokrAuthor struct {
	Nickname string `json:"nickname"` // 昵称
}
