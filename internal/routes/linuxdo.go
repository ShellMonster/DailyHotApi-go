package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// LinuxdoHandler Linux.do 处理器
type LinuxdoHandler struct {
	fetcher *service.Fetcher
}

// NewLinuxdoHandler 创建 Linux.do 处理器
func NewLinuxdoHandler(fetcher *service.Fetcher) *LinuxdoHandler {
	return &LinuxdoHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *LinuxdoHandler) GetPath() string {
	return "/linuxdo"
}

// Handle 处理请求
func (h *LinuxdoHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchLinuxdo(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"linuxdo_weekly",
		"Linux.do",
		"热门文章",
		"Linux.do热门文章列表",
		"https://linux.do",
		nil,
		data,
		!noCache,
	))
}

// fetchLinuxdo 从 Linux.do API 获取数据
func (h *LinuxdoHandler) fetchLinuxdo(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://r.jina.ai/https://linux.do/top.json?period=weekly"

	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "text/plain; charset=utf-8",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 Linux.do API 失败: %w", err)
	}

	jsonBytes, err := extractJinaJSON(body)
	if err != nil {
		return nil, fmt.Errorf("解析 Linux.do 响应失败: %w", err)
	}

	var apiResp LinuxdoAPIResponse
	if err := json.Unmarshal(jsonBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 Linux.do 响应失败: %w", err)
	}

	return h.transformData(apiResp.TopicList.Topics), nil
}

// transformData 将 Linux.do 原始数据转换为统一格式
func (h *LinuxdoHandler) transformData(items []LinuxdoTopic) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 热度(优先使用浏览数,否则点赞数)
		hot := item.Views
		if hot == 0 {
			hot = item.LikeCount
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Desc:      item.Excerpt,
			Author:    item.LastPosterUsername,
			Hot:       hot,
			Timestamp: timeutil.ParseTime(item.CreatedAt),
			URL:       fmt.Sprintf("https://linux.do/t/%d", item.ID),
			MobileURL: fmt.Sprintf("https://linux.do/t/%d", item.ID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是 Linux.do API 的响应结构体定义

// LinuxdoAPIResponse Linux.do API 响应
type LinuxdoAPIResponse struct {
	TopicList LinuxdoTopicList `json:"topic_list"`
}

// LinuxdoTopicList 话题列表
type LinuxdoTopicList struct {
	Topics []LinuxdoTopic `json:"topics"`
}

// LinuxdoTopic 单个话题
type LinuxdoTopic struct {
	ID                 int64  `json:"id"`                   // 话题 ID
	Title              string `json:"title"`                // 标题
	Excerpt            string `json:"excerpt"`              // 摘要
	LastPosterUsername string `json:"last_poster_username"` // 最后回复用户
	CreatedAt          string `json:"created_at"`           // 创建时间
	Views              int64  `json:"views"`                // 浏览数
	LikeCount          int64  `json:"like_count"`           // 点赞数
}

func extractJinaJSON(raw []byte) ([]byte, error) {
	text := string(raw)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("未找到有效 JSON")
	}
	return []byte(text[start : end+1]), nil
}
