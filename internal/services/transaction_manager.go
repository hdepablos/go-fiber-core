// internal/services/transaction_manager.go
package services

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"log"

	"gorm.io/gorm"
)

// TransactionManager contiene las dependencias y la lógica para gestionar transacciones.
type TransactionManager struct {
	Conn *connect.ConnectDTO
}

// NewTransactionManager es el constructor para nuestro gestor de transacciones.
func NewTransactionManager(conn *connect.ConnectDTO) TransactionManager {
	return TransactionManager{Conn: conn}
}

// ExecuteTx encapsula todo el ciclo de vida de una transacción.
func (tm *TransactionManager) ExecuteTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := tm.Conn.ConnectGormWrite.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // ojo validar si es necesario
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			log.Printf("error en rollback: %v, error original: %v", rbErr, err)
		}
		return err
	}

	return tx.Commit().Error
}
