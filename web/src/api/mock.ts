import {
  daysBetween,
  meta,
  plusDays,
  receipts,
  returns,
  snapshotDay,
  type DemoReturn,
} from "./mockData";
import type {
  Filters,
  OperationsReport,
  PriorityCase,
  ReturnsReport,
  StockSummary,
  SupplierStock,
  SupplierTotal,
} from "./types";

type Params = Record<string, string | number>;

function validDay(value: string) {
  const time = Date.parse(value);
  return (
    /^\d{4}-\d{2}-\d{2}$/.test(value) &&
    Number.isFinite(time) &&
    new Date(time).toISOString().slice(0, 10) === value
  );
}

function filters(params: Params): Filters {
  const from = String(params.from || meta.manifest.default_from);
  const to = String(params.to || meta.manifest.default_to);
  if (!validDay(from)) throw new Error("Начало периода должно быть датой ГГГГ-ММ-ДД");
  if (!validDay(to) || to < from)
    throw new Error("Конец периода должен быть датой не раньше начала");
  if (daysBetween(from, to) > 366) throw new Error("Период не должен превышать один год");
  return {
    from,
    to,
    supplier_id: String(params.supplier_id || ""),
    q: String(params.q || "")
      .trim()
      .toLowerCase(),
  };
}

function daysInRange(from: string, to: string) {
  const days: string[] = [];
  for (let day = from; day <= to && day <= snapshotDay; day = plusDays(day, 1)) days.push(day);
  return days;
}

function operations({ from, to, supplier_id }: Filters): OperationsReport {
  const days = new Map(
    daysInRange(from, to).map((date) => [date, { date, count: 0, amount_minor: 0 }]),
  );
  const suppliers = new Map<string, SupplierTotal>();
  const report: OperationsReport = {
    count: 0,
    quantity: 0,
    amount_minor: 0,
    days: [],
    suppliers: [],
  };
  for (const receipt of receipts) {
    const day = days.get(receipt.day);
    if (!day || (supplier_id && receipt.supplier.id !== supplier_id)) continue;
    const supplier = suppliers.get(receipt.supplier.id) ?? {
      ...receipt.supplier,
      count: 0,
      quantity: 0,
      amount_minor: 0,
    };
    supplier.count++;
    supplier.quantity += receipt.quantity;
    supplier.amount_minor += receipt.amount_minor;
    suppliers.set(supplier.id, supplier);
    day.count++;
    day.amount_minor += receipt.amount_minor;
    report.count++;
    report.quantity += receipt.quantity;
    report.amount_minor += receipt.amount_minor;
  }
  report.days = [...days.values()];
  report.suppliers = [...suppliers.values()].sort(
    (a, b) => b.amount_minor - a.amount_minor || a.id.localeCompare(b.id),
  );
  return report;
}

function stockQuantity(item: DemoReturn, day: string) {
  return item.movements.reduce((sum, move) => (move.day <= day ? sum + move.quantity : sum), 0);
}

function priority(item: DemoReturn, day: string, quantity: number, age: number): PriorityCase {
  const { row, policy, events } = item.detail;
  const deadline =
    policy?.confirmed && policy.window_days && row.supplier_shipped_at
      ? plusDays(row.supplier_shipped_at.slice(0, 10), policy.window_days)
      : null;
  const remaining = deadline ? daysBetween(day, deadline) : null;
  const rejected = events
    .filter((event) => event.type === "rejected" && event.recorded_at.slice(0, 10) <= day)
    .reduce((total, event) => total + event.quantity, 0);
  const result: PriorityCase = {
    id: row.id,
    return_id: row.return_id,
    supplier_id: row.supplier_id,
    supplier_name: row.supplier_name,
    product: row.product,
    stock_quantity: quantity,
    stock_value_minor: quantity * row.unit_book_cost_minor,
    age_days: age,
    deadline_date: deadline,
    days_remaining: remaining,
    rejected_quantity: rejected,
    priority: "review",
    reason: "Окно получения поставщиком открыто.",
    next_action: "Согласовать дату доставки с поставщиком.",
  };
  if (remaining === null) {
    result.priority = "clarify";
    result.reason = "Условия поставщика не подтверждены.";
    result.next_action = "Уточнить с закупками условия и срок возврата.";
  } else if (remaining < 0) {
    result.reason = "Окно получения поставщиком истекло.";
    result.next_action = "Согласовать исключение по сроку или оценить повторную продажу.";
  } else if (remaining <= 3) {
    result.priority = "urgent";
    result.reason = "До конца окна получения поставщиком осталось не более трёх дней.";
  }
  if (rejected > 0) {
    result.reason += " Поставщик уже отказывал по этой строке.";
    result.next_action += " Проверить причину отказа.";
  }
  return result;
}

