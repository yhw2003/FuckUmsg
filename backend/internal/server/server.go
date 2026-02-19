package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"chat-assist-backend/internal/auth"
	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/model"
	"chat-assist-backend/internal/onebot"
	"chat-assist-backend/internal/storage"
)

type serverStore interface {
	List(ctx context.Context) ([]model.Todo, error)
	ListLLMFailedMessagesByUser(ctx context.Context, userID int64) ([]model.LLMFailedMessage, error)
	UpdateStatus(ctx context.Context, id int64, status string, completedAt *int64) error
	UpdateContent(ctx context.Context, id int64, title string, detail string) error
	Delete(ctx context.Context, id int64) error
}

type Server struct {
	cfg    config.Config
	store  serverStore
	tokens *auth.TokenStore
	onebot *onebot.Client
	logger *zap.Logger
}

func New(cfg config.Config, store serverStore, tokens *auth.TokenStore, onebot *onebot.Client, logger *zap.Logger) *Server {
	return &Server{cfg: cfg, store: store, tokens: tokens, onebot: onebot, logger: logger}
}

func (s *Server) RegisterRoutes(router *gin.Engine) {
	router.POST("/api/login", s.handleLogin)

	api := router.Group("/api")
	api.Use(auth.Middleware(s.tokens))
	api.GET("/todos", s.handleListTodos)
	api.GET("/failed-messages", s.handleListFailedMessages)
	api.POST("/todos/:id/complete", s.handleComplete)
	api.POST("/todos/:id/reopen", s.handleReopen)
	api.PATCH("/todos/:id", s.handleUpdateTitle)
	api.DELETE("/todos/:id", s.handleDelete)
}

func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Password == "" || req.Password != s.cfg.Server.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	token, expires, err := s.tokens.Issue()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expires.Unix(),
		"expires_in": int64(time.Until(expires).Seconds()),
	})
}

func (s *Server) handleListTodos(c *gin.Context) {
	todos, err := s.store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load todos"})
		return
	}
	s.resolveTodoNames(c.Request.Context(), todos)
	c.JSON(http.StatusOK, gin.H{"todos": todos})
}

func (s *Server) handleListFailedMessages(c *gin.Context) {
	userID, ok := c.Get("auth_user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	failedMessages, err := s.store.ListLLMFailedMessagesByUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load failed messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"failed_messages": failedMessages})
}

func (s *Server) handleComplete(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	now := time.Now().Unix()
	if err := s.store.UpdateStatus(c.Request.Context(), id, storage.StatusDone, &now); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleReopen(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.UpdateStatus(c.Request.Context(), id, storage.StatusOpen, nil); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleUpdateTitle(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	if err := s.store.UpdateContent(c.Request.Context(), id, req.Title, req.Detail); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleDelete(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func parseIDParam(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func (s *Server) resolveTodoNames(ctx context.Context, todos []model.Todo) {
	if s.onebot == nil {
		for i := range todos {
			storedSenderName := strings.TrimSpace(todos[i].SourceName)
			todos[i].SourceName = fallbackSourceName(todos[i])
			todos[i].SenderName = fallbackSenderName(storedSenderName, todos[i].SenderID)
		}
		return
	}
	for i := range todos {
		todo := &todos[i]
		storedSenderName := strings.TrimSpace(todo.SourceName)
		switch todo.SourceType {
		case "group":
			groupName, err := s.onebot.GetGroupName(ctx, todo.SourceID)
			if err != nil || groupName == "" {
				todo.SourceName = todo.SourceID
			} else {
				todo.SourceName = groupName
			}
			if todo.SenderID != 0 {
				name, err := s.onebot.GetGroupMemberName(ctx, todo.SourceID, todo.SenderID)
				if err == nil && name != "" {
					todo.SenderName = name
					break
				}
			}
			todo.SenderName = fallbackSenderName(storedSenderName, todo.SenderID)
		case "private":
			if todo.SenderID != 0 {
				name, err := s.onebot.GetStrangerName(ctx, todo.SenderID)
				if err == nil && name != "" {
					todo.SourceName = name
					todo.SenderName = name
					break
				}
			}
			todo.SourceName = fallbackSourceName(*todo)
			todo.SenderName = fallbackSenderName(storedSenderName, todo.SenderID)
		default:
			todo.SourceName = fallbackSourceName(*todo)
			todo.SenderName = fallbackSenderName(storedSenderName, todo.SenderID)
		}
	}
}

func fallbackSourceName(todo model.Todo) string {
	if todo.SourceID != "" {
		return todo.SourceID
	}
	return "未知来源"
}

func fallbackSenderName(stored string, senderID int64) string {
	if senderID != 0 {
		return strconv.FormatInt(senderID, 10)
	}
	if strings.TrimSpace(stored) != "" {
		return stored
	}
	return "发送者未知"
}
