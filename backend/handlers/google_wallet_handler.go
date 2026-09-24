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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"airres-api/models"
	"github.com/gin-gonic/gin"
)

type googleServiceAccount struct {
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
}

func googleWalletConfigured() bool {
	path := os.Getenv("GOOGLE_WALLET_CREDENTIALS_PATH")
	if os.Getenv("GOOGLE_WALLET_ISSUER_ID") == "" || path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func WalletCapabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"google": googleWalletConfigured(), "apple": appleWalletConfigured(), "generic_pkpass": true})
}

func googleWalletLink(ticket models.Boleto, flight models.Vuelo, origin, destination models.Ciudad, seat models.Asiento) (string, error) {
	if !googleWalletConfigured() {
		return "", errors.New("Google Wallet issuer credentials are not configured")
	}
	issuer := os.Getenv("GOOGLE_WALLET_ISSUER_ID")
	contents, err := os.ReadFile(os.Getenv("GOOGLE_WALLET_CREDENTIALS_PATH"))
	if err != nil {
		return "", err
	}
	var account googleServiceAccount
	if err := json.Unmarshal(contents, &account); err != nil {
		return "", err
	}
	block, _ := pem.Decode([]byte(account.PrivateKey))
	if block == nil {
		return "", errors.New("Google Wallet private key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("Google Wallet private key is not RSA")
	}
	classID := issuer + ".aerolineas_pabon_boarding"
	objectID := issuer + ".ticket_" + strconv.FormatUint(uint64(ticket.IDBoleto), 10)
	localized := func(value string) map[string]any {
		return map[string]any{"defaultValue": map[string]any{"language": "es", "value": value}}
	}
	textField := func(id, label, value string) map[string]any {
		return map[string]any{"id": id, "header": label, "body": value}
	}
	departureZone, err := time.LoadLocation(origin.TimeZone)
	if err != nil {
		departureZone = time.UTC
	}
	arrivalZone, err := time.LoadLocation(destination.TimeZone)
	if err != nil {
		arrivalZone = time.UTC
	}
	class := map[string]any{"id": classID, "issuerName": "Aerolíneas Pabón", "reviewStatus": "UNDER_REVIEW"}
	object := map[string]any{
		"id": objectID, "classId": classID, "state": "ACTIVE",
		"cardTitle": localized("Aerolíneas Pabón"), "header": localized(ticket.NombrePasajero),
		"subheader": localized(origin.Codigo + " → " + destination.Codigo),
		"barcode":   map[string]any{"type": "QR_CODE", "value": verificationURL(ticket)},
		"textModulesData": []any{
			textField("flight", "Vuelo", fmt.Sprintf("AP-%d", flight.ID)),
			textField("seat", "Asiento", seat.Codigo),
			textField("departure", "Salida · "+origin.Codigo, time.Unix(flight.SalidaProgramada, 0).In(departureZone).Format(time.RFC1123)),
			textField("arrival", "Llegada · "+destination.Codigo, time.Unix(flight.LlegadaProgramada, 0).In(arrivalZone).Format(time.RFC1123)),
		},
	}
	claims := map[string]any{"iss": account.ClientEmail, "aud": "google", "typ": "savetowallet", "iat": time.Now().Unix(), "origins": []string{},
		"payload": map[string]any{"genericClasses": []any{class}, "genericObjects": []any{object}}}
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": account.PrivateKeyID}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return "https://pay.google.com/gp/v/save/" + signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func GetGoogleWalletLink(c *gin.Context) {
	conn, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	var flight models.Vuelo
	var origin, destination models.Ciudad
	var seat models.Asiento
	if conn.First(&flight, ticket.IDVuelo).Error != nil || conn.First(&origin, flight.IDOrigen).Error != nil || conn.First(&destination, flight.IDDestino).Error != nil || conn.First(&seat, ticket.IDAsiento).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Datos del pase incompletos"})
		return
	}
	link, err := googleWalletLink(ticket, flight, origin, destination, seat)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": link})
}
