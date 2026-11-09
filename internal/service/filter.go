package service

import (
	"fmt"
	"strings"
	"time"
)

var location = time.FixedZone("Europe/Moscow", 3*60*60)

type Filter struct {
	From       time.Time
	Until      time.Time
	SupplierID string
	Search     string
}

func NewFilter(from, to, supplierID, search string) (Filter, error) {
	start, err := time.ParseInLocation(time.DateOnly, from, location)
	if err != nil {
		return Filter{}, fmt.Errorf("начало периода должно быть датой ГГГГ-ММ-ДД")
	}
	end, err := time.ParseInLocation(time.DateOnly, to, location)
	if err != nil || end.Before(start) {
		return Filter{}, fmt.Errorf("конец периода должен быть датой не раньше начала")
	}
	if end.Sub(start) > 366*24*time.Hour {
		return Filter{}, fmt.Errorf("период не должен превышать один год")
	}
	return Filter{start, end.AddDate(0, 0, 1), supplierID, strings.ToLower(strings.TrimSpace(search))}, nil
}

func (f Filter) Includes(timestamp, supplierID string) bool {
	date, err := time.Parse(time.RFC3339, timestamp)
	return err == nil && !date.Before(f.From) && date.Before(f.Until) && (f.SupplierID == "" || f.SupplierID == supplierID)
}
