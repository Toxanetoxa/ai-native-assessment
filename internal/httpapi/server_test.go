package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"assessment.local/returns/internal/domain"
	"assessment.local/returns/internal/service"
)

type reportSource struct{ data domain.Dataset }

func (s reportSource) Read(context.Context) (domain.Dataset, error) { return s.data, nil }
func (s reportSource) Ping(context.Context) error                   { return nil }

func reportDataset() domain.Dataset {
	return domain.Dataset{
		Manifest: domain.Manifest{
			AsOf: "2026-09-30T23:59:59+03:00", From: "2026-09-01", To: "2026-09-30",
		},
		Suppliers: []domain.Supplier{{ID: "SUP-A", Name: "Поставщик A"}, {ID: "SUP-B", Name: "Поставщик B"}},
		Returns: []domain.ReturnLine{
			{ID: "RL-A", SupplierID: "SUP-A", Posted: true, WarehouseReceivedAt: "2026-04-01T12:00:00+03:00"},
			{ID: "RL-B", SupplierID: "SUP-B", Posted: true, WarehouseReceivedAt: "2026-09-10T12:00:00+03:00"},
		},
		StockMovements: []domain.StockMovement{
			{
				ID: "receipt-A", ReturnLineID: "RL-A", QuantityDelta: 3, ValueDeltaMinor: 30000,
				OccurredAt: "2026-04-01T12:00:00+03:00", RecordedAt: "2026-04-01T12:00:00+03:00",
			},
			{
				ID: "receipt-B", ReturnLineID: "RL-B", QuantityDelta: 2, ValueDeltaMinor: 50000,
				OccurredAt: "2026-09-10T12:00:00+03:00", RecordedAt: "2026-09-10T12:00:00+03:00",
			},
			{
				ID: "dispatch-A", ReturnLineID: "RL-A", QuantityDelta: -1, ValueDeltaMinor: -10000,
				OccurredAt: "2026-09-20T12:00:00+03:00", RecordedAt: "2026-09-20T12:00:00+03:00",
			},
			{
				ID: "late-recording", ReturnLineID: "RL-A", QuantityDelta: -2, ValueDeltaMinor: -20000,
				OccurredAt: "2026-09-10T12:00:00+03:00", RecordedAt: "2026-10-02T12:00:00+03:00",
			},
		},
	}
}

