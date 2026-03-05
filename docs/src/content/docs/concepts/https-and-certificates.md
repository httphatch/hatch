---
title: "HTTPS and Certificates"
description: "How Hatch generates trusted TLS certificates so your local dev server has a green lock."
category: "concepts"
order: 2
lastUpdated: 2025-03-05
---

# HTTPS and Certificates

Hatch gives every `.test` domain a valid TLS certificate trusted by your browser. No warnings, no clicking through "Advanced", no self-signed cert exceptions.

## Why local HTTPS matters

Several browser features only work on secure origins:

- **Cookies** — `SameSite=None; Secure` cookies are rejected over HTTP
- **Mixed content** — HTTPS pages block HTTP subresources
- **Service workers** — require a secure context
- **Web Crypto API** — only available over HTTPS
- **WebAuthn** — requires a secure origin

If your production site uses HTTPS (it should), developing over plain HTTP means you're testing in a different environment than production. Bugs that only appear in production often trace back to this mismatch.

## What `hatch init` does

When you run `hatch init`, Hatch creates a two-tier certificate authority:

```
Root CA (self-signed, trusted in Keychain)
  └── Intermediate CA (signed by Root)
        └── Site certificates (signed by Intermediate)
```

The **root CA** is added to your macOS Keychain as a trusted certificate. This is the step that requires your password. Once trusted, any certificate signed by this CA chain is automatically valid in Safari, Chrome, and other apps that use the system trust store.

The **intermediate CA** signs the actual site certificates. It is not added to Keychain directly; trust chains through the root.

**Site certificates** are generated on-the-fly by the embedded Caddy server when a new domain is first accessed. They're cached and reissued automatically as needed.

## Certificate storage

| File | Purpose |
|------|---------|
| `~/.hatch/certs/rootCA.pem` | Root CA certificate |
| `~/.hatch/certs/rootCA-key.pem` | Root CA private key |
| `~/.hatch/certs/intermediateCA.pem` | Intermediate CA certificate |
| `~/.hatch/certs/intermediateCA-key.pem` | Intermediate CA private key |
| `~/.hatch/caddy/` | Cached site certificates |

Both CAs use ECDSA P-256 keys with 10-year validity.

## Firefox

Firefox uses its own certificate store instead of the system Keychain. You may need to import the root CA manually: Settings > Privacy & Security > Certificates > View Certificates > Import, then select `~/.hatch/certs/rootCA.pem`.

## Re-trusting

If the root CA becomes untrusted (after a system update or `hatch clean`), run:

```bash
hatch trust
```

This re-adds the root CA to your Keychain. If the CA files are missing, it regenerates them first.

See [Architecture](/docs/advanced/architecture) for technical details on the certificate hierarchy.
