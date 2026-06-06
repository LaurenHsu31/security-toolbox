export interface Control {
  key: string
  label: string
  type: 'select' | 'text'
  options?: { value: string; label: string }[]
  default?: string
  placeholder?: string
}

export interface ToolUI {
  inputKey: string
  inputLabel: string
  placeholder: string
  sample?: string
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

export const toolUI: Record<string, ToolUI> = {
  csr: {
    inputKey: 'input',
    inputLabel: 'CSR (PEM / DER / Base64)',
    placeholder: '-----BEGIN CERTIFICATE REQUEST-----\n...'
  },
  cert: {
    inputKey: 'input',
    inputLabel: 'Certificate (PEM / DER / Base64)',
    placeholder: '-----BEGIN CERTIFICATE-----\n...'
  },
  pkcs7: {
    inputKey: 'input',
    inputLabel: 'PKCS#7 / CMS (PEM / DER / Base64)',
    placeholder: '-----BEGIN PKCS7-----\n...'
  },
  jwt: {
    inputKey: 'token',
    inputLabel: 'JWT',
    placeholder: 'eyJhbGciOi...',
    sample:
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c',
    controls: [
      { key: 'secret', label: 'HMAC secret (HS*)', type: 'text', placeholder: 'your-256-bit-secret' }
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
    controls: [fmtControl]
  },
  'ecdsa-sig': {
    inputKey: 'input',
    inputLabel: 'ECDSA signature',
    placeholder: '3045022100...',
    sample: '3006020111020112',
    controls: [
      {
        key: 'from',
        label: 'Convert from',
        type: 'select',
        default: 'der',
        options: [
          { value: 'der', label: 'ASN.1 DER \u2192 raw r||s' },
          { value: 'raw', label: 'raw r||s \u2192 ASN.1 DER' }
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
    controls: [
      { key: 'public', label: 'Peer public key (hex point or PEM)', type: 'text', placeholder: '04abcd... or -----BEGIN PUBLIC KEY-----' },
      curveControl
    ]
  },
  jwk: {
    inputKey: 'input',
    inputLabel: 'Public key (PEM or JWK JSON)',
    placeholder: '-----BEGIN PUBLIC KEY----- ... or {"kty":"EC",...}',
    controls: [
      {
        key: 'direction',
        label: 'Direction',
        type: 'select',
        default: 'pem-to-jwk',
        options: [
          { value: 'pem-to-jwk', label: 'PEM \u2192 JWK' },
          { value: 'jwk-to-pem', label: 'JWK \u2192 PEM' }
        ]
      }
    ]
  },
  eckey: {
    inputKey: 'input',
    inputLabel: 'EC public key (PEM) or point (hex)',
    placeholder: '04abcd... or -----BEGIN PUBLIC KEY-----',
    controls: [curveControl]
  },
  hash: {
    inputKey: 'input',
    inputLabel: 'Input',
    placeholder: 'text to hash',
    sample: 'abc',
    controls: [fmtControl]
  },
  asn1: {
    inputKey: 'input',
    inputLabel: 'DER / BER (PEM / hex / base64)',
    placeholder: '3082...'
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
    placeholder: '00A4040007A0000002471001',
    sample: '00A4040007A0000002471001',
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