func TestReturnReportUsesRequestedCutoffAndSupplier(t *testing.T) {
	handler := New(reportSource{data: reportDataset()})
	for _, tc := range []struct {
		name  string
		query string
		asOf  string
		stock service.StockSummary
		days  int
	}{
		{
			name: "default period includes old stock", asOf: "2026-09-30T23:59:59+03:00", days: 30,
			stock: service.StockSummary{ValueMinor: 70000, AgedValueMinor: 20000, LineCount: 2, Quantity: 4},
		},
		{
			name: "earlier end excludes later dispatch", query: "?from=2026-09-01&to=2026-09-15",
			asOf: "2026-09-15T23:59:59.999999999+03:00", days: 15,
			stock: service.StockSummary{ValueMinor: 80000, AgedValueMinor: 30000, LineCount: 2, Quantity: 5},
		},
		{
			name: "future end capped at dataset snapshot", query: "?from=2026-09-01&to=2026-10-30",
			asOf: "2026-09-30T23:59:59+03:00", days: 30,
			stock: service.StockSummary{ValueMinor: 70000, AgedValueMinor: 20000, LineCount: 2, Quantity: 4},
		},
		{
			name: "supplier filter", query: "?supplier_id=SUP-B", asOf: "2026-09-30T23:59:59+03:00", days: 30,
			stock: service.StockSummary{ValueMinor: 50000, LineCount: 1, Quantity: 2},
		},
		{
			name: "later start shortens trend", query: "?from=2026-09-10&to=2026-09-15",
			asOf: "2026-09-15T23:59:59.999999999+03:00", days: 6,
			stock: service.StockSummary{ValueMinor: 80000, AgedValueMinor: 30000, LineCount: 2, Quantity: 5},
		},
		{
			name: "unknown supplier gives empty distribution", query: "?supplier_id=SUP-X",
			asOf: "2026-09-30T23:59:59+03:00", days: 30,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/returns-report"+tc.query, nil)
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var result struct {
				AsOf               string                  `json:"as_of"`
				Stock              service.StockSummary    `json:"stock"`
				Days               []service.StockDay      `json:"days"`
				Suppliers          []service.SupplierStock `json:"suppliers"`
				Priorities         []service.PriorityCase  `json:"priorities"`
				TotalPriorityCases int                     `json:"total_priority_cases"`
				PriorityRules      struct {
					UrgentWindowDays int    `json:"urgent_window_days"`
					Order            string `json:"order"`
				} `json:"priority_rules"`
				Limitations []string `json:"limitations"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.AsOf != tc.asOf || result.Stock != tc.stock {
				t.Fatalf("report = %+v, want as_of=%s stock=%+v", result, tc.asOf, tc.stock)
			}
			if len(result.Days) != tc.days {
				t.Fatalf("trend has %d days, want %d", len(result.Days), tc.days)
			}
			last := result.Days[len(result.Days)-1]
			if last.Date != tc.asOf[:10] || last.ValueMinor != result.Stock.ValueMinor || last.AgedValueMinor != result.Stock.AgedValueMinor {
				t.Fatalf("last day %+v differs from snapshot %s stock %+v", last, result.AsOf, result.Stock)
			}
			if result.Suppliers == nil {
				t.Fatal("suppliers must be a JSON array")
			}
			var total service.StockSummary
			for _, supplier := range result.Suppliers {
				if id := request.URL.Query().Get("supplier_id"); id != "" && supplier.ID != id {
					t.Fatalf("supplier %s does not match filter %s", supplier.ID, id)
				}
				if supplier.Age0To7Minor+supplier.Age8To30Minor+supplier.Age31To60Minor+supplier.Age61PlusMinor != supplier.ValueMinor {
					t.Fatalf("supplier age groups do not reconcile: %+v", supplier)
				}
				if supplier.Age31To60Minor+supplier.Age61PlusMinor != supplier.AgedValueMinor {
					t.Fatalf("supplier aged stock does not reconcile: %+v", supplier)
				}
				total.ValueMinor += supplier.ValueMinor
				total.AgedValueMinor += supplier.AgedValueMinor
				total.LineCount += supplier.LineCount
				total.Quantity += supplier.Quantity
			}
			if total != result.Stock {
				t.Fatalf("supplier totals = %+v, stock summary = %+v", total, result.Stock)
			}
			if result.Priorities == nil {
				t.Fatal("priorities must be a JSON array")
			}
			if result.TotalPriorityCases != len(result.Priorities) || result.TotalPriorityCases != result.Stock.LineCount {
				t.Fatalf("priority cases = %d, returned = %d, stock lines = %d", result.TotalPriorityCases, len(result.Priorities), result.Stock.LineCount)
			}
			var priorityValue int64
			var priorityQuantity int
			for _, item := range result.Priorities {
				if item.StockQuantity <= 0 || item.ID == "" || item.Reason == "" || item.NextAction == "" {
					t.Fatalf("priority case lacks positive stock or explanation: %+v", item)
				}
				if id := request.URL.Query().Get("supplier_id"); id != "" && item.SupplierID != id {
					t.Fatalf("priority supplier %s does not match filter %s", item.SupplierID, id)
				}
				// У строк этого набора нет условий поставщика: срок нельзя выдумывать.
				if item.Priority != "clarify" || item.DeadlineDate != nil || item.DaysRemaining != nil {
					t.Fatalf("unknown policy must require clarification without a deadline: %+v", item)
				}
				priorityValue += item.StockValueMinor
				priorityQuantity += item.StockQuantity
			}
			if priorityValue != result.Stock.ValueMinor || priorityQuantity != result.Stock.Quantity {
				t.Fatalf("priority totals = %d/%d, stock totals = %d/%d", priorityValue, priorityQuantity, result.Stock.ValueMinor, result.Stock.Quantity)
			}
			if result.PriorityRules.UrgentWindowDays != service.UrgentReturnWindowDays || result.PriorityRules.Order == "" || len(result.Limitations) == 0 {
				t.Fatal("report must explain priority rules and limitations")
			}
		})
	}
}

func TestReturnReportRejectsInvalidDates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		query  string
		asOf   string
		status int
	}{
		{name: "malformed request date", query: "?from=invalid", status: http.StatusBadRequest},
		{name: "reversed period", query: "?from=2026-09-30&to=2026-09-01", status: http.StatusBadRequest},
		{name: "period after snapshot", query: "?from=2026-10-01&to=2026-10-02", status: http.StatusBadRequest},
		{name: "malformed dataset snapshot", asOf: "invalid", status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := reportDataset()
			if tc.asOf != "" {
				data.Manifest.AsOf = tc.asOf
			}
			response := httptest.NewRecorder()
			New(reportSource{data: data}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/returns-report"+tc.query, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tc.status, response.Body.String())
			}
			var result struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Error == "" {
				t.Fatal("error response must explain the failure")
			}
		})
	}
}
