# JWT Authentication Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        JWT Authentication Flow                           │
└─────────────────────────────────────────────────────────────────────────┘

1. REGISTRATION FLOW
════════════════════

    Client                    API Server                  Database
      │                           │                           │
      │  POST /api/auth/register  │                           │
      ├─────────────────────────>│                           │
      │  { username, email,       │                           │
      │    password, role }       │                           │
      │                           │                           │
      │                           │   Hash password           │
      │                           │   (bcrypt)                │
      │                           │                           │
      │                           │   INSERT INTO users       │
      │                           ├─────────────────────────>│
      │                           │                           │
      │                           │<─────────────────────────┤
      │                           │   User created (id: 1)    │
      │                           │                           │
      │                           │   Generate JWT token      │
      │                           │   (sign with secret)      │
      │                           │                           │
      │<─────────────────────────┤                           │
      │  { token, user,           │                           │
      │    expires_in: 86400 }    │                           │
      │                           │                           │


2. LOGIN FLOW
═════════════

    Client                    API Server                  Database
      │                           │                           │
      │  POST /api/auth/login     │                           │
      ├─────────────────────────>│                           │
      │  { username, password }   │                           │
      │                           │                           │
      │                           │   SELECT * FROM users     │
      │                           │   WHERE username = ?      │
      │                           ├─────────────────────────>│
      │                           │                           │
      │                           │<─────────────────────────┤
      │                           │   User data + hash        │
      │                           │                           │
      │                           │   Verify password         │
      │                           │   bcrypt.Compare()        │
      │                           │                           │
      │                           │   ✓ Password valid        │
      │                           │   ✓ User active           │
      │                           │                           │
      │                           │   Generate JWT token      │
      │                           │   { user_id, role, ... }  │
      │                           │                           │
      │<─────────────────────────┤                           │
      │  { token, user,           │                           │
      │    expires_in: 86400 }    │                           │
      │                           │                           │


3. AUTHENTICATED REQUEST FLOW
═════════════════════════════

    Client                    API Server                  Database
      │                           │                           │
      │  GET /api/deliveries      │                           │
      │  Authorization: Bearer    │                           │
      │  eyJhbGc...               │                           │
      ├─────────────────────────>│                           │
      │                           │                           │
      │                           │   Extract token           │
      │                           │   from header             │
      │                           │                           │
      │                           │   Validate JWT            │
      │                           │   - Verify signature      │
      │                           │   - Check expiration      │
      │                           │   - Parse claims          │
      │                           │                           │
      │                           │   ✓ Token valid           │
      │                           │                           │
      │                           │   Add claims to context   │
      │                           │   { user_id, role, ... }  │
      │                           │                           │
      │                           │   Check role permission   │
      │                           │   RequireRole(customer)   │
      │                           │                           │
      │                           │   ✓ Role authorized       │
      │                           │                           │
      │                           │   Query based on role     │
      │                           │   WHERE customer_id = ?   │
      │                           ├─────────────────────────>│
      │                           │                           │
      │                           │<─────────────────────────┤
      │                           │   Filtered results        │
      │                           │                           │
      │<─────────────────────────┤                           │
      │  { deliveries: [...] }    │                           │
      │                           │                           │


4. ROLE-BASED ACCESS CONTROL
════════════════════════════

┌─────────────┬──────────────────────────────────────────────────────┐
│    Role     │                   Permissions                        │
├─────────────┼──────────────────────────────────────────────────────┤
│  Customer   │ ✓ Create deliveries                                  │
│             │ ✓ View own deliveries                                │
│             │ ✗ View other customers' deliveries                   │
│             │ ✗ Update delivery status                             │
│             │ ✗ Update location                                    │
├─────────────┼──────────────────────────────────────────────────────┤
│  Courier    │ ✗ Create deliveries                                  │
│             │ ✓ View assigned deliveries                           │
│             │ ✓ Update location                                    │
│             │ ✓ Update status of assigned deliveries              │
│             │ ✗ View other couriers' deliveries                    │
├─────────────┼──────────────────────────────────────────────────────┤
│   Admin     │ ✓ Full access to all resources                       │
│             │ ✓ Manage users                                       │
│             │ ✓ View all deliveries                                │
│             │ ✓ Access analytics                                   │
│             │ ✓ Override all permissions                           │
└─────────────┴──────────────────────────────────────────────────────┘


