# Driver Location & Dispatch Setup Guide

## Problem
The dispatch service was failing because:
1. **Missing `user_locations` table** - Drivers' locations weren't being stored
2. **Missing `driver_vehicle_types` table** - Driver-to-vehicle relationships weren't tracked

## Solution

### Step 1: Apply Database Migrations ✅ (Done)
The migrations have been automatically applied when your backend restarted. New tables created:
- `user_locations` - Stores driver/user current location (latitude/longitude)
- `driver_vehicle_types` - Maps drivers to available vehicle types

### Step 2: Seed Test Driver Data

Your driver needs:
1. **Location data** - So dispatch can calculate proximity
2. **Vehicle type relationships** - So dispatch knows what vehicles they can drive
3. **Load type relationships** - Already set up during KYC, but verify with:

```sql
-- Check if your driver has vehicle/load types
SELECT d.driver_id, d.vehicle_type_id, l.load_type_id
FROM driver_vehicle_types d
LEFT JOIN driver_load_types l ON d.driver_id = l.driver_id
WHERE d.driver_id = <YOUR_DRIVER_ID>;
```

If rows are empty, manually add them (see Step 4).

### Step 3: One-time Setup - Add Driver Location & Relationships

**Option A: Using SQL directly (Railway Dashboard)**

Go to your Railway app → Database → Connect → SQL Query and run:

```sql
-- Find your approved driver
SELECT id, email FROM users WHERE role = 'driver' AND kyc_status = 'approved';

-- Replace <DRIVER_ID> with the ID above, then run:
INSERT INTO user_locations (user_id, latitude, longitude, updated_at)
VALUES (<DRIVER_ID>, 6.5244, 3.3792, NOW())
ON CONFLICT (user_id) DO UPDATE
SET latitude = 6.5244, longitude = 3.3792, updated_at = NOW();

INSERT INTO driver_vehicle_types (driver_id, vehicle_type_id)
VALUES (<DRIVER_ID>, 1)
ON CONFLICT DO NOTHING;

INSERT INTO driver_load_types (driver_id, load_type_id)
VALUES (<DRIVER_ID>, 3)
ON CONFLICT DO NOTHING;
```

**Option B: Using the API endpoint** (After logging in as driver)

```bash
curl -X POST "http://localhost:8080/api/location" \
  -H "Authorization: Bearer YOUR_DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 6.5244,
    "longitude": 3.3792
  }'
```

Response:
```json
{
  "status": 200,
  "message": "Location updated",
  "data": {
    "id": 1,
    "user_id": 2,
    "latitude": 6.5244,
    "longitude": 3.3792,
    "updated_at": "2026-04-08T11:35:00Z"
  }
}
```

### Step 4: Verify Setup

1. **Check driver has location:**
   ```sql
   SELECT * FROM user_locations WHERE user_id = <DRIVER_ID>;
   ```
   Should return: `id, user_id, latitude, longitude, updated_at`

2. **Check vehicle type relationship:**
   ```sql
   SELECT * FROM driver_vehicle_types WHERE driver_id = <DRIVER_ID>;
   ```
   Should show at least one vehicle type

3. **Check load type relationship:**
   ```sql
   SELECT * FROM driver_load_types WHERE driver_id = <DRIVER_ID>;
   ```
   Should show at least one load type (typically from KYC step 4)

### Step 5: Test Dispatch

1. **Create a new order** with:
   - vehicle_type_id: 1
   - load_type_id: 3
   - pickup coordinates near: 6.5244, 3.3792 (Lagos, Nigeria)

2. **Watch backend logs** for:
   ```
   [Dispatch] Processing order X in geo-cell YYY
   [Dispatch] Assigned driver Z (score: 95.23) to order X
   ```

3. **Check order status:**
   ```bash
   GET /api/orders/<ORDER_ID>
   ```
   Should show:
   - `status`: "driver_assigned"
   - `driver_id`: Your driver ID
   - `fee_units`: Calculated price
   - `estimated_distance_km`: Distance to dropoff

## Coordinate Notes

The order distance calculation is based on **Haversine formula**:
- **Pickup**: 6.5244, 3.3792 (Lagos, Nigeria)
- **Dropoff**: 5.1234, 2.1234 (example - adjust in order)
- **Distance**: ~140 km (realistic for Lagos area)

If you're seeing 12,500+ km distances, check that your order coordinates are valid.

## New API Endpoints

### Update Driver Location (Protected)
```
POST /api/location
Authorization: Bearer <token>
{
  "latitude": 6.5244,
  "longitude": 3.3792
}
```

### Get User Location (Admin/Super Admin)
```
GET /api/location/:user_id
Authorization: Bearer <admin_token>
```

## Troubleshooting

**"ERROR: relation 'user_locations' does not exist"**
- ✅ Fixed - Migrations added the table
- Restart backend to apply changes: `go run ./cmd/main.go`

**"[Dispatch] No suitable driver found for order X"**
- Driver doesn't have the required vehicle/load types
- Verify `driver_vehicle_types` and `driver_load_types` records

**"[Dispatch] Error finding driver: ERROR: relation 'driver_vehicle_types' does not exist"**
- ✅ Fixed - Migration added the table
- Restart backend

**Dispatch still failing after restart?**
- Verify driver has location: `SELECT * FROM user_locations WHERE user_id = <ID>`
- Verify vehicle/load relationships exist
- Check backend logs for specific error messages
