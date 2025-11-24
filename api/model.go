package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleDriver     UserRole = "driver"
	RoleCustomer   UserRole = "customer"
)

// User represents a user in the haulage system
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null" binding:"required,email"`
	Password  string         `json:"-" gorm:"not null" binding:"required,min=6"`
	FirstName string         `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string         `json:"last_name" binding:"required,min=1,max=100"`
	Phone     string         `json:"phone" binding:"required,min=10,max=20"`
	Role      UserRole       `json:"role" gorm:"type:varchar(20);not null;default:'customer'"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate is a GORM hook that hashes the password before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares the provided password with the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// JsonResponse represents a standardized API response
type JsonResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// LoginRequest represents a login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents a user registration request payload
type RegisterRequest struct {
	Email     string   `json:"email" binding:"required,email"`
	Password  string   `json:"password" binding:"required,min=6"`
	FirstName string   `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string   `json:"last_name" binding:"required,min=1,max=100"`
	Phone     string   `json:"phone" binding:"required,min=10,max=20"`
	Role      UserRole `json:"role" binding:"omitempty,oneof=super_admin admin driver customer"`
}

// VerificationCode represents an email verification code
type VerificationCode struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"not null;index"`
	Code      string    `json:"code" gorm:"not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DriverRegisterRequest represents a driver registration request payload
type DriverRegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string `json:"last_name" binding:"required,min=1,max=100"`
	Phone     string `json:"phone" binding:"required,min=10,max=20"`
}

// VerifyEmailRequest represents an email verification request payload
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,min=4,max=10"`
}

// ForgotPasswordRequest represents a forgot password request payload
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyForgotPasswordRequest represents a forgot password verification request payload
type VerifyForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,min=4,max=10"`
}

// ResetPasswordRequest represents a password reset request payload
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,min=4,max=10"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ResponseJSON sends a standardized JSON response
func ResponseJSON(c *gin.Context, status int, message string, data any) {
	response := JsonResponse{
		Status:  status,
		Message: message,
		Data:    data,
	}

	c.JSON(status, response)
}
