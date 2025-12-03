package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/reconciler"
	"print3d-order-bot/internal/telegram"
	"syscall"
	"time"

	"github.com/jackc/pgx"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := pgx.Connect(pgx.ConnConfig{
		Host:     cfg.DB.Host,
		User:     cfg.DB.Username,
		Password: cfg.DB.Password,
		Database: cfg.DB.Database,
	})
	if err != nil {
		log.Fatal(err)
	}

	fileService := file.NewDefaultService(nil, &cfg.FileService)

	orderRepo := order.NewDefaultRepo(conn)
	orderService := order.NewDefaultService(orderRepo, fileService)

	bot, api, err := telegram.NewBot(orderService, &cfg.TelegramCfg)
	if err != nil {
		log.Fatal(err)
	}
	bot.Start(ctx)

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	downloader := file.NewTelegramDownloader(api, httpClient)
	fileService.SetDownloader(downloader)

	reconcilerService := reconciler.NewDefaultService(orderService, fileService)
	reconcilerService.Start(ctx)

	<-ctx.Done()
	slog.Info("Shutting down...")
	ctx, shutdown := context.WithTimeout(context.Background(), time.Second*15)
	defer shutdown()

	if err := reconcilerService.Stop(ctx); err != nil {
		log.Fatal(err)
	}

	if err := conn.Close(); err != nil {
		log.Fatal(err)
	}
}
