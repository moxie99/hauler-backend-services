package main

import (
	"hauler-backend-services/api"

	"github.com/gin-gonic/gin"
)

func main() {
	api.InitDB()
	r := gin.Default()

	// Public routes - Authentication
	r.POST("/api/auth/register", api.Register)
	r.POST("/api/auth/login", api.Login)

	// Driver registration and verification
	r.POST("/api/driver/register", api.DriverRegister)
	r.POST("/api/driver/verify-email", api.VerifyEmail)
	r.POST("/api/driver/resend-verification-code", api.ResendDriverVerificationCode)

	// Password reset flow
	r.POST("/api/auth/forgot-password", api.ForgotPassword)
	r.POST("/api/auth/verify-forgot-password", api.VerifyForgotPasswordCode)
	r.POST("/api/auth/reset-password", api.ResetPassword)
	r.POST("/api/auth/resend-forgot-password-code", api.ResendForgotPasswordCode)

	// Protected routes - User profile
	protected := r.Group("/api", api.JWTAuthMiddleware())
	{
		protected.GET("/profile", api.GetProfile)
	}

	// Admin routes
	_ = r.Group("/api/admin", api.JWTAuthMiddleware(), api.RequireAdmin())
	// Admin-specific routes will be added here

	// Super Admin routes
	_ = r.Group("/api/super-admin", api.JWTAuthMiddleware(), api.RequireSuperAdmin())
	// Super admin-specific routes will be added here

	r.Run(":8080")
}
