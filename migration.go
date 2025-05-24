package migration

import (
	"os"
	"path/filepath"
	"time"
)

type Migration struct {
	name     string // Name
	fileName string // File name
	up       string // Up
	down     string // Down
}

type MigrationRecord struct {
	ID        int       // ID
	Version   string    // Version
	Name      string    // Name
	AppliedAt time.Time // Applied
}

// Create new Migration instance return pointer to Migration
func NewMigration(fileName string) *Migration {
	return &Migration{
		fileName: fileName,
	}
}

// Parse migration file
func (m *Migration) Parse() {
	content, err := os.ReadFile(m.fileName)
	if err != nil {
		panic(err)
	}

	name := filepath.Base(m.fileName)
	m.name = name
	m.up = FindEntry(`(-- up --([^-]+)-- up --)`, string(content))
	m.down = FindEntry(`(-- down --([^-]+)-- down --)`, string(content))
}
