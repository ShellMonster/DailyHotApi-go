package routes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	httpclient "github.com/dailyhot/api/internal/http"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/mmcdole/gofeed"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// PojieHandler 吾爱破解处理器
type PojieHandler struct {
	fetcher *service.Fetcher
}

// NewPojieHandler 创建吾爱破解处理器
func NewPojieHandler(fetcher *service.Fetcher) *PojieHandler {
	return &PojieHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *PojieHandler) GetPath() string {
	return "/52pojie"
}

// Handle 处理请求
func (h *PojieHandler) Handle(c *fiber.Ctx) error {
	pojieType := c.Query("type", "digest")
	noCache := c.Query("cache") == "false"
	typeMap := map[string]string{
		"digest":    "最新精华",
		"hot":       "最新热门",
		"new":       "最新回复",
		"newthread": "最新发表",
	}
	data, actualType, err := h.fetchPojie(c.Context(), pojieType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}
	resp := models.SuccessResponse(
		"52pojie", "吾爱破解", h.getTypeName(actualType), "发现吾爱破解热门讨论",
		"https://www.52pojie.cn/", map[string]interface{}{"type": typeMap, "actualType": actualType},
		data, !noCache,
	)
	return c.JSON(resp)
}

// getTypeName 获取类型名称
func (h *PojieHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"digest":    "最新精华",
		"hot":       "最新热门",
		"new":       "最新回复",
		"newthread": "最新发表",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "最新精华"
}

// fetchPojie 从吾爱破解 RSS 获取数据
func (h *PojieHandler) fetchPojie(ctx context.Context, pojieType string) ([]models.HotData, string, error) {
	apiURL := fmt.Sprintf("https://www.52pojie.cn/forum.php?mod=guide&view=%s&rss=1", pojieType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36",
	}

	data, err := h.fetchPojieWithType(httpClient, apiURL, headers)
	if err != nil {
		return nil, pojieType, err
	}

	// 若默认的 digest 无数据,尝试回退到 hot
	if len(data) == 0 && pojieType == "digest" {
		fallbackType := "hot"
		fallbackURL := fmt.Sprintf("https://www.52pojie.cn/forum.php?mod=guide&view=%s&rss=1", fallbackType)
		if fallbackData, ferr := h.fetchPojieWithType(httpClient, fallbackURL, headers); ferr == nil && len(fallbackData) > 0 {
			return fallbackData, fallbackType, nil
		}
	}

	return data, pojieType, nil
}

func (h *PojieHandler) fetchPojieWithType(httpClient *httpclient.Client, apiURL string, headers map[string]string) ([]models.HotData, error) {
	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求吾爱破解 RSS 失败: %w", err)
	}

	reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("转换 RSS 编码失败: %w", err)
	}

	fp := gofeed.NewParser()
	content := string(utf8Data)
	content = strings.Replace(content, `encoding="gbk"`, `encoding="utf-8"`, 1)
	content = strings.Replace(content, `encoding='gbk'`, `encoding='utf-8'`, 1)
	content = strings.Replace(content, `encoding="GBK"`, `encoding="utf-8"`, 1)
	content = strings.Replace(content, `encoding='GBK'`, `encoding='utf-8'`, 1)

	feed, err := fp.ParseString(content)
	if err != nil {
		return nil, fmt.Errorf("解析 RSS 失败: %w", err)
	}

	return h.transformData(feed.Items), nil
}

// transformData 将 RSS 数据转换为统一格式
func (h *PojieHandler) transformData(items []*gofeed.Item) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for i, item := range items {
		// 获取 ID(使用 GUID 或索引)
		id := item.GUID
		if id == "" {
			id = fmt.Sprintf("%d", i)
		}

		// 描述
		desc := item.Description
		if desc == "" && item.Content != "" {
			desc = item.Content
		}

		// 时间戳
		timestamp := ""
		if item.PublishedParsed != nil {
			timestamp = item.PublishedParsed.Format(time.RFC3339)
		}

		// 作者
		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}

		hotData := models.HotData{
			ID:        id,
			Title:     item.Title,
			Desc:      desc,
			Author:    author,
			Timestamp: timestamp,
			URL:       item.Link,
			MobileURL: item.Link,
		}

		result = append(result, hotData)
	}

	return result
}
