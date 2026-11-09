package service

import (
	"sort"
	"strings"
	"time"

	"assessment.local/returns/internal/domain"
)

type ReturnRow struct {
	domain.ReturnLine
	SupplierName    string `json:"supplier_name"`
	BookValueMinor  int64  `json:"book_value_minor"`
	StockQuantity   int    `json:"stock_quantity"`
	StockValueMinor int64  `json:"stock_value_minor"`
	Status          string `json:"status"`
}

type RegisterResult struct {
	Items []ReturnRow `json:"items"`
	Total int         `json:"total"`
}

type ReturnDetail struct {
	Row       ReturnRow               `json:"row"`
	Policy    *domain.Policy          `json:"policy"`
	Shipments []domain.SupplierReturn `json:"shipments"`
	Events    []domain.Event          `json:"events"`
	Financial []domain.FinancialEntry `json:"financial_entries"`
}

type Register struct{ source Source }

func NewRegister(source Source) *Register { return &Register{source} }

func (s *Register) rows() []ReturnRow {
	data := s.source.Snapshot()
	cutoff, _ := time.Parse(time.RFC3339, data.Manifest.AsOf)
	names := map[string]string{}
	for _, supplier := range data.Suppliers {
		names[supplier.ID] = supplier.Name
	}
	stockQty := map[string]int{}
	stockValue := map[string]int64{}
	for _, movement := range data.StockMovements {
		if domain.KnownAt(movement.OccurredAt, movement.RecordedAt, cutoff) {
			stockQty[movement.ReturnLineID] += movement.QuantityDelta
			stockValue[movement.ReturnLineID] += movement.ValueDeltaMinor
		}
	}
	allocations := map[string]string{}
	dispatched := map[string]int{}
	for _, shipment := range data.SupplierReturns {
		if !shipment.Posted || !domain.KnownAt(shipment.DispatchedAt, shipment.DispatchedAt, cutoff) {
			continue
		}
		for _, allocation := range shipment.Lines {
			allocations[allocation.ID] = allocation.ReturnLineID
			dispatched[allocation.ReturnLineID] += allocation.Quantity
		}
	}
	accepted, rejected := map[string]int{}, map[string]int{}
	for _, event := range domain.VisibleEvents(data.Events, cutoff) {
		id, ok := allocations[event.AllocationID]
		if !ok {
			continue
		}
		switch event.Type {
		case "accepted":
			accepted[id] += event.Quantity
		case "rejected":
			rejected[id] += event.Quantity
		}
	}
	rows := make([]ReturnRow, 0, len(data.Returns))
	for _, line := range data.Returns {
		if !line.Posted || !domain.KnownAt(line.WarehouseReceivedAt, line.WarehouseReceivedAt, cutoff) {
			continue
		}
		status := "on_warehouse"
		switch {
		case accepted[line.ID] == line.Quantity:
			status = "accepted"
		case rejected[line.ID] == line.Quantity:
			status = "rejected"
		case accepted[line.ID]+rejected[line.ID] > 0:
			status = "partial"
		case dispatched[line.ID] > 0:
			status = "pending"
		}
		rows = append(rows, ReturnRow{line, names[line.SupplierID], int64(line.Quantity) * line.UnitBookCostMinor, stockQty[line.ID], stockValue[line.ID], status})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].WarehouseReceivedAt == rows[j].WarehouseReceivedAt {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].WarehouseReceivedAt > rows[j].WarehouseReceivedAt
	})
	return rows
}

func (s *Register) List(filter Filter, offset, limit int) RegisterResult {
	filtered := []ReturnRow{}
	for _, row := range s.rows() {
		if !filter.Includes(row.WarehouseReceivedAt, row.SupplierID) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(row.ID+" "+row.ReturnID+" "+row.Product+" "+row.SupplierName), filter.Search) {
			continue
		}
		filtered = append(filtered, row)
	}
	result := RegisterResult{Items: []ReturnRow{}, Total: len(filtered)}
	if offset < len(filtered) {
		end := min(offset+limit, len(filtered))
		result.Items = filtered[offset:end]
	}
	return result
}

func (s *Register) Detail(id string) (ReturnDetail, bool) {
	data := s.source.Snapshot()
	for _, row := range s.rows() {
		if row.ID != id {
			continue
		}
		result := ReturnDetail{Row: row, Shipments: []domain.SupplierReturn{}, Events: []domain.Event{}, Financial: []domain.FinancialEntry{}}
		for _, policy := range data.Policies {
			if policy.ID == row.PolicyID {
				result.Policy = &policy
			}
		}
		ids := map[string]bool{}
		for _, shipment := range data.SupplierReturns {
			for _, allocation := range shipment.Lines {
				if allocation.ReturnLineID == id {
					result.Shipments = append(result.Shipments, shipment)
					ids[allocation.ID] = true
				}
			}
		}
		for _, event := range data.Events {
			if ids[event.AllocationID] {
				result.Events = append(result.Events, event)
			}
		}
		for _, entry := range data.Financial {
			if entry.ReturnLineID == id {
				result.Financial = append(result.Financial, entry)
			}
		}
		return result, true
	}
	return ReturnDetail{}, false
}
