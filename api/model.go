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
	ID                 uint           `json:"id" gorm:"primaryKey"`
	Email              string         `json:"email" gorm:"uniqueIndex;not null" binding:"required,email"`
	Password           string         `json:"-" gorm:"not null" binding:"required,min=6"`
	FirstName          string         `json:"first_name" binding:"required,min=1,max=100"`
	LastName           string         `json:"last_name" binding:"required,min=1,max=100"`
	Phone              string         `json:"phone" binding:"required,min=10,max=20"`
	Role               UserRole       `json:"role" gorm:"type:varchar(20);not null;default:'customer'"`
	IsActive           bool           `json:"is_active" gorm:"default:true"`
	EmailVerified      bool           `json:"email_verified" gorm:"default:false"`
	KYCStatus          KYCStatus      `json:"kyc_status" gorm:"type:varchar(20);default:null"` // Only for drivers
	CountryID          *uint          `json:"country_id"`
	GenderID           *uint          `json:"gender_id"`
	MustChangePassword bool           `json:"must_change_password" gorm:"default:false"` // For admins created by super admin
	TokenVersion       uint           `json:"-" gorm:"default:0"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`

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
	DocumentType    string                     `json:"document_type" binding:"required,oneof=selfie license_front license_back vehicle_photo vehicle_registration insurance_document roadworthiness_document"`
	Status          DocumentVerificationStatus `json:"status" binding:"required,oneof=approved rejected"`
	RejectionReason string                     `json:"rejection_reason" binding:"required_if=Status rejected"`
	ExpiryDate      string                     `json:"expiry_date,omitempty"` // Optional: Required when approving documents with expiry (license, vehicle_registration, insurance, roadworthiness)
}

// Category represents a vehicle category in the system
type Category struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Code        string    `json:"code" gorm:"uniqueIndex;not null;type:varchar(50)"` // e.g., "motorcycle", "van"
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LoadType represents a type of load/cargo (fragile, liquid, perishable, hazardous, oversized)
type LoadType struct {
	ID                      uint      `json:"id" gorm:"primaryKey"`
	Name                    string    `json:"name" gorm:"uniqueIndex;not null"` // e.g., "Fragile", "Liquid", "Perishable"
	Description             string    `json:"description"`
	RequiresSpecialHandling bool      `json:"requires_special_handling" gorm:"default:false"`
	IsActive                bool      `json:"is_active" gorm:"default:true"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// DriverLoadType represents the many-to-many relationship between drivers and load types
type DriverLoadType struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	DriverID   uint      `json:"driver_id" gorm:"not null;index"`
	LoadTypeID uint      `json:"load_type_id" gorm:"not null;index"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateCategoryRequest represents a request to create a category
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Code        string `json:"code" binding:"required,min=2,max=50"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

// UpdateCategoryRequest represents a request to update a category
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=100"`
	Code        string `json:"code" binding:"omitempty,min=2,max=50"`
	Description string `json:"description" binding:"omitempty,max=500"`
	IsActive    *bool  `json:"is_active"`
}

// CreateLoadTypeRequest represents a request to create a load type
type CreateLoadTypeRequest struct {
	Name                    string `json:"name" binding:"required,min=2,max=100"`
	Description             string `json:"description" binding:"omitempty,max=500"`
	RequiresSpecialHandling bool   `json:"requires_special_handling"`
}

// UpdateLoadTypeRequest represents a request to update a load type
type UpdateLoadTypeRequest struct {
	Name                    string `json:"name" binding:"omitempty,min=2,max=100"`
	Description             string `json:"description" binding:"omitempty,max=500"`
	RequiresSpecialHandling *bool  `json:"requires_special_handling"`
	IsActive                *bool  `json:"is_active"`
}

// VehicleCategory represents the category of a vehicle
type VehicleCategory string

const (
	CategoryMotorcycle   VehicleCategory = "motorcycle"
	CategoryLCV          VehicleCategory = "lcv" // Light Commercial Vehicle
	CategoryMediumTruck  VehicleCategory = "medium_truck"
	CategoryHeavyTruck   VehicleCategory = "heavy_truck"
	CategoryVan          VehicleCategory = "van"
	CategoryPickup       VehicleCategory = "pickup"
	CategoryFlatbed      VehicleCategory = "flatbed"
	CategoryRefrigerated VehicleCategory = "refrigerated"
	CategoryTanker       VehicleCategory = "tanker"
)

