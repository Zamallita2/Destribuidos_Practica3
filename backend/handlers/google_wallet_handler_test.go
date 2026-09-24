package handlers

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"airres-api/models"
)

func TestGoogleWalletLinkSignsPassData(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	account := googleServiceAccount{ClientEmail: "demo@example.iam.gserviceaccount.com", PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})), PrivateKeyID: "test"}
	contents, _ := json.Marshal(account)
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_WALLET_ISSUER_ID", "123456")
	t.Setenv("GOOGLE_WALLET_CREDENTIALS_PATH", path)
	t.Setenv("BOARDING_PASS_SECRET", "test-pass-secret-with-at-least-32-characters")
	t.Setenv("PUBLIC_BASE_URL", "https://example.org")
	link, err := googleWalletLink(models.Boleto{IDBoleto: 11, IDVuelo: 42, NombrePasajero: "张伟", Estado: "SALED"},
		models.Vuelo{ID: 42, SalidaProgramada: 1790200000, LlegadaProgramada: 1790236000},
		models.Ciudad{Codigo: "TYO", TimeZone: "Asia/Tokyo"}, models.Ciudad{Codigo: "FRA", TimeZone: "Europe/Berlin"}, models.Asiento{Codigo: "12A"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(strings.TrimPrefix(link, "https://pay.google.com/gp/v/save/"), ".")
	if len(parts) != 3 {
		t.Fatalf("invalid JWT: %s", link)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatal(err)
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(claimsJSON), "张伟") || !strings.Contains(string(claimsJSON), "12A") || !strings.Contains(string(claimsJSON), "genericObjects") {
		t.Fatalf("missing pass details: %s", claimsJSON)
	}
}
