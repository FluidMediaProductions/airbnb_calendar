package main

import (
	"log"
	"net/http"
	"github.com/lestrrat/go-ical"
	"database/sql"
	_ "github.com/lib/pq"
	"time"
	"github.com/gorilla/mux"
)

const iCalDateFormat = "20060102"

type pgDb struct {
	dbConn *sql.DB
}

var db *pgDb

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
		// Run the insert
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
			// Run the update
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

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	// Create calendar
	c := ical.New()
	// Get rows
	rows, err := db.getEvents()
	if err != nil {
		// Send http error if we cant load events
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Iterare over rows
	for rows.Next() {
		// Setup variables for each row
		var uid, summary string
		var start, end time.Time
		// Pull row data out
		err := rows.Scan(&uid, &summary, &start, &end)
		if err != nil {
			// Send error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Create new event
		e := ical.NewEvent()
		// Convert times to text
		var startTxt, endTxt string
		startTxt = start.Format(iCalDateFormat)
		endTxt = end.Format(iCalDateFormat)
		// Add data
		e.AddProperty("uid", uid)
		e.AddProperty("summary", summary)
		e.AddProperty("dtstart", startTxt)
		e.AddProperty("dtend", endTxt)
		// Ad event to calendar
		c.AddEntry(e)
	}
	if err := rows.Err(); err != nil {
		// Send error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Write out calendar
	ical.NewEncoder(w).Encode(c)
}

func main() {
	var err error
	db, err = initDb()
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

	// Create router
	router := mux.NewRouter().StrictSlash(true)
	// Only accept GET at /calendar/ical.ics requests
	router.Methods("GET").Path("/calendar/ical.ics").HandlerFunc(handleCalendar)
	// Start the server and if it exits log the error
	log.Fatal(http.ListenAndServe(":8080", router))
}
