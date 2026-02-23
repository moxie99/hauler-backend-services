# Pre-Signed URLs - Quick Start Guide

## ✅ What's Done

Your system now uses **secure pre-signed URLs** for all driver documents instead of public S3 access.

## 🎯 Key Changes

### Files Are Now Private
- All documents uploaded to S3 are **private**
- No public access to your S3 bucket
- URLs expire after **60 minutes**

### How It Works
```
Upload → S3 (private) → Store key in DB → Generate pre-signed URL when needed
```

## 🚀 Quick Test

### 1. Upload a Document
```bash
curl -X POST http://localhost:8080/api/driver/kyc/step/5 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "plate_number=ABC-123" \
  -F "brand=Toyota" \
  -F "model=Hilux" \
  -F "year=2022" \
  -F "color=White" \
  -F "insurance_document=@insurance.jpg" \
  -F "roadworthiness_document=@roadworthy.jpg"
```

### 2. Check Response
You'll get URLs like:
```
https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/insurance/uuid.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...&X-Amz-Signature=...
```

### 3. Access URL
- ✅ Works for 60 minutes
- ❌ Expires after 60 minutes (returns Access Denied)

## 📋 No AWS Configuration Needed

Your S3 bucket can remain **completely private** with default settings. No changes needed!

## 🔒 Security Benefits

- ✅ Documents are private
- ✅ URLs expire automatically
- ✅ Can't guess document URLs
- ✅ AWS signature required
- ✅ Audit trail capable

## 💡 Frontend Notes

### Important for Frontend Developers

1. **URLs Expire:** Don't cache URLs for > 60 minutes
2. **Refresh Strategy:** Fetch fresh profile data when needed
3. **Error Handling:** Handle 403 errors (expired URLs)

```typescript
// Good Practice
async function viewDocument(driverId: number) {
  const profile = await fetchDriverProfile(driverId); // Fresh URLs
  window.open(profile.insurance_document_url);
}

// Bad Practice
const url = localStorage.getItem('doc_url'); // Will expire!
window.open(url);
```

## 🔧 Troubleshooting

### "Access Denied" Error
**Cause:** URL expired (after 60 minutes)
**Solution:** Fetch fresh profile data

### URLs Not Generated
**Cause:** AWS credentials missing
**Solution:** Check `.env` file:
```
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_REGION=eu-north-1
AWS_S3_BUCKET=hauler-driver-documents
```

## 📊 What Changed in Code

### S3 Upload
- **Before:** Returns full public URL
- **After:** Returns S3 key (private)

### API Responses
- **Before:** Full URLs in database
- **After:** Pre-signed URLs generated on-the-fly

### All Endpoints Updated
- ✅ KYC Step 1-5 submissions
- ✅ Get KYC Profile
- ✅ Admin get driver
- ✅ Admin review document

## 🎉 Ready to Use

1. ✅ Server is running
2. ✅ All endpoints updated
3. ✅ Backward compatible
4. ✅ No migration needed

Just upload new documents and they'll automatically use secure pre-signed URLs!

## 📚 More Details

See `PRESIGNED_URLS_IMPLEMENTATION.md` for complete documentation.
