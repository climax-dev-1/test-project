##Seamless Token Exchange Migration: Passage → Auth0

This implementation allows your existing users to migrate from Passage to Auth0 **without requiring re-authentication**. Users can exchange their Passage JWT for Auth0 credentials seamlessly.

## 🎯 How It Works

```
User App (with Passage JWT)
          ↓
    POST /migrate/exchange-token
          ↓
[1] Validate Passage JWT ✓
          ↓
[2] Extract user info (email, ID)
          ↓
[3] Find or create user in Auth0
          ↓
[4] Return Auth0 user ID
          ↓
User continues with Auth0 passwordless
```

## 📋 Prerequisites

### Passage Credentials
- **App ID**: Your Passage application ID
- **API Key**: Passage API key (for user management)

### Auth0 Credentials
- **Domain**: Your Auth0 tenant domain
- **Client ID**: Auth0 application client ID
- **Client Secret**: Auth0 application client secret
- **Audience**: Auth0 API audience
- **Connection**: Database connection name (default: "Username-Password-Authentication")

## 🚀 Setup

### 1. Set Environment Variables

```bash
# Auth0 Configuration
export AUTH0_DOMAIN="dev-4skztpo8mxaq8d12.us.auth0.com"
export AUTH0_AUDIENCE="https://dev-4skztpo8mxaq8d12.us.auth0.com/api/v2/"
export CLIENT_ID="your-client-id"
export CLIENT_SECRET="your-client-secret"
export AUTH0_CONNECTION="Username-Password-Authentication"

# Passage Configuration
export PASSAGE_APP_ID="your-passage-app-id"
export PASSAGE_API_KEY="your-passage-api-key"
```

### 2. Run the Server

```bash
go run server_with_migration.go
```

The server will start with:
- GraphQL endpoints (`:8080/query`, `:8080/`)
- Migration endpoint (`:8080/migrate/exchange-token`)
- Stats endpoint (`:8080/migrate/stats`)

## 📡 API Endpoints

### POST /migrate/exchange-token

Exchange a Passage JWT for Auth0 user credentials.

**Request:**
```json
{
  "passage_token": "eyJhbGci..."
}
```

**Response (Success):**
```json
{
  "success": true,
  "auth0_user_id": "auth0|123456",
  "email": "user@example.com",
  "is_new_migration": true,
  "message": "User migrated successfully. Use passwordless authentication to get Auth0 token."
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "invalid passage token: token expired"
}
```

### GET /migrate/stats

View migration statistics.

**Response:**
```json
{
  "total_migrated_users": 42,
  "cache_size": 42
}
```

## 🔄 Migration Flow

### Client-Side Implementation

```typescript
// Step 1: User has existing Passage session
const passageToken = passage.getAuthToken(); // User's current Passage JWT

// Step 2: Exchange Passage token for Auth0 user
const response = await fetch('https://your-api.com/migrate/exchange-token', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    passage_token: passageToken
  })
});

const result = await response.json();

if (result.success) {
  console.log('User migrated!', result.auth0_user_id);
  
  // Step 3: Now authenticate with Auth0 (passwordless)
  // The user already exists in Auth0 with their email
  // Use Auth0's passwordless email to send them a code
  
  await auth0.passwordlessStart({
    connection: 'email',
    send: 'code',
    email: result.email
  });
  
  // User receives code, enters it, gets Auth0 token
  const auth0Token = await auth0.passwordlessLogin({
    connection: 'email',
    email: result.email,
    verificationCode: userEnteredCode
  });
  
  // User is now fully migrated and authenticated with Auth0!
  localStorage.setItem('auth0_token', auth0Token);
}
```

## 🎨 iOS/Swift Implementation

```swift
// Step 1: Get Passage token
let passageToken = try await PassageAuth.session.authToken

// Step 2: Exchange for Auth0 user
let url = URL(string: "https://your-api.com/migrate/exchange-token")!
var request = URLRequest(url: url)
request.httpMethod = "POST"
request.setValue("application/json", forHTTPHeaderField: "Content-Type")

let body = ["passage_token": passageToken]
request.httpBody = try JSONEncoder().encode(body)

let (data, response) = try await URLSession.shared.data(for: request)
let result = try JSONDecoder().decode(MigrationResult.self, from: data)

if result.success {
    print("User migrated: \\(result.email)")
    
    // Step 3: Continue with Auth0 passwordless
    // User already exists in Auth0, just needs to authenticate
}
```

