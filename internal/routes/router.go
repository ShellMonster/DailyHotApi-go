package routes

import (
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// Handler 路由处理器接口
// 所有平台的路由处理器都要实现这个接口
type Handler interface {
	// Handle 处理请求
	Handle(c *fiber.Ctx) error

	// GetPath 获取路由路径,如 "/bilibili"
	GetPath() string
}

// Registry 路由注册表
// 管理所有路由的注册
type Registry struct {
	fetcher  *service.Fetcher   // 数据获取服务
	handlers map[string]Handler // 路由处理器映射表: path -> handler
}

// NewRegistry 创建路由注册表
func NewRegistry(fetcher *service.Fetcher) *Registry {
	return &Registry{
		fetcher:  fetcher,
		handlers: make(map[string]Handler),
	}
}

// Register 注册路由处理器
// 将一个平台的处理器注册到系统中
func (r *Registry) Register(handler Handler) {
	path := handler.GetPath()
	r.handlers[path] = handler
}

// RegisterRoutes 将所有路由注册到 Fiber 应用
// 这个方法会在服务启动时调用
func (r *Registry) RegisterRoutes(app *fiber.App) {
	// 注册所有平台路由
	for path, handler := range r.handlers {
		app.Get(path, handler.Handle)
	}

	// 注册根路径,返回 API 信息
	app.Get("/", r.handleIndex)

	// 注册健康检查接口
	app.Get("/health", r.handleHealth)

	// 注册缓存统计接口
	app.Get("/stats", r.handleStats)

	// 注册所有路由列表接口
	app.Get("/all", r.handleAll)
}

// handleIndex 首页处理器
// 返回 API 的基本信息和可用路由列表
func (r *Registry) handleIndex(c *fiber.Ctx) error {
	// 获取所有可用路由
	routes := make([]string, 0, len(r.handlers))
	for path := range r.handlers {
		routes = append(routes, path)
	}

	c.Set("Content-Type", fiber.MIMEApplicationJSONCharsetUTF8)

	return c.JSON(fiber.Map{
		"code":    200,
		"message": "DailyHotApi - Go 版本",
		"version": "1.0.0",
		"routes":  routes,
		"docs":    "https://github.com/ShellMonster/DailyHotApi-go",
	})
}

// handleHealth 健康检查处理器
// 用于监控系统判断服务是否正常运行
func (r *Registry) handleHealth(c *fiber.Ctx) error {
	c.Set("Content-Type", fiber.MIMEApplicationJSONCharsetUTF8)
	return c.JSON(fiber.Map{
		"status": "healthy",
		"cache":  "ok",
	})
}

// handleStats 缓存统计处理器
// 返回缓存系统的性能统计数据
func (r *Registry) handleStats(c *fiber.Ctx) error {
	stats := r.fetcher.GetCacheStats()
	c.Set("Content-Type", fiber.MIMEApplicationJSONCharsetUTF8)
	return c.JSON(fiber.Map{
		"code":  200,
		"stats": stats,
	})
}

// handleAll 返回所有已注册路由的列表
// 这个接口返回系统中所有可用的 API 端点信息
// 返回格式: { code: 200, count: <数量>, routes: [ { name: "...", path: "..." }, ... ] }
func (r *Registry) handleAll(c *fiber.Ctx) error {
	// 收集所有已注册的路由信息
	routes := make([]fiber.Map, 0, len(r.handlers))

	// 遍历所有已注册的处理器,生成路由信息
	for _, handler := range r.handlers {
		routeInfo := fiber.Map{
			"name": handler.GetPath()[1:], // 移除路径前的 "/" 符号作为名称,例如 "/bilibili" -> "bilibili"
			"path": handler.GetPath(),     // 完整的路径,例如 "/bilibili"
		}
		routes = append(routes, routeInfo)
	}

	// 返回路由列表信息
	c.Set("Content-Type", fiber.MIMEApplicationJSONCharsetUTF8)
	return c.JSON(fiber.Map{
		"code":   200,
		"count":  len(r.handlers),
		"routes": routes,
	})
}

// GetFetcher 获取数据获取服务
// 供路由处理器使用
func (r *Registry) GetFetcher() *service.Fetcher {
	return r.fetcher
}

// GetHandlers 获取所有已注册的路由处理器
// 用于缓存预热等场景
func (r *Registry) GetHandlers() map[string]Handler {
	return r.handlers
}
