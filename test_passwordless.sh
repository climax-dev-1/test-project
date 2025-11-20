#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Auth0 Configuration (from env.example)
AUTH0_DOMAIN="${AUTH0_DOMAIN:-dev-4skztpo8mxaq8d12.us.auth0.com}"
AUTH0_AUDIENCE="${AUTH0_AUDIENCE:-https://dev-4skztpo8mxaq8d12.us.auth0.com/api/v2/}"
CLIENT_ID="${CLIENT_ID:-HFGjKwNvymtShqtohLZhVeN8s9UjPRDi}"
CLIENT_SECRET="${CLIENT_SECRET:-f6twKJm4m47d-o3lQK5JkDbbVfXe6rhHHmnFApBQMhyL3IRZrjD0GGF87QueN5IF}"
GRAPHQL_URL="${GRAPHQL_URL:-http://localhost:8080/query}"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Passwordless OTP Authentication Test${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "  Auth0 Domain: $AUTH0_DOMAIN"
echo "  Auth0 Audience: $AUTH0_AUDIENCE"
echo "  Client ID: $CLIENT_ID"
echo "  Client Secret: ${CLIENT_SECRET:0:20}...${CLIENT_SECRET: -10}"
echo "  GraphQL URL: $GRAPHQL_URL"
echo ""

# Check if email is provided as argument
if [ -n "$1" ]; then
    EMAIL="$1"
else
    echo -n "Enter your email address: "
    read EMAIL
fi

if [ -z "$EMAIL" ]; then
    echo -e "${RED}Error: Email address is required${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Step 1: Requesting OTP code...${NC}"
echo "Sending code to: $EMAIL"
echo ""

RESPONSE=$(curl -s --request POST \
  --url "https://$AUTH0_DOMAIN/passwordless/start" \
  --header 'content-type: application/json' \
  --data "{
    \"client_id\": \"$CLIENT_ID\",
    \"client_secret\": \"$CLIENT_SECRET\",
    \"connection\": \"email\",
    \"email\": \"$EMAIL\",
    \"send\": \"code\"
  }")

echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q "error"; then
    echo -e "${RED}❌ Error requesting OTP code!${NC}"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  1. Passwordless Email not enabled in Auth0 Dashboard"
    echo "     → Go to: Authentication > Passwordless > Email"
    echo "     → Toggle ON and save"
    echo ""
    echo "  2. Connection not enabled for your application"
    echo "     → Go to: Applications > Your App > Connections"
    echo "     → Enable 'email' connection"
    echo ""
    echo "  3. Email provider not configured"
    echo "     → Auth0's default email service is unreliable"
    echo "     → Consider setting up SendGrid, Mailgun, or custom SMTP"
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ OTP request successful!${NC}"
echo -e "${YELLOW}📧 Check your email for the verification code!${NC}"
echo ""
echo -e "${BLUE}Note:${NC} If you don't receive the email:"
echo "  • Check spam/promotions folder"
echo "  • Wait 5-10 minutes (Auth0 free tier can be slow)"
echo "  • Try a different email address"
echo "  • Use Client Credentials mode instead (no email needed)"
echo ""
echo -n "Enter the 6-digit code: "
read OTP_CODE

if [ -z "$OTP_CODE" ]; then
    echo -e "${RED}Error: OTP code is required${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Step 2: Exchanging OTP for access token...${NC}"
echo ""

TOKEN_RESPONSE=$(curl -s --request POST \
  --url "https://$AUTH0_DOMAIN/oauth/token" \
  --header 'content-type: application/json' \
  --data "{
    \"grant_type\": \"http://auth0.com/oauth/grant-type/passwordless/otp\",
    \"client_id\": \"$CLIENT_ID\",
    \"client_secret\": \"$CLIENT_SECRET\",
    \"username\": \"$EMAIL\",
    \"otp\": \"$OTP_CODE\",
    \"realm\": \"email\",
    \"audience\": \"$AUTH0_AUDIENCE\",
    \"scope\": \"openid email profile\"
  }")

echo "$TOKEN_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$TOKEN_RESPONSE"
echo ""

if echo "$TOKEN_RESPONSE" | grep -q "error"; then
    echo -e "${RED}❌ Error getting access token!${NC}"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  • OTP code is incorrect"
    echo "  • OTP code has expired (valid for 5 minutes)"
    echo "  • Try requesting a new code"
    exit 1
fi

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('access_token', ''))" 2>/dev/null)

if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}Failed to extract access token!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Successfully obtained access token!${NC}"
echo "Token: ${ACCESS_TOKEN:0:50}..."
echo ""

# Test if server is running
echo -e "${BLUE}Checking if GraphQL server is running...${NC}"
if ! curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ | grep -q "200"; then
    echo -e "${YELLOW}⚠️  Warning: GraphQL server may not be running on port 8080${NC}"
    echo "Start the server with: AUTH0_DOMAIN=$AUTH0_DOMAIN AUTH0_AUDIENCE=$AUTH0_AUDIENCE go run server.go"
    echo ""
fi

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Testing GraphQL API${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

echo -e "${GREEN}Step 3: Testing createAccountIfNotExists mutation...${NC}"
echo ""

CREATE_RESPONSE=$(curl -s -X POST "$GRAPHQL_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"query": "mutation { createAccountIfNotExists { id email userId createdAt } }"}')

echo "$CREATE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CREATE_RESPONSE"
echo ""

if echo "$CREATE_RESPONSE" | grep -q '"data"'; then
    echo -e "${GREEN}✅ Account created/retrieved successfully!${NC}"
else
    echo -e "${RED}❌ Error creating account!${NC}"
    if echo "$CREATE_RESPONSE" | grep -q "Invalid token"; then
        echo -e "${YELLOW}The server may be configured for a different Auth0 domain.${NC}"
    fi
fi

echo ""
echo -e "${GREEN}Step 4: Testing getAccount query...${NC}"
echo ""

GET_RESPONSE=$(curl -s -X POST "$GRAPHQL_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"query": "query { getAccount { id email userId createdAt } }"}')

echo "$GET_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$GET_RESPONSE"
echo ""

if echo "$GET_RESPONSE" | grep -q '"data"'; then
    echo -e "${GREEN}✅ Account retrieved successfully!${NC}"
else
    echo -e "${RED}❌ Error retrieving account!${NC}"
fi

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Testing Complete!${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "${BLUE}User Information:${NC}"
echo "  Email: $EMAIL"
echo "  Access Token: ${ACCESS_TOKEN:0:30}...${ACCESS_TOKEN: -20}"
echo ""
echo -e "${BLUE}Full Access Token (for manual testing):${NC}"
echo "$ACCESS_TOKEN"
echo ""

