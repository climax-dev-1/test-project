#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Passage → Auth0 Migration Test${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Check if Passage token is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Passage token is required${NC}"
    echo ""
    echo "Usage: $0 <PASSAGE_JWT_TOKEN>"
    echo ""
    echo "Example:"
    echo "  $0 eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    echo ""
    echo "To get a Passage token:"
    echo "  1. Log in to your app with Passage"
    echo "  2. Extract the JWT token from local storage or your Passage SDK"
    echo "  3. Pass it as an argument to this script"
    exit 1
fi

PASSAGE_TOKEN="$1"
MIGRATION_URL="${MIGRATION_URL:-http://localhost:8080/migrate/exchange-token}"
STATS_URL="${STATS_URL:-http://localhost:8080/migrate/stats}"

echo -e "${BLUE}Configuration:${NC}"
echo "  Migration URL: $MIGRATION_URL"
echo "  Stats URL: $STATS_URL"
echo "  Token: ${PASSAGE_TOKEN:0:30}...${PASSAGE_TOKEN: -20}"
echo ""

echo -e "${GREEN}Step 1: Exchanging Passage token for Auth0 user...${NC}"
echo ""

RESPONSE=$(curl -s -X POST "$MIGRATION_URL" \
  -H "Content-Type: application/json" \
  -d "{\"passage_token\": \"$PASSAGE_TOKEN\"}")

echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

# Check if migration was successful
if echo "$RESPONSE" | grep -q '"success": *true'; then
    echo -e "${GREEN}✅ Migration successful!${NC}"
    echo ""
    
    # Extract user info
    AUTH0_USER_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('auth0_user_id', ''))" 2>/dev/null)
    EMAIL=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('email', ''))" 2>/dev/null)
    IS_NEW=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('is_new_migration', False))" 2>/dev/null)
    
    echo -e "${BLUE}User Information:${NC}"
    echo "  Auth0 User ID: $AUTH0_USER_ID"
    echo "  Email: $EMAIL"
    echo "  First Migration: $IS_NEW"
    echo ""
    
    if [ "$IS_NEW" = "True" ]; then
        echo -e "${YELLOW}🎉 This is a new migration!${NC}"
        echo "The user has been created in Auth0."
    else
        echo -e "${BLUE}ℹ️  User was previously migrated${NC}"
        echo "Existing Auth0 user returned."
    fi
    echo ""
    
    echo -e "${GREEN}Step 2: Fetching migration statistics...${NC}"
    echo ""
    
    STATS=$(curl -s "$STATS_URL")
    echo "$STATS" | python3 -m json.tool 2>/dev/null || echo "$STATS"
    echo ""
    
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${GREEN}Next Steps for Complete Migration${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo "The user now exists in Auth0, but needs an Auth0 token."
    echo ""
    echo "Option 1: Passwordless Email (Recommended)"
    echo "  1. Send OTP to user's email:"
    echo "     curl -X POST 'https://dev-4skztpo8mxaq8d12.us.auth0.com/passwordless/start' \\"
    echo "       -H 'Content-Type: application/json' \\"
    echo "       -d '{\"client_id\":\"CLIENT_ID\",\"connection\":\"email\",\"email\":\"$EMAIL\",\"send\":\"code\"}'"
    echo ""
    echo "  2. User enters code"
    echo "  3. Exchange code for Auth0 token"
    echo ""
    echo "Option 2: Implement Auth0 Action"
    echo "  - Create a custom Auth0 Action to issue tokens for migrated users"
    echo "  - See MIGRATION_GUIDE.md for details"
    echo ""
    echo "Option 3: Test with Client Credentials"
    echo "  - Run: ./test_client_credentials.sh"
    echo "  - This bypasses user authentication for testing"
    echo ""
    
else
    echo -e "${RED}❌ Migration failed!${NC}"
    echo ""
    
    ERROR_MSG=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('message', 'Unknown error'))" 2>/dev/null)
    echo -e "${YELLOW}Error:${NC} $ERROR_MSG"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  1. Invalid Passage token"
    echo "     → Verify the token is current and valid"
    echo "     → Check token hasn't expired"
    echo ""
    echo "  2. Passage credentials not configured"
    echo "     → Set PASSAGE_APP_ID environment variable"
    echo "     → Set PASSAGE_API_KEY environment variable"
    echo ""
    echo "  3. Auth0 credentials not configured"
    echo "     → Set CLIENT_ID and CLIENT_SECRET"
    echo "     → Verify Management API permissions"
    echo ""
    echo "  4. Server not running with migration enabled"
    echo "     → Run: go run server_with_migration.go"
    echo "     → Check server logs for errors"
    echo ""
    exit 1
fi

echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Migration Test Complete!${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

