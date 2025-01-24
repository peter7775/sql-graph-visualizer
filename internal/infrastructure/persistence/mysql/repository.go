package mysql

import (
	"github.com/peter7775/alevisualizer/internal/application/ports"
	"database/sql"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]interface{}, error) {
	// Původní implementace FetchData
	return nil, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}
