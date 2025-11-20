# Migration Comparison: Passage to New Provider

## Strategy Comparison

| Metric | **Pure Amazon Cognito** | **Auth0** |
| :--- | :--- | :--- |
| **User Friction** | ⭐⭐⭐⭐⭐ (Very Low) | ⭐⭐⭐ (Moderate) |
| **Reason** | Full control over UI. Supports "Silent Migration" via custom code. | Likely forces "Email Fallback" via Web UI. |
| **Development Speed** | ⭐ (Low) | ⭐⭐⭐⭐ (High) |
| **Reason** | **Hard Mode.** You must write your own FIDO2/WebAuthn server inside AWS Lambda. | Auth0 handles the "Email OTP" flow out of the box. |
| **Cost Effectiveness** | ⭐⭐⭐⭐⭐ (High) | ⭐⭐ (Low) |
| **Reason** | **$0** (Free Tier). | Auth0 Passwordless is expensive at scale. |
| **Maintenance** | ⭐⭐ (High) | ⭐⭐⭐⭐⭐ (Low) |
| **Reason** | You own the crypto code, the DynamoDB tables, and the email delivery logic. | Fully managed SaaS. |

---

## Why "Pure Cognito" is Hard but Powerful

The critical difference is **Ownership**.

*   **With Pure Cognito:** You are essentially building your own Auth Provider using AWS primitives (Lambda + DynamoDB). You have 100% control over the "State Machine" (e.g., Check Passkey -> Fail -> Check Email). This allows you to replicate the exact Passage flow you described.
*   **With Auth0:** You are renting a pre-built flow. Customizing it to this degree (Passkey first, then Email) inside their "Universal Login" is very difficult and often unsupported.

## Summary Recommendation

**If you have strong Backend Engineering resources:**
Go with **Pure Amazon Cognito**. It gives you the exact UX you want (Passkey -> Email Fallback) with zero vendor cost. But be prepared to write ~500-1000 lines of Go code to handle the WebAuthn verification and Lambda orchestration.

**If you want speed:**
Go with **Auth0** (or the Cognito + Authsignal hybrid). Building a secure FIDO2 server from scratch in Lambda is non-trivial and security-critical.
