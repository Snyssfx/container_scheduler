package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/Snyssfx/container_scheduler/internal/api"
	"github.com/Snyssfx/container_scheduler/internal/containersmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var serverPort = flag.Int("port", 9002, "a port that a server should listen for user requests")

func main() {
	// context with graceful shutdown
	ctx, stopFn := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer stopFn()

	flag.Parse()

	log := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "message",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}),
			zapcore.AddSync(os.Stdout),
			zap.NewAtomicLevelAt(zap.DebugLevel),
		),
	).Sugar()

	cm := containersmap.New(log.Named("cm"))
	defer func() {
		errClose := cm.Close()
		if errClose != nil {
			log.Errorf("cannot close containers map: %s", errClose.Error())
		}
	}()

	s := api.NewServer(log.Named("main_server"), cm, *serverPort)
	go s.Serve()
	defer func() {
		errClose := s.Close()
		if errClose != nil {
			log.Errorf("cannot close main server: %s", errClose.Error())
		}
	}()

	log.Info("Server has been started.")
	<-ctx.Done()
	log.Info("See you soon.")
}
