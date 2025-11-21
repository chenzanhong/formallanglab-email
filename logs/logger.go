package email

// var EmailSugar *zap.SugaredLogger

// func InitZapSugarDefault() {
// 	_, filename, _, ok := runtime.Caller(0) // 获取调用者的文件名
// 	if !ok {
// 		log.Fatal("无法获取运行时调用者信息")
// 	}

// 	// 获取当前文件所在的目录
// 	currentDir := filepath.Dir(filename)

// 	logPath := filepath.Join(currentDir, "email.log")
// 	_, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
// 	if err != nil {
// 		log.Fatalf("创建/打开日志文件失败：%v", err.Error())
// 	}

// 	// 配置日志文件
// 	fileLogger := &lumberjack.Logger{
// 		Filename:   logPath,
// 		MaxSize:    20,
// 		MaxBackups: 5,
// 		MaxAge:     360,
// 		Compress:   false,
// 	}

// 	// 编码器配置
// 	encoderConfig := zap.NewProductionEncoderConfig()
// 	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
// 	encoder := zapcore.NewJSONEncoder(encoderConfig)

// 	// 文件输出核心
// 	fileCore := zapcore.NewCore(encoder, zapcore.AddSync(fileLogger), zapcore.InfoLevel)

// 	// 控制台输出核心
// 	consoleCore := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), zapcore.InfoLevel)

// 	// 合并核心
// 	core := zapcore.NewTee(fileCore, consoleCore)

// 	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

// 	EmailSugar = logger.Sugar()
// }
