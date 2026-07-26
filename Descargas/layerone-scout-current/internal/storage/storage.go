package storage

import (
	"time"

	"layerone-scout/internal/model"
)

type Store interface {
	Load() (*Database, error)
	Save(db *Database) error
	Mutate(fn func(*Database) error) (*Database, error)
	Close() error
}

type Database struct {
	SchemaVersion int              `json:"schema_version"`
	Persons       []model.Person   `json:"persons"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}
