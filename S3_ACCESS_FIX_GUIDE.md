# S3 Access Denied Fix Guide

## Problem
When accessing uploaded document URLs like:
```
https://hauler-driver-documents.s3.eu-north-1.amazonaws.com/kyc/roadworthiness/9fc23bf9-2f37-42a5-932f-15449279b0e1.jpg
```

You get an "Access Denied" error because the S3 bucket doesn't allow public access.

## Solution Implemented

### Code Changes Made
Updated `api/s3.go` to:
1. Set uploaded files to `public-read` ACL
2. Added `GeneratePresignedURL()` function for secure temporary access (optional)

## AWS S3 Configuration Required

### Option 1: Public Access (Simple - For Public Documents)

#### Step 1: Disable Block Public Access
1. Go to [AWS S3 Console](https://s3.console.aws.amazon.com/)
2. Select bucket: `hauler-driver-documents`
3. Click **Permissions** tab
4. Under "Block public access (bucket settings)", click **Edit**
5. **UNCHECK** these two options:
   - ☐ Block public access to buckets and objects granted through new access control lists (ACLs)
   - ☐ Block public access to buckets and objects granted through any access control lists (ACLs)
6. Click **Save changes**
7. Type `confirm` when prompted

#### Step 2: Enable ACLs
1. Still in **Permissions** tab
2. Scroll to **Object Ownership** section
3. Click **Edit**
4. Select **ACLs enabled**
5. Select **Bucket owner preferred**
6. Click **Save changes**

#### Step 3: Add Bucket Policy (Recommended)
1. Still in **Permissions** tab
2. Scroll to **Bucket policy** section
3. Click **Edit**
4. Paste this policy:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "PublicReadGetObject",
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::hauler-driver-documents/*"
        }
    ]
}
```

5. Click **Save changes**

#### Step 4: Test
After configuration, restart your Go server and upload a new document. The URL should now be accessible.

**Note:** Existing files uploaded before this change will still be private. You'll need to:
- Re-upload them, OR
- Manually change their ACL in S3 console

---

### Option 2: Pre-Signed URLs (More Secure - For Sensitive Documents)

If you want to keep files private but allow temporary access:

#### Benefits:
- Files remain private in S3
- URLs expire after a set time (e.g., 60 minutes)
- More secure for sensitive documents like driver licenses

#### Implementation:

1. **Keep S3 bucket private** (don't change any settings)

2. **Update your handlers** to generate pre-signed URLs when returning document URLs:

```go
// Example: Update GetDriver handler to return pre-signed URLs
func GetDriver(c *gin.Context) {
    // ... existing code to get driver profile ...
    
    // Generate pre-signed URLs for documents (valid for 60 minutes)
    if profile.LicenseFrontURL != "" {
        key := extractS3KeyFromURL(profile.LicenseFrontURL)
        presignedURL, err := GeneratePresignedURL(key, 60)
        if err == nil {
            profile.LicenseFrontURL = presignedURL
        }
    }
    
    // Repeat for other documents...
    
    ResponseJSON(c, http.StatusOK, "Driver retrieved successfully", gin.H{
        "user":        user,
        "kyc_profile": profile,
    })
}

// Helper function to extract S3 key from full URL
func extractS3KeyFromURL(url string) string {
    // Extract key from: https://bucket.s3.region.amazonaws.com/key
    parts := strings.Split(url, ".amazonaws.com/")
    if len(parts) == 2 {
        return parts[1]
    }
    return ""
}
```

3. **Store S3 keys instead of full URLs** (optional but recommended):

Update your model to store keys:
```go
type DriverProfile struct {
    // Instead of storing full URLs, store S3 keys
    LicenseFrontKey string `json:"-"` // Don't expose in JSON
    LicenseFrontURL string `json:"license_front_url" gorm:"-"` // Computed field
}
```

---

## Recommendation

**For your use case (KYC documents):**

Use **Option 1 (Public Access)** because:
- Simpler implementation
- Documents need to be viewed by admins frequently
- URLs don't expire (better for admin dashboard)
- Still secure (URLs are hard to guess with UUIDs)

**Use Option 2 (Pre-Signed URLs) if:**
- You need audit trails of who accessed documents
- Documents are highly sensitive
- You want URLs to expire after viewing
- Compliance requires private storage

---

## Testing After Configuration

### Test 1: Upload a new document
```bash
# Upload via your API
curl -X POST http://localhost:8080/api/driver/kyc/step/5 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "plate_number=TEST-123" \
  -F "brand=Toyota" \
  -F "model=Hilux" \
  -F "year=2022" \
  -F "color=White" \
  -F "insurance_document=@/path/to/insurance.jpg" \
  -F "roadworthiness_document=@/path/to/roadworthy.jpg"
```

### Test 2: Access the URL directly
Copy the returned URL and paste it in your browser. It should display the image.

### Test 3: Check existing files
For files uploaded before the fix, you need to either:

**Option A: Re-upload them**
- Have drivers re-submit their documents

**Option B: Manually fix ACL in AWS Console**
1. Go to S3 bucket
2. Navigate to the file
3. Click on the file
4. Go to **Permissions** tab
5. Under **Access control list (ACL)**, click **Edit**
6. Add **Everyone (public access)** with **Read** permission
7. Click **Save changes**

**Option C: Bulk fix with AWS CLI**
```bash
# Make all existing files public
aws s3 cp s3://hauler-driver-documents/ s3://hauler-driver-documents/ \
  --recursive \
  --acl public-read \
  --metadata-directive REPLACE
```

---

## Security Considerations

### Public Access is Safe When:
- ✅ File names use UUIDs (hard to guess)
- ✅ No sensitive personal data in file names
- ✅ URLs are not indexed by search engines
- ✅ You have proper authentication on your API

### Additional Security Measures:
1. **Enable S3 Server Access Logging** to track who accesses files
2. **Use CloudFront** in front of S3 for better security and caching
3. **Set CORS policy** to only allow your domain
4. **Enable S3 versioning** to recover from accidental deletions

---

## Troubleshooting

### Issue: Still getting Access Denied after configuration
**Solution:** 
- Wait 1-2 minutes for AWS changes to propagate
- Clear browser cache
- Try in incognito mode
- Check bucket policy is saved correctly

### Issue: New uploads still private
**Solution:**
- Restart your Go server to reload the updated code
- Check that `types.ObjectCannedACLPublicRead` is being used in upload

### Issue: Some files work, others don't
**Solution:**
- Files uploaded before the fix are still private
- Use Option B or C above to fix existing files

---

## Summary

1. ✅ Code updated to set `public-read` ACL on uploads
2. ⏳ Configure AWS S3 bucket (follow Option 1 steps above)
3. ✅ Restart Go server
4. ✅ Test with new upload
5. 🔧 Fix existing files if needed

After completing these steps, all document URLs should be publicly accessible!
