# API Documentation

## Authentication

All authenticated endpoints require a JWT token obtained through the authentication endpoint.

### Get Authentication Token

POST /api/v1/auth/token

Request:
```json
{
    "api_key": "string",     // Your API key
    "api_secret": "string"   // Your API secret
}
```

Response: 200 OK
```json
{
    "success": true,
    "data": {
        "jwt_token": "string",   // JWT token to use for subsequent requests
        "expiration": "string"   // ISO 8601 timestamp
    }
}
```

Error Response: 401 Unauthorized
```json
{
    "success": false,
    "error": {
        "code": "UNAUTHORIZED",
        "message": "Invalid credentials"
    }
}
```

Error Response: 400 Bad Request
```json
{
    "success": false,
    "error": {
        "code": "BAD_REQUEST",
        "message": "Invalid request format"
    }
}
```

Error Response: 500 Internal Server Error
```json
{
    "success": false,
    "error": {
        "code": "INTERNAL_ERROR",
        "message": "An unexpected error occurred"
    }
}
```

## Trading Endpoints

### Create Order

POST /api/v1/orders
Authorization: Bearer <jwt_token>
Idempotency-Key: <unique_key>  // Required to prevent duplicate orders

Request:
```json
{
    "client_id": "string",
    "symbol": "string",
    "side": "BUY" | "SELL",
    "order_type": "MARKET" | "LIMIT",
    "quantity": number,
    "price": number
}
```

Response: 201 Created
```json
{
    "success": true,
    "data": {
        "order_id": "string",
        "client_id": "string",
        "symbol": "string",
        "side": "string",
        "order_type": "string", 
        "quantity": number,
        "price": number,
        "status": "PENDING",
        "created_at": "string",
        "updated_at": "string"
    }
}
```

Error Response: 400 Bad Request
```json
{
    "success": false,
    "error": {
        "code": "BAD_REQUEST",
        "message": "Invalid order details"
    }
}
```

### Get Order Status

GET /api/v1/orders/{order_id}
Authorization: Bearer <jwt_token>

Response: 200 OK
```json
{
    "success": true,
    "data": {
        "order_id": "string",
        "client_id": "string",
        "symbol": "string",
        "side": "string",
        "order_type": "string",
        "quantity": number,
        "price": number,
        "status": "PENDING" | "FILLED" | "CANCELLED",
        "created_at": "string",
        "updated_at": "string"
    }
}
```

Error Response: 404 Not Found
```json
{
    "success": false,
    "error": {
        "code": "NOT_FOUND",
        "message": "Order not found"
    }
}
```

## Internal Endpoints

### Execute Order

POST /api/v1/internal/execution/{order_id}
Idempotency-Key: <unique_key>

Response: 200 OK
```json
{
    "success": true,
    "data": {
        "execution_id": "string",
        "order_id": "string",
        "price": number,
        "quantity": number,
        "side": "string",
        "status": "COMPLETED" | "FAILED",
        "created_at": "string",
        "updated_at": "string"
    }
}
```

### Clear Trade

POST /api/v1/internal/clearing/{trade_id}

Response: 200 OK
```json
{
    "success": true,
    "data": {
        "clearing_id": "string",
        "clearing_status": "PENDING" | "CLEARED" | "FAILED",
        "margin_required": number,
        "net_positions": number,
        "settlement_amount": number,
        "timestamp": "string"
    }
}
```

### Settle Trade

POST /api/v1/internal/settlement/{trade_id}

Response: 200 OK
```json
{
    "success": true,
    "data": {
        "settlement_id": "string",
        "trade_id": "string",
        "client_id": "string",
        "settlement_status": "PENDING" | "SETTLING" | "SETTLED" | "FAILED",
        "settlement_date": "string",
        "final_amount": number,
        "currency": "string",
        "settlement_account": "string",
        "executed_price": number,
        "executed_quantity": number,
        "settlement_fees": number,
        "timestamp": "string"
    }
}
```

## Error Handling

All endpoints follow a consistent error response format:
```json
{
    "success": false,
    "error": {
        "code": string,    // Error code identifier
        "message": string  // Human-readable error message
    }
}
```

Common Error Codes:
- NOT_FOUND: Resource could not be found
- BAD_REQUEST: Invalid input or request format
- UNAUTHORIZED: Invalid or missing authentication
- FORBIDDEN: Valid authentication but insufficient permissions
- INTERNAL_ERROR: Unexpected server error
- VALIDATION_FAILED: Request validation failed
- DUPLICATE_RESOURCE: Resource already exists

Common HTTP Status Codes:
- 200: Successful operation
- 201: Resource created successfully
- 400: Bad request (invalid input)
- 401: Unauthorized (invalid/missing authentication)
- 403: Forbidden (insufficient permissions)
- 404: Resource not found
- 409: Conflict (duplicate resource)
- 500: Internal server error

## Rate Limiting

API requests are rate-limited based on the client API key. The current limits are:
- Authentication endpoints: 10 requests per minute
- Trading endpoints: 100 requests per minute
- Status endpoints: 1000 requests per minute

When rate limit is exceeded, the API will respond with:
```json
{
    "success": false,
    "error": {
        "code": "RATE_LIMIT_EXCEEDED",
        "message": "Rate limit exceeded. Please try again later."
    }
}
```

## Idempotency

All POST endpoints require an Idempotency-Key header to prevent duplicate operations. The key should be a unique string (e.g., UUID) for each unique request. Repeating a request with the same idempotency key will return the result of the original request.

## Best Practices

1. Always include an Idempotency-Key header for POST requests
2. Store JWT tokens securely and refresh before expiration
3. Implement proper error handling for all possible response codes
4. Use appropriate timeouts for API calls
5. Implement exponential backoff for rate limit handling

## Exchange Integration

The system integrates with multiple exchanges with the following characteristics:
- Primary Exchange: Low latency (5-30ms), 0.1% fee rate
- Secondary Exchange: Medium latency (10-50ms), 0.08% fee rate
- Regional Exchange: Higher latency (15-70ms), 0.05% fee rate
- Dark Pool: Highest latency (20-100ms), 0.03% fee rate

## Settlement Process

The settlement process follows T+2 settlement cycle:
1. Trade execution (T)
2. Clearing process (T+1)
3. Final settlement (T+2)

Settlement statuses:
- PENDING: Initial state
- SETTLING: Settlement in progress
- SETTLED: Successfully completed
- FAILED: Settlement failed
