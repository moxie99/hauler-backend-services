# JSON Field Fix - days_of_work Error

## Problem
```
ERROR: invalid input syntax for type json (SQLSTATE 22P02)
```

When submitting KYC Step 1, the `days_of_work` field (JSONB type) was being initialized as an empty string `''` instead of valid JSON.

## Root Cause
PostgreSQL JSONB columns require valid JSON. An empty string is not valid JSON.

## Solution Applied

### 1. Fixed Handler (`api/handlers.go`)
Changed DriverProfile initialization in `SubmitKYCStep1`:

**Before:**
```go
profile = DriverProfile{UserID: userID.(uint)}
```

**After:**
```go
profile = DriverProfile{
    UserID:     userID.(uint),
    DaysOfWork: "[]", // Initialize as empty JSON array
}
```

### 2. Fixed Model (`api/model.go`)
Added default value to the field definition:

**Before:**
```go
DaysOfWork string `json:"days_of_work" gorm:"type:jsonb"`
```

**After:**
```go
DaysOfWork string `json:"days_of_work" gorm:"type:jsonb;default:'[]'"`
```

## How to Apply

**Step 1: Restart Your Server**
```bash
# Stop current server (Ctrl+C)
# Restart:
go run cmd/main.go
```

**Step 2: Test KYC Step 1**
Try submitting KYC Step 1 again - it should work now!

## Why This Happened

JSONB fields in PostgreSQL must contain valid JSON:
- ✅ Valid: `[]`, `{}`, `["Monday"]`, `null`
- ❌ Invalid: `""` (empty string), `undefined`

When creating a new DriverProfile, GORM was setting `days_of_work` to an empty string (Go's default for string type), which PostgreSQL rejected.

## Verification

After restarting, submit KYC Step 1:

```bash
curl -X POST http://localhost:8080/api/driver/kyc/step/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fullName": "Test Driver",
    "phoneNumber": "08012345678",
    "email": "test@example.com",
    "countryId": 1,
    "stateId": 1,
    "genderId": 1,
    "houseAddress": "123 Test St",
    "officeAddress": "456 Office Rd",
    "dateOfBirth": "1990-01-15T00:00:00.000Z"
  }'
```

**Expected:** 200 OK response with profile data

## Summary

✅ **Fixed:** `days_of_work` now initializes as `[]` (empty JSON array)
✅ **Added:** Default value in model definition
✅ **Result:** KYC Step 1 submissions will work correctly

Just restart your server and try again! 🎉
