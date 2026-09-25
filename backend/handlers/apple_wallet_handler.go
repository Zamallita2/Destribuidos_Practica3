package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"airres-api/models"
	"github.com/gin-gonic/gin"
)

func appleWalletConfigured() bool {
	if os.Getenv("APPLE_PASS_TYPE_ID") == "" || os.Getenv("APPLE_TEAM_ID") == "" {
		return false
	}
	for _, name := range []string{"APPLE_PASS_CERT_PATH", "APPLE_PASS_KEY_PATH", "APPLE_WWDR_CERT_PATH"} {
		path := os.Getenv(name)
		if path == "" {
			return false
		}
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	_, err := exec.LookPath("openssl")
	return err == nil
}

func passIcon(size int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	blue := color.RGBA{R: 22, G: 57, B: 111, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, blue)
		}
	}
	// A simple aircraft silhouette keeps the asset self-contained.
	for y := size / 6; y < 5*size/6; y++ {
		for x := size/2 - size/16; x <= size/2+size/16; x++ {
			img.Set(x, y, white)
		}
	}
	for x := size / 6; x < 5*size/6; x++ {
		for y := size/2 - size/16; y <= size/2+size/16; y++ {
			img.Set(x, y, white)
		}
	}
	var buffer bytes.Buffer
	err := png.Encode(&buffer, img)
	return buffer.Bytes(), err
}

func applePassPackage(ticket models.Boleto, flight models.Vuelo, origin, destination models.Ciudad, seat models.Asiento, gate models.Puerta) ([]byte, error) {
	return passPackage(ticket, flight, origin, destination, seat, gate, os.Getenv("APPLE_PASS_TYPE_ID"), os.Getenv("APPLE_TEAM_ID"), verificationURL(ticket), true)
}

// The demo package keeps the standard pass.json/manifest/image layout, while
// intentionally omitting Apple's certificate signature. Third-party readers
// may open it; Apple's own Wallet will reject it.
func demoPassPackage(ticket models.Boleto, flight models.Vuelo, origin, destination models.Ciudad, seat models.Asiento, gate models.Puerta) ([]byte, error) {
	return demoPassPackageForURL(ticket, flight, origin, destination, seat, gate, verificationURL(ticket))
}

func demoPassPackageForURL(ticket models.Boleto, flight models.Vuelo, origin, destination models.Ciudad, seat models.Asiento, gate models.Puerta, barcodeURL string) ([]byte, error) {
	return passPackage(ticket, flight, origin, destination, seat, gate, "pass.edu.aerolineaspabon.demo", "UNIVERSITY", barcodeURL, false)
}

