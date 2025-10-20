package routes

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// DoubanHandler 豆瓣电影处理器
type DoubanHandler struct {
	fetcher *service.Fetcher
}

// NewDoubanHandler 创建豆瓣电影处理器
func NewDoubanHandler(fetcher *service.Fetcher) *DoubanHandler {
	return &DoubanHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *DoubanHandler) GetPath() string {
	return "/douban-movie"
}

// Handle 处理请求
func (h *DoubanHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchDoubanMovieHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"douban-movie",              // name: 平台调用名称
		"豆瓣电影",                      // title: 平台显示名称
		"新片榜",                       // type: 榜单类型
		"发现豆瓣电影热门作品",                // description: 平台描述
		"https://movie.douban.com/", // link: 官方链接
		nil,                         // params: 无参数映射
		data,                        // data: 热榜数据
		!noCache,                    // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchDoubanMovieHot 从豆瓣电影获取数据
func (h *DoubanHandler) fetchDoubanMovieHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://movie.douban.com/chart/"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1",
	})
	if err != nil {
		return nil, fmt.Errorf("请求豆瓣电影失败: %w", err)
	}

	// 解析 HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析 HTML
func (h *DoubanHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	// 匹配每个电影条目
	itemPattern := regexp.MustCompile(`(?s)<tr class="item">(.*?)</tr>`)
	items := itemPattern.FindAllStringSubmatch(html, -1)

	for _, item := range items {
		if len(item) < 2 {
			continue
		}

		itemHTML := item[1]

		// 提取链接和ID
		linkPattern := regexp.MustCompile(`<a href="([^"]+)"`)
		linkMatches := linkPattern.FindStringSubmatch(itemHTML)
		if len(linkMatches) < 2 {
			continue
		}
		url := linkMatches[1]
		id := h.extractID(url)

		// 提取标题
		titlePattern := regexp.MustCompile(`<a[^>]+title="([^"]+)"`)
		titleMatches := titlePattern.FindStringSubmatch(itemHTML)
		if len(titleMatches) < 2 {
			continue
		}
		title := titleMatches[1]

		// 提取评分
		scorePattern := regexp.MustCompile(`<span class="rating_nums">([^<]+)</span>`)
		scoreMatches := scorePattern.FindStringSubmatch(itemHTML)
		score := "0.0"
		if len(scoreMatches) > 1 {
			score = strings.TrimSpace(scoreMatches[1])
		}

		// 提取封面
		cover := ""
		imgPattern := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
		imgMatches := imgPattern.FindStringSubmatch(itemHTML)
		if len(imgMatches) > 1 {
			cover = imgMatches[1]
		}

		// 提取描述
		desc := ""
		descPattern := regexp.MustCompile(`<p class="pl">([^<]+)</p>`)
		descMatches := descPattern.FindStringSubmatch(itemHTML)
		if len(descMatches) > 1 {
			desc = strings.TrimSpace(descMatches[1])
		}

		// 提取评价人数(作为热度)
		hot := int64(0)
		hotPattern := regexp.MustCompile(`<span class="pl">(\d+)人评价</span>`)
		hotMatches := hotPattern.FindStringSubmatch(itemHTML)
		if len(hotMatches) > 1 {
			hot, _ = strconv.ParseInt(hotMatches[1], 10, 64)
		}

		hotData := models.HotData{
			ID:        id,
			Title:     fmt.Sprintf("【%s】%s", score, title),
			Cover:     cover,
			Desc:      desc,
			Hot:       hot,
			URL:       url,
			MobileURL: fmt.Sprintf("https://m.douban.com/movie/subject/%s/", id),
		}

		result = append(result, hotData)
	}

	return result
}

// extractID 从URL中提取ID
func (h *DoubanHandler) extractID(url string) string {
	pattern := regexp.MustCompile(`/subject/(\d+)/`)
	matches := pattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return "0"
}
