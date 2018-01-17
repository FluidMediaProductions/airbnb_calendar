package main

import (
	"time"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/lestrrat/go-ical"
	"fmt"
)

type pgDb struct {
	dbConn *sql.DB
}

type dbConfig struct {
	dbHost string
	dbName string
	dbUser string
	dbPass string
}

func initDb(config *dbConfig) (*pgDb, error) {
	connectString := fmt.Sprintf("user=%s password=%s host=%s dbname=%s",
		config.dbUser, config.dbPass, config.dbHost, config.dbName)
	if dbConn, err := sql.Open("postgres", connectString); err != nil {
		return nil, err
	} else {
		p := &pgDb{dbConn: dbConn}
		if err := p.dbConn.Ping(); err != nil {
			return nil, err
		}
		return p, nil
	}
}

func (p *pgDb) createTablesIfNotExist() error {
	createSql := `
       CREATE TABLE IF NOT EXISTS events (
       uid TEXT NOT NULL PRIMARY KEY,
       summary TEXT NOT NULL,
       dtstart TIMESTAMP NOT NULL,
       dtend TIMESTAMP NOT NULL);
    `
	if rows, err := p.dbConn.Query(createSql); err != nil {
		return err
	} else {
		rows.Close()
	}
	return nil
}

func (p *pgDb) insertOrUpdateEvent(uid *ical.Property, start *ical.Property, end *ical.Property, summary *ical.Property) error {
	lookupSql := `
        SELECT uid, summary, dtstart, dtend FROM events
        WHERE dtstart = $1 AND dtend = $2
    `
	exists := true
	var oldSummary, oldUid string
	var oldStart, oldEnd time.Time
	newStart, err := time.Parse(iCalDateFormat, start.RawValue())
	if err != nil {
		return err
	}
	newEnd, err := time.Parse(iCalDateFormat, end.RawValue())
	if err != nil {
		return err
	}
	r := p.dbConn.QueryRow(lookupSql, newStart, newEnd)
	err = r.Scan(&oldUid, &oldSummary, &oldStart, &oldEnd)
	if err == sql.ErrNoRows {
		exists = false
	} else if err != nil {
		return err
	}
	if !exists {
		insertSql := `
			INSERT INTO events
			(uid, dtstart, dtend, summary)
			VALUES ($1, $2, $3, $4)
        `
		_, err := p.dbConn.Exec(insertSql, uid.RawValue(), newStart, newEnd, summary.RawValue())
		if err != nil {
			return err
		}
	} else {
		if newStart != oldStart || newEnd != oldEnd || summary.RawValue() != oldSummary {
			updateSql := `
				UPDATE events
				SET dtstart = $3, dtend = $4, summary = $5, uid = $2
				WHERE uid = $1
        	`
			_, err := p.dbConn.Exec(updateSql, oldUid, uid.RawValue(), newStart, newEnd, summary.RawValue())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *pgDb) getEvents() (*sql.Rows, error) {
	lookupSql := `
        SELECT uid, summary, dtstart, dtend FROM events
  `
	r, err := p.dbConn.Query(lookupSql)
	if err != nil {
		return nil, err
	}
	return r, nil
}