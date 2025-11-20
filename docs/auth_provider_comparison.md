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
    *   **UX:** You can build standard Swift `UITextField`s for username and password. The user **never leaves your app**.
    *   **Effort:** High. You must handle UI states (loading, error, retry) manually.

*   **Auth0:**
    *   **Mechanism:** Uses **OIDC with PKCE**.
    *   **UX:** Requires opening a system browser modal (`ASWebAuthenticationSession`). The user sees a "Continue" alert, then a web page loads.
    *   **Effort:** Low. `Auth0.swift` SDK handles session and keychain.

*   **Clerk:**
    *   **Mechanism:** Web-wrapper.
    *   **UX:** **Web-View Dependent.** Clerk’s iOS SDK is essentially a wrapper around a web session. It does *not* support fully native fields easily without losing security features like bot protection.
    *   **Effort:** Very Low (if using their pre-built UIs).

*   **Authsignal:**
    *   **Mechanism:** Native Passkeys (WebAuthn).
    *   **UX:** **Best-in-class for MFA.** Provides a native iOS SDK specifically for **FaceID/TouchID**.
    *   **Effort:** Moderate. Used primarily as a second step, not the initial signup.

*   **Authress:**
    *   **Mechanism:** Standard OIDC.
    *   **UX:** Browser redirect flow similar to Auth0.
    *   **Effort:** Moderate.

### B. Backend Integration (Go + GraphQL)

*   **Amazon Cognito:**
    *   **Validation:** Standard JWT verification via JWKS.
    *   **Roles:** **Pre-Token Generation Lambda** (Node/Go) required to inject custom roles into claims.

*   **Auth0:**
    *   **Validation:** Standard JWT verification via JWKS.
    *   **Roles:** **Auth0 Actions** (JS) allow easy injection of custom claims (`https://myapp.com/roles`).

*   **Clerk:**
    *   **Validation:** Mature Go SDK (`github.com/clerkinc/clerk-sdk-go`).
    *   **Roles:** "Session Claims" feature makes adding custom data to tokens very easy via the dashboard.

*   **Authsignal:**
    *   **Validation:** `authsignal-go` SDK.
    *   **Roles:** Not a primary IdP. You use the SDK to check `action_state` (e.g., `ALLOW`, `CHALLENGE_REQUIRED`) rather than validating identity tokens.

*   **Authress:**
    *   **Validation:** Standard JWT verification.
    *   **Roles:** **Best in class.** Real-time permission checking API (`authressClient.UserPermissions.AuthorizeUser(...)`) instead of static token claims.

### C. Back-Office User Lookup (By Email)

*   **Amazon Cognito:**
    *   **API:** `ListUsers` (Filter: `email = "..."`).
    *   **Pros/Cons:** Reliable but strict rate limits.

*   **Auth0:**
    *   **API:** Management API v2 (`GET /api/v2/users-by-email`).
    *   **Pros/Cons:** Very clean SDK, generic rate limits apply.

*   **Clerk:**
    *   **API:** `client.Users().ListAll(params)`.
    *   **Pros/Cons:** Excellent Go SDK support. Intuitive search.

*   **Authress:**
    *   **API:** User Management API.
    *   **Pros/Cons:** Good support, designed for B2B hierarchy lookups.

*   **Authsignal:**
    *   **API:** User API.
    *   **Pros/Cons:** Used to look up *risk status* or *MFA enrollment*, not typically for general user profile data.

### E. Special Use Case: Passkey-First with Email OTP Fallback

You requested a specific flow: **Check Passkey -> (Fallback) -> Email OTP**.

| Provider | Feasibility | Native UI? | Notes |
| :--- | :--- | :--- | :--- |
| **Authsignal** | **Best Fit** | ✅ **Yes** | Authsignal is designed exactly for this "MFA orchestration". You can set up a rule: "If user has passkey, challenge passkey. Else, challenge email." The iOS SDK supports native passkeys directly. |
| **Clerk** | Good Fit | ⚠️ **Webview** | Clerk has "Passkey" and "Email Code" strategies built-in, but the flow is driven by their web components. Doing this *purely natively* is harder. |
| **Auth0** | Doable | ⚠️ **Webview** | Possible via "Universal Login" (Web). Native Passkey support in Auth0 is still maturing and complex to orchestrate manually without the web redirect. |
| **Cognito** | **Hard** | ✅ **Yes** | Cognito supports "Custom Auth Challenges" (Lambda triggers). You *could* build this logic (Check Passkey -> Else Email), but you have to write the FIDO2/WebAuthn server-side verification logic yourself in Go/Lambda. It is not "out of the box". |
| **Authress** | Neutral | ⚠️ **Webview** | Supports the standards, but orchestration would likely happen via their hosted login page. |

**Recommendation for this Flow:**
If you want this **exact flow** with a **Native UI** and minimal backend headache, **Authsignal** is the strongest candidate. You would use a basic User Directory (like Cognito or your own DB) and delegate the entire "Authentication Challenge" (Passkey vs Email) to Authsignal.

---

## 3. Pricing Breakdown (Monthly)

**Scenario:** 10,000 MAUs, 1,000 Logins.

| Provider | Plan | Cost | Notes |
| :--- | :--- | :--- | :--- |
| **Amazon Cognito** | Free Tier | **$0.00** | Free up to 50,000 MAUs indefinitely. |
| **Auth0** | B2C Essentials | **~$250.00+** | Price jumps significantly if you need "MFA" or "Custom Domains". |
| **Authress** | Standard | **Contact** | Likely usage-based, often competitive with Auth0 but less public data. |
| **Authsignal** | Base Plan | **Free** | Free up to 10,000 MAUs. Great for adding MFA cheaply. |
| **Clerk** | Hobby | **Free** | Free up to 10,000 MAUs. $0.02/user/mo after that. |

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

