package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// HuxiuHandler 虎嗅处理器
type HuxiuHandler struct {
	fetcher *service.Fetcher
}

// NewHuxiuHandler 创建虎嗅处理器
func NewHuxiuHandler(fetcher *service.Fetcher) *HuxiuHandler {
	return &HuxiuHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HuxiuHandler) GetPath() string {
	return "/huxiu"
}

// Handle 处理请求
func (h *HuxiuHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchHuxiuHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"huxiu",                  // name: 平台调用名称
		"虎嗅",                     // title: 平台显示名称
		"24小时",                   // type: 榜单类型
		"发现虎嗅平台热门商业资讯",           // description: 平台描述
		"https://www.huxiu.com/", // link: 官方链接
		nil,                      // params: 无参数映射
		data,                     // data: 热榜数据
		!noCache,                 // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchHuxiuHot 从虎嗅获取数据
func (h *HuxiuHandler) fetchHuxiuHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.huxiu.com/moment/"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求虎嗅失败: %w", err)
	}

	// 从HTML中提取JSON数据
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析HTML提取JSON数据
func (h *HuxiuHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return result
	}

	script := doc.Find("script#__NUXT_DATA__").First()
	if script.Length() == 0 {
		return result
	}

	var payload []interface{}
	if err := json.Unmarshal([]byte(script.Text()), &payload); err != nil {
		return result
	}

	postNodes := extractHuxiuPosts(payload)
	for _, post := range postNodes {
		objectID := toString(post["object_id"])
		content := toString(post["content"])
		publishTime := post["publish_time"]
		url := toString(post["url"])

		if url == "" && objectID != "" {
			url = fmt.Sprintf("https://www.huxiu.com/moment/%s.html", objectID)
		}

		if url == "" {
			continue
		}

		author := ""
		if userMap := toMap(post["user_info"]); userMap != nil {
			author = toString(userMap["username"])
		}

		// 处理标题和描述
		title, desc := h.titleProcessing(content)

		timestamp := timeutil.ParseTime(publishTime)
		mobileURL := url
		if objectID != "" {
			mobileURL = fmt.Sprintf("https://m.huxiu.com/moment/%s.html", objectID)
		}

		hotData := models.HotData{
			ID:        objectID,
			Title:     title,
			Desc:      desc,
			Author:    author,
			URL:       url,
			MobileURL: mobileURL,
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// titleProcessing 标题处理(提取第一句作为标题,其余作为描述)
func (h *HuxiuHandler) titleProcessing(text string) (string, string) {
	// 按双换行符分段
	paragraphs := strings.Split(text, "<br><br>")

	if len(paragraphs) == 0 {
		return "", ""
	}

	// 第一段作为标题,去掉末尾的句号
	title := strings.TrimSuffix(paragraphs[0], "。")

	// 其余段落作为描述
	desc := ""
	if len(paragraphs) > 1 {
		desc = strings.Join(paragraphs[1:], "<br><br>")
	}

	return title, desc
}

func extractHuxiuPosts(payload []interface{}) []map[string]interface{} {
	index := -1
	for _, item := range payload {
		if m := toMap(item); m != nil {
			if _, ok := m["moment_list"]; ok {
				index = toInt(m["moment_list"])
				break
			}
		}
	}

	if index < 0 || index >= len(payload) {
		return nil
	}

	momentList := toMap(payload[index])
	if momentList == nil {
		return nil
	}

	resolvedList := resolveRef(momentList["datalist"], payload)
	rawList := toSlice(resolvedList)
	if rawList == nil {
		return nil
	}

	results := make([]map[string]interface{}, 0, len(rawList))

	for _, item := range rawList {
		resolved := resolveRef(item, payload)
		if postMap := toMap(resolved); postMap != nil {
			normalized := make(map[string]interface{}, len(postMap))
			for key, val := range postMap {
				normalized[key] = resolveRef(val, payload)
			}
			results = append(results, normalized)
		}
	}

	return results
}

func toMap(value interface{}) map[string]interface{} {
	if m, ok := value.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func toSlice(value interface{}) []interface{} {
	if arr, ok := value.([]interface{}); ok {
		return arr
	}
	return nil
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return ""
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return -1
	}
}

func resolveRef(value interface{}, payload []interface{}) interface{} {
	return resolveRefDepth(value, payload, 0)
}

func resolveRefDepth(value interface{}, payload []interface{}, depth int) interface{} {
	if depth > 8 {
		return value
	}

	switch v := value.(type) {
	case float64:
		index := int(v)
		if index >= 0 && index < len(payload) {
			return resolveRefDepth(payload[index], payload, depth+1)
		}
	case []interface{}:
		resolved := make([]interface{}, len(v))
		for i, item := range v {
			resolved[i] = resolveRefDepth(item, payload, depth+1)
		}
		return resolved
	case map[string]interface{}:
		resolved := make(map[string]interface{}, len(v))
		for key, item := range v {
			resolved[key] = resolveRefDepth(item, payload, depth+1)
		}
		return resolved
	}

	return value
}
