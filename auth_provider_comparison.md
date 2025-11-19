# Authentication Provider Analysis for Native iOS App

## 1. Executive Summary

| Feature | **Amazon Cognito** | **Auth0** | **Authress** | **Authsignal** | **Clerk** |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Best For** | **Native Experience & Cost** | **Dev Speed & Features** | **Complex Authorization** | **Fraud/MFA & Passwordless** | **Next.js/Web Startups** |
| **Cost (10k MAU)** | **$0.00 / month** | **~$250 - $800 / month** | **Contact Sales** | **Free** (Up to 10k MAU) | **Free** (Up to 10k MAU) |
| **iOS Login UI** | **Fully Native** (Custom Swift UI) | **Web Modal** (SFSafariViewController) | **Web Modal** (Browser Redirect) | **Web Modal** (Browser Redirect) | **Web Modal** (Browser Redirect) |
| **User Lookup** | ✅ Via `ListUsers` API | ✅ Via Management API | ✅ Via User API | ✅ Via User API | ✅ Via User API |
| **Go Integration** | Standard JWKS Verification | Standard JWKS Verification | Specialized SDKs | Standard JWKS / SDK | Standard JWKS / SDK |
| **Custom Claims** | Via Lambda Triggers (Complex) | Via Actions (Easy JS) | Native Permission Model | Limited | Via "Session Claims" (Easy) |
| **Social Login** | ✅ Limited (Google, FB, Apple, Amazon, OIDC) | ✅ Extensive (30+ providers out of box) | ✅ Standard (Google, Github, Microsoft, etc.) | ❌ Focus is on MFA/Passkeys | ✅ Extensive (20+ providers) |

---

## 2. Detailed Technical Comparison

### A. iOS Login Experience (Crucial for "Smoothness")

*   **Amazon Cognito (Winner for "Native" Feel):**
    *   **Mechanism:** Supports the **Secure Remote Password (SRP)** protocol.
    *   **UX:** You can build standard Swift `UITextField`s for username and password. The user **never leaves your app**. There is no browser popup, no "Show allows ‘app’ to use ‘auth0.com’ to sign in" alert.
    *   **Effort:** High. You must handle UI states (loading, error, retry) and flows (forgot password, new password required) manually using the AWS Amplify SDK or AWS Mobile SDK.

*   **Auth0:**
    *   **Mechanism:** Uses **OIDC with PKCE**.
    *   **UX:** Requires opening a system browser modal (`ASWebAuthenticationSession` or `SFSafariViewController`). The user sees a "Continue" system alert, then a web page loads.
    *   **Effort:** Low. The `Auth0.swift` SDK handles the session, keychain storage, and error mapping.

*   **Authress:**
    *   **Mechanism:** Standard OIDC.
    *   **UX:** Similar to Auth0, relies on a browser redirect flow to ensuring security and standard compliance.
    *   **Effort:** Moderate. SDKs are lighter weight but require standard OIDC handling.

### B. Backend Integration (Go + GraphQL)

All three generate **JWTs (JSON Web Tokens)**. Your Go server will validate them similarly using a JWKS (JSON Web Key Set) URL.

*   **Amazon Cognito:**
    *   **Validation:** Use a standard library like `github.com/golang-jwt/jwt` to verify the token signature against `https://cognito-idp.{region}.amazonaws.com/{userPoolId}/.well-known/jwks.json`.
    *   **Roles/Permissions:** To add roles (e.g., `admin`, `editor`) to the JWT, you must write a **Pre-Token Generation Lambda** function (Node.js/Python/Go) in AWS. This adds complexity.

*   **Auth0:**
    *   **Validation:** Similar JWKS verification against `https://{tenant}.auth0.com/.well-known/jwks.json`.
    *   **Roles/Permissions:** Much easier. You can use **Auth0 Actions** (JavaScript snippets) to inject custom claims (e.g., `https://myapp.com/roles`) into the token dynamically.

*   **Authress:**
    *   **Validation:** Standard JWT verification.
    *   **Roles/Permissions:** **Best in class.** Instead of baking roles into a static JWT, Authress provides a dedicated API to check permissions in real-time (e.g., `authressClient.UserPermissions.AuthorizeUser(...)`). This is powerful for complex apps but adds a network call or sidecar dependency.

### C. Back-Office User Lookup (By Email)

*   **Amazon Cognito:**
    *   **API:** `ListUsers`
    *   **Filter Syntax:** `email = "user@example.com"`
    *   **Go Code:** `cognitoClient.ListUsers(context.TODO(), &cognitoidentityprovider.ListUsersInput{Filter: aws.String("email = ...")})`
    *   **Rate Limits:** Default is somewhat low (can be throttled if you do this frequently).

