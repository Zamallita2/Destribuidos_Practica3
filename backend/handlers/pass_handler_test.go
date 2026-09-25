package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"airres-api/models"
	"github.com/gin-gonic/gin"
)

func TestVerificationURLUsesReachableFrontendHost(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
	t.Setenv("QR_LAN_URL_FILE", "")
	t.Setenv("BOARDING_PASS_SECRET", "test-pass-secret-with-at-least-32-characters")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "http://backend:8080/api/boletos/9/qr.png", nil)
	request.Header.Set("X-Forwarded-Host", "192.168.1.55:3001")
	context.Request = request
	url := verificationURLForRequest(context, models.Boleto{IDBoleto: 9, IDVuelo: 12})
	if !strings.HasPrefix(url, "http://192.168.1.55:3001/verificar/9?token=") {
		t.Fatalf("unreachable QR URL: %s", url)
	}
}

func TestQRUsesCurrentWiFiAddressAfterNetworkChange(t *testing.T) {
	file := filepath.Join(t.TempDir(), "qr-lan-url.txt")
	t.Setenv("QR_LAN_URL_FILE", file)
	t.Setenv("PUBLIC_BASE_URL", "http://192.168.0.7:3001")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "http://localhost:8080/api/boletos/9/qr.png", nil)
	for _, address := range []string{"http://10.20.30.40:3001", "http://192.168.55.12:3001"} {
		if err := os.WriteFile(file, []byte(address), 0600); err != nil {
			t.Fatal(err)
		}
		if got := publicBaseURLForRequest(context); got != address {
			t.Fatalf("QR URL after network change: got %q, want %q", got, address)
		}
	}
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:3001")
	stale := time.Now().Add(-time.Minute)
	if err := os.Chtimes(file, stale, stale); err != nil {
		t.Fatal(err)
	}
	if got := publicBaseURLForRequest(context); got != "http://localhost:3001" {
		t.Fatalf("stale Wi-Fi address reused: %s", got)
	}
}

func TestWalletDownloadURLUsesReachableFrontendHost(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "http://backend:8080/api/boletos/9/wallet/qr.png", nil)
	request.Header.Set("X-Forwarded-Host", "192.168.1.55:3001")
	context.Request = request
	url := walletDownloadURLForRequest(context, 9)
	if url != "http://192.168.1.55:3001/pase/9/billetera" {
		t.Fatalf("wallet QR should open the mobile pass download page, got %s", url)
	}
}
