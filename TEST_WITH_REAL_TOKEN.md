# Testing Migration with Real Passage Token

## ✅ Server Status: **RUNNING**

The migration server is successfully running with your Passage credentials:
- **Passage App ID**: `oo7jMcL8o84ecGkZHNxUQQhv`
- **Passage API Key**: Configured ✓
- **Migration endpoint**: `http://localhost:8080/migrate/exchange-token`
- **Stats endpoint**: `http://localhost:8080/migrate/stats`

## 🧪 How to Get a Passage Token for Testing

### Option 1: From Your iOS App

```swift
import PassageKit

// Get the current auth token
let token = try await PassageAuth.session.authToken
print("Passage Token: \\(token)")
```

### Option 2: From Web App

```javascript
// If using Passage JS SDK
const token = passage.getAuthToken();
console.log('Passage Token:', token);
```

### Option 3: From Browser DevTools

1. Open your app that uses Passage authentication
2. Log in with Passage
3. Open Browser DevTools (F12)
4. Go to **Application** > **Local Storage** or **Session Storage**
5. Look for a key containing "passage" or "auth_token"
6. Copy the JWT token value

### Option 4: Using Passage API Directly

```bash
# This would require implementing Passage's authentication flow
# See: https://docs.passage.id/
```

## 🚀 Test the Migration

Once you have a real Passage JWT token:

### Method 1: Using the Test Script

```bash
./test_migration.sh "YOUR_PASSAGE_JWT_TOKEN_HERE"
```

### Method 2: Using cURL

```bash
curl -X POST http://localhost:8080/migrate/exchange-token \
  -H "Content-Type: application/json" \
  -d '{"passage_token": "YOUR_PASSAGE_JWT_TOKEN_HERE"}' | python3 -m json.tool
```

## 📊 Expected Response

### Success Response

```json
{
  "success": true,
  "auth0_user_id": "email|abc123...",
  "email": "user@example.com",
  "is_new_migration": true,
  "message": "User migrated successfully. Use passwordless authentication to get Auth0 token."
}
```

### Error Response (Invalid Token)

```json
{
  "success": false,
  "is_new_migration": false,
  "message": "invalid passage token: token is malformed..."
}
```

## 🔍 Verify Migration

After successful migration, check:

### 1. Migration Stats

```bash
curl http://localhost:8080/migrate/stats | python3 -m json.tool
```

Expected output:
```json
{
  "cache_size": 1,
  "total_migrated_users": 1
}
```

### 2. Auth0 Dashboard

1. Go to https://manage.auth0.com
2. Navigate to **User Management** > **Users**
3. Search for the user's email
4. You should see the newly created user

### 3. GraphQL API

After migration, the user can authenticate with Auth0 and use the GraphQL API:

```bash
# 1. Get Auth0 token (passwordless)
curl -X POST https://dev-4skztpo8mxaq8d12.us.auth0.com/passwordless/start \
  -H 'Content-Type: application/json' \
  -d '{
    "client_id": "HFGjKwNvymtShqtohLZhVeN8s9UjPRDi",
    "client_secret": "f6twKJm4m47d-o3lQK5JkDbbVfXe6rhHHmnFApBQMhyL3IRZrjD0GGF87QueN5IF",
    "connection": "email",
    "email": "user@example.com",
    "send": "code"
  }'

# 2. User receives code and exchanges it for token
# (See test_passwordless.sh for full flow)

# 3. Use token with GraphQL API
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer AUTH0_TOKEN" \
  -d '{"query": "mutation { createAccountIfNotExists { id email userId } }"}'
```

## 🐛 Troubleshooting

### "invalid passage token: token is malformed"
- Token is not a valid JWT
- Check you copied the entire token
- Ensure no extra spaces or line breaks

### "invalid passage token: token has expired"
- The Passage token has expired
- Log in again to get a fresh token
- Passage tokens typically expire after a certain time

### "invalid passage token: signature verification failed"
- The token was not signed by your Passage app
- Verify PASSAGE_APP_ID is correct
- Check the token is from the right environment

### "passage user has no email or phone"
- The Passage user doesn't have an email or phone
- This shouldn't happen for normal users
- Check the user profile in Passage dashboard

### "failed to create auth0 user"
- Check CLIENT_ID and CLIENT_SECRET are correct
- Verify Auth0 Management API permissions
- Ensure database connection exists in Auth0

## 📝 Server Logs

To see real-time logs:

```bash
tail -f server_migration.log
```

## 🔄 Restart Server

If you need to restart the server:

```bash
# Kill existing server
lsof -ti:8080 | xargs kill -9

# Start again
cd /home/ubuntu/Projects/Tom\ Houtchinson/test
AUTH0_DOMAIN="dev-4skztpo8mxaq8d12.us.auth0.com" \
AUTH0_AUDIENCE="https://dev-4skztpo8mxaq8d12.us.auth0.com/api/v2/" \
CLIENT_ID="HFGjKwNvymtShqtohLZhVeN8s9UjPRDi" \
CLIENT_SECRET="f6twKJm4m47d-o3lQK5JkDbbVfXe6rhHHmnFApBQMhyL3IRZrjD0GGF87QueN5IF" \
PASSAGE_APP_ID="oo7jMcL8o84ecGkZHNxUQQhv" \
PASSAGE_API_KEY="twqsfd70i4.rdoXs6RkPcO3VAiuhOH9aEV50a3EKSw0KesDjstvJVYsiy4Eg7freyKK7QhjNZNX" \
./server_migration
```

## 🎯 Next Steps

1. **Get a real Passage JWT token** from your app
2. **Run the test**: `./test_migration.sh "YOUR_TOKEN"`
3. **Verify** the user was created in Auth0
4. **Implement** the client-side migration flow in your iOS app
5. **Test** end-to-end migration in your app

---

**The server is ready and waiting for your real Passage token to test!** 🚀

