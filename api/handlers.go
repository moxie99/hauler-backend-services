/*
Package api provides RESTful handlers and middleware for a haulage/freight service backend.
It includes functionality for user authentication, role-based access control, and managing haulage operations.
*/

package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Register handles user registration.
// It creates a new user account with the provided details and hashes the password.
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Set default role to customer if not provided
	if req.Role == "" {
		req.Role = RoleCustomer
	}

	// Check if email already exists
	var existingUserByEmail User
	if err := DB.Where("email = ?", req.Email).First(&existingUserByEmail).Error; err == nil {
		ResponseJSON(c, http.StatusConflict, "Email already registered", nil)
		return
	}

	// Check if phone number already exists
	var existingUserByPhone User
	if err := DB.Where("phone = ?", req.Phone).First(&existingUserByPhone).Error; err == nil {
		ResponseJSON(c, http.StatusConflict, "Phone number already registered", nil)
		return
	}

	// Validate password strength
	if err := ValidatePassword(req.Password); err != nil {
		ResponseJSON(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create new user
	user := User{
		Email:     req.Email,
		Password:  req.Password, // Will be hashed by BeforeCreate hook
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      req.Role,
		IsActive:  true,
	}

	if err := DB.Create(&user).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	// Don't return password in response
	user.Password = ""
	ResponseJSON(c, http.StatusCreated, "User registered successfully", user)
}

// Login handles user authentication.
// It validates user credentials and returns a JWT token with user information.
func Login(c *gin.Context) {
	var loginRequest LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Find user by email
	var user User
	if err := DB.Where("email = ?", loginRequest.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// Verify password first
	if !user.CheckPassword(loginRequest.Password) {
		ResponseJSON(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// For drivers: Check if email is verified
	if user.Role == RoleDriver && !user.IsActive {
		// Driver hasn't verified email - send verification code and return message
		code := GenerateVerificationCode()
		expiresAt := time.Now().Add(15 * time.Minute)

		// Invalidate any existing unused verification codes
		DB.Model(&VerificationCode{}).Where("email = ? AND used = ?", user.Email, false).Update("used", true)

		// Create new verification code
		verificationCode := VerificationCode{
			Email:     user.Email,
			Code:      code,
			ExpiresAt: expiresAt,
			Used:      false,
		}

		if err := DB.Create(&verificationCode).Error; err != nil {
			log.Printf("Failed to create verification code: %v", err)
		} else {
			// Send verification code via email
			emailService := NewEmailService()
			if err := emailService.SendVerificationCode(user.Email, code); err != nil {
				log.Printf("Failed to send verification email: %v", err)
			}
		}

		ResponseJSON(c, http.StatusForbidden, "Please verify your email address. A verification code has been sent to your email", gin.H{
			"requires_verification": true,
			"email":                user.Email,
		})
		return
	}

	// Check if user is active (for non-drivers or verified drivers)
	if !user.IsActive {
		ResponseJSON(c, http.StatusForbidden, "Account is deactivated", nil)
		return
	}

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour) // 24 hour expiration
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     expirationTime.Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Could not generate token", nil)
		return
	}

	// Don't return password in response
	user.Password = ""
	
	// Prepare response data
	responseData := gin.H{
		"token": tokenString,
		"user":  user,
	}
	
	// For drivers, check KYC status and include in response
	if user.Role == RoleDriver {
		// Set default KYC status if not set
		if user.KYCStatus == "" {
			user.KYCStatus = KYCStatusPending
		}
		
		responseData["kyc_status"] = user.KYCStatus
		
		// Check if KYC is not approved
		if user.KYCStatus != KYCStatusApproved {
			responseData["requires_kyc"] = true
			responseData["kyc_completed"] = false
			
			// Set appropriate message based on KYC status
			var kycMessage string
			switch user.KYCStatus {
			case KYCStatusPending:
				kycMessage = "Please complete your KYC verification to continue"
			case KYCStatusInProgress:
				kycMessage = "Your KYC documents are under review"
			case KYCStatusRejected:
				kycMessage = "Your KYC verification was rejected. Please contact support"
			default:
				kycMessage = "Please complete your KYC verification to continue"
			}
			responseData["kyc_message"] = kycMessage
		} else {
			responseData["requires_kyc"] = false
			responseData["kyc_completed"] = true
		}
	}
	
	ResponseJSON(c, http.StatusOK, "Login successful", responseData)
}

// DriverRegister handles driver registration with email verification.
// It creates a new driver account, generates a verification code, and sends it via email.
func DriverRegister(c *gin.Context) {
	var req DriverRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Check if email already exists
	var existingUserByEmail User
	if err := DB.Where("email = ?", req.Email).First(&existingUserByEmail).Error; err == nil {
		ResponseJSON(c, http.StatusConflict, "Email already registered", nil)
		return
	}

	// Check if phone number already exists
	var existingUserByPhone User
	if err := DB.Where("phone = ?", req.Phone).First(&existingUserByPhone).Error; err == nil {
		ResponseJSON(c, http.StatusConflict, "Phone number already registered", nil)
		return
	}

	// Validate password strength
	if err := ValidatePassword(req.Password); err != nil {
		ResponseJSON(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Create new driver user (initially inactive until email is verified)
	user := User{
		Email:     req.Email,
		Password:  req.Password, // Will be hashed by BeforeCreate hook
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      RoleDriver,
		IsActive:  false, // Inactive until email is verified
	}

	if err := DB.Create(&user).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to create driver account", nil)
		return
	}

	// Explicitly set IsActive to false after creation to ensure it's saved correctly
	// This handles cases where database default might override the struct value
	DB.Model(&user).Update("is_active", false)

	// Generate verification code
	code := GenerateVerificationCode()
	expiresAt := time.Now().Add(15 * time.Minute) // Code expires in 15 minutes

	// Invalidate any existing verification codes for this email
	DB.Model(&VerificationCode{}).Where("email = ? AND used = ?", req.Email, false).Update("used", true)

	// Create verification code record
	verificationCode := VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := DB.Create(&verificationCode).Error; err != nil {
		log.Printf("Failed to create verification code: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to generate verification code", nil)
		return
	}

	// Send verification code via email
	emailService := NewEmailService()
	if err := emailService.SendVerificationCode(req.Email, code); err != nil {
		log.Printf("Failed to send verification email: %v", err)
		// Don't fail the registration, but log the error
		// In production, you might want to handle this differently
	}

	// Don't return password in response
	user.Password = ""
	ResponseJSON(c, http.StatusCreated, "Driver registered successfully. Please check your email for verification code.", gin.H{
		"user":    user,
		"message": "Verification code sent to your email",
	})
}

// VerifyEmail handles email verification for drivers.
// It validates the verification code and activates the user account, then returns a JWT token.
func VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Find the verification code
	var verificationCode VerificationCode
	if err := DB.Where("email = ? AND code = ? AND used = ?", req.Email, req.Code, false).
		First(&verificationCode).Error; err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid or expired verification code", nil)
		return
	}

	// Check if code has expired
	if time.Now().After(verificationCode.ExpiresAt) {
		ResponseJSON(c, http.StatusBadRequest, "Verification code has expired", nil)
		return
	}

	// Find the user
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "User not found", nil)
		return
	}

	// Mark verification code as used
	verificationCode.Used = true
	DB.Save(&verificationCode)

	// Activate the user account
	user.IsActive = true
	
	// For drivers, set KYC status to pending after email verification
	if user.Role == RoleDriver {
		user.KYCStatus = KYCStatusPending
	}
	
	if err := DB.Save(&user).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to activate account", nil)
		return
	}

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour) // 24 hour expiration
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     expirationTime.Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Could not generate token", nil)
		return
	}

	// Don't return password in response
	user.Password = ""
	ResponseJSON(c, http.StatusOK, "Email verified successfully", gin.H{
		"token": tokenString,
		"user":  user,
	})
}

