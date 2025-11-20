# Auth0 + gqlgen GraphQL Server

A GraphQL server built with [gqlgen](https://github.com/99designs/gqlgen) that integrates Auth0 passwordless email authentication.

## Features

- ✅ Auth0 JWT token validation
- ✅ Passwordless email authentication (OTP)
- ✅ GraphQL API with type-safe resolvers
- ✅ In-memory account storage
- ✅ Auto-create accounts on first login

## Prerequisites

- Go 1.21 or higher
- Auth0 account

## Auth0 Setup

### 1. Create an Auth0 Application

1. Go to [Auth0 Dashboard](https://manage.auth0.com/)
2. Navigate to **Applications** → **Applications**
3. Click **Create Application**
4. Name it (e.g., "GraphQL API")
5. Select **Single Page Web Applications**
6. Click **Create**

### 2. Enable Passwordless Email Authentication

1. In Auth0 Dashboard, go to **Authentication** → **Passwordless**
2. Click on **Email**
3. Toggle **Enable** to ON
4. Configure:
   - **OTP Length**: 6 digits
   - **OTP Expiry**: 300 seconds (5 minutes)
5. Click **Save**

### 3. Create an API

1. Go to **Applications** → **APIs**
2. Click **Create API**
3. Fill in:
   - **Name**: GraphQL API
   - **Identifier**: `https://your-api-identifier` (e.g., `https://graphql.example.com`)
   - **Signing Algorithm**: RS256
4. Click **Create**

### 4. Get Your Credentials

From your Auth0 Dashboard:

- **Domain**: Found in Application Settings (e.g., `dev-abc123.us.auth0.com`)
- **Audience**: The API Identifier you created (e.g., `https://graphql.example.com`)

## Installation

```bash
# Clone or create the project
git clone <repository-url>
cd auth0-gqlgen-demo

# Install dependencies
go mod download

# Set environment variables
export AUTH0_DOMAIN="your-domain.auth0.com"
export AUTH0_AUDIENCE="https://your-api-identifier"
export PORT="8080"

# Run the server
go run server.go
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AUTH0_DOMAIN` | Your Auth0 tenant domain | `dev-abc123.us.auth0.com` |
| `AUTH0_AUDIENCE` | Your Auth0 API identifier | `https://graphql.example.com` |
| `PORT` | Server port (optional, default: 8080) | `8080` |

## Testing the API

### 1. Get an Access Token (Passwordless Email)

#### Option A: Using Auth0's Universal Login (Recommended for Testing)

1. Go to: `https://YOUR_DOMAIN/authorize?response_type=token&client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:8080/callback&audience=YOUR_AUDIENCE&scope=openid%20email%20profile&connection=email`

Replace:
- `YOUR_DOMAIN`: Your Auth0 domain
- `YOUR_CLIENT_ID`: Found in Application Settings
- `YOUR_AUDIENCE`: Your API identifier

2. Enter your email address
3. Check your email for the code
4. Enter the code
5. Copy the `access_token` from the URL after redirect

#### Option B: Using Auth0 API (Programmatic)

```bash
# Step 1: Start passwordless flow
curl --request POST \
  --url 'https://YOUR_DOMAIN/passwordless/start' \
  --header 'content-type: application/json' \
  --data '{
    "client_id": "YOUR_CLIENT_ID",
    "connection": "email",
    "email": "user@example.com",
    "send": "code"
  }'

# Step 2: Verify code and get token
curl --request POST \
  --url 'https://YOUR_DOMAIN/oauth/token' \
  --header 'content-type: application/json' \
  --data '{
    "grant_type": "http://auth0.com/oauth/grant-type/passwordless/otp",
    "client_id": "YOUR_CLIENT_ID",
    "username": "user@example.com",
    "otp": "123456",
    "realm": "email",
    "audience": "YOUR_AUDIENCE",
    "scope": "openid email profile"
  }'
```

### 2. Test GraphQL Mutations & Queries

#### Create Account (First Login)

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "query": "mutation { createAccountIfNotExists { id email userId createdAt } }"
  }'
```

**Response:**
```json
{
  "data": {
    "createAccountIfNotExists": {
      "id": "1",
      "email": "user@example.com",
      "userId": "auth0|123456",
      "createdAt": "2024-11-20T10:30:00Z"
    }
  }
}
```

#### Get Account

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "query": "query { getAccount { id email userId createdAt } }"
  }'
```

**Response:**
```json
{
  "data": {
    "getAccount": {
      "id": "1",
      "email": "user@example.com",
      "userId": "auth0|123456",
      "createdAt": "2024-11-20T10:30:00Z"
    }
  }
}
```

### 3. Using GraphQL Playground

1. Open browser to `http://localhost:8080/`
2. Add HTTP Headers:
```json
{
  "Authorization": "Bearer YOUR_ACCESS_TOKEN"
}
```
3. Run queries:

```graphql
# Create or get existing account
mutation {
  createAccountIfNotExists {
    id
    email
    userId
    createdAt
  }
}

# Get current user's account
query {
  getAccount {
    id
    email
    userId
    createdAt
  }
}
```

## Project Structure

```
.
├── auth/
│   └── auth0.go           # Auth0 JWT validation middleware
├── graph/
│   ├── model/
│   │   └── models_gen.go  # Generated GraphQL models
│   ├── resolver.go         # Resolver root
│   └── schema.resolvers.go # Resolver implementations
├── store/
│   └── memory.go          # In-memory account storage
├── schema.graphql         # GraphQL schema definition
├── gqlgen.yml             # gqlgen configuration
├── server.go              # HTTP server entrypoint
├── go.mod
└── README.md
```

## How It Works

### Authentication Flow

1. **User Login (Passwordless Email)**:
   - User enters email in your app
   - Auth0 sends a one-time code to their email
   - User enters the code
   - Auth0 returns a JWT access token

2. **API Request**:
   - Client sends GraphQL request with `Authorization: Bearer <token>` header
   - Server validates JWT against Auth0's JWKS
   - Server extracts `sub` (user ID) and `email` from token claims
   - Server adds user info to request context

3. **Account Creation**:
   - First time: `createAccountIfNotExists` creates a new account
   - Subsequent times: Returns existing account
   - All tied to Auth0's `sub` claim (unique user ID)

### Security

- ✅ JWT signature verification using Auth0's public keys (JWKS)
- ✅ Audience (`aud`) claim validation
- ✅ Issuer (`iss`) claim validation
- ✅ Expiration (`exp`) claim validation
- ✅ No password storage (passwordless authentication)

## Development

### Regenerate GraphQL Code

After modifying `schema.graphql`:

```bash
go run github.com/99designs/gqlgen generate
```

### Run Tests

```bash
go test ./...
```

## Troubleshooting

### "Authorization header required"
- Make sure you're including the `Authorization: Bearer <token>` header

### "Invalid token"
- Token might be expired (default: 24 hours)
- Get a fresh token from Auth0
- Verify `AUTH0_DOMAIN` and `AUTH0_AUDIENCE` are correct

### "unable to find appropriate key"
- Auth0's JWKS might not be accessible
- Check your internet connection
- Verify `AUTH0_DOMAIN` is correct (no `https://` prefix)

### "account not found"
- Call `createAccountIfNotExists` mutation first
- Ensures user account exists in the system

## License

MIT

