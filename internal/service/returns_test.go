package service

import (
	"reflect"
	"testing"
	"time"

	"assessment.local/returns/internal/domain"
)

func TestStockTrendUsesDailyKnowledgeAndMoscowDates(t *testing.T) {
	// Оба аргумента в UTC, но московские даты — с 1 по 4 сентября.
	from := time.Date(2026, time.August, 31, 21, 0, 0, 0, time.UTC)
	cutoff := time.Date(2026, time.September, 3, 22, 0, 0, 0, time.UTC)
	data := domain.Dataset{
		Returns: []domain.ReturnLine{
			{ID: "RL-A", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: "2026-08-01T12:00:00+03:00"},
			{ID: "RL-B", SupplierID: "SUP-B", Posted: true, WarehouseReceivedAt: "2026-08-02T12:00:00+03:00"},
		},
		StockMovements: []domain.StockMovement{
			{
				ID: "receipt-A", ReturnLineID: "RL-A", QuantityDelta: 3, ValueDeltaMinor: 30000,
				OccurredAt: "2026-08-01T12:00:00+03:00", RecordedAt: "2026-08-01T12:00:00+03:00",
			},
			{
				ID: "receipt-B", ReturnLineID: "RL-B", QuantityDelta: 2, ValueDeltaMinor: 50000,
				OccurredAt: "2026-08-02T12:00:00+03:00", RecordedAt: "2026-08-02T12:00:00+03:00",
			},
			{
				ID: "late-dispatch-A", ReturnLineID: "RL-A", QuantityDelta: -1, ValueDeltaMinor: -10000,
				OccurredAt: "2026-09-02T00:00:00+03:00", RecordedAt: "2026-09-03T08:00:00+03:00",
			},
			{
				ID: "after-cutoff-B", ReturnLineID: "RL-B", QuantityDelta: -1, ValueDeltaMinor: -25000,
				OccurredAt: "2026-09-04T15:00:00+03:00", RecordedAt: "2026-09-04T15:00:00+03:00",
			},
		},
	}
	for _, tc := range []struct {
		name       string
		supplierID string
		want       []StockDay
	}{
		{
			name: "all suppliers",
			want: []StockDay{
				{Date: "2026-09-01", ValueMinor: 80000, AgedValueMinor: 30000},
				{Date: "2026-09-02", ValueMinor: 80000, AgedValueMinor: 80000},
				{Date: "2026-09-03", ValueMinor: 70000, AgedValueMinor: 70000},
				{Date: "2026-09-04", ValueMinor: 70000, AgedValueMinor: 70000},
			},
		},
		{
			name: "one supplier", supplierID: "SUP-A",
			want: []StockDay{
				{Date: "2026-09-01", ValueMinor: 30000, AgedValueMinor: 30000},
				{Date: "2026-09-02", ValueMinor: 30000, AgedValueMinor: 30000},
				{Date: "2026-09-03", ValueMinor: 20000, AgedValueMinor: 20000},
				{Date: "2026-09-04", ValueMinor: 20000, AgedValueMinor: 20000},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := StockTrend(data, from, cutoff, tc.supplierID)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("StockTrend() = %+v, want %+v", got, tc.want)
			}
			stock := StockAt(data, cutoff, tc.supplierID)
			last := got[len(got)-1]
			if last.ValueMinor != stock.ValueMinor || last.AgedValueMinor != stock.AgedValueMinor {
				t.Fatalf("last day %+v differs from stock %+v", last, stock)
			}
		})
	}
}

func TestStockTrendEmptyRangeAndZeroStock(t *testing.T) {
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, location)
	if got := StockTrend(domain.Dataset{}, from, from.Add(-time.Nanosecond), ""); got == nil || len(got) != 0 {
		t.Fatalf("reversed range = %+v, want a non-nil empty slice", got)
	}
	want := []StockDay{{Date: "2026-09-01"}, {Date: "2026-09-02"}}
	if got := StockTrend(domain.Dataset{}, from, from.AddDate(0, 0, 1), ""); !reflect.DeepEqual(got, want) {
		t.Fatalf("zero stock trend = %+v, want %+v", got, want)
	}
}

