package db

import (
	"context"
	"database/sql/driver"
	"errors"
	"time"

	"gorm.io/gorm"
)

// flightLockNamespace keeps flight locks apart from any other advisory lock.
const flightLockNamespace int64 = 7_000_000_000_000

// FlightLockOrder returns the PostgreSQL that arbitrates purchases of a flight:
// the region that owns the flight, then the other one if the owner is down.
func FlightLockOrder(flightID uint) []string {
	if flightID >= 1000000000 {
		return []string{"pg_eu", "pg_am"}
	}
	return []string{"pg_am", "pg_eu"}
}

// LockFlight serializes purchases of one flight across every API server.
// Each server writes tickets to the PostgreSQL chosen by the buyer's region,
// so a row lock in that database cannot see a concurrent buyer on the other
// one. A session-level advisory lock in the flight's owner database acts as
// the shared mutex. If the lock holder crashes, PostgreSQL ends its session
// and releases the lock automatically. The returned func releases it.
func LockFlight(ctx context.Context, flightID uint) (func(), string, error) {
	for _, node := range FlightLockOrder(flightID) {
		conn := PGAmerica
		if node == "pg_eu" {
			conn = PGEuropaAsia
		}
		if !IsAvailable(conn) {
			continue
		}
		release, err := advisoryLock(ctx, conn, flightLockNamespace+int64(flightID))
		if err != nil {
			continue
		}
		return release, node, nil
	}
	return nil, "", errors.New("bloqueo_distribuido_no_disponible")
}

func advisoryLock(ctx context.Context, conn *gorm.DB, key int64) (func(), error) {
	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	session, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := session.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		// A cancelled wait may race with the grant; discard the session.
		session.Raw(func(any) error { return driver.ErrBadConn })
		session.Close()
		return nil, err
	}
	return func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := session.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", key); err != nil {
			// Never return a connection that may still hold the lock to the
			// pool: discard it so PostgreSQL ends the session and releases it.
			session.Raw(func(any) error { return driver.ErrBadConn })
		}
		session.Close()
	}, nil
}
