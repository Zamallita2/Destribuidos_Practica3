package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"testing"

	"airres-api/models"
)

func TestDemoPassPackageUnsigned(t *testing.T) {
	t.Setenv("BOARDING_PASS_SECRET", "test-pass-secret-with-at-least-32-characters")
	t.Setenv("PUBLIC_BASE_URL", "https://example.org")
	packageBytes, err := demoPassPackage(
		models.Boleto{IDBoleto: 11, IDVuelo: 42, NombrePasajero: "فاطمة الزهراء", Estado: "SALED"},
		models.Vuelo{ID: 42, SalidaProgramada: 1790200000, LlegadaProgramada: 1790236000},
		models.Ciudad{Codigo: "TYO", TimeZone: "Asia/Tokyo"},
		models.Ciudad{Codigo: "FRA", TimeZone: "Europe/Berlin"},
		models.Asiento{Codigo: "12A"}, models.Puerta{Puerta: "G7"},
	)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(packageBytes), int64(len(packageBytes)))
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte)
	for _, entry := range archive.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name] = content
	}
	if _, signed := files["signature"]; signed {
		t.Fatal("demo pass unexpectedly contains an Apple signature")
	}
	for _, filename := range []string{"pass.json", "manifest.json", "icon.png", "icon@2x.png", "icon@3x.png"} {
		if len(files[filename]) == 0 {
			t.Fatalf("missing %s", filename)
		}
	}
	var pass map[string]any
	if err := json.Unmarshal(files["pass.json"], &pass); err != nil {
		t.Fatal(err)
	}
	if pass["passTypeIdentifier"] != "pass.edu.aerolineaspabon.demo" || pass["teamIdentifier"] != "UNIVERSITY" {
		t.Fatalf("unexpected demo identifiers: %v", pass)
	}
	if !bytes.Contains(files["pass.json"], []byte("فاطمة الزهراء")) || !bytes.Contains(files["pass.json"], []byte("12A")) || !bytes.Contains(files["pass.json"], []byte("PKTransitTypeAir")) {
		t.Fatal("passenger, seat, or air transit details are missing")
	}
	var manifest map[string]string
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	for filename, expected := range manifest {
		digest := sha1.Sum(files[filename])
		if hex.EncodeToString(digest[:]) != expected {
			t.Fatalf("incorrect manifest digest for %s", filename)
		}
	}
}

func TestApplePassPackageStructure(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("OpenSSL is unavailable")
	}
	dir := t.TempDir()
	cert, key := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	command := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-subj", "/CN=Test Wallet", "-keyout", key, "-out", cert, "-days", "1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate temporary signing certificate: %v: %s", err, output)
	}
	t.Setenv("APPLE_PASS_TYPE_ID", "pass.example.test")
	t.Setenv("APPLE_TEAM_ID", "TESTTEAM")
	t.Setenv("APPLE_PASS_CERT_PATH", cert)
	t.Setenv("APPLE_PASS_KEY_PATH", key)
	t.Setenv("APPLE_WWDR_CERT_PATH", cert)
	t.Setenv("BOARDING_PASS_SECRET", "test-pass-secret-with-at-least-32-characters")
	t.Setenv("PUBLIC_BASE_URL", "https://example.org")
	if !appleWalletConfigured() {
		t.Fatal("temporary issuer files were not detected")
	}
	packageBytes, err := applePassPackage(
		models.Boleto{IDBoleto: 11, IDVuelo: 42, NombrePasajero: "张伟", Estado: "SALED"},
		models.Vuelo{ID: 42, SalidaProgramada: 1790200000, LlegadaProgramada: 1790236000},
		models.Ciudad{Codigo: "TYO", TimeZone: "Asia/Tokyo"},
		models.Ciudad{Codigo: "FRA", TimeZone: "Europe/Berlin"},
		models.Asiento{Codigo: "12A"}, models.Puerta{Puerta: "G7"},
	)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(packageBytes), int64(len(packageBytes)))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"pass.json": false, "manifest.json": false, "signature": false, "icon.png": false, "icon@2x.png": false, "icon@3x.png": false}
	for _, file := range archive.File {
		if _, ok := want[file.Name]; ok {
			want[file.Name] = true
		}
		if file.Name == "pass.json" {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(content, []byte("张伟")) || !bytes.Contains(content, []byte("12A")) {
				t.Fatalf("pass details missing: %s", content)
			}
		}
	}
	for filename, present := range want {
		if !present {
			t.Errorf("missing %s", filename)
		}
	}
}
