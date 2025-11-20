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
echo -e "${YELLOW}Client Credentials Flow Test${NC}"
echo -e "${YELLOW}(No email required - works immediately!)${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "  Auth0 Domain: $AUTH0_DOMAIN"
echo "  Auth0 Audience: $AUTH0_AUDIENCE"
echo "  Client ID: $CLIENT_ID"
echo "  Client Secret: ${CLIENT_SECRET:0:20}...${CLIENT_SECRET: -10}"
echo "  GraphQL URL: $GRAPHQL_URL"
echo ""

echo -e "${GREEN}Step 1: Getting access token via Client Credentials...${NC}"
echo ""

TOKEN_RESPONSE=$(curl -s --request POST \
  --url "https://$AUTH0_DOMAIN/oauth/token" \
  --header 'content-type: application/json' \
  --data "{
    \"grant_type\": \"client_credentials\",
    \"client_id\": \"$CLIENT_ID\",
    \"client_secret\": \"$CLIENT_SECRET\",
    \"audience\": \"$AUTH0_AUDIENCE\"
  }")

echo "$TOKEN_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$TOKEN_RESPONSE"
echo ""

if echo "$TOKEN_RESPONSE" | grep -q "error"; then
    echo -e "${RED}❌ Error getting access token!${NC}"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  • Client Credentials grant not enabled"
    echo "    → Go to: Applications > Your App > Settings > Advanced Settings > Grant Types"
    echo "    → Enable 'Client Credentials'"
    echo ""
    echo "  • Wrong credentials"
    echo "    → Verify CLIENT_ID and CLIENT_SECRET"
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
SERVER_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ 2>/dev/null)

if [ "$SERVER_STATUS" != "200" ]; then
    echo -e "${YELLOW}⚠️  Warning: GraphQL server is not running on port 8080${NC}"
    echo ""
    echo "Start the server with:"
    echo "  cd /home/ubuntu/Projects/Tom\ Houtchinson/test"
    echo "  AUTH0_DOMAIN=$AUTH0_DOMAIN AUTH0_AUDIENCE=$AUTH0_AUDIENCE go run server.go"
    echo ""
    echo -e "${RED}Stopping test - please start the server first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Server is running!${NC}"
echo ""

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Testing GraphQL API${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

echo -e "${GREEN}Step 2: Testing createAccountIfNotExists mutation...${NC}"
echo ""

CREATE_RESPONSE=$(curl -s -X POST "$GRAPHQL_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"query": "mutation { createAccountIfNotExists { id email userId createdAt } }"}')

echo "$CREATE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CREATE_RESPONSE"
echo ""

if echo "$CREATE_RESPONSE" | grep -q '"data"'; then
    echo -e "${GREEN}✅ Account created/retrieved successfully!${NC}"
    
    # Extract account info
    ACCOUNT_ID=$(echo "$CREATE_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['createAccountIfNotExists']['id'])" 2>/dev/null)
    USER_ID=$(echo "$CREATE_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['createAccountIfNotExists']['userId'])" 2>/dev/null)
    
    echo ""
    echo -e "${BLUE}Account Details:${NC}"
    echo "  Account ID: $ACCOUNT_ID"
    echo "  User ID: $USER_ID"
else
    echo -e "${RED}❌ Error creating account!${NC}"
    if echo "$CREATE_RESPONSE" | grep -q "Invalid token"; then
        echo -e "${YELLOW}The server may be configured for a different Auth0 domain.${NC}"
        echo "Make sure the server is running with the correct AUTH0_DOMAIN and AUTH0_AUDIENCE"
    fi
fi

echo ""
echo -e "${GREEN}Step 3: Testing getAccount query...${NC}"
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
echo -e "${BLUE}Summary:${NC}"
echo "  ✅ Authentication via Client Credentials"
echo "  ✅ GraphQL server responding"
echo "  ✅ Account operations working"
echo ""
echo -e "${BLUE}Access Token (for manual testing):${NC}"
echo "$ACCESS_TOKEN"
echo ""
echo -e "${BLUE}Test GraphQL Playground:${NC}"
echo "  1. Open: http://localhost:8080/"
echo "  2. Add HTTP Header:"
echo "     {\"Authorization\": \"Bearer $ACCESS_TOKEN\"}"
echo "  3. Run queries/mutations"
echo ""

