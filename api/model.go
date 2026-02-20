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

// KYCStatus represents the KYC verification status
type KYCStatus string

const (
	KYCStatusPending    KYCStatus = "pending"     // Email verified but KYC not started
	KYCStatusInProgress KYCStatus = "in_progress" // KYC documents uploaded, under review
	KYCStatusApproved   KYCStatus = "approved"    // KYC approved, registration complete
	KYCStatusRejected   KYCStatus = "rejected"    // KYC rejected
)

// VerificationCodePurpose represents the purpose of a verification code
type VerificationCodePurpose string

const (
	PurposeEmailVerification VerificationCodePurpose = "email_verification"
	PurposePasswordReset     VerificationCodePurpose = "password_reset"
	PurposeLoginOTP          VerificationCodePurpose = "login_otp"
	PurposeChangePassword    VerificationCodePurpose = "change_password"
)

// User represents a user in the haulage system
type User struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	Email               string         `json:"email" gorm:"uniqueIndex;not null" binding:"required,email"`
	Password            string         `json:"-" gorm:"not null" binding:"required,min=6"`
	FirstName           string         `json:"first_name" binding:"required,min=1,max=100"`
	LastName            string         `json:"last_name" binding:"required,min=1,max=100"`
	Phone               string         `json:"phone" binding:"required,min=10,max=20"`
	Role                UserRole       `json:"role" gorm:"type:varchar(20);not null;default:'customer'"`
	IsActive            bool           `json:"is_active" gorm:"default:true"`
	KYCStatus           KYCStatus      `json:"kyc_status" gorm:"type:varchar(20);default:null"` // Only for drivers
	CountryID           *uint          `json:"country_id"`
	GenderID            *uint          `json:"gender_id"`
	MustChangePassword  bool           `json:"must_change_password" gorm:"default:false"` // For admins created by super admin
	TokenVersion        uint           `json:"-" gorm:"default:0"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Country *Country `json:"country,omitempty" gorm:"foreignKey:CountryID"`
	Gender  *Gender  `json:"gender,omitempty" gorm:"foreignKey:GenderID"`
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
	ID        uint                    `json:"id" gorm:"primaryKey"`
	Email     string                  `json:"email" gorm:"not null;index"`
	Code      string                  `json:"code" gorm:"not null;index"`
	Purpose   VerificationCodePurpose `json:"purpose" gorm:"type:varchar(30);not null;default:'email_verification'"`
	ExpiresAt time.Time               `json:"expires_at" gorm:"not null"`
	Used      bool                    `json:"used" gorm:"default:false"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
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

// ResendCodeRequest represents a resend verification code request payload
type ResendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// UpdateKYCStatusRequest represents a KYC status update request payload
type UpdateKYCStatusRequest struct {
	Status KYCStatus `json:"status" binding:"required,oneof=pending in_progress approved rejected"`
}

// VerifyLoginOTPRequest represents a login OTP verification request payload
type VerifyLoginOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,min=4,max=10"`
}

// CreateAdminRequest represents a request to create an admin user
type CreateAdminRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required,min=1,max=100"`
	LastName  string `json:"last_name" binding:"required,min=1,max=100"`
	Phone     string `json:"phone" binding:"required,min=10,max=20"`
	CountryID uint   `json:"country_id" binding:"required"`
	GenderID  uint   `json:"gender_id" binding:"required"`
}

// ChangePasswordRequest represents a change password request payload
type ChangePasswordRequest struct {
	Code        string `json:"code" binding:"required,min=4,max=10"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateAdminRequest represents a request to update an admin user
type UpdateAdminRequest struct {
	Email     string `json:"email" binding:"omitempty,email"`
	FirstName string `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName  string `json:"last_name" binding:"omitempty,min=1,max=100"`
	Phone     string `json:"phone" binding:"omitempty,min=10,max=20"`
	CountryID *uint  `json:"country_id" binding:"omitempty"`
	GenderID  *uint  `json:"gender_id" binding:"omitempty"`
}

// SuspendDriverRequest represents a request to suspend/activate a driver
type SuspendDriverRequest struct {
	IsActive bool `json:"is_active" binding:"required"`
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

// ReviewDocumentRequest represents a request to review a KYC document
type ReviewDocumentRequest struct {
	DocumentType     string                     `json:"document_type" binding:"required,oneof=selfie license_front license_back vehicle_photo vehicle_registration"`
	Status           DocumentVerificationStatus `json:"status" binding:"required,oneof=approved rejected"`
	RejectionReason  string                     `json:"rejection_reason" binding:"required_if=Status rejected"`
}

// Country represents a country in the system
type Country struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	Code      string    `json:"code" gorm:"uniqueIndex;not null;type:varchar(10)"`
	States    []State   `json:"states,omitempty" gorm:"foreignKey:CountryID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// State represents a state/region within a country
type State struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CountryID uint      `json:"country_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Code      string    `json:"code,omitempty" gorm:"type:varchar(10)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCountryRequest represents a request to create a country
type CreateCountryRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
	Code string `json:"code" binding:"required,min=2,max=10"`
}

// UpdateCountryRequest represents a request to update a country
type UpdateCountryRequest struct {
	Name string `json:"name" binding:"omitempty,min=2,max=100"`
	Code string `json:"code" binding:"omitempty,min=2,max=10"`
}

