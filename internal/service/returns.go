package service

import (
	"sort"
	"time"

	"assessment.local/returns/internal/domain"
)

type StockSummary struct {
	ValueMinor     int64 `json:"value_minor"`
	AgedValueMinor int64 `json:"aged_value_minor"`
	LineCount      int   `json:"line_count"`
	Quantity       int   `json:"quantity"`
}

type StockDay struct {
	Date           string `json:"date"`
	ValueMinor     int64  `json:"value_minor"`
	AgedValueMinor int64  `json:"aged_value_minor"`
}

type SupplierStock struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	StockSummary
	Age0To7Minor   int64 `json:"age_0_7_minor"`
	Age8To30Minor  int64 `json:"age_8_30_minor"`
	Age31To60Minor int64 `json:"age_31_60_minor"`
	Age61PlusMinor int64 `json:"age_61_plus_minor"`
}

// StockBySupplier группирует положительные остатки по поставщику и возрасту.
// Возраст считается в календарных днях по Москве; суммы — в копейках.
func StockBySupplier(data domain.Dataset, cutoff time.Time, supplierID string) []SupplierStock {
	names := make(map[string]string, len(data.Suppliers))
	for _, supplier := range data.Suppliers {
		names[supplier.ID] = supplier.Name
	}
	groups := make(map[string]SupplierStock)
	for _, stock := range stockBalances(data, cutoff, supplierID) {
		if stock.quantity <= 0 {
			continue
		}
		group, found := groups[stock.supplierID]
		if !found {
			name := names[stock.supplierID]
			if name == "" {
				name = stock.supplierID
			}
			group = SupplierStock{ID: stock.supplierID, Name: name}
		}
		group.LineCount++
		group.Quantity += stock.quantity
		group.ValueMinor += stock.valueMinor
		if stock.ageDays > 30 {
			group.AgedValueMinor += stock.valueMinor
		}
		switch {
		case stock.ageDays <= 7:
			group.Age0To7Minor += stock.valueMinor
		case stock.ageDays <= 30:
			group.Age8To30Minor += stock.valueMinor
		case stock.ageDays <= 60:
			group.Age31To60Minor += stock.valueMinor
		default:
			group.Age61PlusMinor += stock.valueMinor
		}
		groups[stock.supplierID] = group
	}

	result := make([]SupplierStock, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ValueMinor == result[j].ValueMinor {
			return result[i].ID < result[j].ID
		}
		return result[i].ValueMinor > result[j].ValueMinor
	})
	return result
}

// StockTrend возвращает остатки, известные на конец каждого московского дня.
// Последняя точка ограничена cutoff, даже если он приходится на середину дня.
func StockTrend(data domain.Dataset, from, cutoff time.Time, supplierID string) []StockDay {
	days := make([]StockDay, 0)
	if from.After(cutoff) {
		return days
	}

	date := moscowDate(from)
	for !date.After(cutoff) {
		nextDate := date.AddDate(0, 0, 1)
		dayCutoff := nextDate.Add(-time.Nanosecond)
		if dayCutoff.After(cutoff) {
			dayCutoff = cutoff
		}
		stock := StockAt(data, dayCutoff, supplierID)
		days = append(days, StockDay{
			Date:           date.Format(time.DateOnly),
			ValueMinor:     stock.ValueMinor,
			AgedValueMinor: stock.AgedValueMinor,
		})
		date = nextDate
	}
	return days
}

// StockAt считает остаток по движениям, известным на дату cutoff включительно.
// Пустой supplierID означает всех поставщиков.
// AgedValueMinor — стоимость остатка старше 30 календарных дней по Москве.
func StockAt(data domain.Dataset, cutoff time.Time, supplierID string) StockSummary {
	var result StockSummary
	for _, stock := range stockBalances(data, cutoff, supplierID) {
		if stock.quantity <= 0 {
			continue
		}
		result.LineCount++
		result.Quantity += stock.quantity
		result.ValueMinor += stock.valueMinor
		if stock.ageDays > 30 {
			result.AgedValueMinor += stock.valueMinor
		}
	}
	return result
}

type stockBalance struct {
	supplierID string
	quantity   int
	valueMinor int64
	ageDays    int
}

func stockBalances(data domain.Dataset, cutoff time.Time, supplierID string) map[string]stockBalance {
	cutoffDate := moscowDate(cutoff)
	balances := make(map[string]stockBalance)
	for _, line := range data.Returns {
		if !line.Posted || (supplierID != "" && line.SupplierID != supplierID) {
			continue
		}
		receivedAt, err := time.Parse(time.RFC3339, line.WarehouseReceivedAt)
		if err != nil || receivedAt.After(cutoff) {
			continue
		}
		balances[line.ID] = stockBalance{
			supplierID: line.SupplierID,
			ageDays:    int(cutoffDate.Sub(moscowDate(receivedAt)) / (24 * time.Hour)),
		}
	}

	for _, movement := range data.StockMovements {
		stock, found := balances[movement.ReturnLineID]
		if !found || !domain.KnownAt(movement.OccurredAt, movement.RecordedAt, cutoff) {
			continue
		}
		stock.quantity += movement.QuantityDelta
		stock.valueMinor += movement.ValueDeltaMinor
		balances[movement.ReturnLineID] = stock
	}

	return balances
}

func moscowDate(value time.Time) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}
