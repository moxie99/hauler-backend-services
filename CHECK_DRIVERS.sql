-- Check all drivers in the system
SELECT id, email, role, kyc_status, is_active, created_at 
FROM users 
WHERE role = 'driver'
ORDER BY created_at DESC;

-- If no drivers found, check all users
SELECT id, email, role, kyc_status, is_active 
FROM users 
ORDER BY created_at DESC 
LIMIT 10;