// ForgotPassword handles forgot password requests.
// It generates a verification code and sends it to the user's email.
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Check if user exists
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Email does not exist
		ResponseJSON(c, http.StatusNotFound, "Email does not exist", nil)
		return
	}

	// Check if user is active
	if !user.IsActive {
		ResponseJSON(c, http.StatusForbidden, "Account is deactivated", nil)
		return
	}

	// Generate verification code
	code := GenerateVerificationCode()
	expiresAt := time.Now().Add(15 * time.Minute) // Code expires in 15 minutes

	// Invalidate any existing verification codes for this email
	DB.Model(&VerificationCode{}).Where("email = ? AND used = ?", req.Email, false).Update("used", true)

	// Create verification code record
	verificationCode := VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := DB.Create(&verificationCode).Error; err != nil {
		log.Printf("Failed to create verification code: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to generate verification code", nil)
		return
	}

	// Send verification code via email
	emailService := NewEmailService()
	if err := emailService.SendPasswordResetCode(req.Email, code); err != nil {
		log.Printf("Failed to send password reset email: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to send password reset email", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Password reset code has been sent to your email", nil)
}

// VerifyForgotPasswordCode verifies the password reset code.
// This step confirms the user has access to their email before allowing password reset.
func VerifyForgotPasswordCode(c *gin.Context) {
	var req VerifyForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// First, check if user exists
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Email does not exist", nil)
		return
	}

	// Check if there's any verification code for this email
	var verificationCode VerificationCode
	if err := DB.Where("email = ? AND used = ?", req.Email, false).
		First(&verificationCode).Error; err != nil {
		ResponseJSON(c, http.StatusBadRequest, "No verification code found for this email. Please request a new password reset code", nil)
		return
	}

	// Check if the code matches
	if verificationCode.Code != req.Code {
		ResponseJSON(c, http.StatusBadRequest, "Invalid verification code", nil)
		return
	}

	// Check if code has expired
	if time.Now().After(verificationCode.ExpiresAt) {
		ResponseJSON(c, http.StatusBadRequest, "Verification code has expired. Please request a new password reset code", nil)
		return
	}

	// Code is valid, but don't mark it as used yet - it will be used in ResetPassword
	// Just return success to allow the user to proceed to password reset
	ResponseJSON(c, http.StatusOK, "Verification code is valid. You can now reset your password", gin.H{
		"verified": true,
	})
}

