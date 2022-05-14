package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = &Logger{}

type Logger struct {
}

func readLoggerProperties() (string, string, string, string, string, string) {
	viper.SetConfigName("logger")
	viper.SetConfigType("properties")
	viper.AddConfigPath("./")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	logFilename := cast.ToString(viper.Get("logFilename"))
	maxSize := cast.ToString(viper.Get("maxSize"))
	maxBackups := cast.ToString(viper.Get("maxBackups"))
	maxAge := cast.ToString(viper.Get("maxAge"))
	compressFlag := cast.ToString(viper.Get("compress"))
	level := cast.ToString(viper.Get("level"))

	return logFilename, maxSize, maxBackups, maxAge, compressFlag, level
}

func (l Logger) Init() {

	logFilename, maxSize, maxBackups, maxAge, compressFlag, level := readLoggerProperties()

	loggerConfig := &lumberjack.Logger{
		Filename:   logFilename,
		MaxSize:    cast.ToInt(maxSize),
		MaxBackups: cast.ToInt(maxBackups),
		MaxAge:     cast.ToInt(maxAge),
		Compress:   cast.ToBool(compressFlag),
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(loggerConfig)

	switch cast.ToString(level) {

	case "Trace":
		logrus.SetLevel(logrus.TraceLevel)

	case "Info":
		logrus.SetLevel(logrus.InfoLevel)

	case "Warn":
		logrus.SetLevel(logrus.WarnLevel)

	case "Error":
		logrus.SetLevel(logrus.ErrorLevel)

	case "Fatal":
		logrus.SetLevel(logrus.FatalLevel)

	default:
		logrus.SetLevel(logrus.DebugLevel)
	}

}

func (l Logger) Info(message string) {
	logrus.Info(message)
	fmt.Println("Info:", message)
}

func (l Logger) Error(message string) {
	logrus.Error(message)
	fmt.Println("Error:", message)
}

func (l Logger) Debug(message string) {
	logrus.Debug(message)
	fmt.Println("Debug:", message)
}

func (l Logger) Warn(message string) {
	logrus.Warn(message)
	fmt.Println("Warn:", message)
}

func (l Logger) Fatal(message string) {
	logrus.Fatal(message)
	fmt.Println("Fatal:", message)
}
