package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/golang-jwt/jwt/v5"
	"github.com/voxmesh/pkg/auth"
	"github.com/voxmesh/pkg/config"
	"github.com/voxmesh/pkg/db"
	slogx "github.com/voxmesh/pkg/log"
	"github.com/voxmesh/pkg/ratelimit"

	wshandler "github.com/voxmesh/ws-gateway/internal/handler"
)

// LiveKitClaims defines the JWT claims for LiveKit SFU access tokens.
// LiveKit uses HS256 JWT with these specific claims.
type LiveKitClaims struct {
	jwt.RegisteredClaims
	Room     string `json:"room"`
	Metadata string `json:"metadata"`
	Video    bool   `json:"video"`
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Load JWT keys for token validation
	if err := auth.LoadKeys(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath); err != nil {
		slogx.Fatal("jwt keys: %v", err)
	}

	redisClient, err := db.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slogx.Fatal("redis: %v", err)
	}
	defer redisClient.Close()

	// WebSocket handler
	wsHandler := wshandler.NewWSHandler(func(userID string) {
		slogx.Info("[ws] user disconnected: %s", userID)
	})
	wsHandler.Run()

	app := fiber.New(fiber.Config{AppName: "voxmesh-ws-gateway"})

	// Prometheus metrics
	fiberprometheus.New("voxmesh-ws-gateway").RegisterAt(app, "/metrics")

	// Rate limiting — strict on auth, moderate on API, skip WebSocket
	rlStore := ratelimit.NewRedisStore(redisClient)
	app.Use(ratelimit.New(ratelimit.Config{
		Store:  rlStore,
		Max:    100,
		Window: time.Minute,
		SkipFunc: func(c *fiber.Ctx) bool {
			return c.Path() == "/ws" || c.Path() == "/health" || c.Path() == "/metrics"
		},
	}))
	// Stricter limit on auth endpoints (login/register/refresh)
	app.Use(ratelimit.New(ratelimit.Config{
		Store:  rlStore,
		Max:    5,
		Window: time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return "strict:" + c.IP()
		},
		SkipFunc: func(c *fiber.Ctx) bool {
			p := c.Path()
			return p != "/api/v1/auth/login" && p != "/api/v1/auth/register" && p != "/api/v1/auth/refresh"
		},
	}))

	// CORS — allow localhost by default, extend via CORS_ORIGINS env var
	corsOrigins := "http://localhost:5173,http://127.0.0.1:5173"
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		corsOrigins = corsOrigins + "," + extra
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		MaxAge:       int((10 * time.Minute).Seconds()),
	}))

	// WebSocket endpoint — JWT validated inline in handler
	app.Get("/ws", wsHandler.Upgrade())

	// REST API proxy to downstream services — preserve full incoming path
	app.All("/api/v1/auth/*", func(c *fiber.Ctx) error {
		return proxy.Do(c, cfg.AuthServiceURL+c.OriginalURL())
	})
	app.All("/api/v1/channels/*", func(c *fiber.Ctx) error {
		return proxy.Do(c, cfg.ChannelServiceURL+c.OriginalURL())
	})

	// ── TURN credentials endpoint ──
	turnSecret := os.Getenv("TURN_SECRET")
	if turnSecret == "" {
		turnSecret = "voxmesh-turn-dev-secret-change-me"
	}
	app.Get("/api/v1/system/turn", func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": 40004, "message": "missing Bearer token"},
			})
		}
		claims, err := auth.ValidateAccessToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil || claims.Subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": 40004, "message": "invalid token"},
			})
		}

		expiry := time.Now().Add(24 * time.Hour).Unix()
		username := fmt.Sprintf("%d:%s", expiry, claims.Subject)

		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(username))
		password := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		turnHost := os.Getenv("TURN_HOST")
		if turnHost == "" {
			turnHost = c.Hostname()
		}
		return c.JSON(fiber.Map{
			"username":  username,
			"password":  password,
			"ttl":       86400,
			"stun_uri":  fmt.Sprintf("stun:%s:3478", turnHost),
			"turn_uri":  fmt.Sprintf("turn:%s:3478", turnHost),
			"turns_uri": fmt.Sprintf("turns:%s:5349", turnHost),
		})
	})

	// ── LiveKit SFU token endpoint ──
	// Generates access tokens for 4+ user voice channels.
	// Frontend calls this when switching from P2P mesh to SFU mode.
	livekitKey, livekitSecret := parseLiveKitKeys(os.Getenv("LIVEKIT_KEYS"))
	app.Get("/api/v1/system/livekit-token", func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": 40004, "message": "missing Bearer token"},
			})
		}
		claims, err := auth.ValidateAccessToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil || claims.Subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": 40004, "message": "invalid token"},
			})
		}
		if livekitKey == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": fiber.Map{"code": 50001, "message": "LiveKit not configured"},
			})
		}

		channelID := c.Query("channel_id")
		if channelID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": 40000, "message": "channel_id query parameter required"},
			})
		}

		now := time.Now()
		lkClaims := LiveKitClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    livekitKey,
				Subject:   claims.Subject,
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
				NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			},
			Room:     channelID,
			Video:    false,
			Metadata: fmt.Sprintf(`{"username":"%s"}`, claims.Username),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, lkClaims)
		signed, err := token.SignedString([]byte(livekitSecret))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"code": 50001, "message": "token generation failed"},
			})
		}

		livekitHost := os.Getenv("LIVEKIT_HOST")
		if livekitHost == "" {
			livekitHost = c.Hostname()
		}
		return c.JSON(fiber.Map{
			"token":    signed,
			"url":      fmt.Sprintf("wss://%s:7880", livekitHost),
			"room":     channelID,
			"username": claims.Username,
		})
	})

	// System endpoints
	app.Get("/api/v1/system/health", func(c *fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if err := redisClient.Ping(c.Context()).Err(); err != nil {
			status = fiber.Map{"status": "degraded", "redis": "unreachable"}
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		return c.JSON(status)
	})
	app.Get("/api/v1/system/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": "0.1.0", "api_version": "v1"})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if err := redisClient.Ping(c.Context()).Err(); err != nil {
			status = fiber.Map{"status": "degraded", "redis": "unreachable"}
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		return c.JSON(status)
	})

	slogx.Info("WS Gateway starting on %s", cfg.Address())

	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			slogx.Fatal("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogx.Info("ws-gateway shutting down")
	app.Shutdown()
}

// parseLiveKitKeys extracts the first API key:secret pair from LIVEKIT_KEYS env var.
// Format: "key1: secret1, key2: secret2"
func parseLiveKitKeys(keysStr string) (key, secret string) {
	if keysStr == "" {
		return "", ""
	}
	// Take the first pair before any comma
	pair := strings.SplitN(keysStr, ",", 2)[0]
	parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// wsAuthMiddleware reads JWT from query param for WebSocket upgrades.
func wsAuthMiddleware(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": "missing token query parameter"},
		})
	}

	claims, err := auth.ValidateAccessToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": "invalid or expired token"},
		})
	}

	c.Locals("claims", claims)
	return c.Next()
}
