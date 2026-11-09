package domain

import "time"

type Manifest struct {
	Version  string            `json:"version"`
	AsOf     string            `json:"as_of"`
	Timezone string            `json:"timezone"`
	Currency string            `json:"currency"`
	From     string            `json:"default_from"`
	To       string            `json:"default_to"`
	Files    map[string]string `json:"files"`
}

type Supplier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Policy struct {
	ID              string  `json:"id"`
	SupplierID      string  `json:"supplier_id"`
	ValidFrom       string  `json:"valid_from"`
	ValidTo         *string `json:"valid_to"`
	Confirmed       bool    `json:"confirmed"`
	WindowDays      *int    `json:"window_days"`
	StartsAt        *string `json:"starts_at"`
	EndsAt          *string `json:"ends_at"`
	IncludeFirstDay *bool   `json:"include_first_day"`
	Calendar        *string `json:"calendar"`
	Packaging       *bool   `json:"requires_packaging"`
	AcceptsGood     *bool   `json:"accepts_good_quality"`
}

type ReturnLine struct {
	ID                  string  `json:"id"`
	ReturnID            string  `json:"return_id"`
	SupplierID          string  `json:"supplier_id"`
	PolicyID            string  `json:"policy_id"`
	CustomerID          string  `json:"customer_id"`
	Product             string  `json:"product"`
	Quantity            int     `json:"quantity"`
	UnitBookCostMinor   int64   `json:"unit_book_cost_minor"`
	SupplierShippedAt   *string `json:"supplier_shipped_at"`
	RequestedAt         *string `json:"requested_at"`
	WarehouseReceivedAt string  `json:"warehouse_received_at"`
	PackagingComplete   *bool   `json:"packaging_complete"`
	Quality             string  `json:"quality"`
	Posted              bool    `json:"posted"`
}

type Allocation struct {
	ID           string `json:"id"`
	ReturnLineID string `json:"return_line_id"`
	Quantity     int    `json:"quantity"`
}

type SupplierReturn struct {
	ID           string       `json:"id"`
	SupplierID   string       `json:"supplier_id"`
	DispatchedAt string       `json:"dispatched_at"`
	Posted       bool         `json:"posted"`
	Lines        []Allocation `json:"lines"`
}

type Event struct {
	ID              string  `json:"event_id"`
	AllocationID    string  `json:"allocation_id"`
	Type            string  `json:"type"`
	OccurredAt      string  `json:"occurred_at"`
	RecordedAt      string  `json:"recorded_at"`
	Quantity        int     `json:"quantity"`
	Reason          *string `json:"reason"`
	ReversesEventID *string `json:"reverses_event_id"`
}

type StockMovement struct {
	ID              string `json:"id"`
	ReturnLineID    string `json:"return_line_id"`
	QuantityDelta   int    `json:"quantity_delta"`
	ValueDeltaMinor int64  `json:"value_delta_minor"`
	OccurredAt      string `json:"occurred_at"`
	RecordedAt      string `json:"recorded_at"`
}

type FinancialEntry struct {
	ID           string `json:"id"`
	ReturnLineID string `json:"return_line_id"`
	Type         string `json:"type"`
	AmountMinor  int64  `json:"amount_minor"`
	OccurredAt   string `json:"occurred_at"`
}

type Shipment struct {
	ID          string `json:"id"`
	SupplierID  string `json:"supplier_id"`
	ShippedAt   string `json:"shipped_at"`
	Quantity    int    `json:"quantity"`
	AmountMinor int64  `json:"amount_minor"`
}

type Dataset struct {
	Manifest        Manifest         `json:"manifest"`
	Suppliers       []Supplier       `json:"suppliers"`
	Policies        []Policy         `json:"supplier_policies"`
	Returns         []ReturnLine     `json:"returns"`
	SupplierReturns []SupplierReturn `json:"supplier_returns"`
	Events          []Event          `json:"events"`
	StockMovements  []StockMovement  `json:"stock_movements"`
	Financial       []FinancialEntry `json:"financial_entries"`
	Shipments       []Shipment       `json:"shipments"`
}

func KnownAt(occurredAt, recordedAt string, cutoff time.Time) bool {
	occurred, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil || occurred.After(cutoff) {
		return false
	}
	recorded, err := time.Parse(time.RFC3339, recordedAt)
	return err == nil && !recorded.After(cutoff)
}

func VisibleEvents(events []Event, cutoff time.Time) []Event {
	seen := make(map[string]bool)
	reversed := make(map[string]bool)
	visible := make([]Event, 0, len(events))
	for _, event := range events {
		if !KnownAt(event.OccurredAt, event.RecordedAt, cutoff) || seen[event.ID] {
			continue
		}
		seen[event.ID] = true
		if event.Type == "reversal" && event.ReversesEventID != nil {
			reversed[*event.ReversesEventID] = true
		} else {
			visible = append(visible, event)
		}
	}
	result := make([]Event, 0, len(visible))
	for _, event := range visible {
		if !reversed[event.ID] {
			result = append(result, event)
		}
	}
	return result
}