func TestStockAtAgedValueUsesMoscowCalendarDates(t *testing.T) {
	// В Москве уже 30 сентября; часовой пояс самого cutoff не меняет правило.
	cutoff := time.Date(2026, time.September, 29, 21, 5, 0, 0, time.UTC)
	for _, tc := range []struct {
		name       string
		receivedAt string
		wantAged   int64
	}{
		{name: "31 calendar days", receivedAt: "2026-08-30T23:59:59+03:00", wantAged: 20000},
		{name: "30 days at midnight", receivedAt: "2026-08-31T00:00:00+03:00"},
		{name: "30 days late evening", receivedAt: "2026-08-31T23:59:59+03:00"},
		{name: "UTC before Moscow boundary", receivedAt: "2026-08-30T20:59:59Z", wantAged: 20000},
		{name: "UTC at Moscow boundary", receivedAt: "2026-08-30T21:00:00Z"},
		{name: "recent receipt", receivedAt: "2026-09-29T12:00:00+03:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := domain.Dataset{
				Returns: []domain.ReturnLine{{
					ID: "RL-1", SupplierID: "SUP-A", Posted: true,
					WarehouseReceivedAt: tc.receivedAt,
				}},
				StockMovements: []domain.StockMovement{
					{
						ID: "receipt", ReturnLineID: "RL-1", QuantityDelta: 3, ValueDeltaMinor: 30000,
						OccurredAt: tc.receivedAt, RecordedAt: tc.receivedAt,
					},
					{
						ID: "dispatch", ReturnLineID: "RL-1", QuantityDelta: -1, ValueDeltaMinor: -10000,
						OccurredAt: cutoff.Format(time.RFC3339), RecordedAt: cutoff.Format(time.RFC3339),
					},
				},
			}
			want := StockSummary{ValueMinor: 20000, AgedValueMinor: tc.wantAged, LineCount: 1, Quantity: 2}
			if got := StockAt(data, cutoff, ""); got != want {
				t.Fatalf("StockAt() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestStockAtUsesKnownStockMovements(t *testing.T) {
	cutoff := time.Date(2026, time.September, 30, 23, 59, 59, 0, location)
	data := domain.Dataset{
		Returns: []domain.ReturnLine{{
			ID: "RL-1", SupplierID: "SUP-A", Posted: true,
			Quantity: 3, UnitBookCostMinor: 99999,
			WarehouseReceivedAt: "2026-04-01T12:00:00+03:00",
		}},
		StockMovements: []domain.StockMovement{
			{
				ID: "receipt", ReturnLineID: "RL-1", QuantityDelta: 3, ValueDeltaMinor: 30000,
				OccurredAt: "2026-04-01T12:00:00+03:00", RecordedAt: "2026-04-01T12:00:00+03:00",
			},
			{
				ID: "dispatch", ReturnLineID: "RL-1", QuantityDelta: -1, ValueDeltaMinor: -10000,
				OccurredAt: "2026-09-15T12:00:00+03:00", RecordedAt: "2026-09-15T12:00:00+03:00",
			},
			{
				ID: "late-recording", ReturnLineID: "RL-1", QuantityDelta: -2, ValueDeltaMinor: -20000,
				OccurredAt: "2026-09-29T12:00:00+03:00", RecordedAt: "2026-10-02T12:00:00+03:00",
			},
			{
				ID: "future-operation", ReturnLineID: "RL-1", QuantityDelta: -2, ValueDeltaMinor: -20000,
				OccurredAt: "2026-10-01T12:00:00+03:00", RecordedAt: "2026-09-30T12:00:00+03:00",
			},
		},
	}
	want := StockSummary{ValueMinor: 20000, AgedValueMinor: 20000, LineCount: 1, Quantity: 2}
	if got := StockAt(data, cutoff, ""); got != want {
		t.Fatalf("StockAt() = %+v, want %+v", got, want)
	}
}

func TestStockAtFiltersLinesAndIncludesCutoff(t *testing.T) {
	cutoff := time.Date(2026, time.September, 30, 23, 59, 59, 0, location)
	stamp := cutoff.Format(time.RFC3339)
	data := domain.Dataset{
		Returns: []domain.ReturnLine{
			{ID: "RL-A", ReturnID: "R-1", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: "2026-08-01T12:00:00+03:00"},
			{ID: "RL-B", ReturnID: "R-1", SupplierID: "SUP-B", Posted: true, WarehouseReceivedAt: stamp},
			{ID: "unposted", SupplierID: "SUP-A", WarehouseReceivedAt: stamp},
			{ID: "future", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: "2026-10-01T00:00:00+03:00"},
			{ID: "empty", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: "2026-08-01T12:00:00+03:00"},
			{ID: "no-movements", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: stamp},
		},
	}
	for _, id := range []string{"RL-A", "RL-B", "unposted", "future", "unknown", "empty"} {
		data.StockMovements = append(data.StockMovements, domain.StockMovement{
			ID: "receipt-" + id, ReturnLineID: id, QuantityDelta: 2, ValueDeltaMinor: 20000,
			OccurredAt: stamp, RecordedAt: stamp,
		})
	}
	data.StockMovements = append(data.StockMovements, domain.StockMovement{
		ID: "dispatch-empty", ReturnLineID: "empty", QuantityDelta: -2, ValueDeltaMinor: -20000,
		OccurredAt: stamp, RecordedAt: stamp,
	})

	for _, tc := range []struct {
		name       string
		supplierID string
		want       StockSummary
	}{
		{name: "all suppliers", want: StockSummary{ValueMinor: 40000, AgedValueMinor: 20000, LineCount: 2, Quantity: 4}},
		{name: "one supplier", supplierID: "SUP-A", want: StockSummary{ValueMinor: 20000, AgedValueMinor: 20000, LineCount: 1, Quantity: 2}},
		{name: "unknown supplier", supplierID: "SUP-X", want: StockSummary{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := StockAt(data, cutoff, tc.supplierID); got != tc.want {
				t.Fatalf("StockAt() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
