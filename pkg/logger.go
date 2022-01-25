package main

import (
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y-cli-microservice/pkg/c8ycli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	PANIC
)

func (l LogLevel) String() string {
	return [...]string{"Debug", "Info", "Warn", "Error", "Panic"}[l]
}

func (l LogLevel) EnumIndex() int {
	return int(l)
}

var Logger *zap.Logger
var basePath string
var runtimeLogPath string
var errLogPath string

func init() {
	exists, _ := Exists("/var/log")
	if exists {
		basePath = "/var/log"
	} else {
		basePath = "./runtime/logs"
	}
	runtimeLogPath, errLogPath = basePath+"/runtime.log", basePath+"/runtime.err"
	Logger, _ = NewLogger()
	Logger.Info("Set Logfile path", zap.String("basePath", basePath), zap.String("runtimeLogPath", runtimeLogPath), zap.String("errorLogPath", errLogPath))
}

func NewLogger() (*zap.Logger, error) {
	os.MkdirAll(basePath, 0700)
	logFile, err := os.OpenFile(runtimeLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error while inializing runtime logfile: ", err)
		return nil, err
	}
	errFile, err := os.OpenFile(errLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error while inializing runtime error logfile: ", err)
		return nil, err
	}

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "file",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return true
		})),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(logFile), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return true
		})),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(errFile), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zap.ErrorLevel
		})),
	)

	logger := zap.New(core)
	return logger, nil
}

func Log(msg string, lvl LogLevel, job *MigrationJob, tenant Tenant, fields []zapcore.Field) {
	if len(fields) == 0 {
		fields = make([]zapcore.Field, 0)
	}
	fields = append(fields, zap.String("tenant", tenant.Url))
	fields = append(fields, zap.String("jobId", job.Id))

	if lvl == ERROR {
		Logger.Error(msg, fields...)
		job.logError(msg, fields)
		return
	} else if lvl == WARN {
		Logger.Warn(msg, fields...)
		job.logWarning(msg, fields)
		return

	}
	Logger.Info(msg, fields...)
}

func ExtractZapLogs(executor *c8ycli.Executor, result *c8ycli.ExecutorResult) []zapcore.Field {
	return []zapcore.Field{zap.String("ExitCode", fmt.Sprint(result.ExitCode)), zap.String("stdout", string(result.Stdout)), zap.String("Command", executor.Command)}
}

func FetchFileLogs() (regLog []byte, errLog []byte, e error) {
	regLog, e = os.ReadFile(runtimeLogPath)
	if e != nil {
		return []byte{}, []byte{}, e
	}
	errLog, e = os.ReadFile(errLogPath)
	if e != nil {
		return regLog, []byte{}, e
	}
	return regLog, errLog, e
}
