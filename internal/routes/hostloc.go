package routes

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// HostlocHandler 全球主机交流处理器
type HostlocHandler struct {
	fetcher *service.Fetcher
}

// NewHostlocHandler 创建全球主机交流处理器
func NewHostlocHandler(fetcher *service.Fetcher) *HostlocHandler {
	return &HostlocHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HostlocHandler) GetPath() string {
	return "/hostloc"
}

// Handle 处理请求
func (h *HostlocHandler) Handle(c *fiber.Ctx) error {
	hostlocType := c.Query("type", "hot") // 默认最新热门
	noCache := c.Query("cache") == "false"

	data, err := h.fetchHostloc(c.Context(), hostlocType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("hostloc_%s", hostlocType),
		"全球主机交流",
		h.getTypeName(hostlocType),
		"全球主机交流热门帖子列表",
		"https://hostloc.com",
		nil,
		data,
		!noCache,
	))
}

// getTypeName 获取类型名称
func (h *HostlocHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"hot":       "最新热门",
		"digest":    "最新精华",
		"new":       "最新回复",
		"newthread": "最新发表",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "最新热门"
}

// fetchHostloc 从全球主机交流 RSS 获取数据
func (h *HostlocHandler) fetchHostloc(ctx context.Context, hostlocType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://r.jina.ai/https://hostloc.com/forum.php?mod=guide&view=%s", hostlocType)

	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":          "text/plain; charset=utf-8",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求全球主机交流页面失败: %w", err)
	}

	return h.parseHostlocMarkdown(string(body)), nil
}

var hostlocThreadPattern = regexp.MustCompile(`\[(?P<title>[^\]]+)\]\((https://hostloc\.com/thread-\d+-\d+-\d+\.html)\)`)

func (h *HostlocHandler) parseHostlocMarkdown(markdown string) []models.HotData {
	matches := hostlocThreadPattern.FindAllStringSubmatch(markdown, -1)
	seen := make(map[string]struct{})
	result := make([]models.HotData, 0, len(matches))

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		title := strings.TrimSpace(m[1])
		link := m[2]

		// 过滤分页或辅助链接
		if title == "" || len(title) <= 2 && strings.IndexFunc(title, func(r rune) bool {
			return r > '9' || r < '0'
		}) == -1 {
			continue
		}
		if strings.EqualFold(title, "new") || strings.HasPrefix(title, "阅读权限") {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}

		hotData := models.HotData{
			ID:        link,
			Title:     title,
			URL:       link,
			MobileURL: link,
		}
		result = append(result, hotData)

		if len(result) >= 30 {
			break
		}
	}

	return result
}
