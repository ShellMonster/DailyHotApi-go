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

// IfanrHandler 爱范儿处理器
type IfanrHandler struct {
	fetcher *service.Fetcher
}

// NewIfanrHandler 创建爱范儿处理器
func NewIfanrHandler(fetcher *service.Fetcher) *IfanrHandler {
	return &IfanrHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *IfanrHandler) GetPath() string {
	return "/ifanr"
}

// Handle 处理请求
func (h *IfanrHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchIfanr(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"ifanr", // name: 平台调用名称
		"爱范儿",   // title: 平台显示名称
		"快讯",    // type: 榜单类型
		"发现爱范儿平台热门科技快讯",          // description: 平台描述
		"https://www.ifanr.com/", // link: 官方链接
		nil,                      // params: 无参数映射
		data,                     // data: 热榜数据
		!noCache,                 // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchIfanr 从爱范儿 API 获取数据
func (h *IfanrHandler) fetchIfanr(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://sso.ifanr.com/api/v5/wp/buzz/?limit=20&offset=0"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求爱范儿 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp IfanrAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析爱范儿响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Objects), nil
}

// transformData 将爱范儿原始数据转换为统一格式
func (h *IfanrHandler) transformData(items []IfanrItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 热度(优先使用点赞数,否则评论数)
		hot := item.LikeCount
		if hot == 0 {
			hot = item.CommentCount
		}

		// URL处理
		postID := item.PostID
		if postID == "" {
			postID = strconv.FormatInt(item.ID, 10)
		}

		url := item.BuzzOriginalURL
		if url == "" {
			url = fmt.Sprintf("https://www.ifanr.com/%s", postID)
		}

		mobileURL := item.BuzzOriginalURL
		if mobileURL == "" {
			mobileURL = fmt.Sprintf("https://www.ifanr.com/digest/%s", postID)
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.PostTitle,
			Desc:      item.PostContent,
			Hot:       hot,
			Timestamp: item.CreatedAt * 1000, // 时间戳转换为毫秒(秒级×1000)
			URL:       url,
			MobileURL: mobileURL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是爱范儿 API 的响应结构体定义

// IfanrAPIResponse 爱范儿 API 响应
type IfanrAPIResponse struct {
	Objects []IfanrItem `json:"objects"`
}

// IfanrItem 单个快讯项
type IfanrItem struct {
	ID              int64  `json:"id"`                // ID
	PostID          string `json:"post_id"`           // 文章 ID(接口返回字符串)
	PostTitle       string `json:"post_title"`        // 标题
	PostContent     string `json:"post_content"`      // 内容
	BuzzOriginalURL string `json:"buzz_original_url"` // 原文链接
	LikeCount       int64  `json:"like_count"`        // 点赞数
	CommentCount    int64  `json:"comment_count"`     // 评论数
	CreatedAt       int64  `json:"created_at"`        // 创建时间(时间戳)
}
