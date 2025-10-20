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

// SinaHandler 新浪网处理器
type SinaHandler struct {
	fetcher *service.Fetcher
}

// NewSinaHandler 创建新浪网处理器
func NewSinaHandler(fetcher *service.Fetcher) *SinaHandler {
	return &SinaHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *SinaHandler) GetPath() string {
	return "/sina"
}

// Handle 处理请求
func (h *SinaHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	hotType := c.Query("type", "all") // 默认新浪热榜
	noCache := c.Query("cache") == "false"

	// 获取热榜数据
	data, err := h.fetchSina(c.Context(), hotType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 准备类型映射表 - 用于前端显示支持的参数选项
	typeMap := map[string]string{
		"all":       "新浪热榜",
		"hotcmnt":   "热议榜",
		"minivideo": "视频热榜",
		"ent":       "娱乐热榜",
		"ai":        "AI热榜",
		"auto":      "汽车热榜",
		"mother":    "育儿热榜",
		"fashion":   "时尚热榜",
		"travel":    "旅游热榜",
		"esg":       "ESG热榜",
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"sina",                  // name: 平台调用名称
		"新浪网",                   // title: 平台显示名称
		h.getTypeName(hotType),  // type: 当前榜单类型
		"发现新浪网热门资讯",             // description: 平台描述
		"https://www.sina.com/", // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// getTypeName 获取榜单类型名称
func (h *SinaHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"all":       "新浪热榜",
		"hotcmnt":   "热议榜",
		"minivideo": "视频热榜",
		"ent":       "娱乐热榜",
		"ai":        "AI热榜",
		"auto":      "汽车热榜",
		"mother":    "育儿热榜",
		"fashion":   "时尚热榜",
		"travel":    "旅游热榜",
		"esg":       "ESG热榜",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "新浪热榜"
}

// fetchSina 从新浪网 API 获取数据
func (h *SinaHandler) fetchSina(ctx context.Context, hotType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://newsapp.sina.cn/api/hotlist?newsId=HB-1-snhs%%2Ftop_news_list-%s", hotType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求新浪网 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp SinaAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析新浪网响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.HotList), nil
}

// transformData 将新浪网原始数据转换为统一格式
func (h *SinaHandler) transformData(items []SinaHotItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		base := item.Base.Base
		info := item.Info

		// 解析中文热度值(如 "10万")
		hot := h.parseChineseNumber(info.HotValue)

		hotData := models.HotData{
			ID:        base.UniqueID,
			Title:     info.Title,
			Hot:       hot,
			URL:       base.URL,
			MobileURL: base.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// parseChineseNumber 解析中文数字(如 "10万" -> 100000)
func (h *SinaHandler) parseChineseNumber(numStr string) int64 {
	// 移除空格
	numStr = regexp.MustCompile(`\s+`).ReplaceAllString(numStr, "")

	// 匹配数字部分
	re := regexp.MustCompile(`([\d.]+)([万亿千百]?)`)
	matches := re.FindStringSubmatch(numStr)

	if len(matches) < 2 {
		return 0
	}

	// 解析基础数字
	baseNum, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	// 处理单位
	if len(matches) >= 3 {
		switch matches[2] {
		case "万":
			baseNum *= 10000
		case "亿":
			baseNum *= 100000000
		case "千":
			baseNum *= 1000
		case "百":
			baseNum *= 100
		}
	}

	return int64(baseNum)
}

// 以下是新浪网 API 的响应结构体定义

// SinaAPIResponse 新浪网 API 响应
type SinaAPIResponse struct {
	Data SinaData `json:"data"`
}

// SinaData 数据部分
type SinaData struct {
	HotList []SinaHotItem `json:"hotList"`
}

// SinaHotItem 单个热榜项
type SinaHotItem struct {
	Base SinaBase `json:"base"`
	Info SinaInfo `json:"info"`
}

// SinaBase 基础信息
type SinaBase struct {
	Base SinaBaseDetail `json:"base"`
}

// SinaBaseDetail 基础详情
type SinaBaseDetail struct {
	UniqueID string `json:"uniqueId"` // 唯一 ID
	URL      string `json:"url"`      // 链接
}

// SinaInfo 信息部分
type SinaInfo struct {
	Title    string `json:"title"`    // 标题
	HotValue string `json:"hotValue"` // 热度值(中文)
}
