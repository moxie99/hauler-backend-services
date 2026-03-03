# Database Migration Fix - Expiry Date Columns

## Problem
Getting 500 error when submitting KYC Step 1:
```
INSERT INTO "driver_profiles" ... [rows:0]
500 | 4.3351512s | POST "/api/driver/kyc/step/1"
```

## Root Cause
The database table `driver_profiles` is missing the new expiry date columns that were added to the model:
- `license_front_expiry_date`
- `license_back_expiry_date`
- `vehicle_registration_expiry_date`
- `insurance_document_expiry_date`
- `roadworthiness_document_expiry_date`

GORM's `AutoMigrate` doesn't always add new columns to existing tables reliably.

## Solution Implemented

### Automatic Migration (Recommended)
I've added a `migrateExpiryDateColumns()` function that runs automatically when the server starts. It:
1. Checks if each expiry date column exists
2. Adds missing columns automatically
3. Logs the results

**To apply:**
1. Stop your server
2. Restart your server: `go run cmd/main.go`
3. Check the logs for messages like:
   ```
   Successfully added column: license_front_expiry_date
   Successfully added column: license_back_expiry_date
   ...
   ```

### Manual Migration (If Automatic Fails)

If the automatic migration doesn't work, run the SQL script manually:

#### Option 1: Using psql Command Line
```bash
psql -h your-database-host -U your-username -d your-database-name -f MANUAL_MIGRATION.sql
```

#### Option 2: Copy-Paste SQL
Connect to your PostgreSQL database and run:

```sql
-- Add expiry date columns
ALTER TABLE driver_profiles 
ADD COLUMN IF NOT EXISTS license_front_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS license_back_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS vehicle_registration_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS insurance_document_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS roadworthiness_document_expiry_date TIMESTAMP WITH TIME ZONE;

-- Verify columns were added
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'driver_profiles'
AND column_name LIKE '%expiry_date%'
ORDER BY column_name;
```

#### Option 3: Using Database GUI (pgAdmin, DBeaver, etc.)
1. Connect to your database
2. Open SQL query window
3. Paste the SQL from Option 2
4. Execute

## Verification

### Check if Columns Exist
Run this query in your database:

```sql
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'driver_profiles'
AND column_name LIKE '%expiry%'
ORDER BY column_name;
```

**Expected Output:**
```
column_name                          | data_type                   | is_nullable
-------------------------------------+-----------------------------+-------------
insurance_document_expiry_date       | timestamp with time zone    | YES
license_back_expiry_date             | timestamp with time zone    | YES
license_front_expiry_date            | timestamp with time zone    | YES
roadworthiness_document_expiry_date  | timestamp with time zone    | YES
vehicle_registration_expiry_date     | timestamp with time zone    | YES
```

### Test the API
After migration, test KYC Step 1:

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

**Expected:** 200 OK response

## For Production (Render, AWS, etc.)

### If Using Render:
1. Go to your Render dashboard
2. Select your database
3. Click "Connect" → "External Connection"
4. Use the connection string with psql:
   ```bash
   psql postgresql://user:password@host:port/database -f MANUAL_MIGRATION.sql
   ```

### If Using AWS RDS:
1. Connect using psql or pgAdmin
2. Run the migration SQL
3. Restart your application

### If Using Heroku:
```bash
heroku pg:psql -a your-app-name < MANUAL_MIGRATION.sql
```

## Troubleshooting

### Issue: "Permission denied" when adding columns
**Cause:** Database user doesn't have ALTER TABLE permission

**Solution:**
```sql
-- Grant permissions (run as superuser)
GRANT ALTER ON TABLE driver_profiles TO your_database_user;
```

### Issue: Columns still not appearing
**Cause:** Connected to wrong database

**Solution:**
1. Verify database name: `SELECT current_database();`
2. List all tables: `\dt` (in psql)
3. Ensure you're in the correct database

### Issue: Server still returns 500 error
**Causes:**
1. Columns not added yet
2. Server not restarted
3. Different error

**Solution:**
1. Verify columns exist (see Verification section)
2. Restart server completely
3. Check server logs for actual error message

## Prevention

To avoid this in the future:

### 1. Always Test Migrations Locally
```bash
# Before deploying
go run cmd/main.go
# Check logs for migration messages
```

### 2. Use Migration Scripts
For production, consider using a migration tool like:
- golang-migrate
- goose
- GORM's Migrator with explicit column additions

### 3. Monitor AutoMigrate
Add logging to see what AutoMigrate does:
```go
DB.AutoMigrate(&DriverProfile{})
// Check if columns exist after
```

## Summary

✅ **Automatic Fix Added:** Server now checks and adds missing columns on startup

✅ **Manual Fix Available:** SQL script provided for manual migration

✅ **Verification Steps:** Query to confirm columns exist

✅ **Production Ready:** Instructions for all major platforms

After applying the fix, your KYC Step 1 submission should work correctly!
