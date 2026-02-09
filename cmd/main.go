package main

import (
	"hauler-backend-services/api"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	api.InitDB()
	r := gin.Default()

	// CORS configuration
	// In production (RENDER is set), only allow the admin frontend
	// In development, allow localhost:3000
	var allowedOrigins []string
	if os.Getenv("RENDER") != "" {
		allowedOrigins = []string{"https://hauler-admin-page.onrender.com"}
	} else {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public routes - Authentication
	r.POST("/api/auth/register", api.Register)
	r.POST("/api/auth/login", api.Login)
	r.POST("/api/auth/verify-login-otp", api.VerifyLoginOTP)

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
		// KYC management
		protected.PUT("/driver/kyc-status", api.UpdateKYCStatus)     // Driver updates own KYC
		protected.PUT("/driver/kyc-status/:id", api.UpdateKYCStatus) // Admin updates driver KYC
		// Change password
		protected.POST("/auth/change-password/request-otp", api.RequestChangePasswordOTP)
		protected.POST("/auth/change-password", api.ChangePassword)
	}

	// Admin routes
	_ = r.Group("/api/admin", api.JWTAuthMiddleware(), api.RequireAdmin())
	// Admin-specific routes will be added here

	// Super Admin routes
	superAdmin := r.Group("/api/super-admin", api.JWTAuthMiddleware(), api.RequireSuperAdmin())
	{
		superAdmin.POST("/create-admin", api.CreateAdmin)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
