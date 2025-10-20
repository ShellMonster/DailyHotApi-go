package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// EarthquakeHandler 中国地震台处理器
type EarthquakeHandler struct {
	fetcher *service.Fetcher
}

// NewEarthquakeHandler 创建中国地震台处理器
func NewEarthquakeHandler(fetcher *service.Fetcher) *EarthquakeHandler {
	return &EarthquakeHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *EarthquakeHandler) GetPath() string {
	return "/earthquake"
}

// Handle 处理请求
func (h *EarthquakeHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchEarthquake(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"earthquake_speedsearch",
		"中国地震台",
		"地震速报",
		"中国地震台地震速报列表",
		"https://news.ceic.ac.cn/speedsearch.html",
		nil,
		data,
		!noCache,
	))
}

// fetchEarthquake 从中国地震台网站获取数据
func (h *EarthquakeHandler) fetchEarthquake(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://news.ceic.ac.cn/speedsearch.html"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求中国地震台失败: %w", err)
	}

	// 使用正则表达式提取 JavaScript 中的数据
	re := regexp.MustCompile(`const newdata = (\[.*?\]);`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("未找到地震数据")
	}

	// 解析 JSON 数组
	var earthquakeList []EarthquakeItem
	if err := json.Unmarshal(matches[1], &earthquakeList); err != nil {
		return nil, fmt.Errorf("解析地震数据失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(earthquakeList), nil
}

// transformData 将地震数据转换为统一格式
func (h *EarthquakeHandler) transformData(items []EarthquakeItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理EpiDepth字段(可能是number或string)
		var depth string
		switch v := item.EpiDepth.(type) {
		case float64:
			depth = fmt.Sprintf("%.0f", v)
		case int64:
			depth = fmt.Sprintf("%d", v)
		case string:
			depth = v
		default:
			depth = "0"
		}

		// 构建描述信息
		desc := fmt.Sprintf("发震时刻(UTC+8)：%s\n参考位置：%s\n震级(M)：%s\n纬度(°)：%s\n经度(°)：%s\n深度(千米)：%s\n录入时间：%s",
			item.OTime, item.LocationC, item.M, item.EpiLat, item.EpiLon, depth, item.SaveTime)

		hotData := models.HotData{
			ID:        item.NewDID,
			Title:     fmt.Sprintf("%s发生%s级地震", item.LocationC, item.M),
			Desc:      desc,
			Timestamp: item.OTime,
			URL:       fmt.Sprintf("https://news.ceic.ac.cn/%s.html", item.NewDID),
			MobileURL: fmt.Sprintf("https://news.ceic.ac.cn/%s.html", item.NewDID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是地震数据的响应结构体定义

// EarthquakeItem 单个地震记录
type EarthquakeItem struct {
	NewDID    string      `json:"NEW_DID"`    // 地震 ID
	OTime     string      `json:"O_TIME"`     // 发震时刻
	LocationC string      `json:"LOCATION_C"` // 参考位置
	M         string      `json:"M"`          // 震级
	EpiLat    string      `json:"EPI_LAT"`    // 纬度
	EpiLon    string      `json:"EPI_LON"`    // 经度
	EpiDepth  interface{} `json:"EPI_DEPTH"`  // 深度 (可能是number或string)
	SaveTime  string      `json:"SAVE_TIME"`  // 录入时间
}
