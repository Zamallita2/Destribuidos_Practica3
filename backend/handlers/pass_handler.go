package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
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
	base := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/api/boletos/%d/validar?token=%s", base, ticket.IDBoleto, url.QueryEscape(passToken(ticket)))
}

// In the classroom demo the frontend may be opened from a phone over the
// local network. Prefer that reachable host over a localhost-only default.
func verificationURLForRequest(c *gin.Context, ticket models.Boleto) string {
	base := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	forwardedHost := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0])
	host := forwardedHost
	if host == "" {
		host = c.Request.Host
	}
	if host != "" && !strings.HasPrefix(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") &&
		(base == "" || strings.Contains(base, "localhost") || strings.Contains(base, "127.0.0.1")) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		base = scheme + "://" + host
	}
	if base == "" {
		base = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/api/boletos/%d/validar?token=%s", base, ticket.IDBoleto, url.QueryEscape(passToken(ticket)))
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