// ResetPassword handles password reset after email verification.
// It validates the code, verifies it hasn't been used, and updates the password.
func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Validate password strength
	if err := ValidatePassword(req.NewPassword); err != nil {
		ResponseJSON(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// First, check if user exists
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Email does not exist", nil)
		return
	}

	// Check if there's any verification code for this email
	var verificationCode VerificationCode
	if err := DB.Where("email = ? AND used = ?", req.Email, false).
		First(&verificationCode).Error; err != nil {
		ResponseJSON(c, http.StatusBadRequest, "No verification code found for this email. Please request a new password reset code", nil)
		return
	}

	// Check if the code matches
	if verificationCode.Code != req.Code {
		ResponseJSON(c, http.StatusBadRequest, "Invalid verification code", nil)
		return
	}

	// Check if code has expired
	if time.Now().After(verificationCode.ExpiresAt) {
		ResponseJSON(c, http.StatusBadRequest, "Verification code has expired. Please request a new password reset code", nil)
		return
	}

	// Mark verification code as used
	verificationCode.Used = true
	DB.Save(&verificationCode)

	// Update password (will be hashed by BeforeUpdate hook if we add one, or we hash it here)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to hash password", nil)
		return
	}

	user.Password = string(hashedPassword)
	if err := DB.Save(&user).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to update password", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Password reset successfully", nil)
}

// ResendDriverVerificationCode resends the email verification code for driver sign-up.
// This is useful when the code expires or the driver didn't receive it.
func ResendDriverVerificationCode(c *gin.Context) {
	var req ResendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Check if user exists and refresh from database to get latest state
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Email does not exist", nil)
		return
	}

	// Refresh user from database to ensure we have the latest IsActive status
	DB.First(&user, user.ID)

	// Check if user is a driver
	if user.Role != RoleDriver {
		ResponseJSON(c, http.StatusBadRequest, "This endpoint is only for driver verification", nil)
		return
	}

	// Check if user is already verified (active)
	// If IsActive is true, the email has been verified
	if user.IsActive {
		ResponseJSON(c, http.StatusBadRequest, "Email is already verified", nil)
		return
	}

	// If user is not active, they can resend the verification code
	// This handles cases where the user hasn't verified yet or needs a new code

	// Generate new verification code
	code := GenerateVerificationCode()
	expiresAt := time.Now().Add(15 * time.Minute) // Code expires in 15 minutes

	// Invalidate any existing unused verification codes for this email
	DB.Model(&VerificationCode{}).Where("email = ? AND used = ?", req.Email, false).Update("used", true)

	// Create new verification code record
	verificationCode := VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := DB.Create(&verificationCode).Error; err != nil {
		log.Printf("Failed to create verification code: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to generate verification code", nil)
		return
	}

	// Send verification code via email
	emailService := NewEmailService()
	if err := emailService.SendVerificationCode(req.Email, code); err != nil {
		log.Printf("Failed to send verification email: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to send verification email", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Verification code has been resent to your email", nil)
}

// ResendForgotPasswordCode resends the password reset verification code.
// This is useful when the code expires or the user didn't receive it.
func ResendForgotPasswordCode(c *gin.Context) {
	var req ResendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Check if user exists and refresh from database to get latest state
	var user User
	if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Email does not exist", nil)
		return
	}

	// Refresh user from database to ensure we have the latest IsActive status
	DB.First(&user, user.ID)

	// Check if user is active
	if !user.IsActive {
		ResponseJSON(c, http.StatusForbidden, "Account is deactivated", nil)
		return
	}

	// Generate new verification code
	code := GenerateVerificationCode()
	expiresAt := time.Now().Add(15 * time.Minute) // Code expires in 15 minutes

	// Invalidate any existing unused verification codes for this email
	DB.Model(&VerificationCode{}).Where("email = ? AND used = ?", req.Email, false).Update("used", true)

	// Create new verification code record
	verificationCode := VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := DB.Create(&verificationCode).Error; err != nil {
		log.Printf("Failed to create verification code: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to generate verification code", nil)
		return
	}

	// Send verification code via email
	emailService := NewEmailService()
	if err := emailService.SendPasswordResetCode(req.Email, code); err != nil {
		log.Printf("Failed to send password reset email: %v", err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to send password reset email", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Password reset code has been resent to your email", nil)
}

var DB *gorm.DB

// InitDB initializes the database connection using environment variables.
// It loads the database configuration from a .env file and migrates the User schema.
func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables:", err)
	}

	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Migrate the schema
	if err := DB.AutoMigrate(&User{}, &VerificationCode{}); err != nil {
		log.Fatal("Failed to migrate schema:", err)
	}

	// Create default super admin if it doesn't exist
	createDefaultSuperAdmin()
}

