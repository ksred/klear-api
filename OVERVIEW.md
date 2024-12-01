# Distributed Trading System Design

## Overview
This system provides a distributed execution, clearing, and settlement infrastructure for trading securities. It simulates the complete lifecycle of a trade from order submission through execution, clearing, and final settlement, while maintaining security and data consistency across all operations.

For simplicity, this is currently presented as a single service. If required in a production implementation, this service would be split out into several smaller services.

## System Flow
1. Client submits trade order through public API
2. System validates order and client details
3. Order is routed to one or more exchanges for execution
4. Executed trades are sent to clearing house (CCP)
5. Cleared trades are sent to central securities depository (CSD) for settlement
6. Status updates available throughout the process

## Network Architecture
- Public subnet: Handles order submission and status inquiries
- Private subnet: Manages execution, clearing, and settlement operations
- API Gateway: Routes and secures all public endpoints

Note: Implementation in this project assumes it is sitting behind an API gateway. We are using a combination of API keys, secrets and JWT tokens for authentication. The same test values are used for both public and private endpoints. We assume that some third party is calling the internal APIs, and this third-party could be the service itself. For now, we call it manually. The service could, of course, not call a private API and instead run the functions directly.

## Client Details
### Base Information
- Client ID (unique identifier)
- Client Name
- Status (active/inactive)

### Trading Limits
- Credit Limit
- Margin Requirements (percentage)
- Daily Trading Limit

### Settlement Details
- Settlement Account Number
- Default Currency

## Authentication
### Credentials
- API Key: Client identifier
- API Secret: Used for request signing
- JWT Token: Used for API access

### Auth Flow
1. Client requests JWT token using API Key/Secret
2. JWT token includes:
   - Client ID
   - Permissions
   - Expiration time
3. All subsequent requests require valid JWT
4. Rate limiting applied per API key

## API Endpoints

### 1. Authentication
```
POST /api/v1/auth/token
- Request:
  - api_key
  - api_secret
- Response:
  - jwt_token
  - expiration
```

### 2. Trade Order
```
POST /api/v1/orders
- Headers:
  - Authorization: Bearer <jwt_token>
- Request:
  - client_id
  - stock_symbol
  - quantity
  - price
  - order_type (market/limit)
  - side (buy/sell)
  - time_in_force
- Response:
  - order_id
  - status
  - timestamp
```

### 3. Trade Execution
```
POST /api/v1/execution/{order_id}
- Internal endpoint
- Simulates execution across multiple exchanges
- Exchange selection based on liquidity
- Response:
  - execution_id
  - executed_price
  - executed_quantity
  - timestamp
  - exchange_id
  - execution_fees
```

### 4. Trade Clearing
```
POST /api/v1/clearing/{trade_id}
- Internal endpoint
- Simulates CCP operations
- Response:
  - clearing_id
  - clearing_status
  - margin_required
  - net_positions
  - settlement_amount
```

### 5. Trade Settlement
```
POST /api/v1/settlement/{trade_id}
- Internal endpoint
- Simulates CSD operations
- Response:
  - settlement_id
  - settlement_status
  - settlement_date
  - final_amount
```

### 6. Order Status
```
GET /api/v1/orders/{order_id}
- Headers:
  - Authorization: Bearer <jwt_token>
- Response:
  - order_id
  - current_status
  - status_history
  - execution_details (if executed)
  - clearing_details (if cleared)
  - settlement_details (if settled)
```

## Order Statuses
- RECEIVED: Initial order received
- VALIDATED: Passed initial checks
- ROUTING: Being sent to exchange(s)
- PARTIALLY_FILLED: Some execution
- FILLED: Fully executed
- CLEARING: At clearing house
- CLEARED: Clearing complete
- SETTLING: At CSD
- SETTLED: Fully settled
- REJECTED: Failed at any stage

## Exchange Simulation
- Multiple mock exchanges (3-5)
- Each exchange simulates:
  - Random latency (5-100ms)
  - Different liquidity pools
  - Price variance
  - Success/failure ratio
- Exchange selection based on:
  - Available liquidity
  - Historical execution quality
  - Current price spreads
  - Fee structures

## Error Handling
- Each endpoint includes specific error codes
- Retry mechanisms with exponential backoff
- Idempotency keys for all POST requests
- Detailed error messages and logging
- Circuit breakers for dependent services

## Security Considerations
- All endpoints secured with JWT authentication
- Rate limiting on public endpoints
- Request signing for sensitive operations
- Secure communication between components
- Input validation and sanitization