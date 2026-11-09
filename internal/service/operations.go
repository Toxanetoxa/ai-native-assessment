package service

import (
	"sort"
	"time"

	"assessment.local/returns/internal/domain"
)

type Source interface{ Snapshot() domain.Dataset }

type DatasetSource domain.Dataset

func (data DatasetSource) Snapshot() domain.Dataset { return domain.Dataset(data) }

type SupplierTotal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
	Quantity    int    `json:"quantity"`
	AmountMinor int64  `json:"amount_minor"`
}

type DayTotal struct {
	Date        string `json:"date"`
	Count       int    `json:"count"`
	AmountMinor int64  `json:"amount_minor"`
}

type OperationsReport struct {
	Count       int             `json:"count"`
	Quantity    int             `json:"quantity"`
	AmountMinor int64           `json:"amount_minor"`
	Days        []DayTotal      `json:"days"`
	Suppliers   []SupplierTotal `json:"suppliers"`
}

type Operations struct{ source Source }

func NewOperations(source Source) *Operations { return &Operations{source} }

func (s *Operations) Report(filter Filter) OperationsReport {
	data := s.source.Snapshot()
	cutoff, _ := time.Parse(time.RFC3339, data.Manifest.AsOf)
	result := OperationsReport{Days: []DayTotal{}, Suppliers: []SupplierTotal{}}
	names := make(map[string]string)
	for _, supplier := range data.Suppliers {
		names[supplier.ID] = supplier.Name
	}
	days := make(map[string]DayTotal)
	suppliers := make(map[string]SupplierTotal)
	for _, shipment := range data.Shipments {
		if !filter.Includes(shipment.ShippedAt, shipment.SupplierID) || !domain.KnownAt(shipment.ShippedAt, shipment.ShippedAt, cutoff) {
			continue
		}
		result.Count++
		result.Quantity += shipment.Quantity
		result.AmountMinor += shipment.AmountMinor
		date, _ := time.Parse(time.RFC3339, shipment.ShippedAt)
		dayKey := date.In(location).Format(time.DateOnly)
		day := days[dayKey]
		day.Date, day.Count, day.AmountMinor = dayKey, day.Count+1, day.AmountMinor+shipment.AmountMinor
		days[dayKey] = day
		supplier := suppliers[shipment.SupplierID]
		supplier.ID, supplier.Name = shipment.SupplierID, names[shipment.SupplierID]
		supplier.Count++
		supplier.Quantity += shipment.Quantity
		supplier.AmountMinor += shipment.AmountMinor
		suppliers[shipment.SupplierID] = supplier
	}
	end := filter.Until
	if cutoff.Before(end) {
		end = cutoff
	}
	for date := filter.From; date.Before(end); date = date.AddDate(0, 0, 1) {
		key := date.Format(time.DateOnly)
		day := days[key]
		day.Date = key
		result.Days = append(result.Days, day)
	}
	for _, supplier := range suppliers {
		result.Suppliers = append(result.Suppliers, supplier)
	}
	sort.Slice(result.Suppliers, func(i, j int) bool {
		if result.Suppliers[i].AmountMinor == result.Suppliers[j].AmountMinor {
			return result.Suppliers[i].ID < result.Suppliers[j].ID
		}
		return result.Suppliers[i].AmountMinor > result.Suppliers[j].AmountMinor
	})
	return result
}
