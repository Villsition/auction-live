package middleware

import (
	"net/http"
	"strings"

	"auction/internal/repository"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func Auth(secret string, db *gorm.DB) gin.HandlerFunc {
	userRepo := repository.NewUserRepo(db)
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			response.Error(c, errcode.ErrUnauthorized, errcode.Message(errcode.ErrUnauthorized))
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			response.Error(c, errcode.ErrTokenExpired, errcode.Message(errcode.ErrTokenExpired))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, errcode.ErrUnauthorized, errcode.Message(errcode.ErrUnauthorized))
			c.Abort()
			return
		}

		uid := uint64(claims["user_id"].(float64))
		tokenVer, _ := claims["ver"].(float64)

		// Check token version against DB (invalidate old tokens)
		user, err := userRepo.GetByID(c.Request.Context(), uid)
		if err != nil || user.TokenVersion != int64(tokenVer) {
			response.Error(c, errcode.ErrTokenExpired, "token expired - please login again")
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", uid)
		if role, exists := claims["role"]; exists {
			c.Set("role", uint8(role.(float64)))
		}
		if nickname, exists := claims["nickname"]; exists {
			c.Set("nickname", nickname.(string))
		}
		if avatar, exists := claims["avatar"]; exists {
			c.Set("avatar", avatar.(string))
		}

		c.Next()
	}
}

// OptionalAuth does not abort on missing token
func OptionalAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.Next()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", uint64(userID.(float64)))
			}
			if role, exists := claims["role"]; exists {
				c.Set("role", uint8(role.(float64)))
			}
			if nickname, exists := claims["nickname"]; exists {
				c.Set("nickname", nickname.(string))
			}
			if avatar, exists := claims["avatar"]; exists {
				c.Set("avatar", avatar.(string))
			}
		}

		c.Next()
	}
}

// SellerOnly middleware - must be placed after Auth. Allows sellers (role=1) and admins (role=2).
func SellerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == nil {
			c.JSON(http.StatusForbidden, gin.H{"code": errcode.ErrForbidden, "message": "seller or admin required"})
			c.Abort()
			return
		}
		r := role.(uint8)
		if r != 1 && r != 2 {
			c.JSON(http.StatusForbidden, gin.H{"code": errcode.ErrForbidden, "message": "seller or admin required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminOnly middleware - must be placed after Auth
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == nil || role.(uint8) != 2 {
			c.JSON(http.StatusForbidden, gin.H{"code": errcode.ErrForbidden, "message": "admin required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