func passPackage(ticket models.Boleto, flight models.Vuelo, origin, destination models.Ciudad, seat models.Asiento, gate models.Puerta, passTypeID, teamID, barcodeURL string, signed bool) ([]byte, error) {
	zone, err := time.LoadLocation(origin.TimeZone)
	if err != nil {
		zone = time.UTC
	}
	arrivalZone, err := time.LoadLocation(destination.TimeZone)
	if err != nil {
		arrivalZone = time.UTC
	}
	field := func(key, label, value string) map[string]any {
		return map[string]any{"key": key, "label": label, "value": value}
	}
	pass := map[string]any{
		"formatVersion": 1, "passTypeIdentifier": passTypeID,
		"serialNumber": fmt.Sprintf("ticket-%d", ticket.IDBoleto), "teamIdentifier": teamID,
		"organizationName": "Aerolíneas Pabón", "description": "Pase de abordar",
		"logoText": "Aerolíneas Pabón", "foregroundColor": "rgb(255,255,255)", "backgroundColor": "rgb(22,57,111)",
		"barcode":      map[string]any{"format": "PKBarcodeFormatQR", "message": barcodeURL, "messageEncoding": "utf-8"},
		"barcodes":     []any{map[string]any{"format": "PKBarcodeFormatQR", "message": barcodeURL, "messageEncoding": "utf-8"}},
		"relevantDate": time.Unix(flight.SalidaProgramada, 0).UTC().Format(time.RFC3339),
		"boardingPass": map[string]any{
			"transitType":     "PKTransitTypeAir",
			"primaryFields":   []any{field("origin", "ORIGEN", origin.Codigo), field("destination", "DESTINO", destination.Codigo)},
			"secondaryFields": []any{field("flight", "VUELO", fmt.Sprintf("AP-%d", flight.ID)), field("seat", "ASIENTO", seat.Codigo)},
			"auxiliaryFields": []any{field("passenger", "PASAJERO", ticket.NombrePasajero), field("gate", "PUERTA", gate.Puerta),
				field("departure", "SALIDA LOCAL", time.Unix(flight.SalidaProgramada, 0).In(zone).Format("2006-01-02 15:04 MST")),
				field("arrival", "LLEGADA LOCAL", time.Unix(flight.LlegadaProgramada, 0).In(arrivalZone).Format("2006-01-02 15:04 MST"))},
		},
	}
	passJSON, err := json.Marshal(pass)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{"pass.json": passJSON}
	for _, asset := range []struct {
		name string
		size int
	}{{"icon.png", 38}, {"icon@2x.png", 76}, {"icon@3x.png", 114}} {
		content, err := passIcon(asset.size)
		if err != nil {
			return nil, err
		}
		files[asset.name] = content
	}
	manifest := map[string]string{}
	for name, content := range files {
		digest := sha1.Sum(content)
		manifest[name] = hex.EncodeToString(digest[:])
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = manifestJSON
	if signed {
		directory, err := os.MkdirTemp("", "aeropabon-pkpass-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(directory)
		manifestPath := filepath.Join(directory, "manifest.json")
		signaturePath := filepath.Join(directory, "signature")
		if err := os.WriteFile(manifestPath, manifestJSON, 0600); err != nil {
			return nil, err
		}
		command := exec.Command("openssl", "smime", "-binary", "-sign", "-signer", os.Getenv("APPLE_PASS_CERT_PATH"),
			"-inkey", os.Getenv("APPLE_PASS_KEY_PATH"), "-certfile", os.Getenv("APPLE_WWDR_CERT_PATH"),
			"-in", manifestPath, "-out", signaturePath, "-outform", "DER")
		if output, err := command.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("Apple pass signature: %w: %s", err, string(output))
		}
		signature, err := os.ReadFile(signaturePath)
		if err != nil {
			return nil, err
		}
		files["signature"] = signature
	}
	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	for name, content := range files {
		entry, err := zipWriter.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(content); err != nil {
			return nil, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return archive.Bytes(), nil
}

func buildDemoWalletPass(c *gin.Context) (uint, []byte, bool) {
	conn, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return 0, nil, false
	}
	var flight models.Vuelo
	var origin, destination models.Ciudad
	var seat models.Asiento
	var gate models.Puerta
	if conn.First(&flight, ticket.IDVuelo).Error != nil || conn.First(&origin, flight.IDOrigen).Error != nil || conn.First(&destination, flight.IDDestino).Error != nil || conn.First(&seat, ticket.IDAsiento).Error != nil || conn.First(&gate, flight.IDPuerta).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Datos del pase incompletos"})
		return 0, nil, false
	}
	packageBytes, err := demoPassPackageForURL(ticket, flight, origin, destination, seat, gate, verificationURLForRequest(c, ticket))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return 0, nil, false
	}
	return ticket.IDBoleto, packageBytes, true
}

func GetDemoWalletPass(c *gin.Context) {
	ticketID, packageBytes, ok := buildDemoWalletPass(c)
	if !ok {
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=AP-%d-demo.pkpass", ticketID))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/vnd.apple.pkpass", packageBytes)
}

// Safari opens a bare .pkpass in Apple Wallet. Wrapping it in ZIP lets the
// visitor save the same demo pass to Files and share it with a third-party app.
func GetDemoWalletPassZip(c *gin.Context) {
	ticketID, packageBytes, ok := buildDemoWalletPass(c)
	if !ok {
		return
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create(fmt.Sprintf("AP-%d-demo.pkpass", ticketID))
	if err == nil {
		_, err = entry.Write(packageBytes)
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo preparar la descarga"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=AP-%d-para-Passbook.zip", ticketID))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/zip", archive.Bytes())
}

func GetAppleWalletPass(c *gin.Context) {
	if !appleWalletConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Apple Wallet issuer certificate is not configured"})
		return
	}
	conn, ticket, ok := loadPurchasedTicket(c)
	if !ok {
		return
	}
	var flight models.Vuelo
	var origin, destination models.Ciudad
	var seat models.Asiento
	var gate models.Puerta
	if conn.First(&flight, ticket.IDVuelo).Error != nil || conn.First(&origin, flight.IDOrigen).Error != nil || conn.First(&destination, flight.IDDestino).Error != nil || conn.First(&seat, ticket.IDAsiento).Error != nil || conn.First(&gate, flight.IDPuerta).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Datos del pase incompletos"})
		return
	}
	packageBytes, err := applePassPackage(ticket, flight, origin, destination, seat, gate)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=AP-%d.pkpass", ticket.IDBoleto))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/vnd.apple.pkpass", packageBytes)
}
