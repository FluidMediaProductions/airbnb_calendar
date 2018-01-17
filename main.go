package main

import (
	"log"
	"net/http"
	"github.com/lestrrat/go-ical"
	"database/sql"
	_ "github.com/lib/pq"
	"time"
)

const iCalDateFormat = "20060102"

type pgDb struct {
	dbConn *sql.DB
}

func initDb() (*pgDb, error) {
	if dbConn, err := sql.Open("postgres", "user=postgres password=4TdBwvaXnpXxTzGMYQA3 host=127.0.0.1 dbname=airbnb_cal sslmode=disable"); err != nil {
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
        WHERE uid = $1
    `
    exists := true
    var oldSummary string
    var oldStart, oldEnd time.Time
    r := p.dbConn.QueryRow(lookupSql, uid.RawValue())
    err := r.Scan(nil, &oldSummary, &oldStart, &oldEnd)
    if err == sql.ErrNoRows {
    	exists = false
	} else if err != nil {
		return err
	}
	var newStart, newEnd time.Time
	newStart, err = time.Parse(iCalDateFormat, start.RawValue())
	if err != nil {
		return err
	}
	newEnd, err = time.Parse(iCalDateFormat, end.RawValue())
	if err != nil {
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
                SET dtstart = $2, dtend = $3, summary = $4
                WHERE uid = $1
            `
            _, err := p.dbConn.Exec(updateSql, uid.RawValue(), newStart, newEnd, summary.RawValue())
            if err != nil {
            	return err
			}
		}
	}
    return nil
}

func main() {
	db, err := initDb()
	if err != nil {
		log.Fatalf("Error initializing database: %v\n", err)
	}

	err = db.createTablesIfNotExist()
	if err != nil {
		log.Fatalf("Error creating database tables: %v\n", err)
	}

	response, err := http.Get("https://www.airbnb.co.uk/calendar/ical/22759834.ics?s=f7c72662a1b98e5db6601e55732e1154")
	if err != nil {
		log.Fatal(err)
	} else {
		defer response.Body.Close()
		p := ical.NewParser()
		c, err := p.Parse(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		for e := range c.Entries() {
			ev, ok := e.(*ical.Event)
			if !ok {
				continue
			}
			start, _ := ev.GetProperty("dtstart")
			end, _ := ev.GetProperty("dtend")
			uid, _ := ev.GetProperty("uid")
			summary, _ := ev.GetProperty("summary")
			err := db.insertOrUpdateEvent(uid, start, end, summary)
			if err != nil {
				log.Fatalf("Can't insert event: %v\n", err)
			}
		}
	}
}
