package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// KuaishouHandler 快手处理器
type KuaishouHandler struct {
	fetcher *service.Fetcher
}

// NewKuaishouHandler 创建快手处理器
func NewKuaishouHandler(fetcher *service.Fetcher) *KuaishouHandler {
	return &KuaishouHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *KuaishouHandler) GetPath() string {
	return "/kuaishou"
}

// Handle 处理请求
func (h *KuaishouHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchKuaishouHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"kuaishou_hot",
		"快手",
		"热榜",
		"快手热榜列表",
		"https://www.kuaishou.com",
		nil,
		data,
		!noCache,
	))
}

// fetchKuaishouHot 从快手获取热榜数据
func (h *KuaishouHandler) fetchKuaishouHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.kuaishou.com/?isHome=1"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("请求快手失败: %w", err)
	}

	// 从HTML中提取JSON数据
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析HTML提取JSON数据
func (h *KuaishouHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	// 正则匹配 window.__APOLLO_STATE__
	pattern := regexp.MustCompile(`window\.__APOLLO_STATE__=(.*?);\(function\(\)`)
	matches := pattern.FindStringSubmatch(html)

	if len(matches) < 2 {
		return result
	}

	// 解析JSON
	var apolloState map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &apolloState); err != nil {
		return result
	}

	// 获取defaultClient
	defaultClient, ok := apolloState["defaultClient"].(map[string]interface{})
	if !ok {
		return result
	}

	// 获取热榜items
	hotRankKey := `$ROOT_QUERY.visionHotRank({"page":"home"})`
	hotRank, ok := defaultClient[hotRankKey].(map[string]interface{})
	if !ok {
		return result
	}

	items, ok := hotRank["items"].([]interface{})
	if !ok {
		return result
	}

	// 提取每个热榜项
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		itemID, ok := itemMap["id"].(string)
		if !ok {
			continue
		}

		// 获取详细数据
		hotItem, ok := defaultClient[itemID].(map[string]interface{})
		if !ok {
			continue
		}

		// 提取字段
		id, _ := hotItem["id"].(string)
		name, _ := hotItem["name"].(string)
		poster, _ := hotItem["poster"].(string)
		hotValue, _ := hotItem["hotValue"].(string)

		// 提取photoId
		photoID := ""
		if photoIds, ok := hotItem["photoIds"].(map[string]interface{}); ok {
			if jsonStr, ok := photoIds["json"].([]interface{}); ok && len(jsonStr) > 0 {
				photoID, _ = jsonStr[0].(string)
			}
		}

		// 解析热度值
		hot := h.parseChineseNumber(hotValue)

		hotData := models.HotData{
			ID:        id,
			Title:     name,
			Cover:     poster,
			Hot:       hot,
			URL:       fmt.Sprintf("https://www.kuaishou.com/short-video/%s", photoID),
			MobileURL: fmt.Sprintf("https://www.kuaishou.com/short-video/%s", photoID),
		}

		result = append(result, hotData)
	}

	return result
}

// parseChineseNumber 解析中文数字(如 "1.2万" -> 12000)
func (h *KuaishouHandler) parseChineseNumber(s string) int64 {
	if s == "" {
		return 0
	}

	// 提取数字部分
	numPattern := regexp.MustCompile(`([\d.]+)`)
	numMatches := numPattern.FindStringSubmatch(s)
	if len(numMatches) < 2 {
		return 0
	}

	numStr := numMatches[1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// 判断单位
	if regexp.MustCompile(`万`).MatchString(s) {
		return int64(num * 10000)
	} else if regexp.MustCompile(`亿`).MatchString(s) {
		return int64(num * 100000000)
	}

	return int64(num)
}
