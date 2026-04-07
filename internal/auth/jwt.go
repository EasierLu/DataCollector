package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims JWT 自定义声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager JWT 管理器
type JWTManager struct {
	secret     []byte
	expiration time.Duration
}

// NewJWTManager 创建新的 JWT 管理器
func NewJWTManager(secret string, expiration time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// GenerateToken 生成 JWT token
// 返回 (token_string, expires_in_seconds, error)
func (j *JWTManager) GenerateToken(userID int64, username, role string) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(j.expiration)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secret)
	if err != nil {
		return "", 0, err
	}

	expiresIn := int64(j.expiration.Seconds())
	return tokenString, expiresIn, nil
}

// ValidateToken 验证 JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken 刷新 token（当剩余有效期不足 2 小时时允许刷新）
func (j *JWTManager) RefreshToken(tokenString string) (string, int64, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", 0, err
	}

	// 检查剩余有效期是否不足 2 小时
	if claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 2*time.Hour {
			return "", 0, errors.New("token can only be refreshed when less than 2 hours remaining")
		}
	}

	// 生成新 token
	return j.GenerateToken(claims.UserID, claims.Username, claims.Role)
}

// HashPassword 使用 bcrypt 加密密码（cost=12）
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
