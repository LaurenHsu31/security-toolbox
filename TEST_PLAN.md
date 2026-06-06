# Test plan

The goal is confidence that every converter is correct, that the service never
talks to the network, and that the UI behaves under real and malformed input.

## 1. Backend unit tests (`backend/internal/tools/*_test.go`)

Table-driven tests that **generate their own inputs at runtime** (keys, CSRs,
certificates, signatures), so they need no fixture files and no network. Run:

```bash
cd backend && go test ./... -v
```

Covered today: CSR decode, X.509 decode, JWT (HMAC verify + claim parsing),
JSON beautify + error path, ECDSA signature DER⇄raw round-trip, Base64/Hex
encode, SHA-256 vector, APDU command parse, JWK⇄PEM round-trip, CBOR map
decode, ASN.1 top-level structure. CCC primitives are checked against published
vectors: AES-CMAC (RFC 4493), HKDF (RFC 5869 TC1), AES-GCM (NIST GCM TC2, plus
a wrong-key authentication-failure case) and ECDH (P-256 agreement symmetry).

### Cross-check against OpenSSL (manual / CI golden)

These confirm we agree with the reference implementation:

```bash
# CSR
openssl req -new -newkey rsa:2048 -nodes -keyout k.pem -out req.pem -subj "/CN=test"
#   paste req.pem into the CSR tool → subject CN, key size match openssl req -text

# Certificate fingerprints
openssl x509 -in cert.pem -noout -fingerprint -sha256
#   compare with the tool's fingerprints.sha256

# ECDSA signature shape (P-256 raw must be 64 bytes)
openssl asn1parse -in sig.der -inform DER
```

## 2. API tests (`net/http/httptest`)

Spin up `server.New` with the embedded FS and assert:

- `GET /api/v1/tools` returns the registry.
- `POST /api/v1/run/json` with valid/invalid bodies returns 200 / 422.
- Unknown tool returns 404.
- **Every response carries the locking CSP header** (`connect-src 'self'`).
- Body larger than 8 MiB is rejected.

## 3. Frontend component tests (Vitest + @vue/test-utils)

```bash
cd frontend && npm run test:unit
```

`ResultView` renders objects as humanized rows, arrays as lists, and recurses
into nested structures. Add cases for the API client's error unwrapping.

## 4. End-to-end (Playwright)

```bash
docker compose up --build -d
cd frontend && npm run test:e2e
```

User flows: select a tool → paste input → see decoded result; auto-detect
feedback shown; invalid JSON shows a line/column error; "Copy all" works.

## 5. No-network / privacy verification

- Run the container with the network disabled and confirm the app still works
  end-to-end (it should, since nothing is fetched):

  ```bash
  docker run --rm -p 8080:8080 --network none security-toolbox   # note: use host port mapping caveats
  ```

- In the browser devtools Network tab, confirm the only requests are to the
  app's own origin.
- Verify the CSP header on every response:

  ```bash
  curl -sI http://localhost:8080/ | grep -i content-security-policy
  ```

## 6. Fuzz / robustness (nice to have)

`go test -fuzz` on the TLV and ASN.1 parsers with random byte slices to ensure
they never panic on malformed input (they should only return errors).
