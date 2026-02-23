# Pre-Signed URLs Implementation Guide

## Overview
Implemented secure pre-signed URLs for all driver KYC documents instead of public S3 access. This ensures sensitive driver documents (licenses, vehicle registration, insurance, etc.) remain private in S3 while still being accessible through temporary URLs.

## What Changed

### 1. S3 Upload Function (`api/s3.go`)
**Before:** Files were uploaded with `public-read` ACL
**After:** Files are uploaded as private and S3 keys are returned

```go
// Now returns S3 key instead of full URL
func UploadFileToS3(file *multipart.FileHeader, folder string) (string, error) {
    // ... upload logic ...
    // Returns: "kyc/insurance/uuid.jpg" instead of full URL
    return key, nil
}
```

### 2. Pre-Signed URL Generation (`api/s3.go`)
Added two new functions:

#### `GeneratePresignedURL(s3Key string, expirationMinutes int)`
- Generates temporary URLs for private S3 objects
- URLs expire after specified time (default: 60 minutes)
- Returns signed URL with AWS signature

#### `ConvertProfileURLsToPresigned(profile *DriverProfile, expirationMinutes int)`
- Converts all S3 keys in a DriverProfile to pre-signed URLs
- Handles all document types:
  - Selfie
  - License Front/Back
  - Vehicle Photo
  - Vehicle Registration
  - Insurance Document
  - Roadworthiness Document

### 3. Handler Updates (`api/handlers.go`)
All endpoints that return DriverProfile now convert S3 keys to pre-signed URLs:

- ✅ `SubmitKYCStep1` - Step 1 completion
- ✅ `SubmitKYCStep2` - Step 2 completion (selfie)
- ✅ `SubmitKYCStep3` - Step 3 completion (license & vehicle docs)
- ✅ `SubmitKYCStep4` - Step 4 completion (work preferences)
- ✅ `SubmitKYCStep5` - Step 5 completion (vehicle details & docs)
- ✅ `GetKYCProfile` - Driver profile retrieval
- ✅ `GetDriver` - Admin view driver details
- ✅ `ReviewDocument` - Admin document review

## How It Works

### Upload Flow
```
1. Driver uploads document
   ↓
2. File uploaded to S3 (private)
   ↓
3. S3 key stored in database
   Example: "kyc/insurance/9fc23bf9-2f37-42a5-932f-15449279b0e1.jpg"
```

### Retrieval Flow
```
1. API endpoint fetches profile from database
   ↓
2. ConvertProfileURLsToPresigned() called
   ↓
3. Each S3 key converted to pre-signed URL
   ↓
4. Response contains temporary URLs (valid 60 minutes)
   Example: "https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/9fc23bf9...jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=..."
```

## Benefits

### Security
- ✅ Files remain private in S3
- ✅ URLs expire after 60 minutes
- ✅ No public access to bucket
- ✅ AWS signature required for access
- ✅ Can't guess or enumerate document URLs

### Compliance
- ✅ Meets data privacy requirements
- ✅ Audit trail via S3 access logs
- ✅ Time-limited access
- ✅ Revocable access (change AWS credentials)

### Flexibility
- ✅ Can change expiration time per use case
- ✅ Can add IP restrictions if needed
- ✅ Can track access patterns
- ✅ Can implement rate limiting

## AWS Configuration Required

### NO CHANGES NEEDED! 🎉

Since files are now private by default, you don't need to:
- ❌ Disable Block Public Access
- ❌ Enable ACLs
- ❌ Add Bucket Policy

Your S3 bucket can remain completely private with default settings.

### Optional: Enable S3 Access Logging (Recommended)

For audit trails, enable S3 server access logging:

1. Go to AWS S3 Console
2. Select bucket: `hauler-driver-documents`
3. Go to **Properties** tab
4. Scroll to **Server access logging**
5. Click **Edit**
6. Enable logging
7. Choose a target bucket for logs
8. Save changes

This will log all access to your documents for compliance.

## Testing

### Test 1: Upload a Document
```bash
curl -X POST http://localhost:8080/api/driver/kyc/step/5 \
  -H "Authorization: Bearer YOUR_DRIVER_TOKEN" \
  -F "plate_number=TEST-123" \
  -F "brand=Toyota" \
  -F "model=Hilux" \
  -F "year=2022" \
  -F "color=White" \
  -F "insurance_document=@/path/to/insurance.jpg" \
  -F "roadworthiness_document=@/path/to/roadworthy.jpg"
```

**Expected Response:**
```json
{
  "status": 200,
  "message": "KYC Step 5 completed successfully. Your documents are under review.",
  "data": {
    "profile": {
      "insurance_document_url": "https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/uuid.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
      "roadworthiness_document_url": "https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/roadworthiness/uuid.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&..."
    }
  }
}
```

### Test 2: Access the URL
Copy the URL from the response and paste it in your browser. It should:
- ✅ Display the image
- ✅ Work for 60 minutes
- ❌ Stop working after 60 minutes (Access Denied)

### Test 3: Try Direct S3 URL (Without Signature)
Try accessing: `https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/uuid.jpg`

**Expected:** Access Denied (because file is private)

