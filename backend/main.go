package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"chat-assist-backend/internal/auth"
	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/llm"
	"chat-assist-backend/internal/onebot"
	"chat-assist-backend/internal/server"
	"chat-assist-backend/internal/sse"
	"chat-assist-backend/internal/storage"
)

func main() {
	configFlag := flag.String("config", "", "path to config.toml")
	flag.Parse()

	configPath := config.ResolveConfigPath(strings.TrimSpace(*configFlag))
	if configPath == "" {
		log.Fatal("config.toml not found")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if strings.TrimSpace(cfg.Server.Password) == "" {
		log.Println("warning: server.password is empty, login will always fail")
	}

	db, err := storage.Open(cfg.Storage.SQLitePath)
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	defer sqlDB.Close()

	store := storage.New(db)
	if err := store.Init(context.Background()); err != nil {
		log.Fatalf("init db failed: %v", err)
	}

	llmClient := llm.NewClient(cfg)
	if !llmClient.Enabled() {
		log.Println("openai not configured: skip extraction")
	}

	tokenTTL := time.Duration(cfg.Server.TokenTTLMinutes) * time.Minute
	tokens := auth.NewTokenStore(tokenTTL)

	onebotClient := onebot.NewClient(cfg)
	srv := server.New(cfg, store, tokens, onebotClient)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	srv.RegisterRoutes(router)
	server.RegisterStatic(router, cfg.Server.StaticDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseListener := sse.NewListener(cfg, store, llmClient)
	go sseListener.Start(ctx)

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Println("shutting down...")
	cancel()
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()
	if err := httpServer.Shutdown(ctxTimeout); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
