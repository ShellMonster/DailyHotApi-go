package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// SinaNewsHandler 新浪新闻处理器
type SinaNewsHandler struct {
	fetcher *service.Fetcher
}

// NewSinaNewsHandler 创建新浪新闻处理器
func NewSinaNewsHandler(fetcher *service.Fetcher) *SinaNewsHandler {
	return &SinaNewsHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *SinaNewsHandler) GetPath() string {
	return "/sina-news"
}

// Handle 处理请求
func (h *SinaNewsHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	newsType := c.Query("type", "1") // 默认总排行
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchSinaNews(c.Context(), newsType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"sina-news",                 // name: 平台调用名称
		"新浪新闻",                      // title: 平台显示名称
		h.getTypeName(newsType),     // type: 榜单类型
		"发现新浪新闻热门资讯",                // description: 平台描述
		"https://news.sina.com.cn/", // link: 官方链接
		map[string]interface{}{"type": map[string]string{
			"1": "总排行", "2": "视频排行", "3": "图片排行",
			"4": "国内新闻", "5": "国际新闻", "6": "社会新闻",
			"7": "体育新闻", "8": "财经新闻", "9": "娱乐新闻",
			"10": "科技新闻", "11": "军事新闻",
		}}, // params: 类型映射
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// getTypeName 获取榜单类型名称
func (h *SinaNewsHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"1":  "总排行",
		"2":  "视频排行",
		"3":  "图片排行",
		"4":  "国内新闻",
		"5":  "国际新闻",
		"6":  "社会新闻",
		"7":  "体育新闻",
		"8":  "财经新闻",
		"9":  "娱乐新闻",
		"10": "科技新闻",
		"11": "军事新闻",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "总排行"
}

// getTypeParams 获取榜单参数
func (h *SinaNewsHandler) getTypeParams(typeID string) (string, string) {
	typeMap := map[string]struct {
		www    string
		params string
	}{
		"1":  {"news", "www_www_all_suda_suda"},
		"2":  {"news", "video_news_all_by_vv"},
		"3":  {"news", "total_slide_suda"},
		"4":  {"news", "news_china_suda"},
		"5":  {"news", "news_world_suda"},
		"6":  {"news", "news_society_suda"},
		"7":  {"sports", "sports_suda"},
		"8":  {"finance", "finance_0_suda"},
		"9":  {"ent", "ent_suda"},
		"10": {"tech", "tech_news_suda"},
		"11": {"news", "news_mil_suda"},
	}

	if config, ok := typeMap[typeID]; ok {
		return config.www, config.params
	}
	return "news", "www_www_all_suda_suda"
}

// fetchSinaNews 从新浪新闻 API 获取数据
func (h *SinaNewsHandler) fetchSinaNews(ctx context.Context, typeID string) ([]models.HotData, error) {
	www, params := h.getTypeParams(typeID)

	// 获取当前日期(格式: YYYYMMDD)
	now := time.Now()
	dateStr := now.Format("20060102")

	apiURL := fmt.Sprintf("https://top.%s.sina.com.cn/ws/GetTopDataList.php?top_type=day&top_cat=%s&top_time=%s&top_show_num=50",
		www, params, dateStr)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求新浪新闻 API 失败: %w", err)
	}

	// 解析 JSONP 响应(格式: var data = {...};)
	jsonData, err := h.parseJSONP(string(body))
	if err != nil {
		return nil, fmt.Errorf("解析新浪新闻响应失败: %w", err)
	}

	var apiResp SinaNewsAPIResponse
	if err := json.Unmarshal([]byte(jsonData), &apiResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data), nil
}

// parseJSONP 解析 JSONP 格式数据
func (h *SinaNewsHandler) parseJSONP(data string) (string, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", fmt.Errorf("数据为空")
	}

	// 移除 "var data = " 前缀
	prefix := "var data = "
	if !strings.HasPrefix(data, prefix) {
		return "", fmt.Errorf("数据格式错误: 缺少前缀")
	}

	jsonStr := strings.TrimPrefix(data, prefix)
	jsonStr = strings.TrimSpace(jsonStr)

	// 移除末尾的分号
	jsonStr = strings.TrimSuffix(jsonStr, ";")
	jsonStr = strings.TrimSpace(jsonStr)

	return jsonStr, nil
}

// transformData 将新浪新闻原始数据转换为统一格式
func (h *SinaNewsHandler) transformData(items []SinaNewsItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 解析热度值(去掉逗号)
		hotStr := strings.ReplaceAll(item.TopNum, ",", "")
		hot, _ := strconv.ParseInt(hotStr, 10, 64)

		// 解析时间戳
		timestamp := item.CreateDate + " " + item.CreateTime

		hotData := models.HotData{
			ID:        item.ID,
			Title:     item.Title,
			Author:    item.Media,
			Hot:       hot,
			Timestamp: timestamp,
			URL:       item.URL,
			MobileURL: item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是新浪新闻 API 的响应结构体定义

// SinaNewsAPIResponse 新浪新闻 API 响应
type SinaNewsAPIResponse struct {
	Data []SinaNewsItem `json:"data"`
}

// SinaNewsItem 单个新闻项
type SinaNewsItem struct {
	ID         string `json:"id"`          // 新闻 ID
	Title      string `json:"title"`       // 标题
	Media      string `json:"media"`       // 媒体来源
	TopNum     string `json:"top_num"`     // 热度值(带逗号)
	CreateDate string `json:"create_date"` // 创建日期
	CreateTime string `json:"create_time"` // 创建时间
	URL        string `json:"url"`         // 链接
}
