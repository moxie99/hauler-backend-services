# Hauler Backend Services API Documentation

All endpoints return a shared JSON wrapper:

```json
{
  "status": 200,
  "message": "string",
  "data": { ... } | [ ... ] | null
}
```

Protected routes require `Authorization: Bearer <token>`.

---

## Authentication

### POST `/api/auth/register`
- Auth: none
- Request JSON:
  - `email` string
  - `password` string
  - `first_name` string
  - `last_name` string
  - `phone` string
  - `role` string (optional: `super_admin`, `admin`, `driver`, `customer`)
- Response:
  - `status`: 201 on success
  - `message`
  - `data`: object with `message` and `requires_verification` when email verification is required

### POST `/api/auth/login`
- Auth: none
- Request JSON:
  - `email` string
  - `password` string
- Response:
  - `data`: object containing:
    - `token` string
    - `user` object
    - driver-specific fields: `kyc_status`, `kyc_current_step`, `requires_kyc`, `kyc_completed`, `kyc_message`

### POST `/api/auth/verify-login-otp`
- Auth: none
- Request JSON:
  - `email` string
  - `code` string
- Response:
  - `data`: object containing `token` and `user`

### POST `/api/auth/verify-email`
- Auth: none
- Request JSON:
  - `email` string
  - `code` string
- Response:
  - `data`: object containing `token` and `user`

### POST `/api/auth/resend-verification-code`
- Auth: none
- Request JSON:
  - `email` string
- Response:
  - `data`: null

---

## Driver registration and verification

### POST `/api/driver/register`
- Auth: none
- Request JSON:
  - `email` string
  - `password` string
  - `first_name` string
  - `last_name` string
  - `phone` string
- Response:
  - `status`: 201 on success
  - `data`: object with `user` and `message`

### POST `/api/driver/verify-email`
- Same request/response as `/api/auth/verify-email`

### POST `/api/driver/resend-verification-code`
- Same request/response as `/api/auth/resend-verification-code`

---

## Public lookup endpoints

### GET `/api/countries`
- Auth: none
- Response:
  - `data`: array of `Country` objects

### GET `/api/countries/:id/states`
- Auth: none
- Response:
  - `data`: array of `State` objects

### GET `/api/genders`
- Auth: none
- Response:
  - `data`: array of `Gender` objects

### GET `/api/vehicle-types`
- Auth: none
- Optional query params:
  - `category_id`
  - `active_only=true`
- Response:
  - `data`: array of `VehicleType` objects

### GET `/api/vehicle-types/:id`
- Auth: none
- Response:
  - `data`: single `VehicleType` object

### GET `/api/categories`
- Auth: none
- Optional query params:
  - `active_only=true`
- Response:
  - `data`: array of `Category` objects, each including `min_weight_kg` and `max_weight_kg`

### GET `/api/categories/:id`
- Auth: none
- Response:
  - `data`: single `Category` object including `min_weight_kg` and `max_weight_kg`

### GET `/api/load-types`
- Auth: none
- Optional query params:
  - `active_only=true`
- Response:
  - `data`: array of `LoadType` objects

### GET `/api/load-types/:id`
- Auth: none
- Response:
  - `data`: single `LoadType` object

---

## Password reset flow

### POST `/api/auth/forgot-password`
- Auth: none
- Request JSON:
  - `email` string
- Response:
  - `data`: null

### POST `/api/auth/verify-forgot-password`
- Auth: none
- Request JSON:
  - `email` string
  - `code` string
- Response:
  - `data`: `{ "verified": true }`

### POST `/api/auth/reset-password`
- Auth: none
- Request JSON:
  - `email` string
  - `code` string
  - `new_password` string
- Response:
  - `data`: null

### POST `/api/auth/resend-forgot-password-code`
- Auth: none
- Request JSON:
  - `email` string
- Response:
  - `data`: null

---

## Protected user routes (`/api` with JWT)

### GET `/api/profile`
- Auth: required
- Response:
  - `data`: object with `user`
  - Drivers also receive `kyc_current_step`, `kyc_status`, and `total_steps`

### POST `/api/orders`
- Auth: required
- Request JSON:
  - `pickup_address` string
  - `dropoff_address` string
  - `pickup_latitude` number
  - `pickup_longitude` number
  - `dropoff_latitude` number
  - `dropoff_longitude` number
  - `geo_cell` string
  - `vehicle_type_id` uint
  - `load_type_id` uint
  - `category_id` uint optional
  - `weight_kg` number
  - `requires_special_handling` bool
  - `preferred_pickup_time` string optional
  - `special_instructions` string optional
