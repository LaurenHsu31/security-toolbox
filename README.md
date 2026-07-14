# security-toolbox

A local-only toolbox for decoding and converting security-sensitive data —
certificates, tokens, signatures, keys and smartcard data. It is built for
people who routinely paste CSRs, X.509 certs, JWTs and CCC / digital-key
material into online decoders and would rather **none of it leave their
machine**.

Everything runs in a single, tiny container. There is no database, no
telemetry, and no outbound network: the page is locked to its own origin by a
strict Content-Security-Policy, and the backend never stores a single byte of
your input.

## Quick start

```bash
docker compose up --build
# then open http://localhost:8075
```

That's the whole thing. The image is a `scratch`-based static binary (a few
MB) with the frontend embedded inside it.

## What it can do

| Category | Tools |
| --- | --- |
| Certificates | CSR decoder · X.509 certificate (full extension list, DER hex, PEM, ASN.1 tree) · PKCS#7 / CMS structure |
| Tokens | JWT decode + signature verification (HS / RS / PS / ES) |
| Data | JSON formatter (beautify / minify / validate) · CBOR decoder |
| Crypto | COSE_Sign1 / COSE_Key · ECDSA signature DER ⇄ raw r‖s · AES-CMAC · HKDF · AES-GCM encrypt/decrypt · ECDH shared secret · JWK ⇄ PEM · EC key / point inspector · hashes |
| Encoding | ASN.1 / DER tree dump · Base64 / Hex / Base64URL · OID lookup |
| Smartcard | BER-TLV · ISO 7816-4 APDU command / response |

Most tools auto-detect the input format: PEM (even mangled by copy-paste,
single-lined, or with literal `\n` escapes), raw DER in Base64 (padded or
not), hex with or without a `0x` prefix. Every tool has a **Use sample**
button that fills a known-good input — together with any control values it
needs (e.g. the JWT sample's HMAC secret) — so you can see the expected
output instantly.

### Working with the UI

- **Search.** Filter the sidebar with the search box — press `/` or
  `Cmd/Ctrl+K` from anywhere to jump to it.
- **Favorites.** Hover any tool and tap the ☆ to pin it; pinned tools collect
  in a **Favorites** group at the top, which you can **drag to reorder**.
- **Drop files.** Drag a file onto the input card: text files load as-is,
  binary files (e.g. a raw DER certificate) are converted to Base64.
- **Copy anything.** Click any value in a result to copy it (a ✓ confirms);
  **Copy all** grabs the whole result. Trees support **Expand / Collapse all**
  and full keyboard navigation.
- **It remembers.** Your input is kept per tool while you switch around, and
  the last-used tool, theme (Auto / Light / Dark toggle in the top bar) and
  favorites are saved in the browser's `localStorage` — everything stays on
  your machine (no server, no account) and survives reloads.

> **Digital car key note.** Cross-checked against the CCC Digital Key Technical
> Specification v4.0.0 (CCC-TS-101). The primitives the spec actually relies on
> are covered: X.509/DER certs, ASN.1 dump, TLV/APDU, OID lookup, ECDSA P-256
> (DER⇄raw, SHA-256), **AES-CMAC** (CMAC-AES-128, the spec's PRF and secure-
> channel C-MAC), **HKDF**, **AES-GCM** (id-aes128-GCM payloads) and **ECDH**
> key agreement. The spec does **not** use CBOR, COSE, JWT/JOSE, JWK or
> PKCS#7/CMS — those tools remain as general-purpose extras. **scrypt** (SPAKE2+
> verifier derivation) and a **SPAKE2+** helper are intentionally left out: both
> need non-stdlib code or an interactive protocol that doesn't fit the
> paste-and-decode model of this zero-dependency build.

## Privacy model

- **No outbound connections.** CSP is `default-src 'self'; connect-src 'self'`,
  so even a compromised page cannot exfiltrate your data.
- **No persistence.** Inputs are processed in memory and discarded; there is no
  database and nothing is logged. The only things stored are UI preferences
  (favorites, theme, last-used tool), kept in the browser's own
  `localStorage` — never sent anywhere.
- **Single origin.** Frontend and API are served by the same Go binary on one
  port.
- **Minimal surface.** The backend uses only the Go standard library — zero
  third-party packages — so the supply chain is just Go itself.

## Architecture

```
frontend/  Vite + Vue 3 + TypeScript  ──build──┐
                                                ▼
backend/   Go (stdlib only)  ──embed web/──▶  single static binary
           ├── internal/server  HTTP + CSP + SPA fallback
           └── internal/tools    one file per converter, a small registry
```

The Docker build compiles the frontend, copies `dist/` into `backend/web/`,
then `go build` embeds it with `//go:embed`. The result is one binary that
serves the SPA and a small JSON API at `POST /api/v1/run/{tool}`.

## Local development

Backend (needs Go 1.22+):

```bash
cd backend
go run .            # serves on :8080 (with the web/ placeholder)
```

Frontend (needs Node 20+), in a second terminal:

```bash
cd frontend
npm install
npm run dev         # Vite on :5173, proxies /api to :8080
```

## Tests

```bash
# Backend unit tests (generate their own inputs; no fixtures, no network)
cd backend && go test ./...

# Frontend component tests
cd frontend && npm run test:unit

# End-to-end (against a running container)
docker compose up --build -d
cd frontend && npm run test:e2e
```

The e2e suite includes a guard that clicks **Use sample** on every tool and
fails if any sample stops decoding — run it after touching samples, control
defaults, or backend handlers.

See [TEST_PLAN.md](./TEST_PLAN.md) for the full strategy, including
cross-checking the backend against `openssl`.

## License

MIT — see [LICENSE](./LICENSE).
