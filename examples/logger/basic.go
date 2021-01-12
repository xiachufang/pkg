package main

import (
	"github.com/xiachufang/pkg/v2/logger"
)

func main() {
	l, err := logger.New(&logger.Options{
		Tag:   "examples.logger",
		Debug: true,
	})

	if err != nil {
		panic(err)
	}

	l.Debug("debug message")
	l.Info("info message")
	l.Warn("info message")
	l.Error("info message")
}
