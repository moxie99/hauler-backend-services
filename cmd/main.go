package main

import (
	"fmt"
	"hauler-backend-services/api"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	api.InitDB()
	api.InitEventProducer()
	api.InitWebSocketHub() // Initialize WebSocket hub
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
	r.POST("/api/auth/verify-email", api.VerifyEmail)
	r.POST("/api/auth/resend-verification-code", api.ResendVerificationCode)

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
		// Order management
		protected.POST("/orders", api.CreateOrder)
		protected.GET("/orders", api.GetOrders)
		protected.GET("/orders/:id", api.GetOrder)
		protected.POST("/orders/:id/confirm", api.ConfirmOrder)
		protected.GET("/orders/:id/tracking", api.GetOrderTracking)
		protected.PUT("/orders/:id/tracking", api.UpdateOrderTracking)
		protected.GET("/orders/:id/ws", api.HandleWebSocket) // WebSocket for real-time tracking
		protected.GET("/orders/:id/sse", api.HandleSSE)      // Server-Sent Events for real-time tracking
		// KYC management
		protected.PUT("/driver/kyc-status", api.UpdateKYCStatus)     // Driver updates own KYC
		protected.PUT("/driver/kyc-status/:id", api.UpdateKYCStatus) // Admin updates driver KYC
		// KYC
		protected.GET("/driver/kyc", api.GetKYCProfile)
		protected.POST("/driver/kyc/step/1", api.SubmitKYCStep1)
		protected.POST("/driver/kyc/step/2", api.SubmitKYCStep2)
		protected.POST("/driver/kyc/step/3", api.SubmitKYCStep3)
		protected.POST("/driver/kyc/step/4", api.SubmitKYCStep4)
		protected.POST("/driver/kyc/step/5", api.SubmitKYCStep5)
		// Location management
		protected.POST("/location", api.UpdateLocation)
		protected.GET("/location/:id", api.GetLocation)
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
		admin.GET("/orders", api.GetAllOrders)
		admin.PATCH("/orders/:id/status", api.UpdateOrderStatus)
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

	// Health check routes (public - no authentication required)
	r.GET("/health", api.HealthCheck)
	r.GET("/health/detailed", api.HealthCheckDetailed)
	r.GET("/health/ready", api.ReadinessCheck)
	r.GET("/health/live", api.LivenessCheck)

	// Start streaming services if Kafka is configured
	var dispatchMatcher *api.DispatchMatcher
	var pricingService *api.PricingService
	var trackingService *api.TrackingService
	var notificationService *api.NotificationService
	kafkaBrokers := strings.Split(strings.TrimSpace(os.Getenv("KAFKA_BROKERS")), ",")
	if len(kafkaBrokers) > 0 && kafkaBrokers[0] != "" {
		dispatchMatcher = api.NewDispatchMatcher(kafkaBrokers)
		if err := dispatchMatcher.Start(); err != nil {
			log.Printf("Warning: Failed to start dispatch matcher: %v. Dispatch will not function.", err)
		} else {
			log.Println("[Dispatch] Matcher started successfully")
		}

		pricingService = api.NewPricingService(kafkaBrokers)
		if err := pricingService.Start(); err != nil {
			log.Printf("Warning: Failed to start pricing service: %v. Pricing will not function.", err)
		} else {
			log.Println("[Pricing] Service started successfully")
		}

		trackingService = api.NewTrackingService(kafkaBrokers)
		if err := trackingService.Start(); err != nil {
			log.Printf("Warning: Failed to start tracking service: %v. Tracking will not function.", err)
		} else {
			log.Println("[Tracking] Service started successfully")
		}

		notificationService = api.NewNotificationService(kafkaBrokers)
		if err := notificationService.Start(); err != nil {
			log.Printf("Warning: Failed to start notification service: %v. Notifications will not function.", err)
		} else {
			log.Println("[Notifications] Service started successfully")
		}
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Shutting down gracefully...", sig)

		// Stop all streaming services
		if dispatchMatcher != nil {
			if err := dispatchMatcher.Stop(); err != nil {
				log.Printf("Error stopping dispatch matcher: %v", err)
			} else {
				log.Println("[Dispatch] Matcher stopped")
			}
		}
		if pricingService != nil {
			if err := pricingService.Stop(); err != nil {
				log.Printf("Error stopping pricing service: %v", err)
			} else {
				log.Println("[Pricing] Service stopped")
			}
		}
		if trackingService != nil {
			if err := trackingService.Stop(); err != nil {
				log.Printf("Error stopping tracking service: %v", err)
			} else {
				log.Println("[Tracking] Service stopped")
			}
		}
		if notificationService != nil {
			if err := notificationService.Stop(); err != nil {
				log.Printf("Error stopping notification service: %v", err)
			} else {
				log.Println("[Notifications] Service stopped")
			}
		}

		os.Exit(0)
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	r.Run(":" + port)
}
