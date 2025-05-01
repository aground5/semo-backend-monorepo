package configs

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "@timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "log.level"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"

	var writer zapcore.WriteSyncer

	// stdout_only 옵션에 따라 출력 대상을 결정
	if Configs.Logs.StdoutOnly {
		writer = zapcore.AddSync(os.Stdout)
	} else {
		logFile := Configs.Logs.LogPath
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic("Failed to open log file: " + err.Error())
		}
		// 파일과 stdout으로 동시에 출력하려면 MultiWriteSyncer 사용
		writer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(file), zapcore.AddSync(os.Stdout))
	}

	var logLevel zapcore.Level
	if Configs.Logs.LogLevel == "DEBUG" {
		logLevel = zapcore.DebugLevel
	} else {
		logLevel = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // JSON 포맷
		writer,
		logLevel,
	)

	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	defer Logger.Sync()

	Logger.Info("Logger initialized!")
}
