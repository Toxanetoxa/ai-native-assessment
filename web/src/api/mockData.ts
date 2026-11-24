import type { Meta, ReturnDetail, ReturnRow } from "./types";

export const meta: Meta = {
  manifest: {
    version: "demo-2026.09",
    as_of: "2026-09-30T23:59:59+03:00",
    timezone: "Europe/Moscow",
    currency: "RUB",
    default_from: "2026-09-01",
    default_to: "2026-09-30",
  },
  suppliers: [
    { id: "SUP-A", name: "Автокомплект" },
    { id: "SUP-B", name: "Мотордеталь" },
    { id: "SUP-C", name: "Транзит Авто" },
    { id: "SUP-D", name: "Север Подшипник" },
  ],
};

const products = [
  { name: "Насос водяной", cost: 375000 },
  { name: "Амортизатор", cost: 624000 },
  { name: "Датчик температуры", cost: 135000 },
  { name: "Подшипник ступицы", cost: 289500 },
  { name: "Комплект тормозных колодок", cost: 412000 },
];

const dayMs = 86400000;
export const snapshotDay = meta.manifest.as_of.slice(0, 10);

// All fixtures use Moscow calendar dates. UTC day arithmetic also works in browsers outside Moscow.
export const plusDays = (day: string, offset: number) =>
  new Date(Date.parse(day) + offset * dayMs).toISOString().slice(0, 10);
export const daysBetween = (from: string, to: string) =>
  (Date.parse(to) - Date.parse(from)) / dayMs;
const noon = (day: string) => day + "T12:00:00+03:00";

export interface DemoReturn {
  detail: ReturnDetail;
  movements: { day: string; quantity: number }[];
}

function makeReturn(index: number): DemoReturn {
  const supplier = meta.suppliers[index % meta.suppliers.length];
  const product = products[index % products.length];
  const serial = String(index + 1).padStart(3, "0");
  const id = "RL-" + serial;
  // Older receipts populate all four age bands; September has two returns per day.
  const received =
    index < 12
      ? plusDays("2026-07-08", index * 4)
      : plusDays("2026-09-01", Math.floor((index - 12) / 2));
  const shipped = plusDays(received, -3);
  const quantity = 2 + (index % 3);
  const windowDays = supplier.id === "SUP-C" ? null : supplier.id === "SUP-D" ? 7 : 14;
  const row: ReturnRow = {
    id,
    return_id: "R-" + serial,
    supplier_id: supplier.id,
    supplier_name: supplier.name,
    policy_id: "POL-" + supplier.id.slice(-1),
    product: product.name,
    quantity,
    unit_book_cost_minor: product.cost,
    book_value_minor: quantity * product.cost,
    supplier_shipped_at: noon(shipped),
    requested_at: noon(plusDays(received, -1)),
    warehouse_received_at: noon(received),
    packaging_complete: true,
    quality: "defective",
    stock_quantity: quantity,
    stock_value_minor: quantity * product.cost,
    status: "on_warehouse",
  };
  const detail: ReturnDetail = {
    row,
    policy: {
      confirmed: windowDays !== null,
      window_days: windowDays,
      ends_at: windowDays === null ? null : "supplier_received_at",
      requires_packaging: false,
      accepts_good_quality: true,
    },
    shipments: [],
    events: [],
    financial_entries: [],
  };
  const movements = [{ day: received, quantity }];
  const scenarios = [
    "on_warehouse",
    "pending",
    "accepted",
    "on_warehouse",
    "rejected",
    "partial",
  ] as const;
  const scenario = scenarios[index % scenarios.length];
  const dispatched = plusDays(received, 1);
  if (scenario !== "on_warehouse" && dispatched <= snapshotDay) {
    const sent = scenario === "partial" ? quantity - 1 : quantity;
    const allocation = "AL-" + serial;
    detail.shipments.push({
      id: "SR-" + serial,
      dispatched_at: noon(dispatched),
      posted: true,
      lines: [{ id: allocation, return_line_id: id, quantity: sent }],
    });
    movements.push({ day: dispatched, quantity: -sent });
    row.status = "pending";
    const decisionDay = plusDays(dispatched, 1);
    if (scenario !== "pending" && decisionDay <= snapshotDay) {
      row.status = scenario;
      detail.events.push({
        event_id: "EV-" + serial,
        allocation_id: allocation,
        type: scenario === "rejected" ? "rejected" : "accepted",
        quantity: sent,
        reason: scenario === "rejected" ? "other" : null,
        occurred_at: noon(decisionDay),
        recorded_at: noon(decisionDay),
        reverses_event_id: null,
      });
      detail.financial_entries.push({
        id: "FE-" + serial,
        type: "handling_cost",
        amount_minor: 15000,
        occurred_at: noon(decisionDay),
      });
      const returned = plusDays(decisionDay, 2);
      if (scenario === "rejected" && returned <= snapshotDay)
        movements.push({ day: returned, quantity: sent });
    }
  }
  row.stock_quantity = movements.reduce((sum, movement) => sum + movement.quantity, 0);
  row.stock_value_minor = row.stock_quantity * product.cost;
  return { detail, movements };
}

export const returns = Array.from({ length: 72 }, (_, index) => makeReturn(index));

export const receipts = Array.from({ length: 184 }, (_, index) => {
  const day = plusDays("2026-07-01", Math.floor(index / 2));
  const supplier = meta.suppliers[(index + Math.floor(index / 6)) % meta.suppliers.length];
  const quantity = 1 + (index % 9);
  const product = products[index % products.length];
  return { day, supplier, quantity, amount_minor: quantity * product.cost };
});
