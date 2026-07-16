package licenseverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
)

func makeJWT(t *testing.T, priv ed25519.PrivateKey, kid string, exp int64) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "EdDSA", "kid": kid})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": "forgebit", "iat": time.Now().Unix(), "exp": exp,
		"email": "customer@example.com", "product": "acme-cli", "tier": "pro",
		"environment": "live", "features": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}

	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(priv, []byte(h+"."+p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyJWTValid(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	token := makeJWT(t, priv, "key1", time.Now().Add(time.Hour).Unix())

	claims, err := VerifyJWT(token, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Product != "acme-cli" || claims.Tier != "pro" || claims.Kid != "key1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

// PHP's json_encode(empty array) emits "[]", not "{}" — Forgebit's server
// does this for the "features" claim when a license has no feature flags.
// This reproduces that exact wire format rather than Go's map-based encoder,
// which would never produce it.
func TestVerifyJWTEmptyArrayFeatures(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","kid":"key1"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"iss":"forgebit","iat":1700000000,"exp":` +
			strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) +
			`,"email":"c@example.com","product":"acme-cli","tier":"pro","environment":"live","features":[]}`,
	))
	sig := base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(header+"."+payload)))
	token := header + "." + payload + "." + sig

	claims, err := VerifyJWT(token, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Product != "acme-cli" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyJWTExpired(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	token := makeJWT(t, priv, "key1", time.Now().Add(-time.Hour).Unix())

	_, err := VerifyJWT(token, pub)
	if err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifyJWTTamperedSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	token := makeJWT(t, priv, "key1", time.Now().Add(time.Hour).Unix())

	parts := strings.Split(token, ".")
	tampered := parts[0] + ".eyJ0YW1wZXJlZCI6dHJ1ZX0." + parts[2]

	_, err := VerifyJWT(tampered, pub)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyJWTWrongKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	otherPub, _, _ := ed25519.GenerateKey(nil)
	token := makeJWT(t, priv, "key1", time.Now().Add(time.Hour).Unix())

	_, err := VerifyJWT(token, otherPub)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func makeForgebitBlob(t *testing.T, priv ed25519.PrivateKey, expiresAt int64, metadata map[string]any) string {
	t.Helper()

	licenseID := ulid.Make()
	vendorID := ulid.Make()
	productID := ulid.Make()

	metaJSON := []byte("{}")
	if metadata != nil {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
	}

	header := make([]byte, forgebitHeaderLen)
	copy(header[0:8], "FBLIC001")
	header[8] = 0x01
	copy(header[9:25], licenseID[:])
	copy(header[25:41], vendorID[:])
	copy(header[41:57], productID[:])
	binary.BigEndian.PutUint64(header[57:65], uint64(expiresAt))
	binary.BigEndian.PutUint32(header[65:69], uint32(len(metaJSON)))

	sig := ed25519.Sign(priv, header)

	buf := append([]byte{}, header...)
	buf = append(buf, sig...)
	buf = append(buf, metaJSON...)

	return base64.StdEncoding.EncodeToString(buf)
}

func TestVerifyForgebitBinaryValid(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	blob := makeForgebitBlob(t, priv, time.Now().Add(time.Hour).Unix(), map[string]any{"seats": float64(5)})

	fields, err := VerifyForgebitBinary(blob, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields.LicenseID == "" || fields.VendorID == "" || fields.ProductID == "" {
		t.Fatalf("expected ULIDs to be populated: %+v", fields)
	}
	if fields.Metadata["seats"] != float64(5) {
		t.Fatalf("unexpected metadata: %+v", fields.Metadata)
	}
}

func TestVerifyForgebitBinaryExpired(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	blob := makeForgebitBlob(t, priv, time.Now().Add(-time.Hour).Unix(), nil)

	_, err := VerifyForgebitBinary(blob, pub)
	if err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifyForgebitBinaryTampered(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	blob := makeForgebitBlob(t, priv, time.Now().Add(time.Hour).Unix(), nil)

	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		t.Fatal(err)
	}
	raw[10] ^= 0xFF // flip a bit inside the signed license_id
	tampered := base64.StdEncoding.EncodeToString(raw)

	_, err = VerifyForgebitBinary(tampered, pub)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestDetectFormat(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	jwt := makeJWT(t, priv, "k", time.Now().Add(time.Hour).Unix())
	forgebitBlob := makeForgebitBlob(t, priv, time.Now().Add(time.Hour).Unix(), nil)

	if got := DetectFormat(jwt); got != FormatJWT {
		t.Fatalf("expected FormatJWT, got %v", got)
	}
	if got := DetectFormat(forgebitBlob); got != FormatForgebit {
		t.Fatalf("expected FormatForgebit, got %v", got)
	}
	if got := DetectFormat("ABCD-EFGH-IJKL-MNOP"); got != FormatUnknown {
		t.Fatalf("expected FormatUnknown, got %v", got)
	}
}

func TestParsePublicKeyPEM(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	// Mirror Forgebit's server exactly: PEM-armor the base64 STRING of the
	// raw key (not a DER-encoded structure) — see VendorKeyController::publicPem.
	armored := "-----BEGIN PUBLIC KEY-----\n" + chunk(base64.StdEncoding.EncodeToString(pub), 64) + "-----END PUBLIC KEY-----\n"

	got, err := ParsePublicKeyPEM([]byte(armored))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != ed25519.PublicKeySize {
		t.Fatalf("unexpected key length: %d", len(got))
	}
}

func chunk(s string, width int) string {
	var b strings.Builder
	for i := 0; i < len(s); i += width {
		end := i + width
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
		b.WriteString("\n")
	}
	return b.String()
}
