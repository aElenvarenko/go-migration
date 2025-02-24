package migration

import (
	"database/sql"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"time"

	"github.com/aElenvarenko/go-cmd"

	_ "github.com/lib/pq"
)

type Migrator struct {
	db         *sql.DB
	cmd        *cmd.Cmd
	migrations []*Migration
}

func NewMigrator() *Migrator {
	return &Migrator{
		migrations: make([]*Migration, 0),
	}
}

func (m *Migrator) ParseArgs() {
	m.cmd = cmd.NewCmd()
	m.cmd.SetGreeting("Migration tool").
		SetUsage("migration [flags] <command> [arguments]").
		AddFlag(
			m.cmd.NewFlag().
				SetName("config").
				SetAlias("c").
				SetDesc("choose configuration file"),
		).
		AddFlag(
			m.cmd.NewFlag().
				SetName("dir").
				SetAlias("d").
				SetDesc("choose directory with migrations").
				SetRequired(true),
		).
		AddFlag(
			m.cmd.NewFlag().
				SetName("url").
				SetAlias("u").
				SetDesc("pass connection url"),
		).
		AddCommand(
			m.cmd.NewCommand().
				SetName("create").
				SetDesc("create new migration").
				AddArgument("name").
				SetFunc(func(cmd *cmd.Cmd) {
					m.create()
				}),
		).
		AddCommand(
			m.cmd.NewCommand().
				SetName("up").
				SetDesc("up migrations").
				SetFunc(func(cmd *cmd.Cmd) {
					m.up()
				}),
		).
		AddCommand(
			m.cmd.NewCommand().
				SetName("down").
				SetDesc("down migrations").
				SetFunc(func(cmd *cmd.Cmd) {
					m.down()
				}),
		).
		Parse()
}

func (m *Migrator) create() {
	m.createMigration()
}

func (m *Migrator) up() {
	m.readMigrationsDir()
	m.initConnection()
	m.createMigrationTable()
	m.applyMigrations()
}

func (m *Migrator) down() {
	m.readMigrationsDir()
	m.initConnection()
	m.createMigrationTable()
	m.rollbackMigrations()
}

func (m *Migrator) createMigration() {
	dir := m.cmd.GetFlagValue("dir")
	_, err := os.Stat(dir)
	if err != nil {
		m.print(fmt.Sprintf("error: %s\n", err.Error()))
		return
	}
	nameArgument := m.cmd.GetCommand().GetArgument("name")
	if nameArgument != nil {
		createTime := time.Now()
		err := os.WriteFile(
			fmt.Sprintf(
				"%s/%s_%s.sql",
				dir,
				createTime.Format("20060102150405000"),
				nameArgument.GetValue(),
			),
			[]byte("-- up --\n-- up --\n-- down --\n-- down --\n"),
			os.ModePerm,
		)
		if err != nil {
			log.Printf("%+v\n", err.Error())
		}
	}
}

func (m *Migrator) readMigrationsDir() {
	dir := m.cmd.GetFlagValue("dir")
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		migration := NewMigration(fmt.Sprintf("%s/%s", dir, file.Name()))
		migration.Parse()
		m.migrations = append(m.migrations, migration)
	}
}

func (m *Migrator) initConnection() {
	db, err := sql.Open("postgres", m.cmd.GetFlagValue("url"))
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	m.db = db
}

func (m *Migrator) createMigrationTable() {
	_, err := m.db.Exec(fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s ("+
			"id SERIAL PRIMARY KEY,"+
			"version VARCHAR(50) NOT NULL UNIQUE,"+
			"name VARCHAR(255) NOT NULL,"+
			"applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP"+
			")",
		"migrations",
	))
	if err != nil {
		panic(err)
	}
}

func (m *Migrator) applyMigrations() {
	applied := make([]MigrationRecord, 0)
	rows, err := m.db.Query("SELECT * FROM migrations")
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		migration := MigrationRecord{}
		err := rows.Scan(
			&migration.ID,
			&migration.Version,
			&migration.Name,
			&migration.AppliedAt,
		)
		if err != nil {
			panic(err)
		}
		applied = append(applied, migration)
	}

	appliedTime := time.Now()
	for _, mig := range m.migrations {
		isApplied := false

		for _, migRec := range applied {
			if migRec.Name == mig.name {
				isApplied = true
			}
		}

		if !isApplied {
			startTime := time.Now()
			m.print(fmt.Sprintf("apply migration: %s\n", mig.name))
			m.applyMigration(mig)
			m.print(fmt.Sprintf("migration applied %ds\n", time.Since(startTime)/time.Second))
		}
	}
	m.print(fmt.Sprintf("total applied %ds\n", time.Since(appliedTime)/time.Second))
}

func (m *Migrator) applyMigration(migration *Migration) {
	if migration.up != "" {
		_, err := m.db.Exec(migration.up)
		if err != nil {
			log.Printf("%+v\n", err.Error())
		}
		hasher := crc32.NewIEEE()
		hasher.Write([]byte(migration.name))
		hashValue := hasher.Sum32()
		_, err = m.db.Exec(fmt.Sprintf(
			"INSERT INTO migrations (version, name) VALUES ('%d', '%s')",
			hashValue,
			migration.name,
		))
		if err != nil {
			log.Printf("%+v\n", err.Error())
		}
	}
}

func (m *Migrator) rollbackMigrations() {
	applied := make([]MigrationRecord, 0)
	rows, err := m.db.Query("SELECT * FROM migrations")
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		migration := MigrationRecord{}
		err := rows.Scan(
			&migration.ID,
			&migration.Version,
			&migration.Name,
			&migration.AppliedAt,
		)
		if err != nil {
			panic(err)
		}
		applied = append(applied, migration)
	}

	rollbackTime := time.Now()
	for i := len(applied) - 1; i >= 0; i-- {
		for _, mig := range m.migrations {
			if applied[i].Name == mig.name {
				migrationTime := time.Now()
				m.print(fmt.Sprintf("rollback migration: %s\n", mig.name))
				m.rollbackMigration(mig, &applied[i])
				m.print(fmt.Sprintf("migration rollback %ds\n", time.Since(migrationTime)/time.Second))
			}
		}
	}
	m.print(fmt.Sprintf("total rollback %ds\n", time.Since(rollbackTime)/time.Second))
}

func (m *Migrator) rollbackMigration(migration *Migration, migrationRecord *MigrationRecord) {
	if migration.down != "" {
		_, err := m.db.Exec(migration.down)
		if err != nil {
			log.Printf("%+v\n", err.Error())
		}
		_, err = m.db.Exec(fmt.Sprintf(
			"DELETE FROM migrations WHERE id = %d",
			migrationRecord.ID,
		))
		if err != nil {
			log.Printf("%+v\n", err.Error())
		}
	}
}

func (m *Migrator) print(message string) {
	os.Stdout.Write([]byte(message))
}
