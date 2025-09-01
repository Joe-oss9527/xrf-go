package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

type Logger struct {
	level  LogLevel
	prefix string
	file   *os.File
}

var defaultLogger = &Logger{
	level:  INFO,
	prefix: "XRF",
}

func SetLogLevel(level LogLevel) {
	defaultLogger.level = level
}

func SetLogFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defaultLogger.file = file
	log.SetOutput(file)
	return nil
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	levelStr := ""
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO"
	case WARNING:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	case FATAL:
		levelStr = "FATAL"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logMessage := fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, l.prefix, levelStr, message)

	if l.file != nil {
		fmt.Fprintln(l.file, logMessage)
	} else {
		if level >= ERROR {
			fmt.Fprintln(os.Stderr, logMessage)
		} else {
			fmt.Println(logMessage)
		}
	}

	if level == FATAL {
		os.Exit(1)
	}
}

func Debug(format string, args ...interface{}) {
	defaultLogger.log(DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.log(INFO, format, args...)
}

func Warning(format string, args ...interface{}) {
	defaultLogger.log(WARNING, format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.log(ERROR, format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.log(FATAL, format, args...)
}

func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [%s] [SUCCESS] %s\n", timestamp, defaultLogger.prefix, message)
}