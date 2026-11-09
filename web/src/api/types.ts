export interface Supplier {
  id: string;
  name: string;
}
export interface Manifest {
  version: string;
  as_of: string;
  timezone: string;
  currency: string;
  default_from: string;
  default_to: string;
}
export interface Meta {
  manifest: Manifest;
  suppliers: Supplier[];
}
export interface Filters {
  from: string;
  to: string;
  supplier_id: string;
  q?: string;
}
export interface SupplierTotal extends Supplier {
  count: number;
  quantity: number;
  amount_minor: number;
}
export interface OperationsReport {
  count: number;
  quantity: number;
  amount_minor: number;
  days: { date: string; count: number; amount_minor: number }[];
  suppliers: SupplierTotal[];
}
export interface StockSummary {
  value_minor: number;
  aged_value_minor: number;
  line_count: number;
  quantity: number;
}
export interface StockDay {
  date: string;
  value_minor: number;
  aged_value_minor: number;
}
export interface SupplierStock extends Supplier, StockSummary {
  age_0_7_minor: number;
  age_8_30_minor: number;
  age_31_60_minor: number;
  age_61_plus_minor: number;
}
export interface PriorityCase {
  id: string;
  return_id: string;
  supplier_id: string;
  supplier_name: string;
  product: string;
  stock_quantity: number;
  stock_value_minor: number;
  age_days: number;
  priority: "urgent" | "clarify" | "review";
  reason: string;
  next_action: string;
  deadline_date: string | null;
  days_remaining: number | null;
  rejected_quantity: number;
}
export interface ReturnsReport {
  as_of: string;
  stock: StockSummary;
  days: StockDay[];
  suppliers: SupplierStock[];
  priorities: PriorityCase[];
  total_priority_cases: number;
  priority_rules: {
    urgent_window_days: number;
    order: string;
  };
  limitations: string[];
}
export interface ReturnRow {
  id: string;
  return_id: string;
  supplier_id: string;
  supplier_name: string;
  policy_id: string;
  product: string;
  quantity: number;
  unit_book_cost_minor: number;
  book_value_minor: number;
  supplier_shipped_at: string | null;
  requested_at: string | null;
  warehouse_received_at: string;
  packaging_complete: boolean | null;
  quality: string;
  stock_quantity: number;
  stock_value_minor: number;
  status: "on_warehouse" | "pending" | "accepted" | "rejected" | "partial";
}
export interface ReturnEvent {
  event_id: string;
  allocation_id: string;
  type: string;
  quantity: number;
  reason: string | null;
  occurred_at: string;
  recorded_at: string;
  reverses_event_id: string | null;
}
export interface Policy {
  confirmed: boolean;
  window_days: number | null;
  ends_at: string | null;
  requires_packaging: boolean | null;
  accepts_good_quality: boolean | null;
}
export interface ReturnDetail {
  row: ReturnRow;
  policy: Policy | null;
  shipments: {
    id: string;
    dispatched_at: string;
    posted: boolean;
    lines: { id: string; return_line_id: string; quantity: number }[];
  }[];
  events: ReturnEvent[];
  financial_entries: {
    id: string;
    type: string;
    amount_minor: number;
    occurred_at: string;
  }[];
}
