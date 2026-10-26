/*
 * MIT License
 *
 * Copyright (c) 2026 nuoxi4n, NuoXianTech
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NuoXianTech/blindcrate/internal/logger"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type gormZapLogger struct {
	logLevel                  gormLogger.LogLevel
	ignoreRecordNotFoundError bool
	slowThreshold             time.Duration
}

func (l *gormZapLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	clone := *l
	clone.logLevel = level
	return &clone
}

func (l *gormZapLogger) Info(ctx context.Context, fmt string, args ...interface{}) {
	if l.logLevel >= gormLogger.Info {
		logger.InfoF(ctx, fmt, args...)
	}
}

func (l *gormZapLogger) Warn(ctx context.Context, fmt string, args ...interface{}) {
	if l.logLevel >= gormLogger.Warn {
		logger.WarnF(ctx, fmt, args...)
	}
}

func (l *gormZapLogger) Error(ctx context.Context, fmt string, args ...interface{}) {
	if l.logLevel >= gormLogger.Error {
		logger.ErrorF(ctx, fmt, args...)
	}
}

func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.logLevel >= gormLogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.ignoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			logger.ErrorF(ctx, "%s\n[%.3fms] [rows:%v] %s", err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.ErrorF(ctx, "%s\n[%.3fms] [rows:%v] %s", err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= gormLogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.slowThreshold)
		if rows == -1 {
			logger.WarnF(ctx, "%s\n[%.3fms] [rows:%v] %s", slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.WarnF(ctx, "%s\n[%.3fms] [rows:%v] %s", slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.logLevel == gormLogger.Info:
		sql, rows := fc()
		if rows == -1 {
			logger.InfoF(ctx, "[%.3fms] [rows:%v] %s", float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.InfoF(ctx, "[%.3fms] [rows:%v] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

func parseLogLevel(level string) gormLogger.LogLevel {
	level = strings.ToLower(level)
	switch level {
	case "silent":
		return gormLogger.Silent
	case "error":
		return gormLogger.Error
	case "warn":
		return gormLogger.Warn
	case "info":
		return gormLogger.Info
	default:
		return gormLogger.Info
	}
}
