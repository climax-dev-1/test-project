# Migration Strategy: Passage → AWS Cognito

## Overview
This strategy focuses on migrating from Passage by 1Password to AWS Cognito while maintaining a seamless user experience through a phased, zero-downtime approach.

---

## 1. Pre-Migration Assessment

### 1.1 Data Inventory
- **User Records**: Identify all user accounts in Passage (email, user IDs, metadata)
- **Authentication Methods**: Passkeys, biometrics, device registrations
- **User Attributes**: Profile data, custom claims, roles
- **Active Sessions**: Document existing session durations and refresh token policies

### 1.2 Architecture Analysis
- **Current Flow**: Passage SDK → iOS App → Go Backend
- **Session Storage**: Where tokens are currently stored (Keychain, UserDefaults)
- **API Dependencies**: GraphQL endpoints expecting Passage tokens
- **Third-party Integrations**: Any services using Passage authentication

---

## 2. Migration Architecture

### 2.1 Dual-Provider Bridge Pattern
We'll implement a **hybrid authentication layer** that supports both Passage and Cognito simultaneously during transition.

**Key Components:**
- **Authentication Router** (Go middleware): Validates tokens from both providers
- **User Migration Service**: Background service for bulk user migration
- **Just-in-Time (JIT) Migration**: Migrate users on their first login attempt
- **Feature Flags**: Control rollout percentage and rollback capability

### 2.2 User Migration Methods

#### Method A: Just-in-Time (JIT) Migration (Recommended)
**Best for**: Seamless user experience, no forced re-authentication

**Flow:**
1. User attempts login with Passage credentials
2. Backend validates with Passage API
3. On successful validation:
   - Create corresponding Cognito user
   - Set temporary password
   - Migrate user metadata
   - Return Cognito token to client
4. User is now on Cognito, unaware of migration

**Advantages:**
- Zero user friction
- No mass email campaigns
- Natural migration pace
- Only active users are migrated

**Challenges:**
- Requires maintaining Passage during transition (continued cost)
- Complex dual-authentication logic

#### Method B: Pre-Migration with Soft Launch
**Best for**: Faster complete migration, cost reduction

**Flow:**
1. Bulk export users from Passage
2. Create Cognito users with temporary passwords
3. Send migration emails with magic links
4. Users click link → auto-login → prompt to set up new auth method
5. Gradually sunset Passage over 60-90 days

---

## 3. Detailed Migration Plan

### Phase 1: Infrastructure Setup (Week 1-2)

#### 3.1 Cognito Configuration
- **User Pool Creation**:
  - Region: Same as Go backend (reduce latency)
  - Password Policy: Match or exceed Passage security
  - MFA Settings: Optional SMS/TOTP (to replace Passkey temporarily)
  
- **App Client**:
  - Native iOS client (SRP authentication)
  - Backend server client (for admin operations)
  - Enable `ALLOW_USER_PASSWORD_AUTH` flow
  
- **Attributes Schema**:
  - Standard: email (required, verified), name, phone
  - Custom: `passage_user_id`, `migration_date`, `original_signup_date`
  
- **Lambda Triggers**:
  - **Pre-Authentication**: Log migration attempts
  - **Post-Authentication**: Inject custom claims (roles, permissions)
  - **User Migration Trigger**: Handle JIT migration from Passage

#### 3.2 Backend Infrastructure
- **Go Middleware**: `auth-router` package
  ```
  func AuthMiddleware(ctx context.Context, token string) (*User, error) {
    // Try Cognito first (new system)
    if user, err := cognitoValidator.Validate(token); err == nil {
      return user, nil
    }
    
    // Fallback to Passage (legacy)
    if user, err := passageValidator.Validate(token); err == nil {
      // Trigger async migration
      go migrateUser(user)
      return user, nil
    }
    
    return nil, ErrUnauthorized
  }
  ```

- **Migration Database Table**:
  ```
  migration_status:
    - passage_user_id (PK)
    - cognito_user_id
    - status (pending, in_progress, completed, failed)
    - migration_date
    - error_message
  ```

#### 3.3 iOS App Changes
- **Dual SDK Integration**:
  - Keep Passage SDK (temporarily)
  - Add AWS Amplify Library for Swift
  
