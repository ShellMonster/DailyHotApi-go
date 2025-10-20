package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WeatherAlarmHandler 中央气象台预警处理器
type WeatherAlarmHandler struct {
	fetcher *service.Fetcher
}

// NewWeatherAlarmHandler 创建中央气象台预警处理器
func NewWeatherAlarmHandler(fetcher *service.Fetcher) *WeatherAlarmHandler {
	return &WeatherAlarmHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *WeatherAlarmHandler) GetPath() string {
	return "/weatheralarm"
}

// Handle 处理请求
func (h *WeatherAlarmHandler) Handle(c *fiber.Ctx) error {
	province := c.Query("province", "") // 省份参数(可选)
	noCache := c.Query("cache") == "false"

	subtitle := "全国气象预警"
	if province != "" {
		subtitle = province + "气象预警"
	}

	// 直接调用fetch函数获取数据
	data, err := h.fetchWeatherAlarm(c.Context(), province)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建params
	var params map[string]interface{}
	if province != "" {
		params = map[string]interface{}{
			"province": province,
		}
	}

	// 构建响应
	resp := models.SuccessResponse(
		fmt.Sprintf("weatheralarm_%s", province),
		"中央气象台",
		subtitle,
		"中央气象台预警信息",
		"http://www.nmc.cn",
		params,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchWeatherAlarm 从中央气象台 API 获取预警数据
func (h *WeatherAlarmHandler) fetchWeatherAlarm(ctx context.Context, province string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("http://www.nmc.cn/rest/findAlarm?pageNo=1&pageSize=20&signaltype=&signallevel=&province=%s",
		url.QueryEscape(province))

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求中央气象台 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp WeatherAlarmAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析中央气象台响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.Page.List), nil
}

// transformData 将气象预警数据转换为统一格式
func (h *WeatherAlarmHandler) transformData(items []WeatherAlarmItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 描述包含发布时间
		desc := item.IssueTime + " " + item.Title

		hotData := models.HotData{
			ID:        item.AlertID,
			Title:     item.Title,
			Desc:      desc,
			Cover:     item.Pic,
			Timestamp: item.IssueTime,
			URL:       "http://nmc.cn" + item.URL,
			MobileURL: "http://nmc.cn" + item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是中央气象台 API 的响应结构体定义

// WeatherAlarmAPIResponse 中央气象台 API 响应
type WeatherAlarmAPIResponse struct {
	Data WeatherAlarmData `json:"data"`
}

// WeatherAlarmData 数据部分
type WeatherAlarmData struct {
	Page WeatherAlarmPage `json:"page"`
}

// WeatherAlarmPage 分页信息
type WeatherAlarmPage struct {
	List []WeatherAlarmItem `json:"list"`
}

// WeatherAlarmItem 单个预警项
type WeatherAlarmItem struct {
	AlertID   string `json:"alertid"`   // 预警 ID
	Title     string `json:"title"`     // 标题
	Pic       string `json:"pic"`       // 图片
	IssueTime string `json:"issuetime"` // 发布时间
	URL       string `json:"url"`       // 详情路径
}
