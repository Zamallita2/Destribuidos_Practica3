package db

import (
	"airres-api/models"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NextDomainID allocates in the caller's transaction. A rollback restores
// the counter, unlike PostgreSQL nextval(), and the two regions have disjoint
// numeric ranges even when writes move to a failover replica.
func NextDomainID(tx *gorm.DB, namespace string) (uint, error) {
	var table, column, condition string
	min := uint(0)
	switch namespace {
	case "flight_am":
		table, column, condition = "vuelos", "id", "id < 1000000000"
	case "flight_eu":
		table, column, condition = "vuelos", "id", "id >= 1000000000"
		min = 1000000000
	case "ticket_am":
		table, column, condition = "boletos", "id_boleto", "id_boleto < 1000000000"
	case "ticket_eu":
		table, column, condition = "boletos", "id_boleto", "id_boleto >= 1000000000"
		min = 1000000000
	default:
		return 0, errors.New("invalid_id_namespace")
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.IDAllocator{Name: namespace, NextValue: min + 1}).Error; err != nil {
		return 0, err
	}
	var counter models.IDAllocator
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&counter, "name = ?", namespace).Error; err != nil {
		return 0, err
	}
	var highest uint
	if err := tx.Raw("SELECT COALESCE(MAX(" + column + "),0) FROM " + table + " WHERE " + condition).Scan(&highest).Error; err != nil {
		return 0, err
	}
	value := counter.NextValue
	if value <= highest {
		value = highest + 1
	}
	if value <= min {
		value = min + 1
	}
	if value >= 2000000000 {
		return 0, errors.New("id_range_exhausted")
	}
	counter.NextValue = value + 1
	if err := tx.Save(&counter).Error; err != nil {
		return 0, err
	}
	return value, nil
}
