package logger

import (
	"os"

	"github.com/dailyhot/api/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Init 初始化日志系统
// cfg: 配置对象,包含日志级别、输出路径等参数
// 返回: 初始化好的 logger 实例
func Init(cfg *config.Config) (*zap.Logger, error) {
	// 1. 确定日志级别
	// 就像调节音量,不同级别会输出不同详细程度的日志
	// debug(最详细) > info(常规) > warn(警告) > error(错误)
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 2. 配置编码器(决定日志的输出格式)
	var encoderConfig zapcore.EncoderConfig
	if cfg.Log.Format == "json" {
		// JSON 格式:机器友好,便于日志分析工具处理
		// 输出像: {"level":"info","ts":1234567890,"msg":"服务启动"}
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		// Console 格式:人类友好,便于直接阅读
		// 输出像: 2024-01-01 12:00:00 INFO 服务启动
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 彩色输出
	}

	// 时间格式设置为易读的格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 3. 创建编码器
	var encoder zapcore.Encoder
	if cfg.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 4. 配置输出目标
	// 同时输出到控制台和文件,方便开发和生产环境
	writers := []zapcore.WriteSyncer{
		zapcore.AddSync(os.Stdout), // 控制台输出
	}

	// 如果配置了日志文件路径,则同时写入文件
	if cfg.Log.OutputPath != "" {
		// 使用 lumberjack 实现日志文件自动轮转
		// 就像一个"自动整理的笔记本",写满一页自动翻页
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Log.OutputPath, // 日志文件路径
			MaxSize:    cfg.Log.MaxSize,    // 单个文件最大大小(MB)
			MaxBackups: cfg.Log.MaxBackups, // 保留的旧文件数量
			MaxAge:     cfg.Log.MaxAge,     // 保留天数
			Compress:   cfg.Log.Compress,   // 是否压缩旧文件
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	// 5. 创建 Core(日志系统的核心)
	core := zapcore.NewCore(
		encoder,                                 // 编码器:决定格式
		zapcore.NewMultiWriteSyncer(writers...), // 输出目标:控制台+文件
		level,                                   // 日志级别:过滤器
	)

	// 6. 创建 Logger 实例
	logger := zap.New(
		core,
		zap.AddCaller(),                       // 添加调用者信息(文件名和行号)
		zap.AddCallerSkip(1),                  // 跳过一层调用栈,显示真正的调用位置
		zap.AddStacktrace(zapcore.ErrorLevel), // Error 级别自动添加堆栈跟踪
	)

	globalLogger = logger
	return logger, nil
}

// Get 获取全局 logger 实例
// 在其他模块中可以通过这个函数获取 logger
func Get() *zap.Logger {
	if globalLogger == nil {
		// 如果还没初始化,返回一个默认的 logger
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Sync 刷新日志缓冲区
// 在程序退出前调用,确保所有日志都写入磁盘
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// 下面是一些便捷方法,可以直接使用而不需要先获取 logger

// Debug 输出调试级别日志
// 用于开发时的详细信息,生产环境通常不输出
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 输出信息级别日志
// 用于记录正常的业务流程,比如"服务启动"、"处理请求"等
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 输出警告级别日志
// 用于记录可能的问题,但不影响运行,比如"缓存未命中"
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 输出错误级别日志
// 用于记录错误情况,比如"请求失败"、"数据解析错误"等
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 输出致命错误日志并退出程序
// 用于无法恢复的严重错误,调用后程序会立即退出
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Infof 格式化输出信息日志(类似 fmt.Printf)
// 更方便的日志输出方式,比如: Infof("用户 %s 登录成功", username)
func Infof(format string, args ...interface{}) {
	Get().Sugar().Infof(format, args...)
}

// Warnf 格式化输出警告日志
func Warnf(format string, args ...interface{}) {
	Get().Sugar().Warnf(format, args...)
}

// Errorf 格式化输出错误日志
func Errorf(format string, args ...interface{}) {
	Get().Sugar().Errorf(format, args...)
}

// Debugf 格式化输出调试日志
func Debugf(format string, args ...interface{}) {
	Get().Sugar().Debugf(format, args...)
}

// With 创建带上下文字段的子 logger
// 用于在一系列日志中添加公共字段,比如请求ID、用户ID等
// 例如: reqLogger := logger.With(zap.String("request_id", "abc123"))
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}
