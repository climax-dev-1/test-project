# Migration Strategy: Passage → Auth0

## Overview
This strategy focuses on migrating from Passage by 1Password to Auth0, leveraging Auth0's powerful migration features, extensive identity provider support, and managed authentication flows for a seamless transition.

---

## 1. Pre-Migration Assessment

### 1.1 Data Inventory
- **User Records**: All user accounts in Passage (email, user IDs, profile data)
- **Authentication Methods**: Passkeys, biometric registrations, device trust
- **Social Connections**: Existing social login providers (if any)
- **User Metadata**: Custom user properties, roles, permissions
- **Active Sessions**: Current session management and token lifetimes

### 1.2 Architecture Analysis
- **Current Flow**: Passage SDK → iOS App → Go Backend → GraphQL API
- **Token Format**: Passage JWT structure and claims
- **Authorization Model**: Role-based access control implementation
- **Integration Points**: All services consuming Passage authentication

---

## 2. Migration Architecture

### 2.1 Auth0 Migration Advantages
Auth0 provides **superior migration capabilities** compared to other providers:

- **Automatic User Migration**: Native "Lazy Migration" feature via custom database connection
- **No Token Downtime**: Instant token generation after validation
- **Password Management**: Built-in secure password hashing migration
- **Social Connection Mapping**: Seamless linking of social accounts
- **Universal Login**: Consistent experience across all platforms

### 2.2 Migration Approach: Lazy Migration (Recommended)

Auth0's **Custom Database** feature allows migrating users on-demand without pre-importing:

**How It Works:**
1. Configure Auth0 custom database connection pointing to Passage API
2. User attempts login via Auth0
3. Auth0 checks if user exists in Auth0 database
4. If not found → Auth0 calls your **Login Script** (validates with Passage)
5. If Passage validates successfully → Auth0 imports user automatically
6. User receives Auth0 token, fully migrated
7. Next login: User exists in Auth0, Passage not called

**Key Benefit**: Zero user disruption, no mass migrations, natural sunsetting of Passage

---

## 3. Detailed Migration Plan

### Phase 1: Auth0 Configuration (Week 1-2)

#### 3.1 Tenant Setup
- **Region Selection**: Choose closest to your Go backend (US, EU, AU)
- **Environment**: Production tenant + Development tenant for testing
- **Custom Domain**: Configure `auth.yourdomain.com` (better UX, no Auth0 branding)

#### 3.2 Application Configuration
- **Create Applications**:
  - **Native Application**: iOS app (OAuth 2.0 PKCE flow)
  - **Machine to Machine**: Go backend (for Management API access)
  
- **Connection Settings**:
  - Enable "Username-Password-Authentication" database
  - Configure as Custom Database (for lazy migration)
  - Set password policies (min length, complexity)
  
- **Advanced Settings**:
  - Token expiration: Access token (24h), Refresh token (30d)
  - Grant types: Authorization Code, Refresh Token
  - Allowed Callback URLs: `com.yourapp://callback`
  - Allowed Logout URLs: `com.yourapp://logout`

#### 3.3 Custom Database Scripts

Auth0 requires two Node.js scripts for lazy migration:

**1. Login Script** (validates user with Passage):
```javascript
function login(email, password, callback) {
  const axios = require('axios');
  
  // Validate with Passage API
  axios.post('https://auth.passage.id/v1/apps/YOUR_APP_ID/login', {
    identifier: email,
    // Note: Passage uses passkeys, not passwords
    // This requires adaptation based on your Passage setup
  }, {
    headers: {
      'Authorization': 'Bearer YOUR_PASSAGE_API_KEY'
    }
  })
  .then(response => {
    // Passage validated successfully
    const user = response.data;
    
    // Return user profile to Auth0
    // Auth0 will automatically create this user in its database
    callback(null, {
      user_id: user.id,
      email: user.email,
      email_verified: user.email_verified,
      name: user.user_metadata?.name,
      // Migrate custom attributes
      user_metadata: {
        passage_user_id: user.id,
        migrated_at: new Date().toISOString()
      }
    });
  })
  .catch(error => {
    callback(error);
  });
}
```

