package service

import (
	"reflect"
	"testing"
	"time"

	"assessment.local/returns/internal/domain"
)

func priorityString(value string) *string { return &value }
func priorityInt(value int) *int          { return &value }
func priorityBool(value bool) *bool       { return &value }

func priorityCutoff() time.Time {
	return time.Date(2026, time.September, 30, 12, 0, 0, 0, location)
}

func priorityDataset() domain.Dataset {
	return domain.Dataset{
		Suppliers: []domain.Supplier{{ID: "SUP-A", Name: "Поставщик A"}},
		Policies: []domain.Policy{{
			ID: "POL-A", SupplierID: "SUP-A", Confirmed: true, ValidFrom: "2026-01-01",
			WindowDays: priorityInt(14), StartsAt: priorityString("supplier_shipped_at"),
			EndsAt: priorityString("supplier_received_at"), IncludeFirstDay: priorityBool(false),
			Calendar: priorityString("calendar_days"), Packaging: priorityBool(true), AcceptsGood: priorityBool(true),
		}},
		Returns: []domain.ReturnLine{{
			ID: "RL-1", ReturnID: "R-1", SupplierID: "SUP-A", PolicyID: "POL-A", Product: "Деталь",
			Posted: true, Quantity: 3, UnitBookCostMinor: 10000,
			SupplierShippedAt:   priorityString("2026-09-20T12:00:00+03:00"),
			WarehouseReceivedAt: "2026-09-25T12:00:00+03:00", Quality: "good", PackagingComplete: priorityBool(true),
		}},
		StockMovements: []domain.StockMovement{
			{
				ID: "receipt", ReturnLineID: "RL-1", QuantityDelta: 3, ValueDeltaMinor: 30000,
				OccurredAt: "2026-09-25T12:00:00+03:00", RecordedAt: "2026-09-25T12:00:00+03:00",
			},
			{
				ID: "dispatch", ReturnLineID: "RL-1", QuantityDelta: -1, ValueDeltaMinor: -10000,
				OccurredAt: "2026-09-29T12:00:00+03:00", RecordedAt: "2026-09-29T12:00:00+03:00",
			},
		},
	}
}

func TestReturnPrioritiesCalendarDeadlineAndUrgency(t *testing.T) {
	for _, tc := range []struct {
		name      string
		change    func(*domain.Dataset)
		cutoff    time.Time
		deadline  string
		remaining int
		priority  string
	}{
		{name: "four days is ordinary review", deadline: "2026-10-04", remaining: 4, priority: "review"},
		{
			name: "three days is urgent", deadline: "2026-10-03", remaining: 3, priority: "urgent",
			change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = priorityString("2026-09-19T12:00:00+03:00") },
		},
		{
			name: "deadline day remains open", deadline: "2026-09-30", remaining: 0, priority: "urgent",
			change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = priorityString("2026-09-16T12:00:00+03:00") },
		},
		{
			name: "expired window", deadline: "2026-09-29", remaining: -1, priority: "review",
			change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = priorityString("2026-09-15T12:00:00+03:00") },
		},
		{
			name: "first day included", deadline: "2026-10-03", remaining: 3, priority: "urgent",
			change: func(d *domain.Dataset) { d.Policies[0].IncludeFirstDay = priorityBool(true) },
		},
		{
			name: "historical policy remains applicable", deadline: "2026-10-04", remaining: 4, priority: "review",
			change: func(d *domain.Dataset) { d.Policies[0].ValidTo = priorityString("2026-09-20") },
		},
		{
			name: "Moscow dates for UTC timestamps", deadline: "2026-09-30", remaining: 0, priority: "urgent",
			cutoff: time.Date(2026, time.September, 29, 22, 0, 0, 0, time.UTC),
			change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = priorityString("2026-09-15T21:00:00Z") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := priorityDataset()
			if tc.change != nil {
				tc.change(&data)
			}
			cutoff := tc.cutoff
			if cutoff.IsZero() {
				cutoff = priorityCutoff()
			}
			rows := ReturnPriorities(data, cutoff, "")
			if len(rows) != 1 {
				t.Fatalf("got %d cases, want 1", len(rows))
			}
			row := rows[0]
			if row.DeadlineDate == nil || *row.DeadlineDate != tc.deadline {
				t.Fatalf("deadline = %v, want %s", row.DeadlineDate, tc.deadline)
			}
			if row.DaysRemaining == nil || *row.DaysRemaining != tc.remaining {
				t.Fatalf("remaining = %v, want %d", row.DaysRemaining, tc.remaining)
			}
			if row.Priority != tc.priority || row.Reason == "" || row.NextAction == "" {
				t.Fatalf("case = %+v, want priority %s and an explanation/action", row, tc.priority)
			}
			if row.StockQuantity != 2 || row.StockValueMinor != 20000 {
				t.Fatalf("case uses original amount instead of remaining stock: %+v", row)
			}
		})
	}
}

