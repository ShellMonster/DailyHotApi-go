package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// HistoryHandler 历史上的今天处理器
type HistoryHandler struct {
	fetcher *service.Fetcher
}

// NewHistoryHandler 创建历史上的今天处理器
func NewHistoryHandler(fetcher *service.Fetcher) *HistoryHandler {
	return &HistoryHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HistoryHandler) GetPath() string {
	return "/history"
}

// Handle 处理请求
func (h *HistoryHandler) Handle(c *fiber.Ctx) error {
	// 获取日期参数
	now := time.Now()
	month := c.Query("month", fmt.Sprintf("%d", int(now.Month())))
	day := c.Query("day", fmt.Sprintf("%d", now.Day()))
	noCache := c.Query("cache") == "false"

	data, err := h.fetchHistory(c.Context(), month, day)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("history_%s_%s", month, day),
		"历史上的今天",
		fmt.Sprintf("%s-%s", month, day),
		"历史上的今天事件列表",
		"https://baike.baidu.com",
		nil,
		data,
		!noCache,
	))
}

// fetchHistory 从百度百科获取历史数据
func (h *HistoryHandler) fetchHistory(ctx context.Context, month, day string) ([]models.HotData, error) {
	// 格式化月份为两位数
	monthInt, _ := strconv.Atoi(month)
	monthStr := fmt.Sprintf("%02d", monthInt)

	// 格式化日期为两位数
	dayInt, _ := strconv.Atoi(day)
	dayStr := fmt.Sprintf("%02d", dayInt)

	apiURL := fmt.Sprintf("https://baike.baidu.com/cms/home/eventsOnHistory/%s.json?_=%d", monthStr, time.Now().UnixMilli())

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求历史数据 API 失败: %w", err)
	}

	// 解析 JSON 响应(结构是 {月份: {月日: [事件列表]}})
	var apiResp map[string]map[string][]HistoryEvent
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析历史数据响应失败: %w", err)
	}

	// 获取指定日期的事件列表
	events, ok := apiResp[monthStr][monthStr+dayStr]
	if !ok {
		return []models.HotData{}, nil
	}

	// 转换为统一格式
	return h.transformData(events), nil
}

// transformData 将历史事件转换为统一格式
func (h *HistoryHandler) transformData(items []HistoryEvent) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	// 用于清理 HTML 标签的正则表达式
	htmlTagRe := regexp.MustCompile(`<[^>]*>`)

	for i, item := range items {
		// 清理 HTML 标签
		title := htmlTagRe.ReplaceAllString(item.Title, "")
		desc := htmlTagRe.ReplaceAllString(item.Desc, "")

		// 封面图
		cover := ""
		if item.Cover {
			cover = item.PicShare
		}

		hotData := models.HotData{
			ID:        strconv.Itoa(i),
			Title:     title,
			Desc:      desc,
			Cover:     cover,
			URL:       item.Link,
			MobileURL: item.Link,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是历史数据的响应结构体定义

// HistoryEvent 历史事件
type HistoryEvent struct {
	Title    string `json:"title"`     // 标题
	Desc     string `json:"desc"`      // 描述
	Cover    bool   `json:"cover"`     // 是否有封面
	PicShare string `json:"pic_share"` // 封面图链接
	Link     string `json:"link"`      // 详情链接
	Year     string `json:"year"`      // 年份
}