- **Authentication Manager**:
  - Check for migration flag in Keychain
  - Route to appropriate SDK
  - Handle migration handoff

---

### Phase 2: Pilot Migration (Week 3-4)

#### 3.4 Internal Testing
- **Test Users**: Create 50-100 test accounts
- **Scenarios**:
  - Fresh Cognito signup
  - JIT migration from Passage
  - Failed migration handling
  - Session continuity across migration
  - Token refresh flows

#### 3.5 Beta User Group (5% of users)
- **Selection Criteria**: 
  - Active users with recent logins
  - Mix of iOS versions
  - Users with different authentication patterns
  
- **Monitoring**:
  - CloudWatch dashboards for Lambda errors
  - Migration success rate metrics
  - Average migration time
  - User support tickets

#### 3.6 Rollback Plan
- **Trigger Conditions**:
  - Migration success rate < 95%
  - Authentication latency > 2x baseline
  - Critical bugs in new flow
  
- **Rollback Process**:
  1. Disable JIT migration Lambda
  2. Route all traffic to Passage
  3. Retain migrated Cognito users (no data loss)
  4. Re-enable after fixes

---

### Phase 3: Gradual Rollout (Week 5-8)

#### 3.7 Progressive Deployment
- **Week 5**: 15% of users
- **Week 6**: 35% of users
- **Week 7**: 65% of users
- **Week 8**: 100% of users

#### 3.8 User Communication
- **In-App Notifications** (non-intrusive):
  - "We've upgraded our security. Your next login will be faster!"
  - No action required messaging
  
- **Email Campaign** (for pre-migration approach):
  - Subject: "Important: Authentication Upgrade"
  - Content: Clear steps, benefits, support contact
  - Timing: 2 weeks before forced migration

#### 3.9 Migration Monitoring
- **Key Metrics**:
  - Total users migrated: X / Y (Z%)
  - Daily migration rate
  - Failed migrations with error types
  - User drop-off rate (logins before/after)
  
- **Alerting**:
  - PagerDuty/Slack alerts for migration failures > 5%
  - Daily summary reports to engineering team

---

### Phase 4: Cleanup & Optimization (Week 9-12)

#### 3.10 Passage Decommissioning
- **Week 9**: Disable new Passage authentications (Cognito only)
- **Week 10**: Archive Passage user data (compliance)
- **Week 11**: Remove Passage SDK from iOS app
- **Week 12**: Remove Passage validation from Go backend

#### 3.11 Code Cleanup
- Remove dual-authentication logic
- Simplify middleware to Cognito-only
- Remove feature flags
- Update documentation

#### 3.12 Cost Optimization
- **Cognito Savings**:
  - Free tier: First 50,000 MAUs
  - Estimated savings: $50-150/month (depending on Passage plan)
  
- **Infrastructure**:
  - Remove Passage API calls (reduced latency)
  - Decommission migration Lambda functions
  - Archive migration database

---

## 4. Technical Considerations

### 4.1 Authentication Method Transition
**Challenge**: Passage uses Passkeys/WebAuthn; Cognito native auth uses SRP password flow

**Solution Options**:

1. **Temporary Password Bridge**:
   - Create Cognito users with secure random temporary passwords
   - On JIT migration, auto-login user without password prompt
   - Prompt user to set up new password/MFA after first Cognito login
   
2. **SMS/Email OTP Transition**:
   - Use Cognito's Custom Auth flow
   - Send OTP on migration
   - User verifies identity without password
   - Set password post-verification

3. **Social Login Redirect** (if users had social logins in Passage):
   - Configure Cognito Identity Providers (Google, Apple, etc.)
   - Link Passage social accounts to Cognito social accounts via email matching

### 4.2 Session Continuity
**Goal**: User doesn't get logged out during migration

**Approach**:
1. User logs in with Passage → receives Passage token
2. Backend validates and triggers migration
3. Backend generates Cognito token using Admin APIs
4. Backend returns **both tokens** to client
5. Client stores Cognito token, uses it for next request
6. Passage token expires naturally (no forced logout)

