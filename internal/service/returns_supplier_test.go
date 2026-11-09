package service

import (
	"reflect"
	"testing"
	"time"

	"assessment.local/returns/internal/domain"
)

func TestStockBySupplierAgeGroupsAndTotals(t *testing.T) {
	// По Москве уже 30 сентября, хотя дата UTC — 29 сентября.
	cutoff := time.Date(2026, time.September, 29, 21, 30, 0, 0, time.UTC)
	data := domain.Dataset{Suppliers: []domain.Supplier{
		{ID: "SUP-A", Name: "Поставщик A"},
		{ID: "SUP-B", Name: "Поставщик B"},
		{ID: "SUP-Z", Name: "Нулевая стоимость"},
	}}
	for _, row := range []struct {
		id         string
		supplierID string
		receivedAt string
		quantity   int
		valueMinor int64
		unposted   bool
		recordedAt string
	}{
		{id: "age-0", supplierID: "SUP-A", receivedAt: "2026-09-30T00:15:00+03:00", quantity: 2, valueMinor: 100},
		{id: "age-7", supplierID: "SUP-A", receivedAt: "2026-09-23T00:00:00+03:00", quantity: 2, valueMinor: 200},
		{id: "age-8", supplierID: "SUP-A", receivedAt: "2026-09-22T23:59:59+03:00", quantity: 2, valueMinor: 300},
		{id: "age-30", supplierID: "SUP-A", receivedAt: "2026-08-30T21:00:00Z", quantity: 2, valueMinor: 400},
		{id: "age-31", supplierID: "SUP-A", receivedAt: "2026-08-30T20:59:59Z", quantity: 2, valueMinor: 500},
		{id: "age-60", supplierID: "SUP-A", receivedAt: "2026-08-01T12:00:00+03:00", quantity: 2, valueMinor: 600},
		{id: "age-61", supplierID: "SUP-A", receivedAt: "2026-07-31T12:00:00+03:00", quantity: 3, valueMinor: 2100},
		{id: "supplier-B", supplierID: "SUP-B", receivedAt: "2026-07-01T12:00:00+03:00", quantity: 1, valueMinor: 5000},
		{id: "missing-name", supplierID: "SUP-C", receivedAt: "2026-09-22T12:00:00+03:00", quantity: 1, valueMinor: 3500},
		{id: "zero-value", supplierID: "SUP-Z", receivedAt: "2026-09-30T00:15:00+03:00", quantity: 1},
		{id: "closed", supplierID: "SUP-A", receivedAt: "2026-07-01T12:00:00+03:00", quantity: 2, valueMinor: 100},
		{id: "unposted", supplierID: "SUP-A", receivedAt: "2026-07-01T12:00:00+03:00", quantity: 2, valueMinor: 100, unposted: true},
		{id: "late", supplierID: "SUP-A", receivedAt: "2026-08-01T12:00:00+03:00", quantity: 1, valueMinor: 1000, recordedAt: "2026-10-02T12:00:00+03:00"},
	} {
		recordedAt := row.recordedAt
		if recordedAt == "" {
			recordedAt = row.receivedAt
		}
		data.Returns = append(data.Returns, domain.ReturnLine{
			ID: row.id, SupplierID: row.supplierID, Posted: !row.unposted,
			WarehouseReceivedAt: row.receivedAt, Quantity: row.quantity,
			UnitBookCostMinor: row.valueMinor / int64(row.quantity),
		})
		data.StockMovements = append(data.StockMovements, domain.StockMovement{
			ID: "receipt-" + row.id, ReturnLineID: row.id,
			QuantityDelta: row.quantity, ValueDeltaMinor: row.valueMinor,
			OccurredAt: row.receivedAt, RecordedAt: recordedAt,
		})
	}
	data.StockMovements = append(data.StockMovements,
		domain.StockMovement{
			ID: "partial-dispatch", ReturnLineID: "age-61", QuantityDelta: -1, ValueDeltaMinor: -700,
			OccurredAt: cutoff.Format(time.RFC3339), RecordedAt: cutoff.Format(time.RFC3339),
		},
		domain.StockMovement{
			ID: "full-dispatch", ReturnLineID: "closed", QuantityDelta: -2, ValueDeltaMinor: -100,
			OccurredAt: cutoff.Format(time.RFC3339), RecordedAt: cutoff.Format(time.RFC3339),
		},
	)
	want := []SupplierStock{
		{
			ID: "SUP-B", Name: "Поставщик B",
			StockSummary:   StockSummary{ValueMinor: 5000, AgedValueMinor: 5000, LineCount: 1, Quantity: 1},
			Age61PlusMinor: 5000,
		},
		{
			ID: "SUP-A", Name: "Поставщик A",
			StockSummary: StockSummary{ValueMinor: 3500, AgedValueMinor: 2500, LineCount: 7, Quantity: 14},
			Age0To7Minor: 300, Age8To30Minor: 700, Age31To60Minor: 1100, Age61PlusMinor: 1400,
		},
		{
			ID: "SUP-C", Name: "SUP-C",
			StockSummary:  StockSummary{ValueMinor: 3500, LineCount: 1, Quantity: 1},
			Age8To30Minor: 3500,
		},
		{
			ID: "SUP-Z", Name: "Нулевая стоимость",
			StockSummary: StockSummary{LineCount: 1, Quantity: 1},
		},
	}
	for _, tc := range []struct {
		name       string
		supplierID string
		want       []SupplierStock
	}{
		{name: "all suppliers sorted by value and ID", want: want},
		{name: "supplier filter", supplierID: "SUP-A", want: want[1:2]},
		{name: "positive stock with zero value", supplierID: "SUP-Z", want: want[3:4]},
		{name: "no matching stock", supplierID: "SUP-X", want: []SupplierStock{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := StockBySupplier(data, cutoff, tc.supplierID)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("StockBySupplier() = %+v, want %+v", got, tc.want)
			}
			var total StockSummary
			for _, supplier := range got {
				if supplier.Age0To7Minor+supplier.Age8To30Minor+supplier.Age31To60Minor+supplier.Age61PlusMinor != supplier.ValueMinor {
					t.Fatalf("age groups do not reconcile: %+v", supplier)
				}
				if supplier.Age31To60Minor+supplier.Age61PlusMinor != supplier.AgedValueMinor {
					t.Fatalf("aged stock does not reconcile: %+v", supplier)
				}
				total.ValueMinor += supplier.ValueMinor
				total.AgedValueMinor += supplier.AgedValueMinor
				total.LineCount += supplier.LineCount
				total.Quantity += supplier.Quantity
			}
			if summary := StockAt(data, cutoff, tc.supplierID); total != summary {
				t.Fatalf("supplier totals = %+v, stock summary = %+v", total, summary)
			}
		})
	}
}
