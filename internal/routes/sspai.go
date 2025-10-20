package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// SspaiHandler 少数派处理器
type SspaiHandler struct {
	fetcher *service.Fetcher
}

// NewSspaiHandler 创建少数派处理器
func NewSspaiHandler(fetcher *service.Fetcher) *SspaiHandler {
	return &SspaiHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *SspaiHandler) GetPath() string {
	return "/sspai"
}

// Handle 处理请求
func (h *SspaiHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	tag := c.Query("type", "热门文章")
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchSspaiHot(c.Context(), tag)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"sspai",                            // name: 平台调用名称
		"少数派",                              // title: 平台显示名称
		fmt.Sprintf("热榜 · %s", tag),        // type: 榜单类型
		"发现少数派热门文章",                        // description: 平台描述
		"https://sspai.com/",               // link: 官方链接
		map[string]interface{}{"tag": tag}, // params: 标签参数映射
		data,                               // data: 热榜数据
		!noCache,                           // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchSspaiHot 从少数派 API 获取数据
func (h *SspaiHandler) fetchSspaiHot(ctx context.Context, tag string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://sspai.com/api/v1/article/tag/page/get?limit=40&tag=%s", url.QueryEscape(tag))

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求少数派 API 失败: %w", err)
	}

	var apiResp SspaiAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析少数派响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *SspaiHandler) transformData(items []SspaiItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Desc:      item.Summary,
			Cover:     item.Banner,
			Author:    item.Author.Nickname,
			Hot:       item.LikeCount,
			URL:       fmt.Sprintf("https://sspai.com/post/%d", item.ID),
			MobileURL: fmt.Sprintf("https://sspai.com/post/%d", item.ID),
			Timestamp: item.ReleasedTime * 1000, // 时间戳转换为毫秒级
		}

		result = append(result, hotData)
	}

	return result
}

// SspaiAPIResponse 少数派 API 响应
type SspaiAPIResponse struct {
	Data []SspaiItem `json:"data"`
}

// SspaiItem 文章项
type SspaiItem struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	Banner       string      `json:"banner"`
	Author       SspaiAuthor `json:"author"`
	LikeCount    int64       `json:"like_count"`
	ReleasedTime int64       `json:"released_time"`
}

// SspaiAuthor 作者信息
type SspaiAuthor struct {
	Nickname string `json:"nickname"`
}
