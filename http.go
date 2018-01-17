package main

import (
	"time"
	"github.com/lestrrat/go-ical"
	"net/http"
	"github.com/gorilla/mux"
)

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	c := ical.New()
	rows, err := db.getEvents()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var uid, summary string
		var start, end time.Time
		err := rows.Scan(&uid, &summary, &start, &end)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		e := ical.NewEvent()
		var startTxt, endTxt string
		startTxt = start.Format(iCalDateFormat)
		endTxt = end.Format(iCalDateFormat)
		e.AddProperty("uid", uid)
		e.AddProperty("summary", summary)
		e.AddProperty("dtstart", startTxt)
		e.AddProperty("dtend", endTxt)
		c.AddEntry(e)
	}
	if err := rows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ical.NewEncoder(w).Encode(c)
}

func serveHttp() error {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/calendar/ical.ics").HandlerFunc(handleCalendar)
	return http.ListenAndServe(":8080", router)
}