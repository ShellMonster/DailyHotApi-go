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

// ZhihuHandler 知乎热榜处理器
type ZhihuHandler struct {
	fetcher *service.Fetcher
}

// NewZhihuHandler 创建知乎处理器
func NewZhihuHandler(fetcher *service.Fetcher) *ZhihuHandler {
	return &ZhihuHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *ZhihuHandler) GetPath() string {
	return "/zhihu"
}

// Handle 处理请求
func (h *ZhihuHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	noCache := c.Query("cache") == "false"

	// 获取热榜数据
	data, err := h.fetchZhihuHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"zhihu",                     // name: 平台调用名称
		"知乎",                        // title: 平台显示名称
		"热榜",                        // type: 榜单类型
		"发现知乎热门话题",                  // description: 平台描述
		"https://www.zhihu.com/hot", // link: 官方链接
		nil,                         // params: 无参数映射
		data,                        // data: 热榜数据
		!noCache,                    // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchZhihuHot 从知乎 API 获取热榜数据
func (h *ZhihuHandler) fetchZhihuHot(ctx context.Context) ([]models.HotData, error) {
	// 知乎热榜 API
	apiURL := "https://api.zhihu.com/topstory/hot-lists/total?limit=50"

	// 发起 HTTP 请求，添加必要的请求头
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		"Referer":         "https://www.zhihu.com/hot",
		"X-Requested-With": "XMLHttpRequest",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求知乎 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp ZhihuAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析知乎响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data), nil
}

// transformData 将知乎原始数据转换为统一格式
func (h *ZhihuHandler) transformData(items []ZhihuItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		target := item.Target

		// 从 URL 中提取问题 ID
		// URL 格式: https://api.zhihu.com/questions/123456 -> 需要替换为 https://www.zhihu.com/question/123456
		// 参考 TypeScript: link: e.target.url.replace('api.', 'www.').replace('questions', 'question')
		url := strings.ReplaceAll(target.URL, "api.", "www.")
		url = strings.ReplaceAll(url, "questions", "question")

		// 解析热度值
		// detail_text 格式: "100 万热度" 或 "100万"
		hot := h.parseHot(item.DetailText)

		// 获取封面图
		cover := ""
		if len(item.Children) > 0 {
			cover = item.Children[0].Thumbnail
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(target.ID, 10),
			Title:     target.Title,
			Desc:      target.Excerpt,
			Cover:     cover,
			URL:       url, // 使用正确转换后的 URL
			Hot:       hot,
			Author:    fmt.Sprintf("回答:%d 关注:%d 评论:%d", target.AnswerCount, target.FollowerCount, target.CommentCount),
			Timestamp: target.Created * 1000, // 时间戳转换为毫秒级
			MobileURL: url,
		}

		result = append(result, hotData)
	}

	return result
}

// parseHot 解析热度文本
// 例如: "100 万热度" -> 1000000
func (h *ZhihuHandler) parseHot(detailText string) int64 {
	// 去掉 "热度" 后缀
	text := strings.TrimSuffix(detailText, "热度")
	text = strings.TrimSpace(text)

	// 分离数字和单位
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return 0
	}

	// 解析数字
	numStr := parts[0]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// 处理单位(万)
	if len(parts) > 1 && parts[1] == "万" {
		num *= 10000
	}

	return int64(num)
}

// 以下是知乎 API 的响应结构体定义

// ZhihuAPIResponse 知乎 API 响应
type ZhihuAPIResponse struct {
	Data []ZhihuItem `json:"data"`
}

// ZhihuItem 单个热榜项
type ZhihuItem struct {
	Target     ZhihuTarget  `json:"target"`      // 目标内容
	DetailText string       `json:"detail_text"` // 热度文本,如 "100 万热度"
	Children   []ZhihuChild `json:"children"`    // 子内容(包含封面图)
}

// ZhihuTarget 目标内容
type ZhihuTarget struct {
	ID             int64  `json:"id"`              // 问题 ID
	Title          string `json:"title"`          // 标题
	Excerpt        string `json:"excerpt"`        // 摘要
	URL            string `json:"url"`            // API URL
	Created        int64  `json:"created"`        // 创建时间戳
	AnswerCount    int64  `json:"answer_count"`   // 回答数
	FollowerCount  int64  `json:"follower_count"` // 关注者数
	CommentCount   int64  `json:"comment_count"`  // 评论数
}

// ZhihuChild 子内容
type ZhihuChild struct {
	Thumbnail string `json:"thumbnail"` // 缩略图
}