### Test 4: Admin Review
```bash
curl -X GET http://localhost:8080/api/admin/drivers/123 \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**Expected:** All document URLs should be pre-signed URLs

## URL Expiration

### Default: 60 Minutes
All pre-signed URLs expire after 60 minutes. This is a good balance between:
- Security (short-lived URLs)
- Usability (enough time to view documents)

### Changing Expiration Time

To change expiration for specific use cases, update the `ConvertProfileURLsToPresigned` call:

```go
// For admin dashboard (longer expiration)
ConvertProfileURLsToPresigned(&profile, 120) // 2 hours

// For driver viewing own docs (shorter expiration)
ConvertProfileURLsToPresigned(&profile, 30) // 30 minutes

// For email notifications (very short)
ConvertProfileURLsToPresigned(&profile, 15) // 15 minutes
```

## Database Storage

### What's Stored
S3 keys are stored in the database, NOT full URLs:

```
✅ Stored: "kyc/insurance/9fc23bf9-2f37-42a5-932f-15449279b0e1.jpg"
❌ NOT Stored: "https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/..."
```

### Why?
- Keys never expire
- Can generate new URLs anytime
- Can change S3 bucket without updating database
- Smaller database storage

## Frontend Integration

### Important Notes for Frontend Developers

1. **URLs Expire:** Pre-signed URLs expire after 60 minutes
   - Don't cache URLs for long periods
   - Refresh profile data if URLs expire
   - Show "Link Expired" message if 403 error

2. **URL Format:** URLs will be very long (contains AWS signature)
   - Don't truncate or modify URLs
   - Use full URL as-is
   - URLs contain query parameters (don't strip them)

3. **Refresh Strategy:**
   ```typescript
   // Good: Fetch fresh URLs when needed
   async function viewDocument(driverId: number) {
     const profile = await fetchDriverProfile(driverId);
     window.open(profile.insurance_document_url);
   }
   
   // Bad: Cache URLs for hours
   const cachedUrl = localStorage.getItem('doc_url'); // Will expire!
   ```

4. **Error Handling:**
   ```typescript
   async function displayDocument(url: string) {
     try {
       const response = await fetch(url);
       if (response.status === 403) {
         // URL expired, refresh profile
         alert('Link expired. Refreshing...');
         await refreshProfile();
       }
     } catch (error) {
       console.error('Failed to load document:', error);
     }
   }
   ```

## Troubleshooting

### Issue: "Access Denied" when accessing URL
**Causes:**
1. URL expired (after 60 minutes)
2. AWS credentials incorrect
3. S3 bucket name wrong

**Solution:**
- Fetch fresh profile data to get new pre-signed URL
- Check AWS credentials in `.env`
- Verify S3 bucket name

### Issue: URLs not being generated
**Causes:**
1. S3 client not initialized
2. AWS credentials missing

**Solution:**
```bash
# Check .env file has:
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=eu-north-1
AWS_S3_BUCKET=hauler-driver-documents
```

### Issue: Database still has old full URLs
**Causes:**
- Documents uploaded before this change

**Solution:**
The system handles both:
- Old records: Full URLs are converted to keys automatically
- New records: Keys are stored directly

No migration needed! The `GeneratePresignedURL` function handles both formats.

## Performance Considerations

### Pre-Signed URL Generation
- Very fast (< 1ms per URL)
- No network call to AWS
- Pure computation (HMAC signature)

### Impact on Response Time
- Minimal (< 10ms for all documents in a profile)
- Acceptable for real-time API responses

### Optimization Tips
1. **Batch Generation:** Already implemented in `ConvertProfileURLsToPresigned`
2. **Lazy Loading:** Only generate URLs for documents that exist
3. **Caching:** Frontend can cache URLs for < 60 minutes

## Security Best Practices

### ✅ Implemented
- Private S3 bucket
- Time-limited URLs
- AWS signature verification
- No public access

### 🔒 Additional Recommendations
1. **Enable S3 Access Logging** for audit trails
2. **Use CloudFront** for additional security layer
3. **Implement Rate Limiting** on API endpoints
4. **Monitor S3 Access Patterns** for anomalies
5. **Rotate AWS Credentials** regularly

### 🚫 Don't Do This
- ❌ Don't make bucket public
- ❌ Don't share pre-signed URLs publicly
- ❌ Don't cache URLs for > 60 minutes
- ❌ Don't log full pre-signed URLs (contains signature)

## Migration from Public URLs

### If You Have Existing Public URLs

The system automatically handles both:

**Old Format (Full URL):**
```
https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/uuid.jpg
```

**New Format (S3 Key):**
```
kyc/insurance/uuid.jpg
```

Both will work! The `GeneratePresignedURL` function detects the format and handles accordingly.

### No Database Migration Needed
- Old records continue to work
- New uploads use key format
- Gradual transition happens automatically

## Summary

✅ **Implemented:**
- Private S3 storage
- Pre-signed URL generation
- 60-minute expiration
- All endpoints updated
- Backward compatible

✅ **Benefits:**
- Enhanced security
- Compliance ready
- Audit trail capable
- No public access

✅ **No AWS Changes Needed:**
- Bucket remains private
- Default settings work
- No policy changes required

🎉 **Ready to Use:**
- Restart server
- Test with new uploads
- Existing data works automatically
