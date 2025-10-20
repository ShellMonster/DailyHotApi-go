package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// NewsmthHandler 水木社区处理器
type NewsmthHandler struct {
	fetcher *service.Fetcher
}

// NewNewsmthHandler 创建水木社区处理器
func NewNewsmthHandler(fetcher *service.Fetcher) *NewsmthHandler {
	return &NewsmthHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *NewsmthHandler) GetPath() string {
	return "/newsmth"
}

// Handle 处理请求
func (h *NewsmthHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchNewsmth(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"newsmth",                  // name: 平台调用名称
		"水木社区",                     // title: 平台显示名称
		"热门话题",                     // type: 榜单类型
		"发现水木社区热门话题讨论",             // description: 平台描述
		"https://www.newsmth.net/", // link: 官方链接
		nil,                        // params: 无参数映射
		data,                       // data: 热榜数据
		!noCache,                   // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchNewsmth 从水木社区 API 获取数据
func (h *NewsmthHandler) fetchNewsmth(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://wap.newsmth.net/wap/api/hot/global"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求水木社区 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp NewsmthAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析水木社区响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.Topics), nil
}

// transformData 将水木社区原始数据转换为统一格式
func (h *NewsmthHandler) transformData(items []NewsmthTopic) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		post := item.Article

		// 构建 URL
		boardTitle := ""
		if item.Board != nil {
			boardTitle = item.Board.Title
		}

		topicURL := fmt.Sprintf("https://wap.newsmth.net/article/%s?title=%s&from=home",
			post.TopicID, url.QueryEscape(boardTitle))

		// 作者名称
		author := ""
		if post.Account != nil {
			author = post.Account.Name
		}

		// 处理PostTime字段(可能是number或string)
		var timestamp string
		switch v := post.PostTime.(type) {
		case float64:
			// API返回的是Unix时间戳(秒)
			timestamp = fmt.Sprintf("%d", int64(v))
		case int64:
			timestamp = fmt.Sprintf("%d", v)
		case string:
			timestamp = v
		default:
			timestamp = ""
		}

		hotData := models.HotData{
			ID:        item.FirstArticleID,
			Title:     post.Subject,
			Desc:      post.Body,
			Author:    author,
			Timestamp: timestamp,
			URL:       topicURL,
			MobileURL: topicURL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是水木社区 API 的响应结构体定义

// NewsmthAPIResponse 水木社区 API 响应
type NewsmthAPIResponse struct {
	Data NewsmthData `json:"data"`
}

// NewsmthData 数据部分
type NewsmthData struct {
	Topics []NewsmthTopic `json:"topics"`
}

// NewsmthTopic 话题
type NewsmthTopic struct {
	FirstArticleID string         `json:"firstArticleId"` // 第一条文章 ID
	Article        NewsmthArticle `json:"article"`        // 文章信息
	Board          *NewsmthBoard  `json:"board"`          // 版块信息
}

// NewsmthArticle 文章
type NewsmthArticle struct {
	TopicID  string          `json:"topicId"`  // 话题 ID
	Subject  string          `json:"subject"`  // 标题
	Body     string          `json:"body"`     // 正文
	PostTime interface{}     `json:"postTime"` // 发布时间 (可能是number或string)
	Account  *NewsmthAccount `json:"account"`  // 账号信息
}

// NewsmthBoard 版块
type NewsmthBoard struct {
	Title string `json:"title"` // 版块标题
}

// NewsmthAccount 账号
type NewsmthAccount struct {
	Name string `json:"name"` // 用户名
}
