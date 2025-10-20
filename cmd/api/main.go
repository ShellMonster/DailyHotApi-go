package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dailyhot/api/internal/cache"
	"github.com/dailyhot/api/internal/config"
	"github.com/dailyhot/api/internal/logger"
	"github.com/dailyhot/api/internal/routes"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志系统
	_, err = logger.Init(cfg)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() // 程序退出前刷新日志缓冲区

	logger.Info("应用启动中...",
		zap.String("version", "1.0.0"),
		zap.Int("port", cfg.Server.Port),
	)

	// 3. 初始化缓存系统
	cacheManager, err := cache.NewManager(cfg)
	if err != nil {
		logger.Fatal("初始化缓存失败", zap.Error(err))
	}
	defer cacheManager.Close() // 程序退出前关闭缓存

	// 4. 创建数据获取服务
	fetcher := service.NewFetcher(cacheManager)

	// 5. 创建路由注册表
	registry := routes.NewRegistry(fetcher)

	// 6. 注册各平台路由处理器
	logger.Info("注册路由处理器...")

	// 视频平台
	registry.Register(routes.NewBilibiliHandler(fetcher)) // B站
	registry.Register(routes.NewDouyinHandler(fetcher))   // 抖音
	registry.Register(routes.NewKuaishouHandler(fetcher)) // 快手

	// 社交平台
	registry.Register(routes.NewWeiboHandler(fetcher)) // 微博
	registry.Register(routes.NewZhihuHandler(fetcher)) // 知乎

	// 搜索引擎
	registry.Register(routes.NewBaiduHandler(fetcher)) // 百度

	// 开发者社区
	registry.Register(routes.NewGitHubHandler(fetcher)) // GitHub
	registry.Register(routes.NewJuejinHandler(fetcher)) // 掘金
	registry.Register(routes.NewV2exHandler(fetcher))   // V2EX

	// IT资讯/科技媒体
	registry.Register(routes.NewIthomeHandler(fetcher))     // IT之家
	registry.Register(routes.NewKr36Handler(fetcher))       // 36氪
	registry.Register(routes.NewSspaiHandler(fetcher))      // 少数派
	registry.Register(routes.NewCTO51Handler(fetcher))      // 51CTO
	registry.Register(routes.NewTechCrunchHandler(fetcher)) // TechCrunch
	registry.Register(routes.NewTheVergeHandler(fetcher))   // The Verge
	registry.Register(routes.NewEngadgetHandler(fetcher))   // Engadget

	// 新闻资讯
	registry.Register(routes.NewToutiaoHandler(fetcher))   // 今日头条
	registry.Register(routes.NewNeteaseHandler(fetcher))   // 网易新闻
	registry.Register(routes.NewGuardianHandler(fetcher))  // The Guardian
	registry.Register(routes.NewEconomistHandler(fetcher)) // The Economist

	// 电影/娱乐
	registry.Register(routes.NewDoubanHandler(fetcher)) // 豆瓣电影

	// 数码社区
	registry.Register(routes.NewCoolapkHandler(fetcher)) // 酷安

	// 体育社区
	registry.Register(routes.NewHupuHandler(fetcher)) // 虎扑

	// 开发者社区(续)
	registry.Register(routes.NewCSDNHandler(fetcher))        // CSDN
	registry.Register(routes.NewHelloGitHubHandler(fetcher)) // HelloGitHub
	registry.Register(routes.NewHackerNewsHandler(fetcher))  // Hacker News
	registry.Register(routes.NewGuokrHandler(fetcher))       // 果壳
	registry.Register(routes.NewProductHuntHandler(fetcher)) // Product Hunt

	// 新闻资讯(续)
	registry.Register(routes.NewSinaNewsHandler(fetcher)) // 新浪新闻
	registry.Register(routes.NewThePaperHandler(fetcher)) // 澎湃新闻
	registry.Register(routes.NewQQNewsHandler(fetcher))   // 腾讯新闻
	registry.Register(routes.NewSinaHandler(fetcher))     // 新浪网
	registry.Register(routes.NewNYTimesHandler(fetcher))  // 纽约时报

	// 视频平台(续)
	registry.Register(routes.NewAcfunHandler(fetcher)) // AcFun

	// 社交社区(续)
	registry.Register(routes.NewZhihuDailyHandler(fetcher))  // 知乎日报
	registry.Register(routes.NewTiebaHandler(fetcher))       // 百度贴吧
	registry.Register(routes.NewDoubanGroupHandler(fetcher)) // 豆瓣讨论
	registry.Register(routes.NewNgabbsHandler(fetcher))      // NGA
	registry.Register(routes.NewNewsmthHandler(fetcher))     // 水木社区
	registry.Register(routes.NewLinuxdoHandler(fetcher))     // Linux.do
	registry.Register(routes.NewHostlocHandler(fetcher))     // 全球主机交流
	registry.Register(routes.NewPojieHandler(fetcher))       // 吾爱破解
	registry.Register(routes.NewNodeseekHandler(fetcher))    // NodeSeek
	registry.Register(routes.NewJianshuHandler(fetcher))     // 简书

	// 游戏相关
	registry.Register(routes.NewMiyousheHandler(fetcher)) // 米游社
	registry.Register(routes.NewGenshinHandler(fetcher))  // 原神
	registry.Register(routes.NewHonkaiHandler(fetcher))   // 崩坏3
	registry.Register(routes.NewStarrailHandler(fetcher)) // 星穹铁道
	registry.Register(routes.NewLolHandler(fetcher))      // 英雄联盟
	registry.Register(routes.NewGameresHandler(fetcher))  // GameRes
	registry.Register(routes.NewYystvHandler(fetcher))    // 游研社

	// 生活服务
	registry.Register(routes.NewSmzdmHandler(fetcher))  // 什么值得买
	registry.Register(routes.NewWereadHandler(fetcher)) // 微信读书
	registry.Register(routes.NewDgtleHandler(fetcher))  // 数字尾巴
	registry.Register(routes.NewIfanrHandler(fetcher))  // 爱范儿

	// 科技媒体(续)
	registry.Register(routes.NewGeekParkHandler(fetcher)) // 极客公园
	registry.Register(routes.NewHuxiuHandler(fetcher))    // 虎嗅

	// 特殊功能
	registry.Register(routes.NewHistoryHandler(fetcher))       // 历史上的今天
	registry.Register(routes.NewEarthquakeHandler(fetcher))    // 中国地震台
	registry.Register(routes.NewWeatherAlarmHandler(fetcher))  // 中央气象台
	registry.Register(routes.NewIthomeXijiayiHandler(fetcher)) // IT之家喜加一

	logger.Info("路由注册完成", zap.Int("total", 61))

	// 6.5. 启动缓存预热(后台协程,不阻塞启动)
	go warmUpCacheAsync(registry)

	// 7. 创建 Fiber 应用
	app := fiber.New(fiber.Config{
		// 应用名称
		AppName: "DailyHotApi v1.0.0",

		// 禁用启动横幅(可选)
		DisableStartupMessage: false,

		// Prefork 模式(多进程,生产环境推荐)
		Prefork: cfg.Server.Prefork,

		// 请求体大小限制
		BodyLimit: 4 * 1024 * 1024, // 4MB

		// 读写超时
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,

		// 错误处理器
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			logger.Error("请求错误",
				zap.String("path", c.Path()),
				zap.Int("code", code),
				zap.Error(err),
			)

			return c.Status(code).JSON(fiber.Map{
				"code":    code,
				"message": err.Error(),
			})
		},
	})

	// 8. 注册中间件
	// CORS 跨域支持
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Gzip 压缩
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // 平衡压缩率和速度
	}))

	// Panic 恢复中间件
	app.Use(recover.New())

	// 请求日志中间件
	app.Use(func(c *fiber.Ctx) error {
		start := c.Context().Time()

		// 执行下一个中间件/处理器
		err := c.Next()

		// 记录请求日志
		logger.Info("请求",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", c.Context().Time().Sub(start)),
			zap.String("ip", c.IP()),
		)

		return err
	})

	// 9. 注册所有路由
	registry.RegisterRoutes(app)

	// 10. 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("服务器启动成功",
		zap.String("addr", addr),
		zap.Bool("prefork", cfg.Server.Prefork),
	)

	// 11. 优雅关闭处理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("收到关闭信号,正在优雅关闭服务器...")

		// 关闭 Fiber 服务器
		if err := app.Shutdown(); err != nil {
			logger.Error("服务器关闭失败", zap.Error(err))
		}

		logger.Info("服务器已关闭")
		os.Exit(0)
	}()

	// 启动 HTTP 服务
	if err := app.Listen(addr); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}

