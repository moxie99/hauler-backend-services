-- Seed test driver with location and vehicle/load type relationships
-- This assumes you already have a driver created and KYC approved

-- First, check if a test driver exists (adjust email as needed)
-- Get driver ID
DO $$
DECLARE
    driver_id INT;
    vehicle_type_id INT := 1;
    load_type_id INT := 3;
BEGIN
    -- Find approved driver (adjust this query based on your actual driver)
    SELECT id INTO driver_id FROM users 
    WHERE role = 'driver' AND kyc_status = 'approved' 
    LIMIT 1;
    
    IF driver_id IS NOT NULL THEN
        -- Add location if not exists
        INSERT INTO user_locations (user_id, latitude, longitude, updated_at)
        VALUES (driver_id, 6.5244, 3.3792, NOW())
        ON CONFLICT (user_id) DO UPDATE
        SET latitude = 6.5244, longitude = 3.3792, updated_at = NOW();
        
        -- Add vehicle type relationship if not exists
        INSERT INTO driver_vehicle_types (driver_id, vehicle_type_id)
        VALUES (driver_id, vehicle_type_id)
        ON CONFLICT DO NOTHING;
        
        -- Add load type relationship if not exists  
        INSERT INTO driver_load_types (driver_id, load_type_id)
        VALUES (driver_id, load_type_id)
        ON CONFLICT DO NOTHING;
        
        RAISE NOTICE 'Driver % seeded with location and relationships', driver_id;
    ELSE
        RAISE NOTICE 'No approved driver found';
    END IF;
END $$;
