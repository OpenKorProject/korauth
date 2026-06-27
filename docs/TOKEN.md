# TOKEN.md — korauth JWT Contract

This document **is immutable**: all other OpenKor services reference it.
Changes only if the token contract breaks require `v2`; compatible
additions are released as minor version updates to this document.

---

## 1. Algorithm

| Parameter       | Value                              |
| --------------- | ---------------------------------- |
| Algorithm       | **RS256** (RSASSA-PKCS1-v1_5)      |
| Key size        | 4096 bit (minimum 2048 bit)        |
| Signer          | korauth (private key)              |
| Verifier        | all services (public key from JWKS) |

Private key **never** goes over the network; other services only
obtain public key via `GET /v1/auth/.well-known/jwks.json`.

---

## 2. Access Token

### 2.1 Header

```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "<key-fingerprint>"
}
```

`kid`: first 16 hex characters of the SHA-256 fingerprint of the relevant public key
(e.g., `a3f8c2d1e0b94f7a`). Derived from key content; not assigned externally.

### 2.2 Payload (required claims)

```json
{
  "iss": "openkor-auth",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "7f3d1c00-0000-0000-0000-000000000001",
  "roles": ["admin"],
  "iat": 1719302400,
  "exp": 1719303300
}
```

| Claim       | Type           | Description                                                    |
| ----------- | -------------- | ----------------------------------------------------------- |
| `iss`       | string         | Fixed: `openkor-auth`                                       |
| `sub`       | UUID string    | User ID (`users.id`)                                |
| `tenant_id` | UUID string    | Tenant that the user belongs to (`tenants.id`)               |
| `roles`     | string array   | User's roles. In MVP, one element; empty array invalid. |
| `iat`       | Unix timestamp | Token creation time (UTC)                                   |
| `exp`       | Unix timestamp | `iat + ACCESS_TOKEN_TTL` (default **15 minutes**)         |

Valid role values: `admin`, `operator`, `viewer`

### 2.3 Restrictions

- Forbidden to include password hash, email, IP, or other PII in token.
- `jti` claim is not included in access token (not revocable; short lifespan is sufficient).
- `nbf` claim is not used; clock skew tolerance on verifier side max **30 seconds**.

---

## 3. Refresh Token

Refresh token is **opaque** (not JWT); it is a random string with no claims.

### 3.1 Generation

```
jti  = crypto/rand → 32 bytes → base64url (URL-safe, no padding) → 43 characters
```

### 3.2 Redis Structure

```
Key   : refresh:{jti}
Value : JSON (below)
TTL   : REFRESH_TOKEN_TTL (default 168 hours = 7 days)
```

```json
{
  "user_id":   "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "7f3d1c00-0000-0000-0000-000000000001",
  "roles":     ["admin"],
  "issued_at": 1719302400,
  "expires_at": 1719907200
}
```

When refresh token is used (token rotation):
1. Current `jti` is deleted from Redis.
2. New `jti` is generated and saved; new access token is returned.

On logout, `refresh:{jti}` is directly deleted — token becomes invalid immediately.

---

## 4. JWKS Endpoint

```
GET /v1/auth/.well-known/jwks.json
```

No authentication required. Response:

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "a3f8c2d1e0b94f7a",
      "n":   "<base64url-encoded modulus>",
      "e":   "AQAB"
    }
  ]
}
```

- `keys` array contains all active public keys (may be multiple during key rotation
  transition period).
- Response can be cached with `Cache-Control: public, max-age=3600` header.

### 4.1 Key Rotation Procedure

1. New RSA key pair is generated; new public key is added to JWKS (old is still active).
2. korauth now signs tokens with **new private key**.
3. Valid access tokens signed with old key (max 15 min) are accepted until expiry.
4. After expiry, old key is removed from JWKS.

---

## 5. Verification Flow (Other Services)

```
Request → service middleware
  1. Parse Authorization: Bearer <token> header
  2. Get kid from JWT header
  3. Find public key matching kid in JWKS cache
     → if not found, refresh JWKS endpoint (max 1x per request)
  4. Verify RS256 signature
  5. Check iss == "openkor-auth"
  6. Check exp (clock skew tolerance ≤ 30 sec)
  7. Pass tenant_id and roles claims to business logic
  8. Error → 401 Unauthorized
```

Services cache public key in memory; JWKS endpoint is only called when
`kid` is not found or at startup.

---

## 6. Brute-Force Protection

Rate limit unit: `tenant_id + username` pair.
Applied at: login endpoint, Redis counter.

| Threshold             | Duration    | Action                        |
| ---------------- | ------- | ---------------------------- |
| 5 failed logins | 15 min  | Account locked (soft-lock) |
| Lock duration     | 15 min  | Auto unlock              |

Redis key: `brute:{tenant_id}:{username}` → counter, TTL 15 minutes.

---

## 7. Password Policy

| Rule             | Value                                   |
| ----------------- | --------------------------------------- |
| Minimum length   | 8 characters                              |
| Required complexity | At least 1 uppercase + 1 digit + 1 special character |
| Hash algorithm  | **argon2id** (time=2, memory=64MB, threads=4) |
| Plain password        | Never logged, not written to DB  |

---

## 8. Contract Version

| Version | Date      | Change                |
| -------- | ---------- | ------------------------- |
| 1.0      | 2026-06-25 | Initial release                 |
