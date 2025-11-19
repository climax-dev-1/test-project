# Authentication Provider Analysis for Native iOS App

## 1. Executive Summary

| Feature | **Amazon Cognito** | **Auth0** | **Authress** |
| :--- | :--- | :--- | :--- |
| **Best For** | **Native Experience & Cost** | **Dev Speed & Features** | **Complex Authorization** |
| **Cost (10k MAU)** | **$0.00 / month** | **~$250 - $800 / month** | **Contact Sales** |
| **iOS Login UI** | **Fully Native** (Custom Swift UI) | **Web Modal** (SFSafariViewController) | **Web Modal** (Browser Redirect) |
| **User Lookup** | ✅ Via `ListUsers` API | ✅ Via Management API | ✅ Via User API |
| **Go Integration** | Standard JWKS Verification | Standard JWKS Verification | Specialized SDKs |
| **Custom Claims** | Via Lambda Triggers (Complex) | Via Actions (Easy JS) | Native Permission Model |
| **Social Login** | ✅ Limited (Google, FB, Apple, Amazon, OIDC) | ✅ Extensive (30+ providers out of box) | ✅ Standard (Google, Github, Microsoft, etc.) |

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

