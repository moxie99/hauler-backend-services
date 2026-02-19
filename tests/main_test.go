package tests

import (
	"bytes"
	"encoding/json"
	"hauler-backend-services/api"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var jwtSecret = func() []byte {
	secret := os.Getenv("SECRET_TOKEN")
	if secret == "" {
		secret = "test-secret-key-for-testing-only"
	}
	return []byte(secret)
}()

// setupTestDB initializes a test database using SQLite.
// NOTE: This requires CGO to be enabled. If you get a CGO error, run:
//
//	set CGO_ENABLED=1
//	go test ./tests/...
//
// Alternatively, you can use a PostgreSQL test database or mock the database for unit tests.
func setupTestDB() {
	var err error
	// Use a temporary file-based SQLite database for testing
	// Clean up any existing test database first
	testDBFile := filepath.Join(os.TempDir(), "hauler_test.db")
	os.Remove(testDBFile) // Ignore errors if file doesn't exist

	// Use GORM's sqlite driver (requires CGO)
	api.DB, err = gorm.Open(sqlite.Open(testDBFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect test database: " + err.Error() +
			"\nNote: SQLite driver requires CGO. Set CGO_ENABLED=1 and ensure you have a C compiler installed.")
	}
	if err := api.DB.AutoMigrate(&api.User{}, &api.VerificationCode{}, &api.Country{}, &api.State{}, &api.Gender{}, &api.DriverProfile{}); err != nil {
		panic("failed to migrate test database: " + err.Error())
	}
}

func createTestUser(email, password, firstName, lastName, phone string, role api.UserRole) api.User {
	user := api.User{
		Email:     email,
		Password:  password, // Will be hashed by BeforeCreate hook
		FirstName: firstName,
		LastName:  lastName,
		Phone:     phone,
		Role:      role,
		IsActive:  true,
	}
	api.DB.Create(&user)
	return user
}

func generateValidToken(userID uint, email string, role api.UserRole) string {
	expirationTime := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":       float64(userID),
		"email":         email,
		"role":          string(role),
		"token_version": float64(0),
		"exp":           expirationTime.Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func generateInvalidToken() string {
	expirationTime := time.Now().Add(-1 * time.Hour) // Expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"email":   "test@example.com",
		"role":    "customer",
		"exp":     expirationTime.Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

// TestRegister tests user registration
func TestRegister(t *testing.T) {
	setupTestDB()
	router := gin.Default()
	router.POST("/api/auth/register", api.Register)

	registerRequest := api.RegisterRequest{
		Email:     "test@example.com",
		Password:  "Password@123",
		FirstName: "John",
		LastName:  "Doe",
		Phone:     "+1234567890",
		Role:      api.RoleCustomer,
	}

	jsonValue, _ := json.Marshal(registerRequest)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "User registered successfully" {
		t.Errorf("Expected message 'User registered successfully', got %s", response.Message)
	}

	if response.Data == nil {
		t.Errorf("Expected user data, got nil")
	}

	// Verify user was created in database
	var user api.User
	if err := api.DB.Where("email = ?", "test@example.com").First(&user).Error; err != nil {
		t.Errorf("Expected user to be created in database, got error: %v", err)
	}
}

// TestRegisterDuplicateEmail tests registration with duplicate email
func TestRegisterDuplicateEmail(t *testing.T) {
	setupTestDB()
	createTestUser("test@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)

	router := gin.Default()
	router.POST("/api/auth/register", api.Register)

	registerRequest := api.RegisterRequest{
		Email:     "test@example.com",
		Password:  "Password@123",
		FirstName: "Jane",
		LastName:  "Doe",
		Phone:     "+1234567891",
		Role:      api.RoleCustomer,
	}

	jsonValue, _ := json.Marshal(registerRequest)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Email already registered" {
		t.Errorf("Expected message 'Email already registered', got %s", response.Message)
	}
}

// TestRegisterDefaultRole tests that default role is set to customer when not provided
func TestRegisterDefaultRole(t *testing.T) {
	setupTestDB()
	router := gin.Default()
	router.POST("/api/auth/register", api.Register)

	registerRequest := api.RegisterRequest{
		Email:     "test2@example.com",
		Password:  "Password@123",
		FirstName: "John",
		LastName:  "Doe",
		Phone:     "+1234567890",
		Role:      "", // Empty role should default to customer
	}

	jsonValue, _ := json.Marshal(registerRequest)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
	}

	// Verify user has customer role
	var user api.User
	api.DB.Where("email = ?", "test2@example.com").First(&user)
	if user.Role != api.RoleCustomer {
		t.Errorf("Expected default role to be customer, got %s", user.Role)
	}
}

// TestLogin tests user login
func TestLogin(t *testing.T) {
	setupTestDB()
	user := createTestUser("test@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)

	router := gin.Default()
	router.POST("/api/auth/login", api.Login)

	loginRequest := api.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	jsonValue, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Login successful" {
		t.Errorf("Expected message 'Login successful', got %s", response.Message)
	}

	// Verify token is present in response
	if response.Data == nil {
		t.Errorf("Expected token and user data, got nil")
	}

	dataMap := response.Data.(map[string]interface{})
	if dataMap["token"] == nil || dataMap["token"].(string) == "" {
		t.Errorf("Expected token in response, got nil or empty")
	}

	if dataMap["user"] == nil {
		t.Errorf("Expected user data in response, got nil")
	}

	// Verify user data
	userData := dataMap["user"].(map[string]interface{})
	if userData["email"] != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, userData["email"])
	}
}

