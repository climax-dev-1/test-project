# Migration Strategy: Passage to New Provider

## 1. Executive Summary & Core Constraints

**The Challenge: Non-Exportable Credentials**
Migrating from Passage (1Password) presents a unique challenge compared to traditional password-based migrations. Passage relies primarily on **Passkeys (FIDO2/WebAuthn)**.
*   **Passkeys cannot be exported.** The private keys live securely on the user's device (i.e., in the Secure Enclave on iOS or TPM on Windows) and are never shared with the server.
*   **Result:** You cannot simply "copy and paste" the database to a new provider. **Every user must re-register a credential** (create a new passkey or set a password) on the new platform.

**The Goal: Seamless Transition**
To minimize friction, we will avoid a "hard cutover" where users are locked out until they re-register. Instead, we will use a **Just-In-Time (JIT) Migration** strategy (often called "Lazy Migration").

---

## 2. Strategy A: Passage to Amazon Cognito

Cognito offers a "Native" feel but requires more manual orchestration for migration.

### Phase 1: Co-Existence (The "Bridge")
1.  **Backend Update**: Update your Go/GraphQL backend to validate JWTs from **both** Passage and Cognito.
    *   *Current*: Verifies Passage public keys.
    *   *New*: Checks `iss` (Issuer) claim. If Passage, verify with Passage JWKS. If Cognito, verify with Cognito JWKS.
2.  **User Import (Bulk)**:
    *   Export user list (Emails, IDs) from Passage via API.
    *   Import users into Cognito using CSV import or API (`AdminCreateUser`).
    *   *State*: Users exist in Cognito with status `FORCE_CHANGE_PASSWORD` or equivalent, but they have no credentials.

### Phase 2: Client-Side Migration (The Flow)
1.  **App Update**: Release a new version of the iOS App.
2.  **Login Flow Logic**:
    *   User opens app.
    *   App checks local storage: "Is this user already migrated?"
    *   **Case A: Already Migrated**: Show Native Cognito Login (Username/Password or Custom Passkey flow).
    *   **Case B: Not Migrated**: Show a custom "Sign In" screen that asks for Email.
3.  **The Switch**:
    *   App calls backend `CheckMigrationStatus(email)`.
    *   If user has a Cognito credential setup -> Route to Cognito.
    *   If user is legacy (Passage only) -> **Route to Passage Login** (Web/Native SDK).
4.  **The Capture (Post-Login)**:
    *   User logs in successfully with Passage.
    *   App gets Passage JWT.
    *   **Immediate Action**: Present a "Security Update" screen. "We are upgrading our login system. Please set a new password/passkey to continue."
    *   App calls Cognito `AdminSetUserPassword` (if using backend trust) or prompts user to complete Cognito registration flow using their verified email.
    *   Once Cognito credential is created, save "Migrated" flag locally.

### Phase 3: Sunset
1.  Monitor traffic. Once < 5% of logins are via Passage, send email campaign to remaining users: "Please log in to upgrade account."
2.  Remove Passage SDK code.

**Pros:** High control, users kept within native app as much as possible.
**Cons:** High development effort to build the orchestration logic.

---

## 3. Strategy B: Passage to Auth0

Auth0 excels at abstracting the complexity of migration, potentially allowing for a "Zero-Touch" migration for users if switching to Passwordless Email Links.

### Phase 1: Configuration
1.  **Auth0 Tenant**: Create tenant and configure "Passwordless" (Email Code/Link) connection.
2.  **User Import**:
    *   Use Auth0 Management API to bulk import users with `email_verified: true` (since we trust Passage's verification).

### Phase 2: The "Magic" Switch
Because Auth0 supports **Passwordless Email Links** out of the box, we can potentially skip the "Re-register credential" friction if we switch authentication methods.

1.  **App Update**: Replace Passage SDK with `Auth0.swift`.
2.  **The Flow**:
    *   User clicks "Login".
    *   Auth0 Modal opens (Web-based).
    *   User enters Email.
    *   **Scenario A (Passkey User)**: Since we can't migrate the passkey, we rely on **Email OTP/Link**.
        *   Auth0 sees user exists (from bulk import).
        *   Auth0 sends Email Link/Code.
        *   User clicks link -> Logged In.
    *   **Scenario B (New Credential)**:
        *   Once logged in, Auth0 can prompt "Enable Passkeys" or "Set Password" via a progressive profiling rule or simply let them stay on Email OTP.

### Phase 3: Enhancing Security (Optional)
If you *require* Passkeys and don't want to fall back to Email OTP:
1.  Use **Auth0 Actions** (Post-Login).
2.  If `event.user.app_metadata.migrated != true`:
    *   Redirect user to a page prompting them to enroll a new Passkey (WebAuthn).
    *   Mark `migrated = true`.

**Pros:** Drastically lower code effort. If you switch to Email Magic Links, users barely notice the difference (no "re-registration" needed, just a different login flow).
**Cons:** Monthly cost; reliance on web-modal UX.

---

## 4. Summary & Recommendation

| Feature | Passage -> Cognito Strategy | Passage -> Auth0 Strategy |
| :--- | :--- | :--- |
| **User Friction** | **Medium**. Users *must* set up new credentials explicitly inside the app. | **Low**. Users can seamlessly switch to Email OTP/Links without manual setup. |
| **Dev Effort** | **High**. Building the "check status -> route to correct provider" logic is complex. | **Low**. Replace SDK, bulk import users, rely on email verification. |
| **Cost** | **Low** (AWS Free Tier). | **High** (Auth0 B2C pricing). |

### Recommendation
If **Seamless Experience** is the absolute #1 priority and you have budget: **Choose Auth0**.
*   **Why?** You can bulk-import users and immediately allow them to log in via Email Magic Link. They don't need to "re-register" or "create a password." They just enter their email, get a code, and they are in. You can then upsell Passkeys later.

If **Native UX** and **Cost** are priorities: **Choose Cognito**.
*   **Why?** You maintain full control over the UI. You can build a nice native screen that says "Welcome back! Please secure your account with a new FaceID" and handle the transition elegantly, avoiding the web-redirect context switch.

