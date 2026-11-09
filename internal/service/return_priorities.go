package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"assessment.local/returns/internal/domain"
)

// UrgentReturnWindowDays — порог пилота, а не договорное условие поставщика.
const UrgentReturnWindowDays = 3

type PriorityCase struct {
	ID               string  `json:"id"`
	ReturnID         string  `json:"return_id"`
	SupplierID       string  `json:"supplier_id"`
	SupplierName     string  `json:"supplier_name"`
	Product          string  `json:"product"`
	StockQuantity    int     `json:"stock_quantity"`
	StockValueMinor  int64   `json:"stock_value_minor"`
	AgeDays          int     `json:"age_days"`
	Priority         string  `json:"priority"`
	Reason           string  `json:"reason"`
	NextAction       string  `json:"next_action"`
	DeadlineDate     *string `json:"deadline_date"`
	DaysRemaining    *int    `json:"days_remaining"`
	RejectedQuantity int     `json:"rejected_quantity"`
}

// ReturnPriorities включает все положительные остатки на заданный срез.
// Сначала идут срочные случаи, затем сумма, возраст и идентификатор строки.
func ReturnPriorities(data domain.Dataset, cutoff time.Time, supplierID string) []PriorityCase {
	balances := stockBalances(data, cutoff, supplierID)
	names := make(map[string]string, len(data.Suppliers))
	for _, supplier := range data.Suppliers {
		names[supplier.ID] = supplier.Name
	}
	policies := make(map[string]domain.Policy, len(data.Policies))
	for _, policy := range data.Policies {
		policies[policy.ID] = policy
	}
	rejected := rejectedByReturnLine(data, cutoff)
	result := make([]PriorityCase, 0)
	for _, line := range data.Returns {
		stock, found := balances[line.ID]
		if !found || stock.quantity <= 0 {
			continue
		}
		name := names[line.SupplierID]
		if name == "" {
			name = line.SupplierID
		}
		row := PriorityCase{
			ID: line.ID, ReturnID: line.ReturnID, SupplierID: line.SupplierID,
			SupplierName: name, Product: line.Product,
			StockQuantity: stock.quantity, StockValueMinor: stock.valueMinor,
			AgeDays: stock.ageDays, Priority: "review", RejectedQuantity: rejected[line.ID],
		}
		policy, found := policies[line.PolicyID]
		anchor, issue := returnPolicyAnchor(line, policy, found, cutoff)
		explainPriority(&row, line, policy, anchor, issue, cutoff)
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool {
		iUrgent, jUrgent := result[i].Priority == "urgent", result[j].Priority == "urgent"
		if iUrgent != jUrgent {
			return iUrgent
		}
		if result[i].StockValueMinor != result[j].StockValueMinor {
			return result[i].StockValueMinor > result[j].StockValueMinor
		}
		if result[i].AgeDays != result[j].AgeDays {
			return result[i].AgeDays > result[j].AgeDays
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// Применимость версии условий проверяется на исходную отгрузку, а не на срез.
func returnPolicyAnchor(line domain.ReturnLine, policy domain.Policy, found bool, cutoff time.Time) (time.Time, string) {
	if !found {
		return time.Time{}, "Для строки отсутствует версия условий поставщика."
	}
	if !policy.Confirmed {
		return time.Time{}, "Условия поставщика не подтверждены."
	}
	if policy.SupplierID != line.SupplierID {
		return time.Time{}, "Версия условий относится к другому поставщику."
	}
	if line.SupplierShippedAt == nil {
		return time.Time{}, "Неизвестна дата исходной отгрузки поставщиком."
	}
	anchor, err := time.Parse(time.RFC3339, *line.SupplierShippedAt)
	if err != nil || anchor.After(cutoff) {
		return time.Time{}, "Дата исходной отгрузки поставщиком требует проверки."
	}
	validFrom, err := time.ParseInLocation(time.DateOnly, policy.ValidFrom, location)
	if err != nil {
		return time.Time{}, "Период действия версии условий требует проверки."
	}
	if moscowDate(anchor).Before(validFrom) {
		return time.Time{}, "Период действия версии условий не соответствует исходной отгрузке."
	}
	if policy.ValidTo != nil {
		validTo, err := time.ParseInLocation(time.DateOnly, *policy.ValidTo, location)
		if err != nil {
			return time.Time{}, "Период действия версии условий требует проверки."
		}
		if moscowDate(anchor).After(validTo) {
			return time.Time{}, "Период действия версии условий не соответствует исходной отгрузке."
		}
	}
	return anchor, ""
}

func explainPriority(row *PriorityCase, line domain.ReturnLine, policy domain.Policy, anchor time.Time, policyIssue string, cutoff time.Time) {
	reasons := []string{fmt.Sprintf("Хранение на складе: %d календ. дн.", row.AgeDays)}
	actions := []string{"Сверить историю решений по строке с текущим остатком."}
	if row.RejectedQuantity > 0 {
		reasons = append(reasons, fmt.Sprintf("По строке зафиксированы отказы суммарно на %d шт. по отправкам; связь с текущим остатком требует проверки.", row.RejectedQuantity))
		actions[0] = "Проверить причины отказов, сопоставить отправки с остатком и согласовать возможность повторного возврата."
	}
	if policyIssue != "" {
		row.Priority = "clarify"
		reasons = append(reasons, policyIssue)
		actions = append(actions, "Уточнить с закупками версию условий и дату исходной отгрузки.")
		row.Reason, row.NextAction = strings.Join(reasons, " "), strings.Join(actions, " ")
		return
	}

	needsInfo := false
	if policy.WindowDays == nil || *policy.WindowDays <= 0 || policy.IncludeFirstDay == nil ||
		policy.Calendar == nil || *policy.Calendar != "calendar_days" ||
		policy.StartsAt == nil || *policy.StartsAt != "supplier_shipped_at" ||
		policy.EndsAt == nil || *policy.EndsAt != "supplier_received_at" {
		needsInfo = true
		reasons = append(reasons, "Окно возврата требует уточнения или ручного расчёта по условиям поставщика.")
		actions = append(actions, "Уточнить длительность, календарь, начало и окончание окна возврата.")
	} else {
		window := *policy.WindowDays
		if *policy.IncludeFirstDay {
			window--
		}
		deadline := moscowDate(anchor).AddDate(0, 0, window)
		date := deadline.Format(time.DateOnly)
		remaining := int(deadline.Sub(moscowDate(cutoff)) / (24 * time.Hour))
		row.DeadlineDate, row.DaysRemaining = &date, &remaining
		switch {
		case remaining < 0:
			reasons = append(reasons, fmt.Sprintf("Окно получения поставщиком истекло %d календ. дн. назад.", -remaining))
		case remaining == 0:
			reasons = append(reasons, "Сегодня последний день окна получения поставщиком.")
		default:
			reasons = append(reasons, fmt.Sprintf("До конца окна получения поставщиком: %d календ. дн. (%s).", remaining, date))
		}
	}

	if policy.Packaging == nil {
		needsInfo = true
		reasons = append(reasons, "Неизвестны требования поставщика к упаковке.")
		actions = append(actions, "Уточнить требования к упаковке.")
	} else if *policy.Packaging {
		if line.PackagingComplete == nil {
			needsInfo = true
			reasons = append(reasons, "Комплектность упаковки не проверена.")
			actions = append(actions, "Проверить комплектность и упаковку на складе.")
		} else if !*line.PackagingComplete {
			reasons = append(reasons, "Упаковка или комплектность нарушена при требовании поставщика.")
			actions = append(actions, "Проверить возможность восстановления упаковки и согласовать условия приёмки.")
		}
	}
	switch line.Quality {
	case "good":
		if policy.AcceptsGood == nil {
			needsInfo = true
			reasons = append(reasons, "Неизвестны условия возврата товара надлежащего качества.")
			actions = append(actions, "Подтвердить условия возврата товара надлежащего качества.")
		} else if !*policy.AcceptsGood {
			reasons = append(reasons, "Условия не предусматривают возврат товара надлежащего качества.")
			actions = append(actions, "Согласовать исключение по качеству либо другой способ реализации.")
		}
	case "defective":
		// Ограничение accepts_good_quality не запрещает само по себе возврат брака.
	default:
		needsInfo = true
		reasons = append(reasons, "Качество товара требует проверки.")
		actions = append(actions, "Уточнить состояние товара и применимые условия возврата.")
	}

	if row.DaysRemaining == nil {
		row.Priority = "clarify"
	} else if *row.DaysRemaining < 0 {
		actions = append(actions, "Согласовать исключение по сроку либо оценить другой способ реализации.")
	} else {
		if *row.DaysRemaining <= UrgentReturnWindowDays {
			row.Priority = "urgent"
		} else if needsInfo {
			row.Priority = "clarify"
		}
		actions = append(actions, fmt.Sprintf("Уточнить время доставки и согласовать возможность получения поставщиком до %s включительно.", *row.DeadlineDate))
	}
	row.Reason, row.NextAction = strings.Join(reasons, " "), strings.Join(actions, " ")
}

// Отказы относятся к количеству отправки; это контекст, а не статус остатка.
func rejectedByReturnLine(data domain.Dataset, cutoff time.Time) map[string]int {
	allocations := make(map[string]string)
	for _, shipment := range data.SupplierReturns {
		if !shipment.Posted || !domain.KnownAt(shipment.DispatchedAt, shipment.DispatchedAt, cutoff) {
			continue
		}
		for _, allocation := range shipment.Lines {
			allocations[allocation.ID] = allocation.ReturnLineID
		}
	}
	rejected := make(map[string]int)
	for _, event := range domain.VisibleEvents(data.Events, cutoff) {
		if id, found := allocations[event.AllocationID]; found && event.Type == "rejected" {
			rejected[id] += event.Quantity
		}
	}
	return rejected
}