function emptyStock(): StockSummary {
  return { line_count: 0, quantity: 0, value_minor: 0, aged_value_minor: 0 };
}

function returnsReport({ from, to, supplier_id }: Filters): ReturnsReport {
  const cutoff = to < snapshotDay ? to : snapshotDay;
  if (from > cutoff) throw new Error("Выбранный период находится после доступного среза данных");
  const items = returns.filter(
    (item) => !supplier_id || item.detail.row.supplier_id === supplier_id,
  );
  const groups = new Map<string, SupplierStock>();
  const priorities: PriorityCase[] = [];
  let stock = emptyStock();
  const days = daysInRange(from, cutoff).map((date) => {
    const summary = emptyStock();
    for (const item of items) {
      const quantity = stockQuantity(item, date);
      if (quantity <= 0) continue;
      const { row } = item.detail;
      const value = quantity * row.unit_book_cost_minor;
      const age = daysBetween(row.warehouse_received_at.slice(0, 10), date);
      summary.line_count++;
      summary.quantity += quantity;
      summary.value_minor += value;
      if (age > 30) summary.aged_value_minor += value;
      if (date !== cutoff) continue;

      const group = groups.get(row.supplier_id) ?? {
        id: row.supplier_id,
        name: row.supplier_name,
        ...emptyStock(),
        age_0_7_minor: 0,
        age_8_30_minor: 0,
        age_31_60_minor: 0,
        age_61_plus_minor: 0,
      };
      group.line_count++;
      group.quantity += quantity;
      group.value_minor += value;
      if (age > 30) group.aged_value_minor += value;
      if (age <= 7) group.age_0_7_minor += value;
      else if (age <= 30) group.age_8_30_minor += value;
      else if (age <= 60) group.age_31_60_minor += value;
      else group.age_61_plus_minor += value;
      groups.set(group.id, group);
      priorities.push(priority(item, date, quantity, age));
    }
    if (date === cutoff) stock = summary;
    return { date, value_minor: summary.value_minor, aged_value_minor: summary.aged_value_minor };
  });
  priorities.sort(
    (a, b) =>
      Number(b.priority === "urgent") - Number(a.priority === "urgent") ||
      b.stock_value_minor - a.stock_value_minor ||
      b.age_days - a.age_days ||
      a.id.localeCompare(b.id),
  );
  return {
    as_of: cutoff === snapshotDay ? meta.manifest.as_of : cutoff + "T23:59:59+03:00",
    stock,
    days,
    priorities,
    total_priority_cases: priorities.length,
    suppliers: [...groups.values()].sort(
      (a, b) => b.value_minor - a.value_minor || a.id.localeCompare(b.id),
    ),
    priority_rules: {
      urgent_window_days: 3,
      order:
        "Сначала срочные случаи, затем сумма остатка по убыванию, возраст по убыванию и идентификатор.",
    },
    limitations: [
      "Отчёт построен на демонстрационных документах и складских движениях.",
      "В очереди только строки с положительным остатком. Срок относится к получению поставщиком. Время доставки неизвестно.",
    ],
  };
}

function register(params: Params) {
  const { from, to, supplier_id, q } = filters(params);
  const page = Number(params.page ?? 1);
  const size = Number(params.page_size ?? 25);
  if (
    !Number.isInteger(page) ||
    page < 1 ||
    page > 10000 ||
    !Number.isInteger(size) ||
    size < 1 ||
    size > 100
  )
    throw new Error("Некорректные параметры страницы");
  const rows = returns
    .map((item) => item.detail.row)
    .filter((row) => {
      const day = row.warehouse_received_at.slice(0, 10);
      const text = [row.id, row.return_id, row.product, row.supplier_name].join(" ").toLowerCase();
      return (
        day >= from &&
        day <= to &&
        (!supplier_id || row.supplier_id === supplier_id) &&
        (!q || text.includes(q))
      );
    })
    .sort(
      (a, b) =>
        b.warehouse_received_at.localeCompare(a.warehouse_received_at) || a.id.localeCompare(b.id),
    );
  return { items: rows.slice((page - 1) * size, page * size), total: rows.length };
}

export function mockGet(path: string, params: Params) {
  switch (path) {
    case "/meta":
      return meta;
    case "/operations":
      return operations(filters(params));
    case "/returns-report":
      return returnsReport(filters(params));
    case "/returns":
      return register(params);
  }
  if (path.startsWith("/returns/")) {
    const id = decodeURIComponent(path.slice("/returns/".length));
    const item = returns.find((item) => item.detail.row.id === id);
    if (item) return item.detail;
    throw new Error("Возврат не найден");
  }
  throw new Error("Маршрут не найден");
}