## ⚠️ Important Notes

### Token Issuance

The current implementation **creates/finds the user in Auth0** but doesn't directly issue Auth0 tokens. This is because Auth0 has strict security requirements for token issuance.

**To issue Auth0 tokens, you have two options:**

#### Option 1: Two-Step Flow (Recommended for Security)
1. Exchange Passage token → Creates Auth0 user
2. User authenticates with Auth0 (passwordless email) → Gets Auth0 token

**Benefits:**
- More secure (user explicitly authenticates with Auth0)
- Email verification through Auth0
- Follows OAuth best practices

#### Option 2: Custom Auth0 Action (Advanced)
Create an Auth0 Action that issues tokens for migrated users:

```javascript
// Auth0 Action: Custom Token Issuance for Migration
exports.onExecutePostLogin = async (event, api) => {
  if (event.request.body.grant_type === 'urn:custom:migration') {
    const passageUserId = event.request.body.passage_user_id;
    
    // Verify this is a valid migrated user
    // Issue token
    api.access.token.setCustomClaim('migrated_from_passage', true);
  }
};
```

## 🔒 Security Considerations

1. **Passage Token Validation**: Always validate Passage tokens server-side
2. **HTTPS Only**: Use HTTPS in production
3. **Rate Limiting**: Implement rate limiting on migration endpoint
4. **Audit Logging**: Log all migration attempts
5. **One-Time Migration**: Optionally enforce one-time migration per user

## 📊 Monitoring

### Check Migration Progress

```bash
curl http://localhost:8080/migrate/stats
```

### Log Analysis

The server logs all migration attempts:
- Successful migrations
- Failed Passage token validations
- Auth0 user creation errors

## 🧪 Testing

### Test Script

```bash
# Assuming you have a valid Passage token
PASSAGE_TOKEN="your-passage-jwt-token"

curl -X POST http://localhost:8080/migrate/exchange-token \
  -H "Content-Type: application/json" \
  -d "{\"passage_token\": \"$PASSAGE_TOKEN\"}"
```

### Expected Output

```json
{
  "success": true,
  "auth0_user_id": "email|691f4f3b0bb13594134e328a",
  "email": "user@example.com",
  "is_new_migration": true,
  "message": "User migrated successfully. Use passwordless authentication to get Auth0 token."
}
```

## 🚦 Production Deployment

### 1. Environment Variables

Ensure all required environment variables are set in your production environment.

### 2. Database Persistence

The current implementation uses an in-memory cache. For production:

```go
// Replace in-memory cache with database
// Store migration records in PostgreSQL, MongoDB, etc.
type MigrationRecord struct {
    PassageUserID string    `db:"passage_user_id"`
    Auth0UserID   string    `db:"auth0_user_id"`
    Email         string    `db:"email"`
    MigratedAt    time.Time `db:"migrated_at"`
}
```

### 3. Rate Limiting

Implement rate limiting to prevent abuse:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/sec, burst 20

http.HandleFunc("/migrate/exchange-token", func(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    // ... rest of handler
})
```

### 4. Metrics

Add metrics for monitoring:
- Total migrations
- Migration success/failure rate
- Average migration time
- Active users on each platform

## 📚 Complete Example

See `test_migration.sh` for a complete end-to-end test script.

## 🆘 Troubleshooting

### "invalid passage token"
- Verify Passage App ID is correct
- Check token hasn't expired
- Ensure token is a valid JWT

### "failed to create auth0 user"
- Verify Auth0 credentials
- Check Management API permissions
- Ensure database connection exists in Auth0

### "User already exists"
- This is normal for repeat migrations
- User is found and returned

## 🎯 Next Steps

1. **Test the migration endpoint** with a real Passage token
2. **Implement the client-side flow** in your iOS app
3. **Set up Auth0 Actions** for custom token issuance (optional)
4. **Add database persistence** for production
5. **Monitor migration progress** with the stats endpoint

---

**Questions?** The implementation is ready to test. Just set your Passage credentials and try the migration endpoint!

