package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
)

// Claims carried in VoxMesh access tokens.
type Claims struct {
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

// LoadKeys reads RS256 private and public keys from PEM files.
func LoadKeys(privatePath, publicPath string) error {
	privData, err := os.ReadFile(privatePath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(privData)
	if block == nil {
		return fmt.Errorf("failed to parse private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
	}
	var ok bool
	privateKey, ok = key.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not RSA")
	}

	pubData, err := os.ReadFile(publicPath)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	block, _ = pem.Decode(pubData)
	if block == nil {
		return fmt.Errorf("failed to parse public key PEM")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	publicKey, ok = pubKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	return nil
}

// GenerateAccessToken creates a JWT with 1 hour expiry.
func GenerateAccessToken(userID, username string, roles []string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			Issuer:    "voxmesh-auth",
			ID:        newJTI(),
		},
		Username: username,
		Roles:    roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// GenerateRefreshToken creates a random opaque token (not JWT).
// The SHA-256 hash is stored in the database; raw value returned to the client.
func GenerateRefreshToken() (raw string, hash string) {
	b := make([]byte, 32)
	rand.Read(b)
	raw = hex.EncodeToString(b)
	hashBytes := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(hashBytes[:])
	return raw, hash
}

// ValidateAccessToken parses and validates a JWT.
func ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return publicKey, nil
		},
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// ValidateAPIKey compares a raw API key against a stored SHA-256 hash.
func ValidateAPIKey(raw, storedHash string) bool {
	hashBytes := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hashBytes[:]) == storedHash
}

// HashAPIKey returns the SHA-256 hex hash of an API key.
func HashAPIKey(raw string) string {
	hashBytes := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hashBytes[:])
}

// FiberAuthMiddleware extracts and validates Bearer token from Authorization header.
// On success, stores *Claims in fiber.Ctx.Locals("claims").
func FiberAuthMiddleware(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if header == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": "missing authorization header"},
		})
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": "invalid authorization format"},
		})
	}

	claims, err := ValidateAccessToken(parts[1])
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": "invalid or expired token"},
		})
	}

	c.Locals("claims", claims)
	return c.Next()
}

// RequireRole returns a middleware that checks for a specific role.
func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": 40004, "message": "not authenticated"},
			})
		}
		for _, r := range claims.Roles {
			if r == role || r == "admin" {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{"code": 40004, "message": fmt.Sprintf("requires role %s", role)},
		})
	}
}

func newJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