func TestReturnPrioritiesUnknownDeadlineIsNotInvented(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Dataset)
	}{
		{name: "missing policy", change: func(d *domain.Dataset) { d.Policies = nil }},
		{name: "unconfirmed policy", change: func(d *domain.Dataset) { d.Policies[0].Confirmed = false }},
		{name: "policy belongs to another supplier", change: func(d *domain.Dataset) { d.Policies[0].SupplierID = "SUP-B" }},
		{name: "wrong policy ID", change: func(d *domain.Dataset) { d.Returns[0].PolicyID = "POL-X" }},
		{name: "missing anchor", change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = nil }},
		{name: "malformed anchor", change: func(d *domain.Dataset) { d.Returns[0].SupplierShippedAt = priorityString("invalid") }},
		{name: "unknown window", change: func(d *domain.Dataset) { d.Policies[0].WindowDays = nil }},
		{name: "unknown first day rule", change: func(d *domain.Dataset) { d.Policies[0].IncludeFirstDay = nil }},
		{name: "unsupported calendar", change: func(d *domain.Dataset) { d.Policies[0].Calendar = priorityString("business_days") }},
		{name: "unsupported start", change: func(d *domain.Dataset) { d.Policies[0].StartsAt = priorityString("requested_at") }},
		{name: "unsupported end", change: func(d *domain.Dataset) { d.Policies[0].EndsAt = priorityString("dispatched_at") }},
		{name: "policy starts after shipment", change: func(d *domain.Dataset) { d.Policies[0].ValidFrom = "2026-09-21" }},
		{name: "policy ended before shipment", change: func(d *domain.Dataset) { d.Policies[0].ValidTo = priorityString("2026-09-19") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := priorityDataset()
			tc.change(&data)
			rows := ReturnPriorities(data, priorityCutoff(), "")
			if len(rows) != 1 {
				t.Fatalf("got %d cases, want 1", len(rows))
			}
			row := rows[0]
			if row.DeadlineDate != nil || row.DaysRemaining != nil || row.Priority != "clarify" {
				t.Fatalf("unknown rule produced an asserted deadline or eligibility: %+v", row)
			}
			if row.Reason == "" || row.NextAction == "" {
				t.Fatalf("unknown rule needs a concrete explanation/action: %+v", row)
			}
		})
	}
}

