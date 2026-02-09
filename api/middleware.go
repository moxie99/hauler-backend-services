package api

import (
	"fmt"
	"os"
	"strings"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Secret key for signing JWT
var jwtSecret = func() []byte {
	secret := os.Getenv("SECRET_TOKEN")
	if secret == "" {
		secret = "your-secret-key-change-in-production"
	}
	return []byte(secret)
}()

// JWTAuthMiddleware validates JWT tokens from the Authorization header.
// It extracts user information from the token and stores it in the context.
// It supports both "Bearer <token>" format and direct token format.
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			ResponseJSON(c, http.StatusUnauthorized, "Authorization token required", nil)
			c.Abort()
			return
		}
		// Remove "Bearer " prefix if present
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil {
			ResponseJSON(c, http.StatusUnauthorized, "Invalid token", nil)
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Store user information in context
			var userID uint
			if uid, ok := claims["user_id"].(float64); ok {
				userID = uint(uid)
				c.Set("user_id", userID)
			}
			if email, ok := claims["email"].(string); ok {
				c.Set("email", email)
			}
			if role, ok := claims["role"].(string); ok {
				c.Set("role", role)
			}

			// Verify token version against database to support session invalidation
			if userID > 0 {
				var user User
				if err := DB.First(&user, userID).Error; err != nil {
					ResponseJSON(c, http.StatusUnauthorized, "User not found", nil)
					c.Abort()
					return
				}

				tokenVersion := uint(0)
				if tv, ok := claims["token_version"].(float64); ok {
					tokenVersion = uint(tv)
				}

				if tokenVersion != user.TokenVersion {
					ResponseJSON(c, http.StatusUnauthorized, "Token has been invalidated. Please login again", nil)
					c.Abort()
					return
				}
			}
		} else {
			ResponseJSON(c, http.StatusUnauthorized, "Invalid token claims", nil)
			c.Abort()
			return
		}

		// Token is valid, proceed to the next handler
		c.Next()
	}
}

// RequireRole creates a middleware that requires the user to have one of the specified roles.
func RequireRole(allowedRoles ...UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get("role")
		if !exists {
			ResponseJSON(c, http.StatusUnauthorized, "User role not found in token", nil)
			c.Abort()
			return
		}

		userRole := UserRole(roleStr.(string))
		allowed := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				allowed = true
				break
			}
		}

		if !allowed {
			ResponseJSON(c, http.StatusForbidden, "Insufficient permissions", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin is a convenience middleware that requires super admin role.
func RequireSuperAdmin() gin.HandlerFunc {
	return RequireRole(RoleSuperAdmin)
}

// RequireAdmin is a convenience middleware that requires admin or super admin role.
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(RoleSuperAdmin, RoleAdmin)
}