// warmUpCacheAsync 异步缓存预热函数
// 在后台协程中通过 HTTP 请求预热热门平台的缓存数据
// 目的: 冷启动时提前加载热门平台数据到缓存,提升首次请求响应速度
func warmUpCacheAsync(registry *routes.Registry) {
	// 定义需要预热的热门平台列表
	// 优先级: 高热度平台优先加载
	hotPlatforms := []string{
		"/weibo",      // 微博热搜
		"/toutiao",    // 今日头条
		"/baidu",      // 百度热搜
		"/bilibili",   // B站热榜
		"/douyin",     // 抖音热点
		"/github",     // GitHub趋势
		"/csdn",       // CSDN热门
		"/v2ex",       // V2EX最热
		"/hackernews", // Hacker News
		"/zhihu",      // 知乎热榜
	}

	// 延迟启动预热,等待 HTTP 服务完全启动
	// 并给予配置加载充足时间
	time.Sleep(1 * time.Second)

	logger.Info("开始缓存预热...", zap.Int("platforms", len(hotPlatforms)))
	startTime := time.Now()

	// 使用 WaitGroup 控制并发
	// 最多同时 3 个协程进行预热,避免启动时资源占用过高
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // 最多 3 个并发

	// 对每个热门平台执行预热
	for _, path := range hotPlatforms {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			// 获取信号量(控制并发)
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 设置请求超时
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// 发起 HTTP 请求进行预热
			// 使用 HTTP 客户端而不是直接调用处理器,避免上下文问题
			httpClient := registry.GetFetcher().GetHTTPClient()
			if httpClient == nil {
				logger.Warn("HTTP 客户端未初始化",
					zap.String("platform", p),
				)
				return
			}

			// 构造本地访问 URL
			url := fmt.Sprintf("http://127.0.0.1:6688%s", p)

			// 执行预热请求
			_, err := httpClient.Get(url, map[string]string{
				"User-Agent": "DailyHotApi/CacheWarmer",
			})

			if err != nil {
				logger.Warn("缓存预热失败",
					zap.String("platform", p),
					zap.Error(err),
				)
				return
			}

			logger.Info("缓存预热成功",
				zap.String("platform", p),
				zap.Duration("elapsed", time.Since(startTime)),
			)

			// 释放上下文
			_ = ctx
		}(path)
	}

	// 等待所有预热协程完成
	wg.Wait()

	logger.Info("缓存预热完成",
		zap.Duration("total_time", time.Since(startTime)),
		zap.Int("warmed_platforms", len(hotPlatforms)),
	)
}
