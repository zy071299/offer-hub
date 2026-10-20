package tools

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/service"
)

type tokenAuthenticator interface {
	AuthenticateToken(context.Context, string) (service.TokenClaims, error)
}

type tokenParser interface {
	ParseToken(string) (service.TokenClaims, error)
}

// SoftJWTAuthMiddleware 尝试解析 Bearer token，所有情况均放行且不访问 Redis。
func SoftJWTAuthMiddleware() gin.HandlerFunc {
	return softJWTAuthMiddleware(service.NewLogoutService(nil, config.Conf.JWT))
}

func softJWTAuthMiddleware(parser tokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 游客或无效 token 不得保留客户端伪造的身份。
		c.Request.Header.Del("user_id")
		headers := c.Request.Header.Values("Authorization")
		if len(headers) == 1 {
			parts := strings.Fields(headers[0])
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				if claims, err := parser.ParseToken(parts[1]); err == nil {
					c.Request.Header.Set("user_id", claims.UserID)
				}
			}
		}
		c.Next()
	}
}

// JWTAuthMiddleware 强制验证 Bearer token，并把可信 user_id 写入请求 Header。
// 黑名单仅读取，不写入或刷新 TTL；其 TTL 由登出流程按 token 剩余有效期设置。
func JWTAuthMiddleware() gin.HandlerFunc {
	return jwtAuthMiddleware(service.NewLogoutService(nil, config.Conf.JWT))
}

func jwtAuthMiddleware(authenticator tokenAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 必须先移除客户端提供的身份，避免缺失或无效 token 保留伪造身份。
		c.Request.Header.Del("user_id")
		headers := c.Request.Header.Values("Authorization")
		if len(headers) != 1 {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
			return
		}
		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
			return
		}
		claims, err := authenticator.AuthenticateToken(c.Request.Context(), parts[1])
		if errors.Is(err, service.ErrUnauthenticated) {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 401, "msg": "未认证", "data": nil})
			return
		}
		if err != nil {
			log.Printf("JWT authentication failed: %v", err)
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误", "data": nil})
			return
		}
		c.Request.Header.Set("user_id", claims.UserID)
		c.Next()
	}
}
