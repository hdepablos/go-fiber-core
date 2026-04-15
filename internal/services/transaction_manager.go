// internal/services/transaction_manager.go
package services

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"log"

	"gorm.io/gorm"
)

// TransactionManager define el contrato mínimo del gestor de transacciones.
type TransactionManager interface {
	Connection() *connect.ConnectDTO
	ExecuteTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// transactionManager contiene las dependencias y la lógica para gestionar transacciones.
type transactionManager struct {
	Conn *connect.ConnectDTO
}

// NewTransactionManager es el constructor para nuestro gestor de transacciones.
func NewTransactionManager(conn *connect.ConnectDTO) TransactionManager {
	return &transactionManager{Conn: conn}
}

// ExecuteTx encapsula todo el ciclo de vida de una transacción.
func (tm *transactionManager) ExecuteTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
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

func (tm *transactionManager) Connection() *connect.ConnectDTO {
	return tm.Conn
}
