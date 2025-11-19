# Migration Comparison: Passage to New Provider

## Metrics Comparison

| Metric | **Amazon Cognito + Authsignal** | **Auth0** |
| :--- | :--- | :--- |
| **User Friction** | ⭐⭐⭐⭐⭐ (Very Low) | ⭐⭐⭐ (Moderate) |
| **Reason** | Can support "Legacy Login" (Passage) side-by-side with new login. User uses FaceID to migrate. | Likely forces "Email Fallback" first because bridging two different Passkey providers in Universal Login is complex. |
| **Development Speed** | ⭐⭐⭐ (Moderate) | ⭐⭐⭐⭐ (High) |
| **Reason** | Requires custom Go logic to "Exchange Passage Token for Cognito Token". | Auth0 handles the "Email OTP" flow out of the box, less custom code. |
| **Cost Effectiveness** | ⭐⭐⭐⭐⭐ (High) | ⭐⭐ (Low) |
| **Reason** | Cognito is Free. Authsignal is Free/Cheap. | Auth0 Passwordless/MFA is expensive at scale. |
| **Long-Term UX** | ⭐⭐⭐⭐⭐ (Native) | ⭐⭐⭐ (Web-based) |
| **Reason** | Final state is 100% Native UI (FaceID). | Final state is likely still a Web Modal (Universal Login). |

---

## Why "Cognito + Authsignal" wins the Migration

The critical difference is **Control**.

*   **With Cognito + Authsignal:** You control the iOS ViewControllers. You can render the "Old Passage Button" AND the "New FaceID Button" on the same screen. You can quietly exchange tokens in the background.
*   **With Auth0:** You hand off control to a Web URL (`auth.myapp.com`). You cannot easily inject the "Passage iOS SDK" into that web page. This forces you to fallback to Email OTP for the migration step, which is annoying for users accustomed to FaceID.

## Summary Recommendation

**Go with the [Amazon Cognito + Authsignal] strategy.**

1.  It preserves the **"FaceID-first"** experience your users love.
2.  It allows a "Silent Migration" where the user authenticates with the old system, and you seamlessly upgrade them to the new system in the background.
3.  It avoids the "Email Loop" friction.