// GetProfile retrieves the current user's profile information.
// It requires authentication via JWT middleware.
func GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ResponseJSON(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "User not found", nil)
		return
	}

	user.Password = ""
	ResponseJSON(c, http.StatusOK, "Profile retrieved successfully", user)
}

// UpdateKYCStatus updates the KYC status for a driver.
// Drivers can set their own status to "in_progress" when submitting documents.
// Admins can update any driver's status to "approved" or "rejected" after review.
func UpdateKYCStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		ResponseJSON(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	role, exists := c.Get("role")
	if !exists {
		ResponseJSON(c, http.StatusUnauthorized, "User role not found", nil)
		return
	}

	var req UpdateKYCStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	// Get the current user
	var currentUser User
	if err := DB.First(&currentUser, userID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "User not found", nil)
		return
	}

	// Check if trying to update own KYC or another user's KYC
	targetUserID := userID.(uint)
	if targetIDParam := c.Param("id"); targetIDParam != "" {
		// Admin trying to update another user's KYC
		userRole := UserRole(role.(string))
		if userRole != RoleAdmin && userRole != RoleSuperAdmin {
			ResponseJSON(c, http.StatusForbidden, "Only admins can update other users' KYC status", nil)
			return
		}
		// Parse target user ID from param
		var targetID uint
		if _, err := fmt.Sscanf(targetIDParam, "%d", &targetID); err != nil {
			ResponseJSON(c, http.StatusBadRequest, "Invalid user ID", nil)
			return
		}
		targetUserID = targetID
	}

	// Get the target user
	var targetUser User
	if err := DB.First(&targetUser, targetUserID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Target user not found", nil)
		return
	}

	// Verify target user is a driver
	if targetUser.Role != RoleDriver {
		ResponseJSON(c, http.StatusBadRequest, "KYC status can only be updated for drivers", nil)
		return
	}

	// Check permissions based on role
	userRole := UserRole(role.(string))
	
	// Drivers can only set their own status to "in_progress"
	if userRole == RoleDriver {
		if targetUserID != userID.(uint) {
			ResponseJSON(c, http.StatusForbidden, "You can only update your own KYC status", nil)
			return
		}
		if req.Status != KYCStatusInProgress {
			ResponseJSON(c, http.StatusForbidden, "Drivers can only set KYC status to 'in_progress'", nil)
			return
		}
	}

	// Admins can set status to approved or rejected
	if userRole == RoleAdmin || userRole == RoleSuperAdmin {
		if req.Status != KYCStatusApproved && req.Status != KYCStatusRejected {
			ResponseJSON(c, http.StatusBadRequest, "Admins can only set KYC status to 'approved' or 'rejected'", nil)
			return
		}
	}

	// Update KYC status
	targetUser.KYCStatus = req.Status
	if err := DB.Save(&targetUser).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to update KYC status", nil)
		return
	}

	targetUser.Password = ""
	ResponseJSON(c, http.StatusOK, "KYC status updated successfully", gin.H{
		"user":       targetUser,
		"kyc_status": targetUser.KYCStatus,
	})
}

// createDefaultSuperAdmin creates a default super admin user if none exists.
// This is called during database initialization.
func createDefaultSuperAdmin() {
	var count int64
	DB.Model(&User{}).Where("role = ?", RoleSuperAdmin).Count(&count)

	if count == 0 {
		// Get default credentials from environment or use defaults
		email := os.Getenv("SUPER_ADMIN_EMAIL")
		if email == "" {
			email = "admin@hauler.com"
		}
		password := os.Getenv("SUPER_ADMIN_PASSWORD")
		if password == "" {
			password = "Admin@123"
		}

		superAdmin := User{
			Email:     email,
			Password:  password, // Will be hashed by BeforeCreate hook
			FirstName: "Super",
			LastName:  "Admin",
			Phone:     "+1234567890",
			Role:      RoleSuperAdmin,
			IsActive:  true,
		}

		if err := DB.Create(&superAdmin).Error; err != nil {
			log.Printf("Warning: Failed to create default super admin: %v", err)
		} else {
			log.Printf("Default super admin created with email: %s", email)
		}
	}
}
