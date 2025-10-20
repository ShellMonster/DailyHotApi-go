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

// GeekParkHandler 极客公园处理器
type GeekParkHandler struct {
	fetcher *service.Fetcher
}

// NewGeekParkHandler 创建极客公园处理器
func NewGeekParkHandler(fetcher *service.Fetcher) *GeekParkHandler {
	return &GeekParkHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *GeekParkHandler) GetPath() string {
	return "/geekpark"
}

// Handle 处理请求
func (h *GeekParkHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchGeekParkHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"geekpark_hot",
		"极客公园",
		"热门文章",
		"极客公园热门文章列表",
		"https://www.geekpark.net",
		nil,
		data,
		!noCache,
	))
}

// fetchGeekParkHot 从极客公园 API 获取数据
func (h *GeekParkHandler) fetchGeekParkHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://mainssl.geekpark.net/api/v2"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求极客公园 API 失败: %w", err)
	}

	var apiResp GeekParkAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析极客公园响应失败: %w", err)
	}

	return h.transformData(apiResp.HomepagePosts), nil
}

// transformData 转换数据格式
func (h *GeekParkHandler) transformData(items []GeekParkHomeItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		post := item.Post

		// 提取作者
		author := ""
		if len(post.Authors) > 0 {
			author = post.Authors[0].Nickname
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(post.ID, 10),
			Title:     post.Title,
			Desc:      post.Abstract,
			Cover:     post.CoverURL,
			Author:    author,
			Hot:       post.Views,
			URL:       fmt.Sprintf("https://www.geekpark.net/news/%d", post.ID),
			MobileURL: fmt.Sprintf("https://www.geekpark.net/news/%d", post.ID),
			Timestamp: post.PublishedTimestamp * 1000, // 时间戳转换为毫秒级
		}

		result = append(result, hotData)
	}

	return result
}

// GeekParkAPIResponse 极客公园 API 响应
type GeekParkAPIResponse struct {
	HomepagePosts []GeekParkHomeItem `json:"homepage_posts"`
}

// GeekParkHomeItem 首页项
type GeekParkHomeItem struct {
	Post GeekParkPost `json:"post"`
}

// GeekParkPost 文章
type GeekParkPost struct {
	ID                 int64            `json:"id"`
	Title              string           `json:"title"`
	Abstract           string           `json:"abstract"`
	CoverURL           string           `json:"cover_url"`
	Views              int64            `json:"views"`
	PublishedTimestamp int64            `json:"published_timestamp"`
	Authors            []GeekParkAuthor `json:"authors"`
}

// GeekParkAuthor 作者
type GeekParkAuthor struct {
	Nickname string `json:"nickname"`
}