**2. Get User Script** (fetch user profile):
```javascript
function getByEmail(email, callback) {
  const axios = require('axios');
  
  axios.get(`https://auth.passage.id/v1/apps/YOUR_APP_ID/users?identifier=${email}`, {
    headers: {
      'Authorization': 'Bearer YOUR_PASSAGE_API_KEY'
    }
  })
  .then(response => {
    const user = response.data;
    if (!user) return callback(null);
    
    callback(null, {
      user_id: user.id,
      email: user.email,
      email_verified: user.email_verified,
      name: user.user_metadata?.name
    });
  })
  .catch(error => {
    callback(error);
  });
}
```

**Challenge**: Passage uses **passwordless** authentication (passkeys/biometrics), not password-based. This requires an adaptation strategy (see Section 4.1).

#### 3.4 Rules & Actions
Auth0 Actions (similar to Cognito Lambdas) allow customization:

**Post-Login Action** (add custom claims):
```javascript
exports.onExecutePostLogin = async (event, api) => {
  const namespace = 'https://yourdomain.com';
  
  // Fetch user roles from your database
  const roles = await fetchUserRoles(event.user.user_id);
  
  // Add custom claims to token
  api.idToken.setCustomClaim(`${namespace}/roles`, roles);
  api.accessToken.setCustomClaim(`${namespace}/roles`, roles);
  
  // Track migration
  if (event.user.user_metadata?.passage_user_id) {
    await logMigration(event.user.user_id, 'completed');
  }
};
```

#### 3.5 Social Connections (Optional)
If users had social logins with Passage:
- Configure Google, Apple, Facebook, etc. in Auth0
- Map social account emails to migrated users
- Auth0 automatically links accounts by email

---

### Phase 2: Backend Integration (Week 2-3)

#### 3.6 Go Backend Middleware
**Dual-Authentication Period** (supports both Passage and Auth0):

```go
package auth

import (
  "context"
  "github.com/auth0/go-jwt-middleware/v2"
  "github.com/auth0/go-jwt-middleware/v2/jwks"
)

