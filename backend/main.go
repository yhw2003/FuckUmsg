package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"chat-assist-backend/internal/auth"
	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/llm"
	"chat-assist-backend/internal/logx"
	"chat-assist-backend/internal/onebot"
	"chat-assist-backend/internal/server"
	"chat-assist-backend/internal/sse"
	"chat-assist-backend/internal/storage"
)

func resolveLogOptions(cfg config.Config, envLevel string) (level, format string) {
	level = strings.TrimSpace(cfg.Log.Level)
	if level == "" {
		level = strings.TrimSpace(envLevel)
	}
	if level == "" {
		level = "info"
	}
	format = strings.TrimSpace(cfg.Log.Format)
	if format == "" {
		format = "json"
	}
	return level, format
}
func main() {
	configFlag := flag.String("config", "", "path to config.toml")
	flag.Parse()

	configPath := config.ResolveConfigPath(strings.TrimSpace(*configFlag))
	if configPath == "" {
		os.Stderr.WriteString("config.toml not found\n")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		os.Stderr.WriteString("load config failed: " + err.Error() + "\n")
		os.Exit(1)
	}

	effectiveLevel, effectiveFormat := resolveLogOptions(cfg, os.Getenv("LOG_LEVEL"))
	logger, err := logx.New(effectiveLevel, effectiveFormat)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	if strings.TrimSpace(cfg.Server.Password) == "" {
		logger.Warn("server.password is empty, login will always fail")
	}

	db, err := storage.Open(cfg.Storage.SQLitePath)
	if err != nil {
		logger.Error("open db failed", zap.Error(err))
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("open db failed", zap.Error(err))
		os.Exit(1)
	}
	defer sqlDB.Close()

	store := storage.New(db)
	if err := store.Init(context.Background()); err != nil {
		logger.Error("init db failed", zap.Error(err))
		os.Exit(1)
	}

	llmClient := llm.NewClient(cfg)
	if !llmClient.Enabled() {
		logger.Info("openai not configured: skip extraction")
	}

	tokenTTL := time.Duration(cfg.Server.TokenTTLMinutes) * time.Minute
	tokens := auth.NewTokenStore(tokenTTL)

	onebotClient := onebot.NewClient(cfg)
	selfID := int64(0)
	if onebotClient != nil {
		id, err := onebotClient.GetSelfID(context.Background())
		if err != nil {
			logger.Warn("get onebot self id failed", zap.Error(err))
		} else {
			selfID = id
			logger.Info("onebot self id", zap.Int64("self_id", selfID))
		}
	}
	srv := server.New(cfg, store, tokens, onebotClient, logger.Named("server"))
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	srv.RegisterRoutes(router)
	server.RegisterStatic(router, cfg.Server.StaticDir, cfg.Server.StaticEmbed, logger.Named("static"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseListener := sse.NewListener(cfg, selfID, store, llmClient, logger.Named("sse"))
	go sseListener.Start(ctx)

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", zap.Error(err))
			os.Exit(1)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	logger.Info("shutting down")
	cancel()
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()
	if err := httpServer.Shutdown(ctxTimeout); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
}
