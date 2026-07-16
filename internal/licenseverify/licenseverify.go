// Checks jwt and forgebit license keys locally, offline, no network calls
// Serial/hmac/hwid/encfile keys aren't self-describing and need the API
package licenseverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrInvalidSignature  = errors.New("signature verification failed")
	ErrExpired           = errors.New("license expired")
	ErrUnsupportedFormat = errors.New("key is not offline-verifiable in this CLI (unsupported format)")
)

type Format string

const (
	FormatJWT      Format = "jwt"
	FormatForgebit Format = "forgebit"
	FormatUnknown  Format = "unknown"
)

func DetectFormat(key string) Format {
	if looksLikeJWT(key) {
		return FormatJWT
	}
	if data, err := base64.StdEncoding.DecodeString(key); err == nil && len(data) >= 8 && string(data[0:8]) == "FBLIC001" {
		return FormatForgebit
	}
	return FormatUnknown
}

func looksLikeJWT(key string) bool {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return false
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}
	return header.Alg == "EdDSA"
}

type Claims struct {
	Kid         string
	Issuer      string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Email       string
	Product     string
	Tier        string
	Environment string
	Features    map[string]any
}

// PHP encodes an empty features array as [] instead of {}, which a map can't decode
func decodeFeatures(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		return m
	}
	return nil
}

func VerifyJWT(token string, pub ed25519.PublicKey) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("not a JWT-shaped license key")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("decoding header: %w", err)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decoding payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("decoding signature: %w", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Claims{}, fmt.Errorf("parsing header: %w", err)
	}
	if header.Alg != "EdDSA" {
		return Claims{}, fmt.Errorf("unsupported algorithm %q", header.Alg)
	}

	signingInput := parts[0] + "." + parts[1]
	if !ed25519.Verify(pub, []byte(signingInput), sig) {
		return Claims{}, ErrInvalidSignature
	}

	var payload struct {
		Iss         string          `json:"iss"`
		Iat         int64           `json:"iat"`
		Exp         int64           `json:"exp"`
		Email       string          `json:"email"`
		Product     string          `json:"product"`
		Tier        string          `json:"tier"`
		Environment string          `json:"environment"`
		Features    json.RawMessage `json:"features"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return Claims{}, fmt.Errorf("parsing payload: %w", err)
	}

	claims := Claims{
		Kid:         header.Kid,
		Issuer:      payload.Iss,
		IssuedAt:    time.Unix(payload.Iat, 0),
		Email:       payload.Email,
		Product:     payload.Product,
		Tier:        payload.Tier,
		Environment: payload.Environment,
		Features:    decodeFeatures(payload.Features),
	}
	if payload.Exp != 0 {
		claims.ExpiresAt = time.Unix(payload.Exp, 0)
		if time.Now().After(claims.ExpiresAt) {
			return claims, ErrExpired
		}
	}

	return claims, nil
}

type Fields struct {
	LicenseID string
	VendorID  string
	ProductID string
	ExpiresAt time.Time
	Metadata  map[string]any
}

const forgebitHeaderLen = 69 // magic(8) + version(1) + 3 ULIDs(16 each) + expires_at(8) + metadata_len(4)
const forgebitSignedLen = forgebitHeaderLen + ed25519.SignatureSize

func VerifyForgebitBinary(raw string, pub ed25519.PublicKey) (Fields, error) {
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return Fields{}, fmt.Errorf("decoding base64: %w", err)
	}
	if len(data) < forgebitSignedLen {
		return Fields{}, errors.New("license blob is too short")
	}
	if string(data[0:8]) != "FBLIC001" {
		return Fields{}, errors.New("bad magic bytes")
	}

	licenseID := ulid.ULID(data[9:25]).String()
	vendorID := ulid.ULID(data[25:41]).String()
	productID := ulid.ULID(data[41:57]).String()
	expiresAtRaw := binary.BigEndian.Uint64(data[57:65])
	metaLen := binary.BigEndian.Uint32(data[65:69])
	sig := data[forgebitHeaderLen:forgebitSignedLen]

	if uint64(len(data)) < uint64(forgebitSignedLen)+uint64(metaLen) {
		return Fields{}, errors.New("license blob metadata is truncated")
	}

	if !ed25519.Verify(pub, data[0:forgebitHeaderLen], sig) {
		return Fields{}, ErrInvalidSignature
	}

	fields := Fields{
		LicenseID: licenseID,
		VendorID:  vendorID,
		ProductID: productID,
	}

	if metaLen > 0 {
		metadata := data[forgebitSignedLen : uint64(forgebitSignedLen)+uint64(metaLen)]
		_ = json.Unmarshal(metadata, &fields.Metadata)
	}

	if expiresAtRaw != 0 {
		fields.ExpiresAt = time.Unix(int64(expiresAtRaw), 0)
		if time.Now().After(fields.ExpiresAt) {
			return fields, ErrExpired
		}
	}

	return fields, nil
}

func LoadPublicKeyPEM(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePublicKeyPEM(data)
}

// The PEM body is a raw Ed25519 key, not SPKI/DER, so x509 can't parse it
func ParsePublicKeyPEM(data []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected a raw %d-byte Ed25519 key, got %d bytes", ed25519.PublicKeySize, len(block.Bytes))
	}
	return ed25519.PublicKey(block.Bytes), nil
}

type Result struct {
	Format         Format
	Valid          bool
	Reason         string
	JWTClaims      *Claims
	ForgebitFields *Fields
}

func Verify(key string, pub ed25519.PublicKey) Result {
	switch DetectFormat(key) {
	case FormatJWT:
		claims, err := VerifyJWT(key, pub)
		if err != nil {
			return Result{Format: FormatJWT, Valid: false, Reason: err.Error(), JWTClaims: &claims}
		}
		return Result{Format: FormatJWT, Valid: true, JWTClaims: &claims}
	case FormatForgebit:
		fields, err := VerifyForgebitBinary(key, pub)
		if err != nil {
			return Result{Format: FormatForgebit, Valid: false, Reason: err.Error(), ForgebitFields: &fields}
		}
		return Result{Format: FormatForgebit, Valid: true, ForgebitFields: &fields}
	default:
		return Result{Format: FormatUnknown, Valid: false, Reason: ErrUnsupportedFormat.Error()}
	}
}
