package migration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Migration struct {
	name     string
	fileName string
	up       string
	down     string
}

type MigrationRecord struct {
	ID        int
	Version   string
	Name      string
	AppliedAt time.Time
}

func NewMigration(fileName string) *Migration {
	return &Migration{
		fileName: fileName,
	}
}

func (m *Migration) Parse() {
	context, err := os.ReadFile(m.fileName)
	if err != nil {
		panic(err)
	}

	name := filepath.Base(m.fileName)
	rUp := regexp.MustCompile(`(-- up --([^-]+)-- up --)`)
	rDown := regexp.MustCompile(`(-- down --([^-]+)-- down --)`)

	m.name = name

	allSubMatches := rUp.FindAllStringSubmatch(string(context), -1)
	if len(allSubMatches) > 0 {
		if len(allSubMatches[0]) >= 2 {
			m.up = strings.Trim(allSubMatches[0][2], "\n")
		}
	}

	allSubMatches = rDown.FindAllStringSubmatch(string(context), -1)
	if len(allSubMatches) > 0 {
		if len(allSubMatches[0]) >= 2 {
			m.down = strings.Trim(allSubMatches[0][2], "\n")
		}
	}
}