- Response:
  - `data`: created `Order` object

### GET `/api/orders`
- Auth: required
- Optional query params: `page`, `page_size`, `status`
- Response:
  - `data`: array of `Order` objects for the authenticated customer

### GET `/api/orders/:id/tracking`
- Auth: required
- Response:
  - `data`: object containing order details, driver info, and current driver location

### PUT `/api/orders/:id/tracking`
- Auth: required (assigned driver only)
- Request JSON:
  - `status` string (`picked_up`, `en_route`, `delivered`)
  - `latitude` number optional (driver's current location)
  - `longitude` number optional (driver's current location)
  - `estimated_arrival_mins` number optional
  - `message` string optional
- Response:
  - `data`: tracking update confirmation

### GET `/api/orders/:id/ws`
- Auth: required
- WebSocket endpoint for real-time order tracking updates
- Receives live updates for order status changes and driver location

### GET `/api/orders/:id/sse`
- Auth: required
- Server-Sent Events endpoint for real-time order tracking
- Alternative to WebSocket for browsers without WebSocket support

### PUT `/api/driver/kyc-status`
- Auth: required
- Request JSON:
  - `status` string (`pending`, `in_progress`, `approved`, `rejected`)
- Response:
  - `data`: object with `user` and `kyc_status`

### PUT `/api/driver/kyc-status/:id`
- Auth: required
- Admin or super admin only
- Same request body as above

### GET `/api/driver/kyc`
- Auth: required
- Response:
  - `data`: `{ profile, current_step, total_steps }`

### POST `/api/driver/kyc/step/1`
- Auth: required
- Request JSON:
  - `fullName` string
  - `phoneNumber` string
  - `email` string
  - `countryId` uint
  - `stateId` uint
  - `genderId` uint
  - `houseAddress` string
  - `officeAddress` string
  - `dateOfBirth` string (ISO 8601)
- Response:
  - `data`: `{ profile, current_step, total_steps }`

### POST `/api/driver/kyc/step/2`
- Auth: required
- Request: multipart/form-data
  - `selfie` file
- Response:
  - `data`: `{ profile, current_step, total_steps }`

### POST `/api/driver/kyc/step/3`
- Auth: required
- Request: multipart/form-data
  - `license_front` file
  - `license_back` file
  - `vehicle_photo` file
  - `vehicle_registration` file
- Response:
  - `data`: `{ profile, current_step, total_steps }`

### POST `/api/driver/kyc/step/4`
- Auth: required
- Request JSON:
  - `daysOfWork` array of strings
  - `vehicleTypeId` uint
  - `loadTypeIds` array of uints (optional)
  - `workStartTime` string
  - `workEndTime` string
- Response:
  - `data`: `{ profile, current_step, total_steps, load_types, days_of_work }`

### POST `/api/driver/kyc/step/5`
- Auth: required
- Request: multipart/form-data
  - `plate_number` string
  - `brand` string
  - `model` string
  - `year` string
  - `color` string
  - `insurance_document` file
  - `roadworthiness_document` file
- Response:
  - `data`: `{ profile, current_step, total_steps }`

### POST `/api/auth/refresh-token`
- Auth: required
- Response:
  - `data`: `{ token }`

### POST `/api/auth/change-password/request-otp`
- Auth: required
- Response:
  - `data`: null

### POST `/api/auth/change-password`
- Auth: required
- Request JSON:
  - `code` string
  - `old_password` string
  - `new_password` string
- Response:
  - `data`: `{ token }`

### POST `/api/auth/logout`
- Auth: required
- Response:
  - `data`: `{ "logged_out": true }`

---

## Super Admin routes

### POST `/api/super-admin/create-admin`
- Auth: required
- Super admin only
- Request JSON:
  - `email` string
  - `password` string
  - `first_name` string
  - `last_name` string
  - `phone` string
  - `country_id` uint
  - `gender_id` uint
- Response:
  - `data`: created admin `User` object

### GET `/api/super-admin/admins`
- Auth: required
- Super admin only
- Optional query params: `page`, `page_size`
- Response:
  - `data`: pagination object with admins array

### PUT `/api/super-admin/admins/:id`
- Auth: required
- Super admin only
- Request JSON:
  - optional `email`, `first_name`, `last_name`, `phone`
  - optional `country_id`, `gender_id`
- Response:
  - `data`: updated admin `User` object

### DELETE `/api/super-admin/admins/:id`
- Auth: required
- Super admin only
- Response:
  - `data`: null

### GET `/api/super-admin/drivers`
- Auth: required
- Super admin only
- Query params: `page`, `page_size`, `kyc_status`, `kyc_step`, `step_status`, `document_status`
- Response:
  - `data`: pagination object with drivers

### GET `/api/super-admin/drivers/:id`
- Auth: required
- Super admin only
- Response:
  - `data`: `{ driver, kyc_profile }`

### PUT `/api/super-admin/drivers/:id/suspend`
- Auth: required
- Super admin only
- Request JSON:
  - `is_active` bool
- Response:
  - `data`: updated driver object

### POST `/api/super-admin/drivers/:id/review-document`
- Auth: required
- Super admin only
- Request JSON:
  - `document_type` string
    - one of `selfie`, `license_front`, `license_back`, `vehicle_photo`, `vehicle_registration`, `insurance_document`, `roadworthiness_document`
  - `status` string (`approved`, `rejected`)
  - `rejection_reason` string required if status is `rejected`
  - `expiry_date` string optional, format `YYYY-MM-DD`
- Response:
  - `data`: `{ driver, kyc_profile }`

---

## Admin routes

### POST `/api/admin/countries`
- Auth: required
- Admin or super admin
- Request JSON:
  - `name` string
  - `code` string
- Response:
  - `data`: created `Country` object

### PUT `/api/admin/countries/:id`
- Auth: required
- Admin or super admin
- Request JSON:
  - optional `name`, `code`
- Response:
  - `data`: updated `Country` object

### DELETE `/api/admin/countries/:id`
- Auth: required
- Admin or super admin
- Response:
  - `data`: null

### POST `/api/admin/states`
- Auth: required
- Admin or super admin
- Request JSON:
  - `country_id` uint
  - `name` string
  - `code` string optional
- Response:
  - `data`: created `State` object

### PUT `/api/admin/states/:id`
- Auth: required
- Admin or super admin
- Request JSON:
  - optional `name`, `code`
- Response:
  - `data`: updated `State` object

### DELETE `/api/admin/states/:id`
- Auth: required
- Admin or super admin
- Response:
  - `data`: null

### GET `/api/admin/drivers`
- Auth: required
- Admin or super admin
- Same behaviour as `/api/super-admin/drivers`, but filtered by admin country
- Response:
  - `data`: pagination object

### GET `/api/admin/drivers/:id`
- Auth: required
- Admin or super admin
- Response:
  - `data`: `{ driver, kyc_profile }`

### PUT `/api/admin/drivers/:id/suspend`
- Auth: required
- Admin or super admin
- Request JSON:
  - `is_active` bool
- Response:
  - `data`: updated driver object

### POST `/api/admin/drivers/:id/review-document`
- Auth: required
- Admin or super admin
- Same request JSON as super admin review document
- Response:
  - `data`: `{ driver, kyc_profile }`

### GET `/api/admin/orders`
- Auth: required
- Admin or super admin
- Query parameters:
  - `page` int optional (default: 1)
  - `page_size` int optional (default: 20, max: 100)
  - `status` string optional (filter by order status)
  - `customer_id` uint optional (filter by customer ID)
  - `driver_id` uint optional (filter by driver ID)
  - `vehicle_type_id` uint optional (filter by vehicle type ID)
  - `geo_cell` string optional (filter by geo cell)
  - `pickup_date` string optional (filter by pickup date, format: YYYY-MM-DD)
- Response:
  - `data`: object containing:
    - `orders` array of `Order` objects with preloaded relationships
    - `page` int
    - `page_size` int
    - `total` int (total number of orders)
    - `total_pages` int

### PATCH `/api/admin/orders/:id/status`
- Auth: required
- Admin or super admin
- Request JSON:
  - `status` string (`pricing_requested`, `on_hold`, `dispatch_requested`, `driver_assigned`, `picked_up`, `delivered`, `cancelled`)
  - `driver_id` uint optional
  - `fee_units` int optional
  - `driver_rate_units` int optional
  - `estimated_time_mins` number optional
  - `estimated_distance_km` number optional
- Response:
  - `data`: updated `Order` object

### POST `/api/admin/vehicle-types`
- Auth: required
- Admin or super admin
- Request: multipart/form-data
  - `name` string
  - `category_id` uint
  - `description` string optional
  - `image` file optional
  - `max_payload_kg` number
  - `cargo_length_m` number optional
  - `cargo_width_m` number optional
  - `cargo_height_m` number optional
  - `cargo_volume_m3` number optional
  - `is_temperature_controlled` bool
  - `is_enclosed` bool
  - `has_tail_lift` bool
  - `has_crane` bool
  - `requires_special_license` bool
- Response:
  - `data`: created `VehicleType` object

### PUT `/api/admin/vehicle-types/:id`
- Auth: required
- Admin or super admin
- Request: multipart/form-data with any fields from create
- Response:
  - `data`: updated `VehicleType` object

### DELETE `/api/admin/vehicle-types/:id`
- Auth: required
- Admin or super admin
- Response:
  - `data`: null

### POST `/api/admin/categories`
- Auth: required
- Admin or super admin
- Request JSON:
  - `name` string
  - `code` string
  - `description` string optional
  - `min_weight_kg` number optional
  - `max_weight_kg` number optional
- Response:
  - `data`: created `Category` object

### PUT `/api/admin/categories/:id`
- Auth: required
- Admin or super admin
- Request JSON:
  - optional `name`, `code`, `description`
  - optional `min_weight_kg` number
  - optional `max_weight_kg` number
  - optional `is_active` bool
- Response:
  - `data`: updated `Category` object

### DELETE `/api/admin/categories/:id`
- Auth: required
- Admin or super admin
- Response:
  - `data`: null

### POST `/api/admin/load-types`
- Auth: required
- Admin or super admin
- Request JSON:
  - `name` string
  - `description` string optional
  - `requires_special_handling` bool
- Response:
  - `data`: created `LoadType` object

### PUT `/api/admin/load-types/:id`
- Auth: required
- Admin or super admin
- Request JSON:
  - optional `name`, `description`
  - optional `requires_special_handling` bool
  - optional `is_active` bool
- Response:
  - `data`: updated `LoadType` object

---

## Event-driven architecture

This service publishes and consumes Kafka events to support real-time dispatch, pricing, tracking, and notifications.

### Topics
- `order-events`
  - publishes `OrderPlaced` and `OrderStatusUpdated` events
- `dispatch-events`
  - publishes `DriverAssigned` events
- `pricing-events`
  - publishes `PricingCalculated` events
- `tracking-events`
  - publishes `TrackingUpdated` events
- `notifications`
  - publishes `NotificationEvent` payloads for downstream delivery

### Event contracts
- `OrderPlaced`
  - emitted after order creation
  - contains order details, geo-cell, vehicle/load type, weight, and location
- `OrderStatusUpdated`
  - emitted after status transitions
  - contains status, driver assignment, fee, ETA, and distance
- `DriverAssigned`
  - emitted when a driver is matched to an order
- `PricingCalculated`
  - emitted when pricing has been calculated for an order
- `TrackingUpdated`
  - emitted when tracking or status metadata changes
- `NotificationEvent`
  - emitted to the notifications topic for email/SMS/push delivery

---

## Health check endpoints

### GET `/health`
- Auth: none
- Response:
  - `data`: `{ status, timestamp }`

### GET `/health/detailed`
- Auth: none
- Response:
  - `data`: health object including `database`, `s3`, and `environment` checks

### GET `/health/ready`
- Auth: none
- Response:
  - `data`: `{ ready, checks, timestamp }`

### GET `/health/live`
- Auth: none
- Response:
  - `data`: `{ alive, timestamp }`

---

## Main payload object summaries

### User
- `id`
- `email`
- `first_name`
- `last_name`
- `phone`
- `role`
- `is_active`
- `email_verified`
- `kyc_status`
- `country_id`
- `gender_id`
- `created_at`
- `updated_at`

### Country
- `id`
- `name`
- `code`
- `states` optional

### State
- `id`
- `country_id`
- `name`
- `code`

### Gender
- `id`
- `name`

### VehicleType
- `id`
- `name`
- `category_id`
- `description`
- `image_url`
- `max_payload_kg`
- `cargo_length_m`
- `cargo_width_m`
- `cargo_height_m`
- `cargo_volume_m3`
- `is_temperature_controlled`
- `is_enclosed`
- `has_tail_lift`
- `has_crane`
- `requires_special_license`
- `is_active`
- `created_at`
- `updated_at`

### Category
- `id`
- `name`
- `code`
- `description`
- `min_weight_kg`
- `max_weight_kg`
- `is_active`
- `created_at`
- `updated_at`

### LoadType
- `id`
- `name`
- `description`
- `requires_special_handling`
- `is_active`
- `created_at`
- `updated_at`

### DriverProfile
- `id`
- `user_id`
- `current_step`
- personal info fields, document URLs/statuses, work preferences, vehicle details
- `created_at`
- `updated_at`
