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

// JianshuHandler 简书处理器
type JianshuHandler struct {
	fetcher *service.Fetcher
}

// NewJianshuHandler 创建简书处理器
func NewJianshuHandler(fetcher *service.Fetcher) *JianshuHandler {
	return &JianshuHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *JianshuHandler) GetPath() string {
	return "/jianshu"
}

// Handle 处理请求
func (h *JianshuHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchJianshuHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"jianshu_hot",
		"简书",
		"热门推荐",
		"简书热门推荐列表",
		"https://www.jianshu.com",
		nil,
		data,
		!noCache,
	))
}

// fetchJianshuHot 从简书获取数据
func (h *JianshuHandler) fetchJianshuHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.jianshu.com/"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"Referer":    "https://www.jianshu.com",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("请求简书失败: %w", err)
	}

	// 解析HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析HTML
func (h *JianshuHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	// 先找到 ul.note-list
	ulPattern := regexp.MustCompile(`(?s)<ul[^>]+class="[^"]*note-list[^"]*"[^>]*>(.*?)</ul>`)
	ulMatches := ulPattern.FindStringSubmatch(html)

	if len(ulMatches) < 2 {
		return result
	}

	ulContent := ulMatches[1]

	// 匹配所有 li 项（不要求data-note-id属性）
	liPattern := regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)
	liMatches := liPattern.FindAllStringSubmatch(ulContent, -1)

	for _, liMatch := range liMatches {
		if len(liMatch) < 2 {
			continue
		}

		liHTML := liMatch[1]
		noteID := "" // 稍后从href或data-note-id中提取

		// 提取链接 href
		hrefPattern := regexp.MustCompile(`<a[^>]+href="(/p/[^"]+)"`)
		hrefMatches := hrefPattern.FindStringSubmatch(liHTML)
		href := ""
		if len(hrefMatches) > 1 {
			href = hrefMatches[1]
		}

		// 提取标题
		titlePattern := regexp.MustCompile(`<a[^>]+class="title"[^>]*>([^<]+)</a>`)
		titleMatches := titlePattern.FindStringSubmatch(liHTML)
		title := ""
		if len(titleMatches) > 1 {
			title = strings.TrimSpace(titleMatches[1])
		}

		// 提取封面
		coverPattern := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
		coverMatches := coverPattern.FindStringSubmatch(liHTML)
		cover := ""
		if len(coverMatches) > 1 {
			cover = coverMatches[1]
		}

		// 提取描述
		descPattern := regexp.MustCompile(`<p[^>]+class="abstract"[^>]*>([^<]+)</p>`)
		descMatches := descPattern.FindStringSubmatch(liHTML)
		desc := ""
		if len(descMatches) > 1 {
			desc = strings.TrimSpace(descMatches[1])
		}

		// 提取作者
		authorPattern := regexp.MustCompile(`<a[^>]+class="nickname"[^>]*>([^<]+)</a>`)
		authorMatches := authorPattern.FindStringSubmatch(liHTML)
		author := ""
		if len(authorMatches) > 1 {
			author = strings.TrimSpace(authorMatches[1])
		}

		// 从href中提取ID
		if href != "" {
			idPattern := regexp.MustCompile(`([^/]+)$`)
			idMatches := idPattern.FindStringSubmatch(href)
			if len(idMatches) > 1 {
				noteID = idMatches[1]
			}
		}

		// 如果还没有ID，尝试从data-note-id属性提取
		if noteID == "" {
			noteIDPattern := regexp.MustCompile(`data-note-id="([^"]+)"`)
			noteIDMatches := noteIDPattern.FindStringSubmatch(liHTML)
			if len(noteIDMatches) > 1 {
				noteID = noteIDMatches[1]
			}
		}

		// 如果没有标题或链接，跳过这个项目
		if title == "" || href == "" {
			continue
		}

		url := "https://www.jianshu.com" + href

		hotData := models.HotData{
			ID:        noteID,
			Title:     title,
			Cover:     cover,
			Desc:      desc,
			Author:    author,
			URL:       url,
			MobileURL: url,
		}

		result = append(result, hotData)
	}

	return result
}