type AuthService struct {
  auth0Validator   *jwtmiddleware.JWTMiddleware
  passageValidator *PassageValidator
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (*User, error) {
  // Try Auth0 first (new system)
  if user, err := s.validateAuth0Token(token); err == nil {
    return user, nil
  }
  
  // Fallback to Passage (legacy)
  if user, err := s.passageValidator.Validate(token); err == nil {
    // Log that user hasn't migrated yet
    // They will migrate on next login via Auth0
    return user, nil
  }
  
  return nil, ErrUnauthorized
}

func (s *AuthService) validateAuth0Token(token string) (*User, error) {
  // Use Auth0's Go JWT middleware
  // Validates signature against Auth0 JWKS endpoint
  // Extracts claims and returns user
}
```

#### 3.7 Management API Integration
For user lookup in back-office:

```go
import "github.com/auth0/go-auth0/management"

func NewAuth0Client() (*management.Management, error) {
  return management.New(
    "your-tenant.auth0.com",
    management.WithClientCredentials(clientID, clientSecret),
  )
}

func FindUserByEmail(email string) (*management.User, error) {
  client, _ := NewAuth0Client()
  
  users, err := client.User.ListByEmail(email)
  if err != nil {
    return nil, err
  }
  
  if len(users) == 0 {
    return nil, ErrUserNotFound
  }
  
  return users[0], nil
}
```

---

### Phase 3: iOS App Integration (Week 3-4)

#### 3.8 Auth0.swift SDK Integration

**Installation** (Swift Package Manager):
```swift
dependencies: [
  .package(url: "https://github.com/auth0/Auth0.swift", from: "2.0.0")
]
```

**Authentication Manager**:
```swift
import Auth0

class AuthenticationManager {
  private let auth0 = Auth0.webAuth(
    clientId: "YOUR_CLIENT_ID",
    domain: "your-tenant.auth0.com"
  )
  
  func login() async throws -> Credentials {
    return try await auth0
      .audience("https://yourdomain.com/api")
      .scope("openid profile email offline_access")
      .start()
  }
  
  func logout() async throws {
    try await auth0.clearSession()
  }
}
```

**Migration Handling**:
During the transition period, the app needs to handle both authentication methods:

```swift
class MigrationManager {
  func checkMigrationStatus() -> AuthProvider {
    // Check keychain for existing tokens
    if let auth0Token = Keychain.getAuth0Token(), !auth0Token.isExpired {
      return .auth0
    }
    
    if let passageToken = Keychain.getPassageToken(), !passageToken.isExpired {
      // User still on Passage, prompt to migrate
      return .passage
    }
    
    // New user, use Auth0
    return .auth0
  }
  
  func migrateSession() async throws {
    // User has Passage token but needs to migrate
    // Show gentle prompt: "Sign in for improved security"
    try await AuthenticationManager.shared.login()
    
    // Once Auth0 login succeeds, clear Passage token
    Keychain.deletePassageToken()
  }
}
```

#### 3.9 UI/UX Considerations
**Auth0 uses browser-based login** (ASWebAuthenticationSession):
- User taps "Sign In"
- Safari modal appears (cannot be avoided on iOS)
- User sees Auth0 Universal Login page
- After authentication, redirects back to app

**Mitigation for "Web Modal" UX**:
1. **Custom Domain**: Use your own domain instead of `tenant.auth0.com`
2. **Branding**: Customize Universal Login to match app design
3. **Explanation**: In-app message: "For your security, we use industry-standard authentication"

---

### Phase 4: Testing & Pilot (Week 4-5)

#### 3.10 Test Scenarios
1. **Fresh User Signup**:
   - User signs up via Auth0
   - Receives email verification
   - Logs in successfully
   - Token validated by Go backend

2. **Lazy Migration (Critical)**:
   - Existing Passage user logs in via Auth0
   - Auth0 calls Passage Login Script
   - User imported to Auth0 automatically
   - User receives Auth0 token
   - Next login: Direct Auth0, no Passage call

3. **Social Login Migration**:
   - User previously used "Sign in with Google" via Passage
   - Logs in with Google via Auth0
   - Auth0 links accounts by email
   - User migrated seamlessly

4. **Session Continuity**:
   - User has active Passage token
   - Backend accepts both Passage and Auth0 tokens
   - User eventually logs in via Auth0
   - Old Passage token expires naturally

5. **Role/Permissions Migration**:
   - User with admin role logs in
   - Auth0 Action injects role claims
   - Go backend correctly identifies admin
   - GraphQL resolvers respect permissions

#### 3.11 Beta Testing (10% of users)
- **Selection**: Active users with diverse usage patterns
- **Monitoring**:
  - Auth0 Dashboard: Login attempts, migration success rate
  - Backend logs: Token validation errors
  - iOS analytics: Login flow completion rate
  
- **Success Criteria**:
  - Migration success rate > 95%
  - No increase in support tickets
  - Authentication latency < 3 seconds (p95)

---

### Phase 5: Full Rollout (Week 6-8)

#### 3.12 Progressive Deployment

**Week 6**: 30% of users
- **Trigger**: Update iOS app to prefer Auth0
- **Fallback**: Keep Passage SDK in app (disabled by default)
- **Monitoring**: Hourly checks of error rates

**Week 7**: 70% of users
- **User Communication**: In-app notification about improved security
- **Support Prep**: Brief support team on migration FAQs

**Week 8**: 100% of users
- **Forced Migration**: All new logins route to Auth0
- **Passage Deprecation Notice**: Email to inactive users

#### 3.13 Communication Strategy

**In-App Message** (non-blocking):
```
🔒 Security Upgrade
We've enhanced our authentication for better security.
Sign in now to experience faster, more secure access.
[Continue] [Learn More]
```

**Email Campaign** (for inactive users):
```
Subject: Action Required: Update Your [App Name] Login

Hi [Name],

We're upgrading to a more secure authentication system.

What you need to do:
1. Open the [App Name] app
2. Sign in with your email
3. You're all set!

Your account data is safe and will be automatically transferred.

Questions? Contact support@yourdomain.com
```

---

### Phase 6: Cleanup & Optimization (Week 9-12)

#### 3.14 Passage Decommissioning

**Week 9**: Disable Passage validation
- Remove Passage fallback from Go backend
- Monitor for any authentication errors
- Keep Passage API access (read-only) for data audit

**Week 10**: Remove Passage SDK from iOS
- Ship app update without Passage dependency
- Reduce app size (~5-10MB savings)
- Simplify authentication code

**Week 11**: Disable Custom Database Connection
- Once all users migrated, switch Auth0 to native database
- Remove Login/GetUser scripts
- Improves authentication performance (no external API calls)

**Week 12**: Complete Passage Shutdown
- Export final user list for records
- Cancel Passage subscription
- Archive migration logs

#### 3.15 Performance Optimization

**Auth0 Caching**:
- Enable Management API rate limit optimizations
- Use Auth0 Organizations (if B2B) for better tenant isolation
- Configure token expiration based on security requirements

**Backend Optimization**:
- Remove dual-authentication logic
- Simplify middleware to Auth0-only validation
- Cache JWKS endpoint response (reduces latency)

---

## 4. Technical Considerations

### 4.1 Passkey to Password Migration Challenge

**Problem**: Passage uses passkeys (FIDO2/WebAuthn); Auth0 custom database expects password validation.

**Solution Options**:

#### Option A: One-Time Password (OTP) Bridge
1. User enters email in Auth0 Universal Login
2. Custom database Login Script sends OTP via Passage
3. User enters OTP in Auth0
4. Script validates OTP with Passage
5. User imported to Auth0
6. Prompt user to set password for future logins

**Implementation**:
```javascript
// Auth0 Custom Database Login Script
function login(email, password, callback) {
  // password field contains the OTP user entered
  const otp = password;
  
  axios.post('https://auth.passage.id/v1/apps/YOUR_APP_ID/verify-otp', {
    email: email,
    otp: otp
  })
  .then(response => {
    // OTP valid, import user
    callback(null, {
      user_id: response.data.user_id,
      email: email,
      // Force password reset on first Auth0 login
      email_verified: true
    });
  })
  .catch(err => callback(err));
}
```

#### Option B: Magic Link Migration
1. Send email to all users: "Upgrade your account security"
2. Email contains Auth0 passwordless magic link
3. User clicks link → auto-login to Auth0
4. User imported during passwordless flow
5. Prompt to set password (optional)

#### Option C: Temporary Admin Import (Faster, Less Seamless)
1. Export all users from Passage via API
2. Bulk import to Auth0 via Management API
3. Set all users as "password reset required"
4. Send password reset emails
5. Users click reset link, set new password

**Recommendation**: Option A (OTP Bridge) for best UX, Option C for fastest migration.

---

### 4.2 Custom Claims & Authorization

**Current State**: Roles likely stored in your database or Passage metadata.

**Auth0 Approach**: 
1. Store roles in Auth0 `user_metadata` or `app_metadata`
2. Use Auth0 Actions to inject into tokens
3. Backend validates claims from JWT

**Migration of Roles**:
```javascript
// In Custom Database Login Script
function login(email, password, callback) {
  // ... validate with Passage ...
  
  // Fetch roles from your database
  const roles = await fetchRolesFromYourDB(user.id);
  
  callback(null, {
    user_id: user.id,
    email: email,
    app_metadata: {
      roles: roles,  // Admin, User, Premium, etc.
      permissions: calculatePermissions(roles)
    }
  });
}
```

```javascript
// Post-Login Action
exports.onExecutePostLogin = async (event, api) => {
  const roles = event.user.app_metadata?.roles || [];
  api.accessToken.setCustomClaim('https://yourdomain.com/roles', roles);
};
```

---

### 4.3 Session Continuity Strategy

**Goal**: User doesn't experience forced logout during migration.

**Approach**:
1. **Dual Token Support** (Weeks 1-8):
   - Backend accepts both Passage and Auth0 tokens
   - Tokens validated independently
   - No forced logout
   
2. **Graceful Token Expiration**:
   - Passage tokens expire naturally (7-30 days)
   - App prompts login when token expires
   - Login now goes through Auth0
   - User migrated seamlessly
   
3. **Proactive Migration Prompt**:
   - Detect users still on Passage (via token type)
   - Show gentle in-app prompt: "Upgrade your security"
   - User taps → Auth0 login → migrated
   - No data loss, no forced logout

---

### 4.4 Data Integrity & Compliance

**GDPR/Privacy Considerations**:
- **Data Export**: Export user data from Passage before migration
- **User Consent**: Update privacy policy to mention Auth0 as processor
- **Right to Deletion**: Ensure deletion requests cascade to both systems during transition

**Audit Trail**:
- Log every migration in your database:
  ```sql
  CREATE TABLE user_migrations (
    passage_user_id VARCHAR,
    auth0_user_id VARCHAR,
    migration_date TIMESTAMP,
    migration_method VARCHAR,  -- 'lazy', 'bulk', 'manual'
    status VARCHAR  -- 'pending', 'completed', 'failed'
  );
  ```

**Data Validation**:
- Verify email addresses before import
- Check for duplicate accounts (same email in Passage and Auth0)
- Validate custom attribute lengths (Auth0 has limits)

---

## 5. Risk Mitigation

### 5.1 High-Risk Scenarios

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Lazy migration failure | High | Medium | Comprehensive testing of custom DB scripts; fallback to bulk import |
| User confusion (web modal) | Medium | High | Clear in-app explanation; branded custom domain |
| Passage API downtime during migration | High | Low | Cache user data; implement retry logic; have emergency bulk import ready |
| Token validation errors | Critical | Medium | Extensive integration testing; dual-auth period; monitoring |
| Social login account conflicts | Medium | Medium | Email verification; manual account linking for edge cases |
| Custom domain SSL issues | Medium | Low | Test domain setup in dev environment first; have Auth0 default as fallback |

### 5.2 Rollback Strategy

**Scenario 1**: Custom database scripts fail (>10% error rate)
- **Action**: Disable Auth0 login, revert to Passage
- **Time**: 5 minutes (feature flag flip)
- **Impact**: Users see "Login temporarily unavailable"

**Scenario 2**: Token validation issues in backend
- **Action**: Temporarily accept only Passage tokens
- **Time**: 10 minutes (deploy backend config change)
- **Impact**: Users on Auth0 must re-login with Passage

**Scenario 3**: Mass user complaints about UX
- **Action**: Pause rollout, improve Universal Login branding
- **Time**: 1-2 days
- **Impact**: New users delayed, existing users unaffected

---

## 6. Success Metrics

### 6.1 Technical KPIs
- **Migration Success Rate**: ≥ 97% (lazy migration)
- **Authentication Latency**: ≤ 2 seconds (p95) including web modal
- **Token Validation Accuracy**: 100%
- **API Error Rate**: ≤ 0.5% during migration period
- **Custom Database Script Performance**: < 500ms execution time

### 6.2 User Experience KPIs
- **Login Completion Rate**: ≥ 90% (considering web modal drop-off)
- **User Complaints**: < 2% of migrated users
- **Support Ticket Volume**: < 30% increase
- **App Store Rating**: Maintain ≥ 4.5 stars

### 6.3 Business KPIs
- **Migration Timeline**: Completed within 12 weeks
- **Cost Impact**: +$250-800/month (Auth0 subscription)
- **Downtime**: 0 minutes
- **Engineering Time**: ≤ 160 hours (Auth0 is easier to integrate than Cognito)

---

## 7. Migration Timeline Summary

```
Week 1-2:   Auth0 Configuration
            ├─ Tenant setup & custom domain
            ├─ Custom database scripts (Passage integration)
            ├─ Actions for custom claims
            └─ Social connections configuration

Week 2-3:   Backend Integration (Parallel)
            ├─ Go middleware with dual-auth support
            ├─ Management API integration
            └─ Migration logging database

Week 3-4:   iOS Integration
            ├─ Auth0.swift SDK installation
            ├─ Migration manager implementation
            ├─ UI/UX for auth flow
            └─ Keychain management

Week 4-5:   Testing & Pilot
            ├─ Internal testing (50 users)
            ├─ Beta rollout (10% of users)
            └─ Monitoring & optimization

Week 6-8:   Full Rollout
            ├─ Week 6: 30% of users
            ├─ Week 7: 70% of users
            └─ Week 8: 100% of users

Week 9-12:  Cleanup
            ├─ Disable Passage validation
            ├─ Remove Passage SDK from iOS
            ├─ Disable custom database (native Auth0)
            └─ Cancel Passage subscription
```

---

## 8. Auth0 vs Cognito: Migration Comparison

| Aspect | Auth0 | Cognito |
|--------|-------|---------|
| **Migration Ease** | ⭐⭐⭐⭐⭐ Lazy migration built-in | ⭐⭐⭐ Requires custom Lambda |
| **iOS UX** | ⭐⭐⭐ Web modal (browser-based) | ⭐⭐⭐⭐⭐ Fully native |
| **Setup Complexity** | ⭐⭐⭐⭐ Simpler, better docs | ⭐⭐ More complex, steeper learning curve |
| **Cost** | ⭐⭐ $250-800/month | ⭐⭐⭐⭐⭐ Free (up to 50k MAU) |
| **Developer Experience** | ⭐⭐⭐⭐⭐ Excellent SDKs, docs | ⭐⭐⭐ Good but AWS-centric |
| **Customization** | ⭐⭐⭐⭐ Actions, Rules, Hooks | ⭐⭐⭐⭐ Lambda triggers (more control) |
| **Social Logins** | ⭐⭐⭐⭐⭐ 30+ out-of-box | ⭐⭐⭐ Limited providers |
| **Support** | ⭐⭐⭐⭐ Excellent community + paid support | ⭐⭐⭐ AWS support + forums |

**Recommendation**:
- Choose **Auth0** if: Developer speed, feature richness, and migration ease are priorities. Acceptable to pay for premium service.
- Choose **Cognito** if: Cost is critical, you want native iOS UX, and you're already invested in AWS ecosystem.

---

## 9. Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    MIGRATION ARCHITECTURE DIAGRAM                        │
└─────────────────────────────────────────────────────────────────────────┘


                    PHASE 1-3: LAZY MIGRATION WITH CUSTOM DATABASE
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  ┌──────────────┐                                                         │
│  │   iOS App    │                                                         │
│  │              │         User taps "Sign In"                             │
│  │ ┌──────────┐ │               │                                         │
│  │ │ Auth0    │─┼───────────────┘                                         │
│  │ │  SDK     │ │                                                         │
│  │ └──────────┘ │                                                         │
│  │              │         Opens ASWebAuthenticationSession                │
│  └──────────────┘               │                                         │
│                                 ▼                                         │
│                   ┌──────────────────────────┐                            │
│                   │ Auth0 Universal Login    │                            │
│                   │ (Custom Domain)          │                            │
│                   │  auth.yourdomain.com     │                            │
│                   └────────────┬─────────────┘                            │
│                                │ User enters email                        │
│                                ▼                                         │
│                   ┌──────────────────────────┐                            │
│                   │ Auth0 checks database:   │                            │
│                   │ "Does user exist?"       │                            │
│                   └────────────┬─────────────┘                            │
│                                │                                           │
│                 ┌──────────────┴──────────────┐                           │
│                 │                             │                           │
│         ┌───────▼───────┐             ┌───────▼───────┐                   │
│         │ User Exists   │             │ User NOT Found│                   │
│         │ in Auth0      │             │ (First Login) │                   │
│         └───────┬───────┘             └───────┬───────┘                   │
│                 │                             │                           │
│                 │                             ▼                           │
│                 │               ┌──────────────────────────┐              │
│                 │               │ Custom Database Script   │              │
│                 │               │ "Login" (Node.js)        │              │
│                 │               └────────────┬─────────────┘              │
│                 │                            │                            │
│                 │                            ▼                            │
│                 │               ┌──────────────────────────┐              │
│                 │               │  Call Passage API        │              │
│                 │               │  Validate User           │              │
│                 │               │                          │              │
│                 │               │  POST /login or          │              │
│                 │               │  Send OTP                │              │
│                 │               └────────────┬─────────────┘              │
│                 │                            │                            │
│                 │         ┌──────────────────┴─────────────┐              │
│                 │         │                                │              │
│                 │   ┌─────▼──────┐                  ┌──────▼──────┐       │
│                 │   │  Success   │                  │   Failure   │       │
│                 │   │            │                  │             │       │
│                 │   └─────┬──────┘                  └──────┬──────┘       │
│                 │         │                                │              │
│                 │         ▼                                ▼              │
│                 │   ┌──────────────────────┐     ┌──────────────┐        │
│                 │   │ Import User to Auth0 │     │ Show Error   │        │
│                 │   │ (Automatic)          │     └──────────────┘        │
│                 │   │                      │                             │
│                 │   │ • user_id            │                             │
│                 │   │ • email              │                             │
│                 │   │ • user_metadata      │                             │
│                 │   │ • app_metadata       │                             │
│                 │   └──────────┬───────────┘                             │
│                 │              │                                          │
│                 └──────────────┼──────────────┘                           │
│                                │                                          │
│                                ▼                                          │
│                   ┌──────────────────────────┐                            │
│                   │ Auth0 Post-Login Action  │                            │
│                   │ (Add Custom Claims)      │                            │
│                   └────────────┬─────────────┘                            │
│                                │                                          │
│                                ▼                                          │
│                   ┌──────────────────────────┐                            │
│                   │ Generate JWT Tokens      │                            │
│                   │ • Access Token           │                            │
│                   │ • ID Token               │                            │
│                   │ • Refresh Token          │                            │
│                   └────────────┬─────────────┘                            │
│                                │                                          │
│                                ▼                                          │
│                   ┌──────────────────────────┐                            │
│                   │ Redirect to iOS App      │                            │
│                   │ com.yourapp://callback   │                            │
│                   └────────────┬─────────────┘                            │
│                                │                                          │
│  ┌──────────────┐              │                                          │
│  │   iOS App    │◀─────────────┘                                          │
│  │              │                                                         │
│  │ Stores token │                                                         │
│  │ in Keychain  │                                                         │
│  └──────┬───────┘                                                         │
│         │                                                                 │
│         │ API Request with Token                                          │
│         ▼                                                                 │
│  ┌────────────────────────┐                                               │
│  │  Go Backend API        │                                               │
│  │                        │                                               │
│  │  ┌──────────────────┐ │        ┌──────────────────┐                   │
│  │  │ Auth Middleware  │─┼───────▶│ Validate JWT     │                   │
│  │  │                  │ │        │ (JWKS)           │                   │
│  │  │ • Try Auth0      │ │        │                  │                   │
│  │  │ • Fallback:      │ │        │ https://         │                   │
│  │  │   Passage (temp) │ │        │ your-tenant      │                   │
│  │  └──────────────────┘ │        │ .auth0.com/      │                   │
│  │           │            │        │ .well-known/jwks │                   │
│  │           ▼            │        └──────────────────┘                   │
│  │  ┌──────────────────┐ │                                               │
│  │  │ GraphQL Resolvers│ │                                               │
│  │  │ (Authorized)     │ │                                               │
│  │  └──────────────────┘ │                                               │
│  └────────────────────────┘                                               │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                        LAZY MIGRATION FLOW (User Perspective)
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│   Existing Passage User                    New Auth0 User                 │
│          │                                        │                        │
│          │ First login after migration            │ First time signup     │
│          ▼                                        ▼                        │
│   ┌─────────────┐                          ┌─────────────┐                │
│   │ Tap Sign In │                          │ Tap Sign Up │                │
│   └──────┬──────┘                          └──────┬──────┘                │
│          │                                        │                        │
│          ▼                                        ▼                        │
│   ┌──────────────────────┐              ┌──────────────────────┐          │
│   │ Safari modal opens   │              │ Safari modal opens   │          │
│   │ "Continue to         │              │ "Continue to         │          │
│   │  auth.yourdomain.com"│              │  auth.yourdomain.com"│          │
│   └──────┬───────────────┘              └──────┬───────────────┘          │
│          │                                     │                           │
│          ▼                                     ▼                           │
│   ┌──────────────────────┐              ┌──────────────────────┐          │
│   │ Enter email          │              │ Enter email          │          │
│   │ user@example.com     │              │ newuser@example.com  │          │
│   └──────┬───────────────┘              └──────┬───────────────┘          │
│          │                                     │                           │
│          ▼                                     ▼                           │
│   ┌──────────────────────┐              ┌──────────────────────┐          │
│   │ Auth0: "User not in  │              │ Enter password       │          │
│   │  database, checking  │              │ ********             │          │
│   │  Passage..."         │              └──────┬───────────────┘          │
│   │                      │                     │                           │
│   │ (Happens invisibly   │                     ▼                           │
│   │  in <1 second)       │              ┌──────────────────────┐          │
│   └──────┬───────────────┘              │ Account created      │          │
│          │                               │ in Auth0             │          │
│          ▼                               └──────┬───────────────┘          │
│   ┌──────────────────────┐                     │                           │
│   │ Receive OTP email    │                     │                           │
│   │ or enter password    │                     │                           │
│   │ (bridge method)      │                     │                           │
│   └──────┬───────────────┘                     │                           │
│          │                                     │                           │
│          ▼                                     │                           │
│   ┌──────────────────────┐                     │                           │
│   │ Validated by Passage │                     │                           │
│   └──────┬───────────────┘                     │                           │
│          │                                     │                           │
│          ▼                                     │                           │
│   ┌──────────────────────┐                     │                           │
│   │ User IMPORTED to     │                     │                           │
│   │ Auth0 automatically  │                     │                           │
│   └──────┬───────────────┘                     │                           │
│          │                                     │                           │
│          └─────────────┬───────────────────────┘                           │
│                        │                                                   │
│                        ▼                                                   │
│              ┌──────────────────────┐                                      │
│              │ Redirect to app      │                                      │
│              │ (Logged in!)         │                                      │
│              └──────┬───────────────┘                                      │
│                     │                                                      │
│                     ▼                                                      │
│              ┌──────────────────────┐                                      │
│              │ Subsequent logins:   │                                      │
│              │ Direct Auth0         │                                      │
│              │ (No Passage call)    │                                      │
│              └──────────────────────┘                                      │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                         PHASE 4-6: POST-MIGRATION (AUTH0 NATIVE)
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  ┌──────────────┐                                                         │
│  │   iOS App    │                                                         │
│  │              │                                                         │
│  │ ┌──────────┐ │         ┌────────────────────────┐                      │
│  │ │ Auth0    │─┼────────▶│ Auth0 Universal Login  │                      │
│  │ │  SDK     │ │         │ (Web Modal)            │                      │
│  │ │ (PKCE)   │ │         │                        │                      │
│  │ └──────────┘ │         │ Custom Domain:         │                      │
│  │              │         │ auth.yourdomain.com    │                      │
│  └──────────────┘         └───────────┬────────────┘                      │
│         │                              │                                   │
│         │ Token received               ▼                                   │
│         │                   ┌────────────────────────┐                     │
│         │                   │ Auth0 Native Database  │                     │
│         │                   │ (No custom DB script)  │                     │
│         │                   │                        │                     │
│         │                   │ All users migrated     │                     │
│         │                   └───────────┬────────────┘                     │
│         │                               │                                  │
│         │                               ▼                                  │
│         │                   ┌────────────────────────┐                     │
│         │                   │ Post-Login Action      │                     │
│         │                   │ (Custom Claims)        │                     │
│         │                   └───────────┬────────────┘                     │
│         │                               │                                  │
│         │                               ▼                                  │
│         │                   ┌────────────────────────┐                     │
│         │◀──────────────────│ JWT Tokens             │                     │
│         │                   └────────────────────────┘                     │
│         │                                                                  │
│         ▼                                                                  │
│  ┌────────────────────────┐                                               │
│  │  Go Backend API        │                                               │
│  │  (Simplified)          │                                               │
│  │                        │      ┌──────────────────┐                     │
│  │  ┌──────────────────┐ │      │ Auth0 JWKS       │                     │
│  │  │ Auth Middleware  │─┼─────▶│ Validation       │                     │
│  │  │ (Auth0 Only)     │ │      │                  │                     │
│  │  └────────┬─────────┘ │      │ Cached for       │                     │
│  │           │            │      │ performance      │                     │
│  │           ▼            │      └──────────────────┘                     │
│  │  ┌──────────────────┐ │                                               │
│  │  │ GraphQL API      │ │      ┌──────────────────┐                     │
│  │  │ (Fully Migrated) │─┼─────▶│ Auth0 Management │                     │
│  │  │                  │ │      │ API (User Lookup)│                     │
│  │  └──────────────────┘ │      └──────────────────┘                     │
│  └────────────────────────┘                                               │
│                                                                            │
│  Passage SDK: REMOVED                                                     │
│  Passage API: DECOMMISSIONED                                              │
│  Custom Database: DISABLED                                                │
│  Migration Time: 0ms (no external calls)                                  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                              ROLLOUT TIMELINE
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  Week 1-2  │████████│ Auth0 Configuration                                 │
│            │        │ • Tenant & custom domain                            │
│            │        │ • Custom database scripts                           │
│            │        │ • Actions & rules                                   │
│            │        │                                                     │
│  Week 2-3  │████████│ Backend Integration (Parallel)                      │
│            │        │ • Go dual-auth middleware                           │
│            │        │ • Management API                                    │
│            │        │                                                     │
│  Week 3-4  │████████│ iOS Integration                                     │
│            │        │ • Auth0.swift SDK                                   │
│            │        │ • Migration manager                                 │
│            │        │ • UI/UX implementation                              │
│            │        │                                                     │
│  Week 4-5  │████████│ Testing & Pilot                                     │
│            │        │ • Internal: 50 users                                │
│            │        │ • Beta: 10% rollout                                 │
│            │        │                                                     │
│  Week 6    │████    │ 30% Rollout                                         │
│  Week 7    │██████████│ 70% Rollout                                       │
│  Week 8    │██████████████│ 100% Rollout                                  │
│            │              │                                               │
│  Week 9-12 │████████│ Cleanup                                             │
│            │        │ • Disable Passage                                   │
│            │        │ • Native Auth0 database                             │
│            │        │ • Cost optimization                                 │
│            │        │                                                     │
│            0%      25%      50%      75%      100%                         │
│                   Users Migrated                                          │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                         KEY MIGRATION ADVANTAGES: AUTH0
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  ✅ LAZY MIGRATION                                                         │
│     Built-in feature via Custom Database                                  │
│     No bulk export/import needed                                          │
│     Users migrated transparently on first login                           │
│                                                                            │
│  ✅ SUPERIOR DEVELOPER EXPERIENCE                                          │
│     Excellent documentation & SDKs                                         │
│     Intuitive dashboard                                                   │
│     Fast integration (~50% less code than Cognito)                        │
│                                                                            │
│  ✅ FEATURE-RICH                                                           │
│     30+ social providers out-of-box                                        │
│     Advanced security (breached password detection, bot detection)        │
│     Customizable Universal Login (no-code branding)                       │
│                                                                            │
│  ✅ ROBUST MIGRATION TOOLS                                                 │
│     User Import/Export API                                                 │
│     Bulk import with password hashes                                      │
│     Account linking (merge duplicate accounts)                            │
│                                                                            │
│  ⚠️  TRADE-OFFS                                                            │
│     Web modal UX (not fully native like Cognito)                          │
│     Higher cost ($250-800/month vs Cognito free tier)                     │
│     Vendor lock-in (though standards-compliant OIDC)                      │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Appendix

### 10.1 Required Auth0 Resources
- **Tenant**: 1x Production, 1x Development (recommended)
- **Applications**: 2x (Native iOS, M2M for backend)
- **Database Connection**: 1x (custom for lazy migration)
- **Actions**: 2-3x (Post-Login, Pre-Registration)
- **Rules** (deprecated, use Actions): 0x
- **Custom Domain**: 1x (e.g., `auth.yourdomain.com`)

### 10.2 Cost Breakdown
**Auth0 Essentials Plan** (10,000 MAU):
- Base: ~$240/month
- Custom Domain: Included
- Unlimited Social Connections: Included
- MFA (SMS): Additional cost per SMS (~$0.05/SMS)
- Enterprise SSO: Additional

**Total Estimated**: $250-800/month depending on features

### 10.3 Team Responsibilities
- **Backend Engineer** (40 hours): Middleware, Management API, migration logging
- **iOS Engineer** (50 hours): SDK integration, UI/UX, keychain management
- **Full-Stack Engineer** (30 hours): Custom database scripts, Actions
- **DevOps** (20 hours): Domain setup, monitoring, deployment
- **QA** (30 hours): Test scenarios, regression, user acceptance
- **Support** (10 hours): Documentation, user communication, ticket handling

**Total**: ~160 hours

### 10.4 Communication Templates

**In-App Prompt**:
```
🔐 Security Upgrade in Progress

We're migrating to a more secure authentication system.

Your account is safe and will be automatically transferred.

Next time you log in, you might see a brief verification step.

Questions? Tap here for FAQs.
```

**Migration Email**:
```
Subject: [App Name] Security Enhancement

Hi [Name],

We're upgrading our authentication to provide you with:
✓ Faster login
✓ Enhanced security
✓ More login options (social, biometric)

What you need to do: Nothing!

Your account will be automatically upgraded the next time you sign in.

Need help? Visit support.yourdomain.com/migration

Thanks,
The [App Name] Team
```

---

## Conclusion

This Auth0 migration strategy leverages **lazy migration** to provide the most seamless possible user experience. The custom database connection allows validating users with Passage while simultaneously importing them to Auth0 in real-time.

**Key Success Factors:**
1. **Lazy Migration**: No forced user action, transparent migration
2. **Custom Database Scripts**: Well-tested Passage API integration
3. **Dual-Auth Period**: Risk-free transition with fallback
4. **Progressive Rollout**: Gradual deployment with monitoring
5. **Clear Communication**: Proactive user notification

**Why Choose Auth0 Over Cognito:**
- **Faster Development**: ~50% less implementation time
- **Better Migration Tools**: Native lazy migration support
- **Superior DX**: Cleaner APIs, better documentation
- **Feature Richness**: 30+ social providers, advanced security

**Trade-off**: Higher cost (~$3,000-10,000/year) and web-modal UX vs Cognito's native iOS experience.

**Estimated Total Effort**: 150-180 engineering hours across 12 weeks
**Risk Level**: Low (lazy migration reduces friction significantly)
**User Impact**: Minimal (one-time web modal login, then seamless)