// TestLoginInvalidCredentials tests login with invalid credentials
func TestLoginInvalidCredentials(t *testing.T) {
	setupTestDB()
	createTestUser("test@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)

	router := gin.Default()
	router.POST("/api/auth/login", api.Login)

	loginRequest := api.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	jsonValue, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Invalid email or password" {
		t.Errorf("Expected message 'Invalid email or password', got %s", response.Message)
	}
}

// TestLoginInactiveUser tests login with inactive user
func TestLoginInactiveUser(t *testing.T) {
	setupTestDB()
	user := createTestUser("test@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)
	user.IsActive = false
	api.DB.Save(&user)

	router := gin.Default()
	router.POST("/api/auth/login", api.Login)

	loginRequest := api.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	jsonValue, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Account is deactivated" {
		t.Errorf("Expected message 'Account is deactivated', got %s", response.Message)
	}
}

// TestGetProfile tests getting user profile
func TestGetProfile(t *testing.T) {
	setupTestDB()
	user := createTestUser("test@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)
	token := generateValidToken(user.ID, user.Email, user.Role)

	router := gin.Default()
	protected := router.Group("/api", api.JWTAuthMiddleware())
	protected.GET("/profile", api.GetProfile)

	req, _ := http.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Profile retrieved successfully" {
		t.Errorf("Expected message 'Profile retrieved successfully', got %s", response.Message)
	}

	if response.Data == nil {
		t.Errorf("Expected user data, got nil")
	}

	userData := response.Data.(map[string]interface{})
	if userData["email"] != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, userData["email"])
	}
}

// TestGetProfileWithoutToken tests getting profile without authentication
func TestGetProfileWithoutToken(t *testing.T) {
	setupTestDB()
	router := gin.Default()
	protected := router.Group("/api", api.JWTAuthMiddleware())
	protected.GET("/profile", api.GetProfile)

	req, _ := http.NewRequest("GET", "/api/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, status)
	}

	var response api.JsonResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Authorization token required" {
		t.Errorf("Expected message 'Authorization token required', got %s", response.Message)
	}
}

// TestGetProfileWithInvalidToken tests getting profile with invalid token
func TestGetProfileWithInvalidToken(t *testing.T) {
	setupTestDB()
	router := gin.Default()
	protected := router.Group("/api", api.JWTAuthMiddleware())
	protected.GET("/profile", api.GetProfile)

	req, _ := http.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, status)
	}
}

// TestGetProfileWithExpiredToken tests getting profile with expired token
func TestGetProfileWithExpiredToken(t *testing.T) {
	setupTestDB()
	router := gin.Default()
	protected := router.Group("/api", api.JWTAuthMiddleware())
	protected.GET("/profile", api.GetProfile)

	expiredToken := generateInvalidToken()
	req, _ := http.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, status)
	}
}

// TestRequireAdminMiddleware tests admin middleware
func TestRequireAdminMiddleware(t *testing.T) {
	setupTestDB()
	router := gin.Default()

	// Create a test endpoint that requires admin
	protected := router.Group("/api/admin", api.JWTAuthMiddleware(), api.RequireAdmin())
	protected.GET("/test", func(c *gin.Context) {
		api.ResponseJSON(c, http.StatusOK, "Access granted", nil)
	})

	// Test with customer role (should fail)
	customer := createTestUser("customer@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)
	customerToken := generateValidToken(customer.ID, customer.Email, customer.Role)

	req, _ := http.NewRequest("GET", "/api/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+customerToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, status)
	}

	// Test with admin role (should succeed)
	admin := createTestUser("admin@example.com", "password123", "Admin", "User", "+1234567890", api.RoleAdmin)
	adminToken := generateValidToken(admin.ID, admin.Email, admin.Role)

	req, _ = http.NewRequest("GET", "/api/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	// Test with super admin role (should succeed)
	superAdmin := createTestUser("superadmin@example.com", "password123", "Super", "Admin", "+1234567890", api.RoleSuperAdmin)
	superAdminToken := generateValidToken(superAdmin.ID, superAdmin.Email, superAdmin.Role)

	req, _ = http.NewRequest("GET", "/api/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}
}

// TestRequireSuperAdminMiddleware tests super admin middleware
func TestRequireSuperAdminMiddleware(t *testing.T) {
	setupTestDB()
	router := gin.Default()

	// Create a test endpoint that requires super admin
	protected := router.Group("/api/super-admin", api.JWTAuthMiddleware(), api.RequireSuperAdmin())
	protected.GET("/test", func(c *gin.Context) {
		api.ResponseJSON(c, http.StatusOK, "Access granted", nil)
	})

	// Test with customer role (should fail)
	customer := createTestUser("customer@example.com", "password123", "John", "Doe", "+1234567890", api.RoleCustomer)
	customerToken := generateValidToken(customer.ID, customer.Email, customer.Role)

	req, _ := http.NewRequest("GET", "/api/super-admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+customerToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, status)
	}

	// Test with admin role (should fail)
	admin := createTestUser("admin@example.com", "password123", "Admin", "User", "+1234567890", api.RoleAdmin)
	adminToken := generateValidToken(admin.ID, admin.Email, admin.Role)

	req, _ = http.NewRequest("GET", "/api/super-admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, status)
	}

	// Test with super admin role (should succeed)
	superAdmin := createTestUser("superadmin@example.com", "password123", "Super", "Admin", "+1234567890", api.RoleSuperAdmin)
	superAdminToken := generateValidToken(superAdmin.ID, superAdmin.Email, superAdmin.Role)

	req, _ = http.NewRequest("GET", "/api/super-admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}
}
