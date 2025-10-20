package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// NgabbsHandler NGA 处理器
type NgabbsHandler struct {
	fetcher *service.Fetcher
}

// NewNgabbsHandler 创建 NGA 处理器
func NewNgabbsHandler(fetcher *service.Fetcher) *NgabbsHandler {
	return &NgabbsHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *NgabbsHandler) GetPath() string {
	return "/ngabbs"
}

// Handle 处理请求
func (h *NgabbsHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchNgabbs(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"ngabbs_hot",
		"NGA",
		"论坛热帖",
		"NGA论坛热帖列表",
		"https://bbs.nga.cn",
		nil,
		data,
		!noCache,
	))
}

// fetchNgabbs 从 NGA API 获取数据
func (h *NgabbsHandler) fetchNgabbs(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://ngabbs.com/nuke.php?__lib=load_topic&__act=load_topic_reply_ladder2&opt=1&all=1"

	// 构建 POST 数据
	formData := url.Values{}
	formData.Set("__output", "14")

	// 发起 HTTP POST 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"Accept":          "*/*",
		"Host":            "ngabbs.com",
		"Referer":         "https://ngabbs.com/",
		"Connection":      "keep-alive",
		"Content-Type":    "application/x-www-form-urlencoded",
		"User-Agent":      "Apifox/1.0.0 (https://apifox.com)",
		"X-User-Agent":    "NGA_skull/7.3.1(iPhone13,2;iOS 17.2.1)",
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "zh-Hans-CN;q=1",
	}

	body, err := httpClient.Post(apiURL, []byte(formData.Encode()), headers)
	if err != nil {
		return nil, fmt.Errorf("请求 NGA API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp NgabbsAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 NGA 响应失败: %w", err)
	}

	// 检查数据
	if len(apiResp.Result) == 0 {
		return nil, fmt.Errorf("NGA 数据为空")
	}

	items, err := decodeNgabbsItems(apiResp.Result[0])
	if err != nil {
		return nil, fmt.Errorf("解析 NGA 列表失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(items), nil
}

// transformData 将 NGA 原始数据转换为统一格式
func (h *NgabbsHandler) transformData(items []NgabbsItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		tid, _ := item.Tid.Int64()
		replies, _ := item.Replies.Int64()
		postdate, _ := item.Postdate.Int64()

		hotData := models.HotData{
			ID:        strconv.FormatInt(tid, 10),
			Title:     normalizeNgabbsSubject(item.Subject),
			Author:    item.Author,
			Hot:       replies,
			Timestamp: postdate * 1000, // 时间戳转换为毫秒(秒级×1000)
			URL:       fmt.Sprintf("https://bbs.nga.cn%s", item.Tpcurl),
			MobileURL: fmt.Sprintf("https://bbs.nga.cn%s", item.Tpcurl),
		}

		result = append(result, hotData)
	}

	return result
}

func normalizeNgabbsSubject(subject interface{}) string {
	switch v := subject.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return ""
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// 以下是 NGA API 的响应结构体定义

type NgabbsAPIResponse struct {
	Result []json.RawMessage `json:"result"`
}

// NgabbsItem 单个帖子项
type NgabbsItem struct {
	Tid      json.Number `json:"tid"`      // 帖子 ID
	Subject  interface{} `json:"subject"`  // 标题(可能为空)
	Author   string      `json:"author"`   // 作者
	Replies  json.Number `json:"replies"`  // 回复数
	Postdate json.Number `json:"postdate"` // 发帖时间(时间戳)
	Tpcurl   string      `json:"tpcurl"`   // 帖子路径
}

func decodeNgabbsItems(raw json.RawMessage) ([]NgabbsItem, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var items []NgabbsItem
	if err := decoder.Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}