*   **Auth0:**
    *   **API:** Management API v2 (`GET /api/v2/users-by-email`)
    *   **Go Code:** Uses the `auth0` management SDK. Very clean.
    *   **Rate Limits:** Generous for backend-to-backend calls, but strict limits on free/lower tiers.

### D. Emerging Contenders (Clerk & Authsignal)

These two are newer players with distinct philosophies.

#### **Clerk** (The "Frontend-First" Choice)
*   **Philosophy:** "User Management should be a UI component, not just an API."
*   **iOS Experience:**
    *   **Status:** ⚠️ **Web-View Dependent.**
    *   Clerk’s iOS SDK (`ClerkSDK`) is essentially a wrapper around a web browser session. It does *not* currently support a fully native implementation (like generic `UITextFields` sending JSON to an API) easily without losing security features like bot protection.
    *   **Pros:** You get "Profile Management," "Avatar Upload," and "Organization Switcher" screens for free (as web views).
    *   **Cons:** It feels like a web app inside a native shell.
*   **Go Integration:**
    *   **SDK:** `github.com/clerkinc/clerk-sdk-go`
    *   **Middleware:** Very mature. `clerk.WithSession(http.Handler)` automatically injects user context.
    *   **User Lookup:** Excellent. `client.Users().ListAll(params)` is intuitive and allows searching by email, phone, or external ID easily.
    *   **Pricing:** **Free for 10k MAU.** Very generous.

#### **Authsignal** (The "Fraud & MFA" Specialist)
*   **Philosophy:** "You already have a login (e.g., Cognito/Auth0/Custom), but you need secure MFA (Passkeys) and Fraud rules."
*   **iOS Experience:**
    *   **Status:** ✅ **Native Passkey Support.**
    *   They provide a native iOS SDK specifically for **Passkeys (FaceID/TouchID)**. This allows you to use Cognito for the "base" user and Authsignal for the "MFA" layer.
    *   **UX:** Highly native feel for the verification step.
*   **Go Integration:**
    *   **SDK:** `github.com/authsignal/authsignal-go`
    *   **Role:** It is **not** a primary IdP (Identity Provider). You wouldn't typically "look up a user by email" in Authsignal to find their profile data; you would look them up to check their *risk score* or *MFA status*.
*   **Best Use Case:** Use **Cognito** for the user directory (free) + **Authsignal** for high-security actions (e.g., "Withdraw Funds" triggers a FaceID check).

---

## 3. Pricing Breakdown (Monthly)

**Scenario:** 10,000 MAUs, 1,000 Logins.

| Provider | Plan | Cost | Notes |
| :--- | :--- | :--- | :--- |
| **Amazon Cognito** | Free Tier | **$0.00** | Free up to 50,000 MAUs indefinitely. |
| **Auth0** | B2C Essentials | **~$250.00+** | Price jumps significantly if you need "MFA" or "Custom Domains". |
| **Authress** | Standard | **Contact** | Likely usage-based, often competitive with Auth0 but less public data. |

---

## 4. Final Recommendation

### **Primary Recommendation: Amazon Cognito**

**Why?**
1.  **Native UI Control:** It is the *only* feasible way to get a truly native iOS login screen (no browser redirects) without compromising security, thanks to SRP support in the AWS SDKs.
2.  **Cost:** It saves you ~$3,000 - $10,000 per year compared to Auth0 at this scale.
3.  **Infrastructure:** Since you are likely hosting the Go API on AWS (EC2/Lambda/Fargate), keeping auth within the same VPC/IAM boundary simplifies security.

**Implementation Strategy:**
1.  **iOS:** Use **AWS Amplify Library for Swift** (`Amplify.Auth`). Build custom UI ViewControllers that call `Amplify.Auth.signIn(username:password:)`.
2.  **Go:** Use `github.com/golang-jwt/jwt/v5` to validate tokens.
3.  **Lookup:** Create a dedicated IAM user for your Back-office API with permission only for `cognito-idp:ListUsers`.

### **Alternative: Auth0**

**Why?**
*   If your team is very small and you want to outsource *every* aspect of login (forgot password flows, email templates, brute force protection) and are okay with the "Browser Modal" UX.

### **Alternative: Authress**

**Why?**
*   If your application is B2B (selling to other companies) and requires complex "User Management" features where your customers need to manage their own teams and permissions. Authress excels here.

### **Alternative: Authsignal**

**Why?**
*   If you want to build your own native UI but need a specialized "Passwordless" or "MFA-only" layer to sit alongside it. Authsignal is often used as an add-on for fraud detection and MFA rather than a primary identity provider for social login.

### **Alternative: Clerk**

**Why?**
*   If developer speed is the #1 priority and you are okay with a web-based login modal. Clerk has the best "Developer Experience" of the bunch but is heavily optimized for React/Web stacks. Their iOS SDK is newer and relies on web redirects, meaning you lose the "Native" feel compared to Cognito.