func TestReturnPrioritiesConditionsPreserveKnownDeadline(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func(*domain.Dataset)
		priority string
	}{
		{name: "incomplete packaging", priority: "review", change: func(d *domain.Dataset) { d.Returns[0].PackagingComplete = priorityBool(false) }},
		{name: "unknown packaging", priority: "clarify", change: func(d *domain.Dataset) { d.Returns[0].PackagingComplete = nil }},
		{name: "packaging not required", priority: "review", change: func(d *domain.Dataset) {
			d.Policies[0].Packaging = priorityBool(false)
			d.Returns[0].PackagingComplete = nil
		}},
		{name: "good quality not accepted", priority: "review", change: func(d *domain.Dataset) { d.Policies[0].AcceptsGood = priorityBool(false) }},
		{name: "unknown quality", priority: "clarify", change: func(d *domain.Dataset) { d.Returns[0].Quality = "unknown" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := priorityDataset()
			tc.change(&data)
			rows := ReturnPriorities(data, priorityCutoff(), "")
			if len(rows) != 1 {
				t.Fatalf("got %d cases, want 1", len(rows))
			}
			row := rows[0]
			if row.DeadlineDate == nil || *row.DeadlineDate != "2026-10-04" || row.DaysRemaining == nil || *row.DaysRemaining != 4 {
				t.Fatalf("condition discarded a known deadline: %+v", row)
			}
			if row.Priority != tc.priority || row.Reason == "" || row.NextAction == "" {
				t.Fatalf("case = %+v, want priority %s with explanation/action", row, tc.priority)
			}
		})
	}
	data := priorityDataset()
	data.Returns[0].Quality = "defective"
	acceptedGood := ReturnPriorities(data, priorityCutoff(), "")[0]
	data.Policies[0].AcceptsGood = priorityBool(false)
	refusedGood := ReturnPriorities(data, priorityCutoff(), "")[0]
	if refusedGood.Priority != acceptedGood.Priority || refusedGood.Reason != acceptedGood.Reason || refusedGood.NextAction != acceptedGood.NextAction {
		t.Fatal("restriction for good quality incorrectly changes the defective-goods case")
	}
	data.Returns[0].SupplierShippedAt = priorityString("2026-09-19T12:00:00+03:00")
	data.Returns[0].PackagingComplete = priorityBool(false)
	if row := ReturnPriorities(data, priorityCutoff(), "")[0]; row.Priority != "urgent" {
		t.Fatalf("known obstacle hid a closing window requiring urgent human review: %+v", row)
	}
}

func TestReturnPrioritiesSortAndFilterRemainingStock(t *testing.T) {
	var data domain.Dataset
	for _, tc := range []struct {
		id, supplierID, priority string
		value                    int64
		age                      int
	}{
		{id: "B-young", supplierID: "SUP-A", value: 8000, age: 5},
		{id: "C-old", supplierID: "SUP-A", priority: "clarify", value: 8000, age: 10},
		{id: "R-expensive", supplierID: "SUP-A", value: 10000, age: 5},
		{id: "U-cheap", supplierID: "SUP-A", priority: "urgent", value: 100, age: 1},
		{id: "A-young", supplierID: "SUP-A", value: 8000, age: 5},
		{id: "other-supplier", supplierID: "SUP-B", value: 5000, age: 5},
	} {
		part := priorityDataset()
		line := part.Returns[0]
		line.ID, line.ReturnID, line.SupplierID, line.PolicyID = tc.id, "R-"+tc.id, tc.supplierID, "POL-"+tc.id
		received := moscowDate(priorityCutoff()).AddDate(0, 0, -tc.age).Add(12 * time.Hour)
		line.WarehouseReceivedAt = received.Format(time.RFC3339)
		line.SupplierShippedAt = priorityString(received.AddDate(0, 0, -1).Format(time.RFC3339))
		policy := part.Policies[0]
		policy.ID, policy.SupplierID, policy.WindowDays = line.PolicyID, line.SupplierID, priorityInt(120)
		if tc.priority == "clarify" {
			policy.Confirmed = false
		}
		if tc.priority == "urgent" {
			line.SupplierShippedAt = priorityString("2026-09-19T12:00:00+03:00")
			policy.WindowDays = priorityInt(14)
		}
		data.Returns = append(data.Returns, line)
		data.Policies = append(data.Policies, policy)
		data.StockMovements = append(data.StockMovements,
			domain.StockMovement{ID: "receipt-" + tc.id, ReturnLineID: tc.id, QuantityDelta: 3, ValueDeltaMinor: tc.value + 1000, OccurredAt: line.WarehouseReceivedAt, RecordedAt: line.WarehouseReceivedAt},
			domain.StockMovement{ID: "dispatch-" + tc.id, ReturnLineID: tc.id, QuantityDelta: -1, ValueDeltaMinor: -1000, OccurredAt: priorityCutoff().Format(time.RFC3339), RecordedAt: priorityCutoff().Format(time.RFC3339)},
		)
	}
	for _, tc := range []struct {
		name, supplierID string
		want             []string
	}{
		{name: "all cases", want: []string{"U-cheap", "R-expensive", "C-old", "A-young", "B-young", "other-supplier"}},
		{name: "supplier filter", supplierID: "SUP-A", want: []string{"U-cheap", "R-expensive", "C-old", "A-young", "B-young"}},
		{name: "unknown supplier", supplierID: "SUP-X", want: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := ReturnPriorities(data, priorityCutoff(), tc.supplierID)
			if rows == nil {
				t.Fatal("empty queue must serialize as []")
			}
			ids := make([]string, 0, len(rows))
			var value int64
			for _, row := range rows {
				ids = append(ids, row.ID)
				value += row.StockValueMinor
				if row.StockQuantity != 2 {
					t.Fatalf("wrong remaining quantity: %+v", row)
				}
			}
			if !reflect.DeepEqual(ids, tc.want) {
				t.Fatalf("case order = %v, want %v", ids, tc.want)
			}
			if stock := StockAt(data, priorityCutoff(), tc.supplierID); stock.LineCount != len(rows) || stock.ValueMinor != value {
				t.Fatalf("queue does not reconcile with remaining stock: %+v", stock)
			}
		})
	}
	data = priorityDataset()
	data.StockMovements = append(data.StockMovements, domain.StockMovement{
		ID: "close", ReturnLineID: "RL-1", QuantityDelta: -2, ValueDeltaMinor: -20000,
		OccurredAt: priorityCutoff().Format(time.RFC3339), RecordedAt: priorityCutoff().Format(time.RFC3339),
	})
	if rows := ReturnPriorities(data, priorityCutoff(), ""); len(rows) != 0 {
		t.Fatalf("closed stock appears in queue: %+v", rows)
	}
}