// CreateStateRequest represents a request to create a state
type CreateStateRequest struct {
	CountryID uint   `json:"country_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=2,max=100"`
	Code      string `json:"code" binding:"omitempty,max=10"`
}

// UpdateStateRequest represents a request to update a state
type UpdateStateRequest struct {
	Name string `json:"name" binding:"omitempty,min=2,max=100"`
	Code string `json:"code" binding:"omitempty,max=10"`
}

// Gender represents a gender option in the system
type Gender struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DocumentVerificationStatus represents the verification status of a document
type DocumentVerificationStatus string

const (
	DocStatusPending  DocumentVerificationStatus = "pending"   // Uploaded, awaiting review
	DocStatusApproved DocumentVerificationStatus = "approved"  // Verified and approved
	DocStatusRejected DocumentVerificationStatus = "rejected"  // Rejected, needs reupload
)

// KYCStepStatus represents the overall status of a KYC step
type KYCStepStatus string

const (
	StepStatusNotStarted KYCStepStatus = "not_started" // Step not started
	StepStatusPending    KYCStepStatus = "pending"     // Documents uploaded, under review
	StepStatusApproved   KYCStepStatus = "approved"    // Step approved
	StepStatusRejected   KYCStepStatus = "rejected"    // Step rejected, needs correction
)

// DriverProfile stores KYC data for drivers across all steps
type DriverProfile struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	UserID      uint `json:"user_id" gorm:"uniqueIndex;not null"`
	CurrentStep uint `json:"current_step" gorm:"default:0"`

	// Step 1: Personal Information
	FullName      string     `json:"full_name"`
	PhoneNumber   string     `json:"phone_number"`
	Email         string     `json:"email"`
	CountryID     *uint      `json:"country_id"`
	StateID       *uint      `json:"state_id"`
	GenderID      *uint      `json:"gender_id"`
	HouseAddress  string     `json:"house_address"`
	OfficeAddress string     `json:"office_address"`
	DateOfBirth   *time.Time `json:"date_of_birth"`
	Step1Status   KYCStepStatus `json:"step1_status" gorm:"type:varchar(20);default:'not_started'"`
	Step1RejectionReason string `json:"step1_rejection_reason,omitempty"`

	// Step 2: Selfie
	SelfieURL            string                     `json:"selfie_url"`
	SelfieStatus         DocumentVerificationStatus `json:"selfie_status" gorm:"type:varchar(20);default:'pending'"`
	SelfieRejectionReason string                    `json:"selfie_rejection_reason,omitempty"`
	Step2Status          KYCStepStatus              `json:"step2_status" gorm:"type:varchar(20);default:'not_started'"`

	// Step 3: Vehicle Documentation & Driver License
	LicenseFrontURL              string                     `json:"license_front_url"`
	LicenseFrontStatus           DocumentVerificationStatus `json:"license_front_status" gorm:"type:varchar(20);default:'pending'"`
	LicenseFrontRejectionReason  string                     `json:"license_front_rejection_reason,omitempty"`
	
	LicenseBackURL               string                     `json:"license_back_url"`
	LicenseBackStatus            DocumentVerificationStatus `json:"license_back_status" gorm:"type:varchar(20);default:'pending'"`
	LicenseBackRejectionReason   string                     `json:"license_back_rejection_reason,omitempty"`
	
	VehiclePhotoURL              string                     `json:"vehicle_photo_url"`
	VehiclePhotoStatus           DocumentVerificationStatus `json:"vehicle_photo_status" gorm:"type:varchar(20);default:'pending'"`
	VehiclePhotoRejectionReason  string                     `json:"vehicle_photo_rejection_reason,omitempty"`
	
	VehicleRegistrationURL       string                     `json:"vehicle_registration_url"`
	VehicleRegistrationStatus    DocumentVerificationStatus `json:"vehicle_registration_status" gorm:"type:varchar(20);default:'pending'"`
	VehicleRegistrationRejectionReason string                `json:"vehicle_registration_rejection_reason,omitempty"`
	
	Step3Status                  KYCStepStatus              `json:"step3_status" gorm:"type:varchar(20);default:'not_started'"`

	// Admin review tracking
	ReviewedBy   *uint      `json:"reviewed_by,omitempty"` // Admin user ID who last reviewed
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`

	// Relationships
	Country    *Country `json:"country,omitempty" gorm:"foreignKey:CountryID"`
	State      *State   `json:"state,omitempty" gorm:"foreignKey:StateID"`
	Gender     *Gender  `json:"gender,omitempty" gorm:"foreignKey:GenderID"`
	ReviewedByUser *User `json:"reviewed_by_user,omitempty" gorm:"foreignKey:ReviewedBy"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// KYCStep1Request represents the request payload for KYC step 1
type KYCStep1Request struct {
	FullName      string `json:"fullName" binding:"required,min=2,max=200"`
	PhoneNumber   string `json:"phoneNumber" binding:"required,min=10,max=20"`
	Email         string `json:"email" binding:"required,email"`
	CountryID     uint   `json:"countryId" binding:"required"`
	StateID       uint   `json:"stateId" binding:"required"`
	GenderID      uint   `json:"genderId" binding:"required"`
	HouseAddress  string `json:"houseAddress" binding:"required,min=5,max=500"`
	OfficeAddress string `json:"officeAddress" binding:"required,min=5,max=500"`
	DateOfBirth   string `json:"dateOfBirth" binding:"required"`
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
