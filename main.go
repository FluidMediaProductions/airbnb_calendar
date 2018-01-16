package main

import (
	"fmt"
	"net/http"
	"github.com/lestrrat/go-ical"
)

func main() {
	response, err := http.Get("https://www.airbnb.co.uk/calendar/ical/22759834.ics?s=f7c72662a1b98e5db6601e55732e1154")
	if err != nil {
		panic(err)
	} else {
		defer response.Body.Close()
		p := ical.NewParser()
		c, err := p.Parse(response.Body)
		if err != nil {
			panic(err)
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
			fmt.Print(start.Name(), " ", start.RawValue(), "\r\n")
			fmt.Print(end.Name(), " ", end.RawValue(), "\r\n")
			fmt.Print(uid.Name(), " ", uid.RawValue(), "\r\n")
			fmt.Print(summary.Name(), " ", summary.RawValue(), "\r\n")
		}
	}
}