func TestReturnPrioritiesRejectionEvidenceRespectsVisibility(t *testing.T) {
	data := priorityDataset()
	data.SupplierReturns = []domain.SupplierReturn{
		{ID: "SR-1", SupplierID: "SUP-A", Posted: true, DispatchedAt: "2026-09-28T12:00:00+03:00", Lines: []domain.Allocation{{ID: "AL-1", ReturnLineID: "RL-1", Quantity: 2}}},
		{ID: "SR-draft", SupplierID: "SUP-A", DispatchedAt: "2026-09-28T12:00:00+03:00", Lines: []domain.Allocation{{ID: "AL-draft", ReturnLineID: "RL-1", Quantity: 1}}},
	}
	stamp := "2026-09-29T12:00:00+03:00"
	data.Events = []domain.Event{
		{ID: "active", AllocationID: "AL-1", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: stamp},
		{ID: "active", AllocationID: "AL-1", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: stamp},
		{ID: "cancelled", AllocationID: "AL-1", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: stamp},
		{ID: "reversal", AllocationID: "AL-1", Type: "reversal", ReversesEventID: priorityString("cancelled"), OccurredAt: stamp, RecordedAt: stamp},
		{ID: "late", AllocationID: "AL-1", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: "2026-10-02T12:00:00+03:00"},
		{ID: "future", AllocationID: "AL-1", Type: "rejected", Quantity: 1, OccurredAt: "2026-10-01T12:00:00+03:00", RecordedAt: stamp},
		{ID: "draft", AllocationID: "AL-draft", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: stamp},
		{ID: "orphan", AllocationID: "AL-unknown", Type: "rejected", Quantity: 1, OccurredAt: stamp, RecordedAt: stamp},
	}
	rows := ReturnPriorities(data, priorityCutoff(), "")
	if len(rows) != 1 {
		t.Fatalf("got %d cases, want 1", len(rows))
	}
	if row := rows[0]; row.RejectedQuantity != 1 || row.StockQuantity != 2 || row.StockValueMinor != 20000 {
		t.Fatalf("rejection history is duplicated or changes current stock: %+v", row)
	}
}
