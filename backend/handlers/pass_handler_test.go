package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"airres-api/models"
	"github.com/gin-gonic/gin"
)

func TestVerificationURLUsesReachableFrontendHost(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
	t.Setenv("BOARDING_PASS_SECRET", "test-pass-secret-with-at-least-32-characters")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "http://backend:8080/api/boletos/9/qr.png", nil)
	request.Header.Set("X-Forwarded-Host", "192.168.1.55:3001")
	context.Request = request
	url := verificationURLForRequest(context, models.Boleto{IDBoleto: 9, IDVuelo: 12})
	if !strings.HasPrefix(url, "http://192.168.1.55:3001/api/boletos/9/validar?token=") {
		t.Fatalf("unreachable QR URL: %s", url)
	}
}
