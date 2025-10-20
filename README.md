# DailyHotApi - Go 版本

DailyHotApi 的 Go 语言高性能重构版本,提供多平台热榜/资讯聚合 API 服务(当前已接入 **60+** 渠道)。

> 🙏 **致谢**  
> 本项目是在 [imsyy/DailyHotApi](https://github.com/imsyy/DailyHotApi) 基础上的 Go 语言重构版本。原项目由 @imsyy 使用 TypeScript 精心打造，功能完善、更新及时。  
> 如果本项目对你有帮助，强烈推荐前往原仓库为作者点个 Star、提交 Issue 或 PR 支持原项目的持续发展。  
> 同时也欢迎大家关注原项目的部署文档、工具生态，以便获得更完整的使用体验。

## ✨ 特性

- 🚀 **极致性能**: 基于 Go + Fiber 框架,QPS 可达 100,000+
- 💾 **智能缓存**: BigCache + Redis 双层缓存,L1 零 GC 设计
- 🔒 **类型安全**: 完整的类型定义,编译期错误检查
- 🌏 **覆盖广泛**: 覆盖国内外资讯/社区/科技等 60+ 热点源
- 📦 **容器化**: Docker 一键部署,镜像仅 20MB
- 🎯 **标准化**: 统一的 API 响应格式,易于集成
- 📊 **可观测**: 结构化日志 + 性能监控接口

## 📊 性能与部署概览

- 基于 Go + Fiber，配合 BigCache/Redis 双层缓存，在 8C16G 服务器上可稳定提供 **~15k QPS**(缓存命中率 >95%)。
- 冷启动场景下通过异步预热热门路由，平均响应保持在 **<20ms**；非缓存请求通常 **<200ms**，具体取决于上游源站。
- 服务常驻内存约 **35MB**(启用缓存)；不启用 Redis 时仍可通过 BigCache 提供快速 L1 缓存。
- 官方示例部署地址: **https://apinews.geekaso.com/** (仅供演示，请勿滥用)。

## 🏗️ 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| Web 框架 | Fiber v2 | Express 风格,性能极致 |
| HTTP 客户端 | Resty | 链式调用,自动重试 |
| 内存缓存 | BigCache | 零 GC,亿级数据 |
| 分布式缓存 | Redis | 可选,支持集群 |
| JSON 解析 | encoding/json | 标准库,稳定可靠 |
| RSS/Atom 解析 | gofeed | 统一解析多种订阅格式 |
| 日志系统 | Zap + Lumberjack | 结构化日志 & 按天轮转 |
| 配置管理 | Viper | 支持多格式&环境变量 |
| 缓存 | BigCache + go-redis/v9 | 内存+分布式双层缓存 |

## 📁 项目结构

```
DailyHotApi-Go/
├── cmd/                    # 程序入口
│   └── api/
│       └── main.go         # 主程序
├── internal/               # 内部模块(不对外暴露)
│   ├── cache/              # 缓存系统
│   ├── config/             # 配置管理
│   ├── http/               # HTTP 客户端
│   ├── logger/             # 日志系统
│   ├── models/             # 数据模型
│   ├── routes/             # 路由处理器
│   │   ├── router.go       # 路由注册表 & 公共接口
│   │   ├── *.go            # 各平台路由处理器
│   └── service/            # 业务服务层
├── pkg/                    # 公共工具库
│   └── utils/              # 平台工具(WBI、时间解析等)
├── config.yaml             # 默认配置
├── docker-compose.yml      # 一键启动(可选 redis)
├── docker-compose.redis.yml# 启动 redis 示例
├── Dockerfile              # 构建镜像
├── logs/                   # 默认日志目录
├── README.md               # 英文/中文主文档
├── 快速上手.md              # Docker 推送与部署指南
└── 镜像发布指南.md          # 镜像发布说明
```

## 🚀 快速开始

### 方式一:Docker Compose(推荐)

```bash
# 1. 克隆项目
git clone <repository-url>
cd DailyHotApi-Go

# 2. 一键启动(包含 Redis)
docker-compose up -d

# 3. 查看日志
docker-compose logs -f api

# 4. 验证示例
curl https://apinews.geekaso.com/bilibili
```

### 方式二:本地运行

```bash
# 1. 安装依赖
go mod download

# 2. 运行程序
go run cmd/api/main.go

# 3. 验证示例
curl https://apinews.geekaso.com/bilibili
```

### 方式三:编译部署

```bash
# 1. 编译
go build -o dailyhot-api cmd/api/main.go

# 2. 运行
./dailyhot-api
```

## ⚙️ 配置说明

编辑 `config.yaml`:

```yaml
server:
  port: 6688              # 服务端口
  host: "0.0.0.0"         # 监听地址
  prefork: false          # 多进程模式

cache:
  enabled: true           # 启用内存缓存
  default_expire: 5m      # 缓存时间
  max_entries: 10000      # 最大条目数

redis:
  enabled: false          # 启用 Redis(可选)
  host: "localhost"
  port: 6379

log:
  level: "info"           # 日志级别
  format: "console"       # 输出格式
```

也可以通过**环境变量**覆盖配置:

```bash
export DAILYHOT_SERVER_PORT=8080
export DAILYHOT_REDIS_ENABLED=true
export DAILYHOT_REDIS_HOST=redis
```

## 📡 API 接口

### 基础信息

```bash
GET /
```

返回 API 版本和可用路由列表。

### 健康检查

```bash
GET /health
```

返回服务健康状态,用于监控系统。

### 缓存统计

```bash
GET /stats
```

返回缓存性能统计数据。

### 已实现的平台接口

下方仅列出常用/新增平台,完整列表可访问 `/all` 查看。

#### 热榜 / 社交
- `/weibo` 微博热搜
- `/zhihu` 知乎热榜
- `/douyin` 抖音热点
- `/bilibili` B站热榜
- `/baidu?type=realtime` 百度热搜(支持 realtime/novel/movie/teleplay/car/game)
- `/github?type=daily` GitHub Trending(daily/weekly/monthly)
- `/juejin?type=1` 掘金热门(分类 ID)
- `/v2ex?type=hot` V2EX(最热/最新)
- `/52pojie` 吾爱破解(默认精华,无数据时自动回退热门,响应 `params.actualType` 标记实际来源)

#### 科技 / 创业媒体
- `/ithome` IT之家
- `/36kr?type=hot` 36氪(人气/视频/热议/收藏)
- `/sspai?type=热门文章` 少数派(多标签)
- `/ifanr` 爱范儿
- `/geekpark` 极客公园
- `/huxiu` 虎嗅
- `/techcrunch` TechCrunch
- `/theverge` The Verge
- `/engadget` Engadget
- `/economist` The Economist 最新

#### 新闻资讯
- `/toutiao` 今日头条
- `/netease` 网易新闻
- `/sinanews` 新浪新闻
- `/thepaper` 澎湃新闻
- `/qqnews` 腾讯新闻
- `/theguardian` The Guardian World News
- `/nytimes?type=china` 纽约时报(中文/全球)

#### 其他垂类示例
- `/smzdm` 什么值得买
- `/coolapk` 酷安热榜
- `/weread` 微信读书
- `/miyoushe` 米游社
- `/yystv` 游研社
- `/earthquake` 中国地震台
- `/weatheralarm` 中央气象台
- `/history` 历史上的今天

> 小贴士: 大多数接口都支持 `cache=false` 参数强制刷新源数据(默认启用缓存)。

### 响应格式

所有接口返回统一格式:

```json
{
  "code": 200,
  "message": "success",
  "name": "平台名称",
  "title": "平台显示名称",
  "type": "榜单类型",
  "updateTime": "2024-01-01T12:00:00Z",
  "total": 20,
  "fromCache": true,
  "data": [
    {
      "id": "唯一标识",
      "title": "标题",
      "desc": "描述",
      "cover": "封面图",
      "url": "详情链接",
      "hot": 热度值,
      "author": "作者",
      "timestamp": "时间戳"
    }
  ]
}
```

额外扩展字段:
- `params.actualType`: 对部分存在自动降级的来源(如 `/52pojie`)标记当前真实使用的榜单类型。

## 🔧 性能优化

本项目采用了多项性能优化技术:

### 1. 对象池化 (sync.Pool)

复用 HTTP 请求对象,减少内存分配:

```go
var httpRequestPool = sync.Pool{
    New: func() interface{} {
        return &HTTPRequest{}
    },
}
```

### 2. 协程池 (ants)

限制 Goroutine 数量,防止资源耗尽:

```go
pool, _ := ants.NewPool(10000)
```

### 3. 零拷贝缓存 (BigCache)

L1 缓存采用 BigCache,零 GC 设计,支持亿级数据:

```go
cache, _ := bigcache.New(context.Background(), config)
```

### 4. 多级缓存架构

```
请求 → L1(BigCache) → L2(Redis) → 原始 API
         ↓ 命中          ↓ 命中        ↓ 请求
       <10μs          <1ms         ~100ms
```

## 🐛 添加新平台

1. **创建路由处理器**

在 `internal/routes/` 下创建新文件,如 `weibo.go`:

```go
package routes

type WeiboHandler struct {
    fetcher *service.Fetcher
}

func NewWeiboHandler(fetcher *service.Fetcher) *WeiboHandler {
    return &WeiboHandler{fetcher: fetcher}
}

func (h *WeiboHandler) GetPath() string {
    return "/weibo"
}

func (h *WeiboHandler) Handle(c *fiber.Ctx) error {
    resp, err := h.fetcher.GetData(
        c.Context(),
        "weibo_hot",
        "微博",
        "热搜榜",
        5*time.Minute,
        h.fetchWeiboHot,
    )

    if err != nil {
        return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
    }

    return c.JSON(resp)
}

func (h *WeiboHandler) fetchWeiboHot(ctx context.Context) ([]models.HotData, error) {
    // 实现数据获取逻辑
    // ...
}
```

2. **注册路由**

在 `cmd/api/main.go` 中添加:

```go
registry.Register(routes.NewWeiboHandler(fetcher))
```

## 📝 开发进度

### 基础架构 ✅
- [x] 配置管理(Viper)
- [x] 日志系统(Zap)
- [x] 双层缓存(BigCache + Redis)
- [x] HTTP 客户端(Resty)
- [x] 路由管理系统

### 已实现平台概览

当前已接入 **61** 个平台,覆盖视频、社交、资讯、科技、游戏、电商等多个垂类。常用分类如下:

- **视频**: B站、抖音、快手、AcFun、米哈游系游戏社区等
- **社交/社区**: 微博、知乎、V2EX、吾爱破解、NGA、NodeSeek、Hostloc、Linux.do…
- **资讯**: 今日头条、澎湃、腾讯新闻、The Guardian、纽约时报、The Economist…
- **生活/电商**: 什么值得买、微信读书、数字尾巴、汽车之家等
- **工具/特殊**: 历史上的今天、中央气象台预警、中国地震台、IT之家喜加一等
- **IT 资讯/科技媒体**: IT之家、36氪、少数派、爱范儿、极客公园、虎嗅、TechCrunch、The Verge、Engadget、The Economist 等

完整列表可通过 `/all` 接口或 `cmd/api/main.go` 注册信息查看。

## 🤝 贡献指南

欢迎提交 Pull Request 添加更多平台支持!

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/new-platform`)
3. 提交更改 (`git commit -m 'Add: xxx平台支持'`)
4. 推送到分支 (`git push origin feature/new-platform`)
5. 提交 Pull Request

## 📄 许可证

MIT License

## 🙏 致谢

本项目基于 [DailyHotApi](https://github.com/imsyy/DailyHotApi) 使用 Go 语言重构。

感谢原作者的创意和开源精神!