5. MIDDLEWARE CHAIN
═══════════════════

Request
   │
   ▼
┌────────────────────────────────────────┐
│      Extract Token from Header         │
│   Authorization: Bearer <token>        │
└────────────────────────────────────────┘
   │
   ▼
┌────────────────────────────────────────┐
│         Validate JWT Token             │
│  - Verify signature                    │
│  - Check expiration                    │
│  - Parse claims                        │
└────────────────────────────────────────┘
   │
   ├─────[Invalid]────> 401 Unauthorized
   │
   ▼ [Valid]
┌────────────────────────────────────────┐
│       Add Claims to Context            │
│  ctx = context.WithValue(ctx, ...)     │
└────────────────────────────────────────┘
   │
   ▼
┌────────────────────────────────────────┐
│       Check Role Requirements          │
│  RequireRole(customer, admin)          │
└────────────────────────────────────────┘
   │
   ├─────[Forbidden]───> 403 Forbidden
   │
   ▼ [Authorized]
┌────────────────────────────────────────┐
│         Execute Handler                │
│  Get claims from context               │
│  Apply role-based filtering            │
└────────────────────────────────────────┘
   │
   ▼
Response


6. TOKEN STRUCTURE
══════════════════

JWT Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI...

Decoded:
┌─────────────────────────────────────────────────────────────┐
│ HEADER                                                      │
│ {                                                           │
│   "alg": "HS256",                                          │
│   "typ": "JWT"                                             │
│ }                                                           │
├─────────────────────────────────────────────────────────────┤
│ PAYLOAD (Claims)                                           │
│ {                                                           │
│   "user_id": 1,                                            │
│   "username": "john_doe",                                  │
│   "email": "john@example.com",                             │
│   "role": "customer",                                      │
│   "customer_id": 1,                                        │
│   "exp": 1704196800,  // Expiration timestamp             │
│   "iat": 1704110400,  // Issued at timestamp              │
│   "nbf": 1704110400   // Not before timestamp             │
│ }                                                           │
├─────────────────────────────────────────────────────────────┤
│ SIGNATURE                                                   │
│ HMACSHA256(                                                │
│   base64UrlEncode(header) + "." +                          │
│   base64UrlEncode(payload),                                │
│   secret_key                                               │
│ )                                                           │
└─────────────────────────────────────────────────────────────┘


7. ACCESS CONTROL LOGIC
═══════════════════════

Delivery Access Check:

Customer accessing delivery #123
   │
   ▼
┌────────────────────────────────────────┐
│  Get delivery details from database    │
│  delivery.customer_id = 1              │
│  delivery.courier_id = 5               │
└────────────────────────────────────────┘
   │
   ▼
┌────────────────────────────────────────┐
│  Check user role and permissions       │
│  user.role = "customer"                │
│  user.customer_id = 1                  │
└────────────────────────────────────────┘
   │
   ├─────[Admin]───────────────────────> ✓ Allow (admin can access all)
   │
   ├─────[Customer]
   │       │
   │       ├─[customer_id matches]─────> ✓ Allow
   │       │
   │       └─[customer_id different]───> ✗ Deny (403 Forbidden)
   │
   └─────[Courier]
           │
           ├─[courier_id matches]───────> ✓ Allow
           │
           └─[courier_id different]─────> ✗ Deny (403 Forbidden)


8. ERROR HANDLING
═════════════════

┌─────────────────────────┬────────────┬──────────────────────────┐
│       Scenario          │   Status   │        Response          │
├─────────────────────────┼────────────┼──────────────────────────┤
│ No token provided       │    401     │ "Unauthorized"           │
│ Invalid token format    │    401     │ "Invalid token"          │
│ Token expired           │    401     │ "Token has expired"      │
│ Invalid signature       │    401     │ "Invalid token"          │
│ User not active         │    401     │ "Unauthorized"           │
│ Insufficient role       │    403     │ "Forbidden"              │
│ Wrong credentials       │    401     │ "Invalid credentials"    │
│ Resource not found      │    404     │ "Not found"              │
└─────────────────────────┴────────────┴──────────────────────────┘
```
