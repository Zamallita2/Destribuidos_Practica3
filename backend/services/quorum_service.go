package services

import (
	"context"
	"errors"
	"time"

	"airres-api/db"
	"airres-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

// ConfirmTicketReplication gives a purchase one additional durable copy
// before it is reported as confirmed. The outbox retries any missing node.
func ConfirmTicketReplication(source *gorm.DB, ticket models.Boleto) error {
	_ = ApplySyncEvent(SyncEvent{Action: "CREATE", Entity: "Boleto", Data: &ticket,
		LamportClock: ticket.LamportClock, VectorClock: ticket.VectorClock, NodeID: ticket.SourceNode})
	other := db.PGAmerica
	if source == db.PGAmerica {
		other = db.PGEuropaAsia
	}
	if db.IsAvailable(other) {
		var replica models.Boleto
		if other.First(&replica, ticket.IDBoleto).Error == nil && replica.Estado == ticket.Estado {
			return nil
		}
	}
	if db.IsMongoAvailable() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var replica models.Boleto
		if db.MongoDatabase.Collection("boletos").FindOne(ctx, bson.M{"id_boleto": ticket.IDBoleto}).Decode(&replica) == nil && replica.Estado == ticket.Estado {
			return nil
		}
	}
	return errors.New("compra_guardada_localmente_pendiente_de_replicacion")
}
