package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"airres-api/db"
	"airres-api/models"
	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func passToken(ticket models.Boleto) string {
	secret := os.Getenv("BOARDING_PASS_SECRET")
	if len(secret) < 32 {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "airres-pass:%d:%d", ticket.IDBoleto, ticket.IDVuelo)
	return hex.EncodeToString(mac.Sum(nil))
}

func loadPurchasedTicket(c *gin.Context) (*gorm.DB, models.Boleto, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return nil, models.Boleto{}, false
	}
	conn, ticket, err := db.GetDBForTicketWrite(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
		return nil, ticket, false
	}
	if ticket.Estado != "SALED" {
		c.JSON(http.StatusGone, gin.H{"error": "El pase requiere un boleto comprado y vigente"})
		return nil, ticket, false
	}
	if passToken(ticket) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "BOARDING_PASS_SECRET debe configurarse con 32 caracteres o más"})
		return nil, ticket, false
	}
	return conn, ticket, true
}

func verificationURL(ticket models.Boleto) string {
	base := currentLANBaseURL()
	if base == "" {
		base = strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	}
	if base == "" {
		base = "http://localhost:3001"
	}
	return fmt.Sprintf("%s/verificar/%d?token=%s", base, ticket.IDBoleto, url.QueryEscape(passToken(ticket)))
}

// In the classroom demo the frontend may be opened from a phone over the
// local network. Prefer that reachable host over a localhost-only default.
func verificationURLForRequest(c *gin.Context, ticket models.Boleto) string {
	return fmt.Sprintf("%s/verificar/%d?token=%s", publicBaseURLForRequest(c), ticket.IDBoleto, url.QueryEscape(passToken(ticket)))
}

func currentLANBaseURL() string {
	path := os.Getenv("QR_LAN_URL_FILE")
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > 45*time.Second {
		return ""
	}
	address, err := url.Parse(strings.TrimSpace(string(content)))
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.Path != "" || address.RawQuery != "" {
		return ""
	}
	ip := net.ParseIP(address.Hostname())
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || address.Port() == "" {
		return ""
	}
	return address.String()
}

func localRequestHost(host string) bool {
	parsed, err := url.Parse("http://" + host)
	if err != nil {
		return true
	}
	name := strings.ToLower(parsed.Hostname())
	return name == "localhost" || name == "backend" || name == "host.docker.internal" || name == "127.0.0.1" || name == "::1"
}

func publicBaseURLForRequest(c *gin.Context) string {
	forwardedHost := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0])
	host := forwardedHost
	if host == "" {
		host = c.Request.Host
	}
	if host != "" && !localRequestHost(host) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		return scheme + "://" + host
	}
	base := currentLANBaseURL()
	if base == "" {
		base = strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	}
	if base == "" {
		base = "http://localhost:3001"
	}
	return base
}

func walletDownloadURLForRequest(c *gin.Context, ticketID uint) string {
	return fmt.Sprintf("%s/pase/%d/billetera", publicBaseURLForRequest(c), ticketID)
}

func GetQRNetworkAddress(c *gin.Context) {
	address := publicBaseURLForRequest(c)
	parsed, err := url.Parse(address)
	ready := err == nil && !localRequestHost(parsed.Host)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"url": address, "ready": ready})
}

func GetBoardingPass(c *gin.Context) {
	conn, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	var flight models.Vuelo
	var origin, destination models.Ciudad
	var seat models.Asiento
	var gate models.Puerta
	if conn.First(&flight, ticket.IDVuelo).Error != nil || conn.First(&origin, flight.IDOrigen).Error != nil || conn.First(&destination, flight.IDDestino).Error != nil || conn.First(&seat, ticket.IDAsiento).Error != nil || conn.First(&gate, flight.IDPuerta).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Datos del vuelo incompletos"})
		return
	}
	departureZone, err := time.LoadLocation(origin.TimeZone)
	if err != nil {
		departureZone = time.UTC
	}
	arrivalZone, err := time.LoadLocation(destination.TimeZone)
	if err != nil {
		arrivalZone = time.UTC
	}
	qrPath := fmt.Sprintf("/api/boletos/%d/qr.png?token=%s", ticket.IDBoleto, url.QueryEscape(passToken(ticket)))
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"id_boleto": ticket.IDBoleto, "pasajero": ticket.NombrePasajero, "clase": ticket.Clase,
		"vuelo": flight.ID, "origen": origin.Codigo, "destino": destination.Codigo,
		"asiento": seat.Codigo, "puerta": gate.Puerta,
		"salida_local":  time.Unix(flight.SalidaProgramada, 0).In(departureZone).Format(time.RFC3339),
		"llegada_local": time.Unix(flight.LlegadaProgramada, 0).In(arrivalZone).Format(time.RFC3339),
		"zona_salida":   origin.TimeZone, "zona_llegada": destination.TimeZone,
		"qr_url": qrPath, "validation_url": verificationURLForRequest(c, ticket),
	})
}

func GetBoardingQRCode(c *gin.Context) {
	_, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.Query("token")), []byte(passToken(ticket))) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	png, err := qrcode.Encode(verificationURLForRequest(c, ticket), qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

// GetWalletDownloadQRCode points a phone to the pass file. The QR inside the
// pass remains a separate boarding validation code.
func GetWalletDownloadQRCode(c *gin.Context) {
	_, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	png, err := qrcode.Encode(walletDownloadURLForRequest(c, ticket.IDBoleto), qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

func ValidateBoardingPass(c *gin.Context) {
	_, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.Query("token")), []byte(passToken(ticket))) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": "Token inválido"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "id_boleto": ticket.IDBoleto, "id_vuelo": ticket.IDVuelo, "id_asiento": ticket.IDAsiento, "pasajero": ticket.NombrePasajero})
}
