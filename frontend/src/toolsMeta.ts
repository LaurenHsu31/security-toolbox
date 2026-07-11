export interface Control {
  key: string
  label: string
  type: 'select' | 'text' | 'password'
  options?: { value: string; label: string }[]
  default?: string
  placeholder?: string
}

export interface ToolUI {
  inputKey: string
  inputLabel: string
  placeholder: string
  sample?: string
  // Control values applied together with `sample` when the user clicks
  // "Use sample", for tools whose sample only decodes with matching controls.
  sampleControls?: Record<string, string>
  controls?: Control[]
  monoOutputKey?: string
}

const fmtControl: Control = {
  key: 'inputFormat',
  label: 'Input format',
  type: 'select',
  default: 'auto',
  options: [
    { value: 'auto', label: 'Auto-detect' },
    { value: 'hex', label: 'Hex' },
    { value: 'base64', label: 'Base64' },
    { value: 'base64url', label: 'Base64URL' },
    { value: 'utf8', label: 'UTF-8 text' }
  ]
}

const curveControl: Control = {
  key: 'curve',
  label: 'Curve',
  type: 'select',
  default: 'P-256',
  options: [
    { value: 'P-256', label: 'P-256' },
    { value: 'P-384', label: 'P-384' },
    { value: 'P-521', label: 'P-521' }
  ]
}

// A small self-signed P-256 certificate (CN=demo.example) plus its CSR and a
// degenerate PKCS#7 wrapping it — generated once with OpenSSL for the
// "Use sample" buttons. Nothing secret: it is demo material only.
const sampleCert = `-----BEGIN CERTIFICATE-----
MIIB4zCCAYigAwIBAgIUKpuin7o6qkbh+zrg4XmyOJYNTK4wCgYIKoZIzj0EAwIw
OjELMAkGA1UEBhMCVFcxFDASBgNVBAoMC0V4YW1wbGUgTGFiMRUwEwYDVQQDDAxk
ZW1vLmV4YW1wbGUwHhcNMjYwNzExMDcyNTQ3WhcNMzYwNzA4MDcyNTQ3WjA6MQsw
CQYDVQQGEwJUVzEUMBIGA1UECgwLRXhhbXBsZSBMYWIxFTATBgNVBAMMDGRlbW8u
ZXhhbXBsZTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABMXSxtxVE095z4cLW7xl
ayVQxJw7oZYviaokIj6WLeJWE6NJiH9j1bVxwJLg4U4mtML21ivEtmTE12zLxIrA
l0ujbDBqMB0GA1UdDgQWBBSWuIERHXLBZ45ossbQBN6Kr2axtTAfBgNVHSMEGDAW
gBSWuIERHXLBZ45ossbQBN6Kr2axtTAPBgNVHRMBAf8EBTADAQH/MBcGA1UdEQQQ
MA6CDGRlbW8uZXhhbXBsZTAKBggqhkjOPQQDAgNJADBGAiEA/+LLPja6I882DuPm
4V1baeST8Mb+AH8cALeh1Z+86FoCIQCpYq8dV7ux+mwIrlKfeNVS/hmP/cEYYuVf
oOlNZg4ulw==
-----END CERTIFICATE-----`

const sampleCSR = `-----BEGIN CERTIFICATE REQUEST-----
MIH1MIGcAgEAMDoxCzAJBgNVBAYTAlRXMRQwEgYDVQQKDAtFeGFtcGxlIExhYjEV
MBMGA1UEAwwMZGVtby5leGFtcGxlMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
xdLG3FUTT3nPhwtbvGVrJVDEnDuhli+JqiQiPpYt4lYTo0mIf2PVtXHAkuDhTia0
wvbWK8S2ZMTXbMvEisCXS6AAMAoGCCqGSM49BAMCA0gAMEUCIDR+H0gqcw6KTauK
vNAEAbXYEQQTvn8Gy6m4PVq2uhyuAiEAvupYX8j/ID0jJcmPF99HcrTDEVeZqRQi
9R0XqLMeFSI=
-----END CERTIFICATE REQUEST-----`