// VehicleType represents a type of vehicle available for hauling
type VehicleType struct {
	ID                      uint      `json:"id" gorm:"primaryKey"`
	Name                    string    `json:"name" gorm:"not null;uniqueIndex"`
	CategoryID              uint      `json:"category_id" gorm:"not null"`
	Description             string    `json:"description"`
	ImageURL                string    `json:"image_url"`
	MaxPayloadKg            float64   `json:"max_payload_kg"`
	CargoLengthM            float64   `json:"cargo_length_m"`
	CargoWidthM             float64   `json:"cargo_width_m"`
	CargoHeightM            float64   `json:"cargo_height_m"`
	CargoVolumeM3           float64   `json:"cargo_volume_m3"`
	IsTemperatureControlled bool      `json:"is_temperature_controlled" gorm:"default:false"`
	IsEnclosed              bool      `json:"is_enclosed" gorm:"default:true"`
	HasTailLift             bool      `json:"has_tail_lift" gorm:"default:false"`
	HasCrane                bool      `json:"has_crane" gorm:"default:false"`
	RequiresSpecialLicense  bool      `json:"requires_special_license" gorm:"default:false"`
	IsActive                bool      `json:"is_active" gorm:"default:true"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	// Relationships
	Category *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

// CreateVehicleTypeRequest represents a request to create a vehicle type
type CreateVehicleTypeRequest struct {
	Name                    string  `json:"name" binding:"required,min=2,max=100"`
	CategoryID              uint    `json:"category_id" binding:"required"`
	Description             string  `json:"description" binding:"omitempty,max=500"`
	ImageURL                string  `json:"image_url" binding:"omitempty,url"`
	MaxPayloadKg            float64 `json:"max_payload_kg" binding:"required,min=0"`
	CargoLengthM            float64 `json:"cargo_length_m" binding:"omitempty,min=0"`
	CargoWidthM             float64 `json:"cargo_width_m" binding:"omitempty,min=0"`
	CargoHeightM            float64 `json:"cargo_height_m" binding:"omitempty,min=0"`
	CargoVolumeM3           float64 `json:"cargo_volume_m3" binding:"omitempty,min=0"`
	IsTemperatureControlled bool    `json:"is_temperature_controlled"`
	IsEnclosed              bool    `json:"is_enclosed"`
	HasTailLift             bool    `json:"has_tail_lift"`
	HasCrane                bool    `json:"has_crane"`
	RequiresSpecialLicense  bool    `json:"requires_special_license"`
}

// UpdateVehicleTypeRequest represents a request to update a vehicle type
type UpdateVehicleTypeRequest struct {
	Name                    string   `json:"name" binding:"omitempty,min=2,max=100"`
	CategoryID              *uint    `json:"category_id" binding:"omitempty"`
	Description             string   `json:"description" binding:"omitempty,max=500"`
	ImageURL                string   `json:"image_url" binding:"omitempty,url"`
	MaxPayloadKg            *float64 `json:"max_payload_kg" binding:"omitempty,min=0"`
	CargoLengthM            *float64 `json:"cargo_length_m" binding:"omitempty,min=0"`
	CargoWidthM             *float64 `json:"cargo_width_m" binding:"omitempty,min=0"`
	CargoHeightM            *float64 `json:"cargo_height_m" binding:"omitempty,min=0"`
	CargoVolumeM3           *float64 `json:"cargo_volume_m3" binding:"omitempty,min=0"`
	IsTemperatureControlled *bool    `json:"is_temperature_controlled"`
	IsEnclosed              *bool    `json:"is_enclosed"`
	HasTailLift             *bool    `json:"has_tail_lift"`
	HasCrane                *bool    `json:"has_crane"`
	RequiresSpecialLicense  *bool    `json:"requires_special_license"`
	IsActive                *bool    `json:"is_active"`
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
	DocStatusPending  DocumentVerificationStatus = "pending"  // Uploaded, awaiting review
	DocStatusApproved DocumentVerificationStatus = "approved" // Verified and approved
	DocStatusRejected DocumentVerificationStatus = "rejected" // Rejected, needs reupload
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
	FullName             string        `json:"full_name"`
	PhoneNumber          string        `json:"phone_number"`
	Email                string        `json:"email"`
	CountryID            *uint         `json:"country_id"`
	StateID              *uint         `json:"state_id"`
	GenderID             *uint         `json:"gender_id"`
	HouseAddress         string        `json:"house_address"`
	OfficeAddress        string        `json:"office_address"`
	DateOfBirth          *time.Time    `json:"date_of_birth"`
	Step1Status          KYCStepStatus `json:"step1_status" gorm:"type:varchar(20);default:'not_started'"`
	Step1RejectionReason string        `json:"step1_rejection_reason,omitempty"`

	// Step 2: Selfie
	SelfieURL             string                     `json:"selfie_url"`
	SelfieStatus          DocumentVerificationStatus `json:"selfie_status" gorm:"type:varchar(20);default:'pending'"`
	SelfieRejectionReason string                     `json:"selfie_rejection_reason,omitempty"`
	Step2Status           KYCStepStatus              `json:"step2_status" gorm:"type:varchar(20);default:'not_started'"`

	// Step 3: Vehicle Documentation & Driver License
	LicenseFrontURL             string                     `json:"license_front_url"`
	LicenseFrontStatus          DocumentVerificationStatus `json:"license_front_status" gorm:"type:varchar(20);default:'pending'"`
	LicenseFrontRejectionReason string                     `json:"license_front_rejection_reason,omitempty"`
	LicenseFrontExpiryDate      *time.Time                 `json:"license_front_expiry_date,omitempty"`

	LicenseBackURL             string                     `json:"license_back_url"`
	LicenseBackStatus          DocumentVerificationStatus `json:"license_back_status" gorm:"type:varchar(20);default:'pending'"`
	LicenseBackRejectionReason string                     `json:"license_back_rejection_reason,omitempty"`
	LicenseBackExpiryDate      *time.Time                 `json:"license_back_expiry_date,omitempty"`

	VehiclePhotoURL             string                     `json:"vehicle_photo_url"`
	VehiclePhotoStatus          DocumentVerificationStatus `json:"vehicle_photo_status" gorm:"type:varchar(20);default:'pending'"`
	VehiclePhotoRejectionReason string                     `json:"vehicle_photo_rejection_reason,omitempty"`

	VehicleRegistrationURL             string                     `json:"vehicle_registration_url"`
	VehicleRegistrationStatus          DocumentVerificationStatus `json:"vehicle_registration_status" gorm:"type:varchar(20);default:'pending'"`
	VehicleRegistrationRejectionReason string                     `json:"vehicle_registration_rejection_reason,omitempty"`
	VehicleRegistrationExpiryDate      *time.Time                 `json:"vehicle_registration_expiry_date,omitempty"`

	Step3Status KYCStepStatus `json:"step3_status" gorm:"type:varchar(20);default:'not_started'"`

	// Step 4: Work Preferences
	DaysOfWork    string        `json:"days_of_work" gorm:"type:jsonb;default:'[]'"` // JSON array: ["Monday", "Tuesday", ...]
	VehicleTypeID *uint         `json:"vehicle_type_id"`                             // Single vehicle type
	WorkStartTime string        `json:"work_start_time"`                             // Format: "09:00"
	WorkEndTime   string        `json:"work_end_time"`                               // Format: "17:00"
	Step4Status   KYCStepStatus `json:"step4_status" gorm:"type:varchar(20);default:'not_started'"`

	// Step 5: Vehicle Details & Documents
	PlateNumber                           string                     `json:"plate_number" gorm:"uniqueIndex"`
	VehicleBrand                          string                     `json:"vehicle_brand"`
	VehicleModel                          string                     `json:"vehicle_model"`
	VehicleYear                           string                     `json:"vehicle_year"`
	VehicleColor                          string                     `json:"vehicle_color"`
	InsuranceDocumentURL                  string                     `json:"insurance_document_url"`
	InsuranceDocumentStatus               DocumentVerificationStatus `json:"insurance_document_status" gorm:"type:varchar(20);default:'pending'"`
	InsuranceDocumentRejectionReason      string                     `json:"insurance_document_rejection_reason,omitempty"`
	InsuranceDocumentExpiryDate           *time.Time                 `json:"insurance_document_expiry_date,omitempty"`
	RoadworthinessDocumentURL             string                     `json:"roadworthiness_document_url"`
	RoadworthinessDocumentStatus          DocumentVerificationStatus `json:"roadworthiness_document_status" gorm:"type:varchar(20);default:'pending'"`
	RoadworthinessDocumentRejectionReason string                     `json:"roadworthiness_document_rejection_reason,omitempty"`
	RoadworthinessDocumentExpiryDate      *time.Time                 `json:"roadworthiness_document_expiry_date,omitempty"`
	Step5Status                           KYCStepStatus              `json:"step5_status" gorm:"type:varchar(20);default:'not_started'"`

	// Admin review tracking
	ReviewedBy *uint      `json:"reviewed_by,omitempty"` // Admin user ID who last reviewed
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`

	// Relationships
	Country        *Country     `json:"country,omitempty" gorm:"foreignKey:CountryID"`
	State          *State       `json:"state,omitempty" gorm:"foreignKey:StateID"`
	Gender         *Gender      `json:"gender,omitempty" gorm:"foreignKey:GenderID"`
	VehicleType    *VehicleType `json:"vehicle_type,omitempty" gorm:"foreignKey:VehicleTypeID"`
	ReviewedByUser *User        `json:"reviewed_by_user,omitempty" gorm:"foreignKey:ReviewedBy"`

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

// KYCStep4Request represents the request payload for KYC step 4
type KYCStep4Request struct {
	DaysOfWork    []string `json:"daysOfWork" binding:"required,min=1"` // ["Monday", "Tuesday", ...]
	VehicleTypeID uint     `json:"vehicleTypeId" binding:"required"`    // Single vehicle type ID
	LoadTypeIDs   []uint   `json:"loadTypeIds"`                         // Optional: [1, 2] or empty for "All"
	WorkStartTime string   `json:"workStartTime" binding:"required"`    // "09:00"
	WorkEndTime   string   `json:"workEndTime" binding:"required"`      // "17:00"
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
