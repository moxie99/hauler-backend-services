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

	// Public routes - Countries, States, and Genders
	r.GET("/api/countries", api.GetCountries)
	r.GET("/api/countries/:id/states", api.GetStatesByCountry)
	r.GET("/api/genders", api.GetGenders)
	r.GET("/api/vehicle-types", api.GetAllVehicleTypes)
	r.GET("/api/vehicle-types/:id", api.GetVehicleType)
	r.GET("/api/categories", api.GetAllCategories)
	r.GET("/api/categories/:id", api.GetCategory)
	r.GET("/api/load-types", api.GetAllLoadTypes)
	r.GET("/api/load-types/:id", api.GetLoadType)

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
		// KYC
		protected.GET("/driver/kyc", api.GetKYCProfile)
		protected.POST("/driver/kyc/step/1", api.SubmitKYCStep1)
		protected.POST("/driver/kyc/step/2", api.SubmitKYCStep2)
		protected.POST("/driver/kyc/step/3", api.SubmitKYCStep3)
		protected.POST("/driver/kyc/step/4", api.SubmitKYCStep4)
		// Token refresh
		protected.POST("/auth/refresh-token", api.RefreshToken)
		// Change password
		protected.POST("/auth/change-password/request-otp", api.RequestChangePasswordOTP)
		protected.POST("/auth/change-password", api.ChangePassword)
		// Logout
		protected.POST("/auth/logout", api.Logout)
	}

	// Super Admin routes
	superAdmin := r.Group("/api/super-admin", api.JWTAuthMiddleware(), api.RequireSuperAdmin())
	{
		superAdmin.POST("/create-admin", api.CreateAdmin)
		superAdmin.GET("/admins", api.GetAllAdmins)
		superAdmin.PUT("/admins/:id", api.UpdateAdmin)
		superAdmin.DELETE("/admins/:id", api.DeleteAdmin)
		superAdmin.GET("/drivers", api.GetAllDrivers)
		superAdmin.GET("/drivers/:id", api.GetDriverDetails)
		superAdmin.PUT("/drivers/:id/suspend", api.SuspendDriver)
		superAdmin.POST("/drivers/:id/review-document", api.ReviewDocument)
	}

	// Admin routes - Country and State management + Driver management
	admin := r.Group("/api/admin", api.JWTAuthMiddleware(), api.RequireAdmin())
	{
		admin.POST("/countries", api.CreateCountry)
		admin.PUT("/countries/:id", api.UpdateCountry)
		admin.DELETE("/countries/:id", api.DeleteCountry)
		admin.POST("/states", api.CreateState)
		admin.PUT("/states/:id", api.UpdateState)
		admin.DELETE("/states/:id", api.DeleteState)
		admin.GET("/drivers", api.GetAllDrivers)
		admin.GET("/drivers/:id", api.GetDriverDetails)
		admin.PUT("/drivers/:id/suspend", api.SuspendDriver)
		admin.POST("/drivers/:id/review-document", api.ReviewDocument)
		admin.POST("/vehicle-types", api.CreateVehicleType)
		admin.PUT("/vehicle-types/:id", api.UpdateVehicleType)
		admin.DELETE("/vehicle-types/:id", api.DeleteVehicleType)
		admin.POST("/categories", api.CreateCategory)
		admin.PUT("/categories/:id", api.UpdateCategory)
		admin.DELETE("/categories/:id", api.DeleteCategory)
		admin.POST("/load-types", api.CreateLoadType)
		admin.PUT("/load-types/:id", api.UpdateLoadType)
		admin.DELETE("/load-types/:id", api.DeleteLoadType)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
