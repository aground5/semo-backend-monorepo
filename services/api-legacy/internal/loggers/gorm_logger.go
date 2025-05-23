package loggers

import (
	"context"
	"errors"
	"time"

	// "semo-server/configs-legacy" // 이 import는 더 이상 필요 없을 수 있습니다. 확인 필요.

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type ZapGormLogger struct {
	ZapLogger                 *zap.Logger // zap.Logger 필드 추가
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

// NewZapGormLogger 함수가 zap.Logger를 인자로 받도록 수정
func NewZapGormLogger(logger *zap.Logger, level gormlogger.LogLevel, slowThreshold time.Duration, ignoreRecordNotFoundError bool) *ZapGormLogger {
	return &ZapGormLogger{
		ZapLogger:                 logger, // 전달받은 로거 설정
		LogLevel:                  level,
		SlowThreshold:             slowThreshold,
		IgnoreRecordNotFoundError: ignoreRecordNotFoundError,
	}
}

func (z *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *z
	newLogger.LogLevel = level
	return &newLogger
}

func (z *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if z.LogLevel < gormlogger.Info {
		return
	}
	z.ZapLogger.Sugar().Infof(msg, data...) // 구조체의 ZapLogger 사용
}

func (z *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if z.LogLevel < gormlogger.Warn {
		return
	}
	z.ZapLogger.Sugar().Warnf(msg, data...) // 구조체의 ZapLogger 사용
}

func (z *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if z.LogLevel < gormlogger.Error {
		return
	}
	z.ZapLogger.Sugar().Errorf(msg, data...) // 구조체의 ZapLogger 사용
}

func (z *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// z.ZapLogger nil 체크 추가 (방어 코드)
	if z.ZapLogger == nil {
		// zap logger가 초기화되지 않은 경우 처리 (예: 표준 출력)
		// fmt.Printf("GORM Trace: ZapLogger not initialized. SQL: %s, Rows: %d, Elapsed: %v, Error: %v\n", sql, rows, elapsed, err)
		return
	}


	if deadline, ok := ctx.Deadline(); ok {
		z.ZapLogger.Debug("Context deadline", zap.Time("deadline", deadline)) // 구조체의 ZapLogger 사용
	}

	if err != nil && (!z.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		z.ZapLogger.Error("GORM Trace Error", // 구조체의 ZapLogger 사용
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Stack("stack"),
		)
		return
	}

	if z.SlowThreshold != 0 && elapsed > z.SlowThreshold {
		z.ZapLogger.Warn("GORM Slow Query", // 구조체의 ZapLogger 사용
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
		return
	}

	if z.LogLevel >= gormlogger.Info {
		z.ZapLogger.Info("GORM Query", // 구조체의 ZapLogger 사용
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
	}
}