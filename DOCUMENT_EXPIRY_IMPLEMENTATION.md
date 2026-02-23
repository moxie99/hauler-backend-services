# Document Expiry Date Implementation

## Overview
Implemented expiry date tracking for KYC documents to enable automated notifications and driver eligibility checks.

## Changes Made

### 1. Model Updates (`api/model.go`)

#### Added Expiry Date Fields to DriverProfile:
- `LicenseFrontExpiryDate` - Driver's license front expiry
- `LicenseBackExpiryDate` - Driver's license back expiry  
- `VehicleRegistrationExpiryDate` - Vehicle registration expiry
- `InsuranceDocumentExpiryDate` - Insurance document expiry
- `RoadworthinessDocumentExpiryDate` - Roadworthiness certificate expiry

All fields are `*time.Time` (nullable) and marked as `omitempty` in JSON.

#### Updated ReviewDocumentRequest:
Added optional `ExpiryDate` field (string format: YYYY-MM-DD) for admins to provide when approving documents.

```go
type ReviewDocumentRequest struct {
    DocumentType     string                     `json:"document_type"`
    Status           DocumentVerificationStatus `json:"status"`
    RejectionReason  string                     `json:"rejection_reason"`
    ExpiryDate       string                     `json:"expiry_date,omitempty"` // NEW
}
```

### 2. Handler Updates (`api/handlers.go`)

#### ReviewDocument Handler:
- Parses `expiry_date` from request (format: YYYY-MM-DD)
- Validates date format
- Saves expiry date when approving documents:
  - `license_front` → `LicenseFrontExpiryDate`
  - `license_back` → `LicenseBackExpiryDate`
  - `vehicle_registration` → `VehicleRegistrationExpiryDate`
  - `insurance_document` → `InsuranceDocumentExpiryDate`
  - `roadworthiness_document` → `RoadworthinessDocumentExpiryDate`

## API Usage

### Review Document with Expiry Date

**Endpoint:** `POST /api/admin/drivers/:id/review-document`

**Request Body (Approve with Expiry):**
```json
{
  "document_type": "license_front",
  "status": "approved",
  "expiry_date": "2027-12-31"
}
```

**Request Body (Approve Insurance):**
```json
{
  "document_type": "insurance_document",
  "status": "approved",
  "expiry_date": "2026-06-30"
}
```

**Request Body (Reject):**
```json
{
  "document_type": "roadworthiness_document",
  "status": "rejected",
  "rejection_reason": "Document is not clear"
}
```

## Documents with Expiry Tracking

1. **Driver's License (Front)** - `license_front`
2. **Driver's License (Back)** - `license_back`
3. **Vehicle Registration** - `vehicle_registration`
4. **Insurance Document** - `insurance_document`
5. **Roadworthiness Certificate** - `roadworthiness_document`

Note: `selfie` and `vehicle_photo` do NOT have expiry dates.

## Response Format

When fetching driver details, expiry dates are included:

```json
{
  "status": 200,
  "message": "Driver retrieved successfully",
  "data": {
    "user": { ... },
    "kyc_profile": {
      "license_front_url": "...",
      "license_front_status": "approved",
      "license_front_expiry_date": "2027-12-31T00:00:00Z",
      "insurance_document_url": "...",
      "insurance_document_status": "approved",
      "insurance_document_expiry_date": "2026-06-30T00:00:00Z",
      ...
    }
  }
}
```

## Next Steps (Future Implementation)

### 1. Expiry Notification System
Create a background job (cron/scheduler) to:
- Check documents expiring within 7 days
- Send SMS notifications to drivers
- Send push notifications via mobile app
- Optionally send email reminders

### 2. Driver Eligibility Check
Create endpoint/middleware to:
- Check if driver has any expired documents
- Block order assignment to drivers with expired documents
- Return eligibility status with reasons

### 3. Frontend Integration
- Display expiry dates in driver profile
- Show warning badges for documents expiring soon
- Highlight expired documents in red
- Allow drivers to upload renewed documents

## Database Migration

The new fields are automatically added to the `driver_profiles` table when the server starts (GORM AutoMigrate).

## Testing

Test the implementation:

1. **Approve document with expiry:**
```bash
curl -X POST http://localhost:8080/api/admin/drivers/1/review-document \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "document_type": "license_front",
    "status": "approved",
    "expiry_date": "2027-12-31"
  }'
```

2. **Get driver details to verify expiry date saved:**
```bash
curl -X GET http://localhost:8080/api/admin/drivers/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

3. **Approve without expiry (optional):**
```bash
curl -X POST http://localhost:8080/api/admin/drivers/1/review-document \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "document_type": "vehicle_photo",
    "status": "approved"
  }'
```
