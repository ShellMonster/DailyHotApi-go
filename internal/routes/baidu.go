package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// BaiduHandler 百度热搜处理器
type BaiduHandler struct {
	fetcher *service.Fetcher
}

// NewBaiduHandler 创建百度处理器
func NewBaiduHandler(fetcher *service.Fetcher) *BaiduHandler {
	return &BaiduHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *BaiduHandler) GetPath() string {
	return "/baidu"
}

// Handle 处理请求
func (h *BaiduHandler) Handle(c *fiber.Ctx) error {
	// 获取类型参数 (实时/小说/电影等)
	hotType := c.Query("type", "realtime")
	noCache := c.Query("cache") == "false"

	// 类型映射表
	typeMap := map[string]string{
		"realtime": "热搜",
		"novel":    "小说",
		"movie":    "电影",
		"teleplay": "电视剧",
		"car":      "汽车",
		"game":     "游戏",
	}

	// 获取当前类型名称
	typeName := h.getTypeName(hotType)

	// 获取数据
	data, err := h.fetchBaiduHot(c.Context(), hotType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"baidu",                  // name: 平台调用名称
		"百度",                     // title: 平台显示名称
		typeName,                 // type: 当前类型(只返回类型名称,不需要前缀)
		"发现百度热门搜索内容",             // description: 平台描述
		"https://top.baidu.com/", // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// getTypeName 获取类型中文名称
func (h *BaiduHandler) getTypeName(hotType string) string {
	typeMap := map[string]string{
		"realtime": "热搜",
		"novel":    "小说",
		"movie":    "电影",
		"teleplay": "电视剧",
		"car":      "汽车",
		"game":     "游戏",
	}

	if name, ok := typeMap[hotType]; ok {
		return name
	}
	return "热搜"
}

// fetchBaiduHot 从百度获取热搜数据
func (h *BaiduHandler) fetchBaiduHot(ctx context.Context, hotType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://top.baidu.com/board?tab=%s", hotType)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/1.0 Mobile/12F69 Safari/605.1.15",
	})
	if err != nil {
		return nil, fmt.Errorf("请求百度 API 失败: %w", err)
	}

	// 百度的数据嵌入在 HTML 注释中,需要用正则提取
	pattern := regexp.MustCompile(`<!--s-data:(.*?)-->`)
	matches := pattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("未找到百度数据")
	}

	// 解析 JSON
	var dataWrapper BaiduDataWrapper
	if err := json.Unmarshal(matches[1], &dataWrapper); err != nil {
		return nil, fmt.Errorf("解析百度数据失败: %w", err)
	}

	// 检查数据结构
	if len(dataWrapper.Cards) == 0 || dataWrapper.Cards[0].Content == nil {
		return nil, fmt.Errorf("百度数据为空")
	}

	return h.transformData(dataWrapper.Cards[0].Content), nil
}

// transformData 转换数据格式
func (h *BaiduHandler) transformData(items []BaiduItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理Show字段(可能是string或array)
		author := ""
		switch v := item.Show.(type) {
		case string:
			author = v
		case []interface{}:
			if len(v) > 0 {
				if str, ok := v[0].(string); ok {
					author = str
				}
			}
		}

		// 处理HotScore字段（可能是int64或string）
		var hotScore int64
		switch v := item.HotScore.(type) {
		case float64:
			hotScore = int64(v)
		case int64:
			hotScore = v
		case string:
			// 尝试将字符串转换为int64
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				hotScore = parsed
			}
		default:
			hotScore = 0
		}

		hotData := models.HotData{
			ID:        strconv.Itoa(item.Index),
			Title:     item.Word,
			Desc:      item.Desc,
			Cover:     item.Img,
			Hot:       hotScore,
			URL:       fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(item.Query)),
			MobileURL: item.RawURL,
			Author:    author,
		}

		result = append(result, hotData)
	}

	return result
}

// BaiduDataWrapper 百度数据包装
type BaiduDataWrapper struct {
	Cards []BaiduCard `json:"cards"`
}

// BaiduCard 卡片
type BaiduCard struct {
	Content []BaiduItem `json:"content"`
}

// BaiduItem 热搜项
type BaiduItem struct {
	Index    int         `json:"index"`    // 排名
	Word     string      `json:"word"`     // 热搜词
	Query    string      `json:"query"`    // 查询词
	Desc     string      `json:"desc"`     // 描述
	Img      string      `json:"img"`      // 图片
	Show     interface{} `json:"show"`     // 来源 (可能是string或array)
	RawURL   string      `json:"rawUrl"`   // 移动端 URL
	HotScore interface{} `json:"hotScore"` // 热度分数（可能是int64或string）
}