### 4.3 Custom Claims & Roles
**Passage Approach**: (varies by implementation)
**Cognito Approach**: Pre-Token Generation Lambda

**Migration Steps**:
1. Export user roles/permissions from current system
2. Store in separate database (DynamoDB/PostgreSQL)
3. Lambda function queries database and injects claims:
   ```javascript
   exports.handler = async (event) => {
     const userId = event.request.userAttributes.sub;
     const roles = await getRolesFromDB(userId);
     
     event.response = {
       claimsOverrideDetails: {
         claimsToAddOrOverride: {
           'custom:roles': JSON.stringify(roles),
           'custom:permissions': getPermissions(roles)
         }
       }
     };
     return event;
   };
   ```

### 4.4 Data Integrity
**Critical Safeguards**:
- **Idempotency**: Ensure re-running migration doesn't create duplicate users
- **Email Verification**: Transfer verified email status from Passage
- **Audit Trail**: Log every migration with timestamp, source, and result
- **Data Validation**: 
  - Email format validation
  - Required field checks before Cognito user creation
  - Attribute length limits (Cognito has strict limits)

---

## 5. Risk Mitigation

### 5.1 High-Risk Scenarios

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Mass migration failure | Critical | Low | Staged rollout, comprehensive testing, instant rollback |
| User data loss | Critical | Very Low | Multiple backups, dry-run migrations, verification checks |
| Extended downtime | High | Low | Zero-downtime design, dual-provider bridge |
| User confusion/support load | Medium | Medium | Clear communication, in-app guides, support team briefing |
| Regulatory compliance issues | High | Low | Legal review of data export, GDPR compliance check |
| Token validation failures | High | Medium | Extensive integration testing, monitoring, graceful fallbacks |

### 5.2 Rollback Triggers
- Authentication success rate drops below 95%
- Migration failure rate exceeds 10%
- Increased error rates in backend APIs (>2x normal)
- Critical security vulnerability discovered
- User complaints exceed threshold (>50 tickets/day)

---

## 6. Success Metrics

### 6.1 Technical KPIs
- **Migration Success Rate**: ≥ 99%
- **Authentication Latency**: ≤ 500ms (p95)
- **Zero data loss**: 100% user data integrity
- **API Error Rate**: ≤ 0.1% during migration
- **Session Continuity**: ≥ 98% users not forcibly logged out

### 6.2 User Experience KPIs
- **User Complaints**: < 1% of migrated users
- **Authentication Drop-off**: < 2% increase
- **Support Ticket Volume**: < 20% increase
- **App Store Rating**: Maintain or improve

### 6.3 Business KPIs
- **Migration Timeline**: Completed within 12 weeks
- **Cost Reduction**: $600-1,800/year savings
- **Downtime**: 0 minutes
- **Engineering Time**: ≤ 200 hours

---

## 7. Migration Timeline Summary

```
Week 1-2:   Infrastructure Setup
            ├─ Cognito User Pool configuration
            ├─ Lambda functions development
            ├─ Go backend dual-auth middleware
            └─ iOS app SDK integration

Week 3-4:   Pilot Migration
            ├─ Internal testing (100 test users)
            ├─ Beta group (5% of users)
            └─ Monitoring & iteration

Week 5-8:   Gradual Rollout
            ├─ Week 5: 15%
            ├─ Week 6: 35%
            ├─ Week 7: 65%
            └─ Week 8: 100%

Week 9-12:  Cleanup & Optimization
            ├─ Disable Passage authentication
            ├─ Remove legacy code
            ├─ Decommission migration infrastructure
            └─ Documentation & post-mortem
```

---

## 8. Appendix

### 8.1 Required AWS Resources
- **User Pool**: 1x in production region
- **Lambda Functions**: 4x (Pre-Auth, Post-Auth, User Migration, Custom Claims)
- **IAM Roles**: 3x (Lambda execution, App client, Admin operations)
- **CloudWatch**: Log groups, dashboards, alarms
- **DynamoDB**: Migration tracking table (optional, can use RDS)

