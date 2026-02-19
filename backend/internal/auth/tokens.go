package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenEntry struct {
	expiresAt time.Time
	userID    int64
}

type TokenStore struct {
	ttl    time.Duration
	mu     sync.Mutex
	tokens map[string]tokenEntry
}

func NewTokenStore(ttl time.Duration) *TokenStore {
	return &TokenStore{
		ttl:    ttl,
		tokens: make(map[string]tokenEntry),
	}
}

func (s *TokenStore) Issue() (string, time.Time, error) {
	return s.IssueForUser(0)
}

func (s *TokenStore) IssueForUser(userID int64) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(s.ttl)
	s.mu.Lock()
	s.tokens[token] = tokenEntry{expiresAt: expires, userID: userID}
	s.mu.Unlock()
	return token, expires, nil
}

func (s *TokenStore) Validate(token string) bool {
	_, ok := s.ValidateAndGetUserID(token)
	return ok
}

func (s *TokenStore) ValidateAndGetUserID(token string) (int64, bool) {
	if token == "" {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.tokens[token]
	if !ok {
		return 0, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.tokens, token)
		return 0, false
	}
	return entry.userID, true
}

func Middleware(tokens *TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c.Request)
		userID, ok := tokens.ValidateAndGetUserID(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("auth_user_id", userID)
		c.Next()
	}
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
