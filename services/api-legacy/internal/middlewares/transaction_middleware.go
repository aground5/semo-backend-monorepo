package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const txKey = "tx"

// TransactionMiddleware starts a database transaction and adds it to the context.
// It automatically commits the transaction if the handler returns successfully,
// or rolls it back if an error occurs.
func TransactionMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Begin transaction
			tx := db.Begin()
			if tx.Error != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to begin transaction"})
			}

			// Add transaction to context
			c.Set(txKey, tx)

			// Handle panic to ensure rollback
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
					panic(r) // re-throw panic after rollback
				}
			}()

			// Execute the handler
			err := next(c)

			// Commit or rollback based on the error
			if err != nil {
				tx.Rollback()
				return err
			}

			// Commit transaction
			if err := tx.Commit().Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to commit transaction"})
			}

			return nil
		}
	}
}

// GetTransactionFromContext retrieves the transaction from the Echo context
func GetTransactionFromContext(c echo.Context) (*gorm.DB, error) {
	tx := c.Get(txKey)
	if tx == nil {
		return nil, errors.New("transaction not found in context")
	}
	
	gormDB, ok := tx.(*gorm.DB)
	if !ok {
		return nil, errors.New("invalid transaction type in context")
	}
	
	return gormDB, nil
}

// GetTxFromStandardContext retrieves the transaction from the standard context
func GetTxFromStandardContext(ctx context.Context) (*gorm.DB, error) {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if !ok || tx == nil {
			return nil, errors.New("transaction not found in context")
	}
	return tx, nil
}

// WithTransaction is a helper function for repository methods that need a transaction
// If tx is nil, it creates a new transaction, otherwise it uses the provided transaction
func WithTransaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	// Check if we already have a transaction in the context
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if ok && tx != nil {
		// Use the existing transaction
		return fn(tx)
	}
	
	// Begin a new transaction if none exists
	tx = db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}
