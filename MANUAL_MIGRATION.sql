-- Manual Migration Script for Adding Expiry Date Columns
-- Run this SQL script directly on your PostgreSQL database

-- Add expiry date columns to driver_profiles table
ALTER TABLE driver_profiles 
ADD COLUMN IF NOT EXISTS license_front_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS license_back_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS vehicle_registration_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS insurance_document_expiry_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS roadworthiness_document_expiry_date TIMESTAMP WITH TIME ZONE;

-- Add email_verified column to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE;

-- Verify the columns were added
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'driver_profiles'
AND column_name LIKE '%expiry_date%'
ORDER BY column_name;
