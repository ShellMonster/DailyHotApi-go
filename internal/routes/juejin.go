package routes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// JuejinHandler 掘金热榜处理器
type JuejinHandler struct {
	fetcher *service.Fetcher
}

// NewJuejinHandler 创建掘金处理器
func NewJuejinHandler(fetcher *service.Fetcher) *JuejinHandler {
	return &JuejinHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *JuejinHandler) GetPath() string {
	return "/juejin"
}

// Handle 处理请求
func (h *JuejinHandler) Handle(c *fiber.Ctx) error {
	// 支持不同分类: 1-综合, 6809637767543259144-后端, 等
	categoryID := c.Query("type", "1")
	noCache := c.Query("cache") == "false"

	// 获取热榜数据
	data, err := h.fetchJuejinHot(c.Context(), categoryID)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 准备类型映射表 - 用于前端显示支持的分类
	typeMap := map[string]string{
		"1":                   "综合",
		"6809637767543259144": "后端",
		"6809637767559319566": "前端",
		"6809637769859440654": "iOS",
		"6809637769895981454": "Android",
		"6809637773895446728": "DevOps",
		"6809637774852677639": "人工智能",
		"6809637776263692295": "开源",
	}

	// 获取当前分类名称
	categoryName := typeMap[categoryID]
	if categoryName == "" {
		categoryName = "综合"
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"juejin",                           // name: 平台调用名称
		"掘金",                               // title: 平台显示名称
		fmt.Sprintf("热榜·%s", categoryName), // type: 当前分类
		"发现掘金热门技术内容",                       // description: 平台描述
		"https://juejin.cn/",               // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchJuejinHot 从掘金 API 获取热榜数据
func (h *JuejinHandler) fetchJuejinHot(ctx context.Context, categoryID string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://api.juejin.cn/content_api/v1/content/article_rank?category_id=%s&type=hot", categoryID)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("请求掘金 API 失败: %w", err)
	}

	var apiResp JuejinAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析掘金响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *JuejinHandler) transformData(items []JuejinItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		content := item.Content
		author := item.Author
		counter := item.ContentCounter

		hotData := models.HotData{
			ID:        content.ContentID,
			Title:     content.Title,
			Author:    author.Name,
			Hot:       counter.HotRank,
			URL:       fmt.Sprintf("https://juejin.cn/post/%s", content.ContentID),
			MobileURL: fmt.Sprintf("https://juejin.cn/post/%s", content.ContentID),
		}

		result = append(result, hotData)
	}

	return result
}

// JuejinAPIResponse 掘金 API 响应
type JuejinAPIResponse struct {
	Data []JuejinItem `json:"data"`
}

// JuejinItem 文章项
type JuejinItem struct {
	Content        JuejinContent        `json:"content"`
	Author         JuejinAuthor         `json:"author"`
	ContentCounter JuejinContentCounter `json:"content_counter"`
}

// JuejinContent 文章内容
type JuejinContent struct {
	ContentID string `json:"content_id"`
	Title     string `json:"title"`
}

// JuejinAuthor 作者信息
type JuejinAuthor struct {
	Name string `json:"name"`
}

// JuejinContentCounter 统计信息
type JuejinContentCounter struct {
	HotRank int64 `json:"hot_rank"`
}