### 8.2 Team Responsibilities
- **Backend Engineer**: Go middleware, Lambda functions, monitoring
- **iOS Engineer**: Dual SDK integration, UI/UX for migration prompts
- **DevOps**: Infrastructure provisioning, deployment pipelines, monitoring
- **QA**: Test scenarios, regression testing, user acceptance testing
- **Support**: User communication, ticket handling, escalation procedures
- **Product Manager**: Rollout decisions, user communication, success metrics

### 8.3 Communication Templates
- **Internal Kickoff Email**
- **Beta User Invitation**
- **General User Migration Notice**
- **Support Team Migration FAQ**
- **Post-Migration Success Summary**

---

## 9. Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    MIGRATION ARCHITECTURE DIAGRAM                        │
└─────────────────────────────────────────────────────────────────────────┘

                            PHASE 1-2: DUAL PROVIDER BRIDGE
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  ┌──────────────┐                                                         │
│  │   iOS App    │                                                         │
│  │              │                                                         │
│  │ ┌──────────┐ │                                                         │
│  │ │ Passage  │ │   ┌──────────────────┐                                 │
│  │ │   SDK    │─┼──▶│  Auth Router     │                                 │
│  │ └──────────┘ │   │  (Go Middleware) │                                 │
│  │              │   │                  │                                 │
│  │ ┌──────────┐ │   │  ┌────────────┐ │      ┌──────────────────┐       │
│  │ │ Cognito  │─┼──▶│  │  Validate  │─┼─────▶│  Passage API     │       │
│  │ │Amplify   │ │   │  │   Token    │ │      │  (Legacy)        │       │
│  │ └──────────┘ │   │  └────────────┘ │      └──────────────────┘       │
│  └──────────────┘   │         │        │               │                 │
│                     │         ▼        │               ▼                 │
│                     │  ┌────────────┐ │      ┌──────────────────┐       │
│                     │  │   Cognito  │─┼─────▶│  AWS Cognito     │       │
│                     │  │  Validator │ │      │  User Pool       │       │
│                     │  └────────────┘ │      └──────────────────┘       │
│                     │         │        │               │                 │
│                     │         ▼        │               │                 │
│                     │  ┌────────────┐ │               │                 │
│                     │  │  If Valid  │ │               │                 │
│                     │  │   Pass to  │ │               │                 │
│                     │  │  GraphQL   │ │      ┌────────▼─────────┐       │
│                     │  └────────────┘ │      │ Migration Lambda │       │
│                     └─────────┬────────┘      │  (JIT Trigger)   │       │
│                               │               └────────┬─────────┘       │
│                               ▼                        │                 │
│                     ┌──────────────────┐               │                 │
│                     │  Go Backend API  │               │                 │
│                     │    (GraphQL)     │               │                 │
│                     └──────────────────┘               │                 │
│                                                        │                 │
│                                          ┌─────────────▼──────────────┐  │
│                                          │  Migration Status DB       │  │
│                                          │  (DynamoDB/PostgreSQL)     │  │
│                                          └────────────────────────────┘  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                         PHASE 3-4: COGNITO ONLY (POST-MIGRATION)
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  ┌──────────────┐                                                         │
│  │   iOS App    │                                                         │
│  │              │                                                         │
│  │ ┌──────────┐ │   ┌──────────────────┐      ┌──────────────────┐       │
│  │ │ Cognito  │─┼──▶│  Auth Middleware │─────▶│  AWS Cognito     │       │
│  │ │Amplify   │ │   │  (Simplified)    │      │  User Pool       │       │
│  │ │  (SRP)   │ │   │                  │      │                  │       │
│  │ └──────────┘ │   │  ┌────────────┐ │      │  ┌────────────┐  │       │
│  └──────────────┘   │  │  Validate  │ │      │  │Pre-Token   │  │       │
│                     │  │   JWKS     │ │      │  │ Generation │  │       │
│                     │  │            │ │      │  │  Lambda    │  │       │
│                     │  └────────────┘ │      │  └────────────┘  │       │
│                     │         │        │      │         │        │       │
│                     │         ▼        │      │         ▼        │       │
│                     └─────────────────┘      │  ┌────────────┐  │       │
│                               │               │  │ Custom     │  │       │
│                               │               │  │ Claims     │  │       │
│                               │               │  └────────────┘  │       │
│                               │               └──────────────────┘       │
│                               │                                           │
│                               ▼                                           │
│                     ┌──────────────────┐                                 │
│                     │  Go Backend API  │                                 │
│                     │    (GraphQL)     │                                 │
│                     └──────────────────┘                                 │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                            JIT MIGRATION FLOW DETAIL
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│   User Login Attempt                                                      │
│          │                                                                 │
│          ▼                                                                 │
│   ┌─────────────┐                                                         │
│   │  Passage    │                                                         │
│   │  Auth       │                                                         │
│   └──────┬──────┘                                                         │
│          │                                                                 │
│          ▼                                                                 │
│   ┌─────────────────┐      Success        ┌──────────────────┐           │
│   │ Validate with   │─────────────────────▶│ Check Migration  │           │
│   │ Passage API     │                      │ Status in DB     │           │
│   └─────────────────┘                      └────────┬─────────┘           │
│          │                                           │                     │
│          │ Failure                                   │                     │
│          ▼                                           ▼                     │
│   ┌─────────────┐                          ┌─────────────────┐            │
│   │Return Error │                          │  Not Migrated?  │            │
│   └─────────────┘                          └────────┬────────┘            │
│                                                      │ Yes                 │
│                                                      ▼                     │
│                                            ┌──────────────────┐            │
│                                            │ Create Cognito   │            │
│                                            │ User via Admin   │            │
│                                            │ API              │            │
│                                            └────────┬─────────┘            │
│                                                     │                      │
│                                                     ▼                      │
│                                            ┌──────────────────┐            │
│                                            │ Migrate User     │            │
│                                            │ Attributes       │            │
│                                            └────────┬─────────┘            │
│                                                     │                      │
│                                                     ▼                      │
│                                            ┌──────────────────┐            │
│                                            │ Generate Cognito │            │
│                                            │ Token (Admin)    │            │
│                                            └────────┬─────────┘            │
│                                                     │                      │
│                                                     ▼                      │
│                                            ┌──────────────────┐            │
│                                            │ Update Migration │            │
│                                            │ Status: Complete │            │
│                                            └────────┬─────────┘            │
│                                                     │                      │
│                                                     ▼                      │
│                                            ┌──────────────────┐            │
│                                            │ Return Cognito   │            │
│                                            │ Token to Client  │            │
│                                            └──────────────────┘            │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘


                              ROLLOUT TIMELINE
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│  Week 1-2  │████████│ Infrastructure Setup                                │
│            │        │ • Cognito config                                    │
│            │        │ • Lambda functions                                  │
│            │        │ • Backend middleware                                │
│            │        │ • iOS dual SDK                                      │
│            │        │                                                     │
│  Week 3-4  │████████│ Pilot Testing                                       │
│            │        │ • Internal: 100 users                               │
│            │        │ • Beta: 5% of users                                 │
│            │        │ • Monitoring & fixes                                │
│            │        │                                                     │
│  Week 5    │█████   │ 15% Rollout                                         │
│  Week 6    │████████│ 35% Rollout                                         │
│  Week 7    │███████████│ 65% Rollout                                      │
│  Week 8    │██████████████│ 100% Rollout                                  │
│            │              │                                               │
│  Week 9-12 │████████│ Cleanup                                             │
│            │        │ • Disable Passage                                   │
│            │        │ • Remove legacy code                                │
│            │        │ • Cost optimization                                 │
│            │        │                                                     │
│            0%      25%      50%      75%      100%                         │
│                   Users Migrated                                          │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## Conclusion

This migration strategy prioritizes **zero downtime**, **user experience continuity**, and **risk mitigation** through a gradual, monitored rollout. The Just-in-Time migration approach ensures users experience no disruption, while the dual-provider bridge allows for instant rollback if issues arise.

**Key Success Factors:**
1. Comprehensive testing with pilot groups
2. Real-time monitoring and alerting
3. Clear rollback procedures
4. Progressive deployment with feature flags
5. Proactive user communication

**Estimated Total Effort**: 180-220 engineering hours across 12 weeks
**Risk Level**: Low (with proper testing and staged rollout)
**User Impact**: Minimal to none (transparent migration for 98%+ of users)