const samplePKCS7 = `-----BEGIN PKCS7-----
MIICEgYJKoZIhvcNAQcCoIICAzCCAf8CAQExADALBgkqhkiG9w0BBwGgggHnMIIB
4zCCAYigAwIBAgIUKpuin7o6qkbh+zrg4XmyOJYNTK4wCgYIKoZIzj0EAwIwOjEL
MAkGA1UEBhMCVFcxFDASBgNVBAoMC0V4YW1wbGUgTGFiMRUwEwYDVQQDDAxkZW1v
LmV4YW1wbGUwHhcNMjYwNzExMDcyNTQ3WhcNMzYwNzA4MDcyNTQ3WjA6MQswCQYD
VQQGEwJUVzEUMBIGA1UECgwLRXhhbXBsZSBMYWIxFTATBgNVBAMMDGRlbW8uZXhh
bXBsZTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABMXSxtxVE095z4cLW7xlayVQ
xJw7oZYviaokIj6WLeJWE6NJiH9j1bVxwJLg4U4mtML21ivEtmTE12zLxIrAl0uj
bDBqMB0GA1UdDgQWBBSWuIERHXLBZ45ossbQBN6Kr2axtTAfBgNVHSMEGDAWgBSW
uIERHXLBZ45ossbQBN6Kr2axtTAPBgNVHRMBAf8EBTADAQH/MBcGA1UdEQQQMA6C
DGRlbW8uZXhhbXBsZTAKBggqhkjOPQQDAgNJADBGAiEA/+LLPja6I882DuPm4V1b
aeST8Mb+AH8cALeh1Z+86FoCIQCpYq8dV7ux+mwIrlKfeNVS/hmP/cEYYuVfoOlN
Zg4ulzEA
-----END PKCS7-----`

const samplePublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAExdLG3FUTT3nPhwtbvGVrJVDEnDuh
li+JqiQiPpYt4lYTo0mIf2PVtXHAkuDhTia0wvbWK8S2ZMTXbMvEisCXSw==
-----END PUBLIC KEY-----`

// The P-256 base point G, as an uncompressed SEC1 point.
const p256GeneratorPoint =
  '046b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c2964fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5'

export const toolUI: Record<string, ToolUI> = {
  csr: {
    inputKey: 'input',
    inputLabel: 'CSR (PEM / DER / Base64)',
    placeholder: '-----BEGIN CERTIFICATE REQUEST-----\n...',
    sample: sampleCSR
  },
  cert: {
    inputKey: 'input',
    inputLabel: 'Certificate (PEM / DER / Base64)',
    placeholder: '-----BEGIN CERTIFICATE-----\n...',
    sample: sampleCert
  },
  pkcs7: {
    inputKey: 'input',
    inputLabel: 'PKCS#7 / CMS (PEM / DER / Base64)',
    placeholder: '-----BEGIN PKCS7-----\n...',
    sample: samplePKCS7
  },
  jwt: {
    inputKey: 'token',
    inputLabel: 'JWT',
    placeholder: 'eyJhbGciOi...',
    sample:
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c',
    // The sample token is signed with this well-known demo secret, so the
    // signature verifies as "valid" out of the box.
    sampleControls: { secret: 'your-256-bit-secret' },
    controls: [
      { key: 'secret', label: 'HMAC secret (HS*)', type: 'password', placeholder: 'your-256-bit-secret' }
    ]
  },
  json: {
    inputKey: 'input',
    inputLabel: 'JSON',
    placeholder: '{"hello":"world"}',
    sample: '{"a":1,"b":[2,3],"nested":{"x":true}}',
    monoOutputKey: 'formatted',
    controls: [
      {
        key: 'mode',
        label: 'Mode',
        type: 'select',
        default: 'beautify',
        options: [
          { value: 'beautify', label: 'Beautify' },
          { value: 'minify', label: 'Minify' },
          { value: 'validate', label: 'Validate only' }
        ]
      }
    ]
  },
  cbor: {
    inputKey: 'input',
    inputLabel: 'CBOR (hex / base64)',
    placeholder: 'a26161016162820203',
    sample: 'a26161016162820203',
    controls: [fmtControl]
  },
  cose: {
    inputKey: 'input',
    inputLabel: 'COSE (hex / base64)',
    placeholder: 'd2845...',
    // COSE_Sign1 example from the COSE spec test vectors (sign1-pass-01).
    sample:
      'd28443a10126a10442313154546869732069732074686520636f6e74656e742e58408eb33e4ca31d1c465ab05aac34cc6b23d58fef5c083106c4d25a91aef0b0117e2af9a291aa32e14ab834dc56ed2a223444547e01f11d3b0916e5a4c345cacb36',
    controls: [fmtControl]
  },
  'ecdsa-sig': {
    inputKey: 'input',
    inputLabel: 'ECDSA signature',
    placeholder: '3045022100...',
    // A real P-256 ECDSA-SHA256 signature in ASN.1 DER form.
    sample:
      '3046022100e024587b140a51bd52a371767e700fd441c7ecae55560703b1f6073165cb76590221008b0df7f6446ff72f44d3d01f690aa936e5fc904678286f0e831d261ac6638af0',
    controls: [
      {
        key: 'from',
        label: 'Convert from',
        type: 'select',
        default: 'der',
        options: [
          { value: 'der', label: 'ASN.1 DER → raw r||s' },
          { value: 'raw', label: 'raw r||s → ASN.1 DER' }
        ]
      },
      curveControl,
      fmtControl
    ]
  },
  cmac: {
    inputKey: 'input',
    inputLabel: 'Message',
    placeholder: '6bc1bee22e409f96e93d7e117393172a',
    // RFC 4493 test vector: with the default key this yields tag
    // 070a16b46b4d4144f79bdd9dd04a287c.
    sample: '6bc1bee22e409f96e93d7e117393172a',
    controls: [
      { key: 'key', label: 'AES key (hex / base64)', type: 'text', default: '2b7e151628aed2a6abf7158809cf4f3c' },
      {
        key: 'from',
        label: 'Message format',
        type: 'select',
        default: 'hex',
        options: [
          { value: 'auto', label: 'Auto-detect' },
          { value: 'hex', label: 'Hex' },
          { value: 'base64', label: 'Base64' },
          { value: 'utf8', label: 'UTF-8 text' }
        ]
      },
      { key: 'tagLen', label: 'Truncate tag to (bytes)', type: 'text', placeholder: '16' }
    ]
  },
  hkdf: {
    inputKey: 'ikm',
    inputLabel: 'Input keying material (IKM)',
    placeholder: '0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b',
    // RFC 5869 test case 1 with the default salt/info/length.
    sample: '0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b',
    controls: [
      { key: 'salt', label: 'Salt (hex / base64, optional)', type: 'text', default: '000102030405060708090a0b0c' },
      { key: 'info', label: 'Info (hex / base64, optional)', type: 'text', default: 'f0f1f2f3f4f5f6f7f8f9' },
      { key: 'length', label: 'Output length (bytes)', type: 'text', default: '42' },
      {
        key: 'hash',
        label: 'Hash',
        type: 'select',
        default: 'SHA-256',
        options: [
          { value: 'SHA-256', label: 'SHA-256' },
          { value: 'SHA-384', label: 'SHA-384' },
          { value: 'SHA-512', label: 'SHA-512' }
        ]
      }
    ]
  },
  'aes-gcm': {
    inputKey: 'data',
    inputLabel: 'Data (ciphertext+tag to decrypt, or plaintext to encrypt)',
    placeholder: '0388dace60b6a392f328c2b971b2fe78ab6e47d42cec13bdf53a67b21257bddf',
    // NIST GCM test case 2: decrypts to 16 zero bytes with the default
    // all-zero key and IV.
    sample: '0388dace60b6a392f328c2b971b2fe78ab6e47d42cec13bdf53a67b21257bddf',
    controls: [
      {
        key: 'mode',
        label: 'Mode',
        type: 'select',
        default: 'decrypt',
        options: [
          { value: 'decrypt', label: 'Decrypt' },
          { value: 'encrypt', label: 'Encrypt' }
        ]
      },
      { key: 'key', label: 'AES key (hex / base64)', type: 'text', default: '00000000000000000000000000000000' },
      { key: 'nonce', label: 'Nonce / IV (hex / base64)', type: 'text', default: '000000000000000000000000' },
      { key: 'aad', label: 'AAD (optional)', type: 'text' },
      { key: 'tag', label: 'Tag (decrypt, if separate)', type: 'text' },
      { key: 'tagLen', label: 'Tag length (bytes)', type: 'text', placeholder: '16' }
    ]
  },
  ecdh: {
    inputKey: 'private',
    inputLabel: 'Your private key (hex scalar or PEM)',
    placeholder: '-----BEGIN PRIVATE KEY----- ... or 32-byte hex scalar',
    // RFC 6979 A.2.5 P-256 private key; the sample peer key is the base
    // point G, so the shared secret is reproducible.
    sample: 'c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721',
    sampleControls: { public: p256GeneratorPoint },
    controls: [
      { key: 'public', label: 'Peer public key (hex point or PEM)', type: 'text', placeholder: '04abcd... or -----BEGIN PUBLIC KEY-----' },
      curveControl
    ]
  },
  jwk: {
    inputKey: 'input',
    inputLabel: 'Public key (PEM or JWK JSON)',
    placeholder: '-----BEGIN PUBLIC KEY----- ... or {"kty":"EC",...}',
    sample: samplePublicKeyPEM,
    controls: [
      {
        key: 'direction',
        label: 'Direction',
        type: 'select',
        default: 'pem-to-jwk',
        options: [
          { value: 'pem-to-jwk', label: 'PEM → JWK' },
          { value: 'jwk-to-pem', label: 'JWK → PEM' }
        ]
      }
    ]
  },
  eckey: {
    inputKey: 'input',
    inputLabel: 'EC public key (PEM) or point (hex)',
    placeholder: '04abcd... or -----BEGIN PUBLIC KEY-----',
    sample: p256GeneratorPoint,
    controls: [curveControl]
  },
  hash: {
    inputKey: 'input',
    inputLabel: 'Input',
    placeholder: 'text to hash',
    sample: 'abc',
    // Default to UTF-8: with auto-detect, short text like "abc" is valid
    // base64 and would silently hash the wrong bytes.
    controls: [
      {
        key: 'from',
        label: 'Interpret input as',
        type: 'select',
        default: 'utf8',
        options: [
          { value: 'utf8', label: 'UTF-8 text' },
          { value: 'hex', label: 'Hex' },
          { value: 'base64', label: 'Base64' },
          { value: 'base64url', label: 'Base64URL' },
          { value: 'auto', label: 'Auto-detect' }
        ]
      }
    ]
  },
  asn1: {
    inputKey: 'input',
    inputLabel: 'DER / BER (PEM / hex / base64)',
    placeholder: '3082...',
    // AlgorithmIdentifier for rsaEncryption: SEQUENCE { OID, NULL }.
    sample: '300d06092a864886f70d0101010500'
  },
  encode: {
    inputKey: 'input',
    inputLabel: 'Input',
    placeholder: 'hello',
    sample: 'hello',
    controls: [
      {
        key: 'from',
        label: 'Interpret input as',
        type: 'select',
        default: 'auto',
        options: [
          { value: 'auto', label: 'Auto-detect' },
          { value: 'utf8', label: 'UTF-8 text' },
          { value: 'hex', label: 'Hex' },
          { value: 'base64', label: 'Base64' },
          { value: 'base64url', label: 'Base64URL' }
        ]
      }
    ]
  },
  oid: {
    inputKey: 'input',
    inputLabel: 'OID (dotted) or name',
    placeholder: '2.5.4.3  or  commonName',
    sample: '2.5.4.3'
  },
  tlv: {
    inputKey: 'input',
    inputLabel: 'Hex data',
    placeholder: '6F0E8407A0000002471001A503880102',
    // A BER-TLV FCI template (matches the default BER-TLV mode; switch the
    // mode to APDU to parse command/response frames instead).
    sample: '6F0E8407A0000002471001A503880102',
    controls: [
      {
        key: 'mode',
        label: 'Mode',
        type: 'select',
        default: 'tlv',
        options: [
          { value: 'tlv', label: 'BER-TLV' },
          { value: 'apdu-command', label: 'APDU command' },
          { value: 'apdu-response', label: 'APDU response' }
        ]
      }
    ]
  }
}
