import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchMeta,
  fetchOperations,
  fetchReturn,
  fetchReturns,
  fetchReturnsReport,
} from "./workspace";

const defaults = { from: "", to: "", supplier_id: "" };

beforeEach(() => {
  vi.stubEnv("VITE_API_MODE", "mock");
  vi.stubGlobal(
    "fetch",
    vi.fn(() => {
      throw new Error("Unexpected HTTP request");
    }),
  );
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("demo API", () => {
  it("filters receipts and keeps the daily and supplier totals consistent", async () => {
    const { manifest, suppliers } = await fetchMeta();
    const all = await fetchOperations(defaults);
    const selected = await fetchOperations({
      ...defaults,
      from: manifest.default_to,
      supplier_id: all.suppliers[0].id,
    });
    expect(all.count).toBeGreaterThan(0);
    expect(all.amount_minor).toBe(all.days.reduce((sum, day) => sum + day.amount_minor, 0));
    expect(all.amount_minor).toBe(
      all.suppliers.reduce((sum, supplier) => sum + supplier.amount_minor, 0),
    );
    expect(suppliers.length).toBeGreaterThan(1);
    expect(selected.days.map((day) => day.date)).toEqual([manifest.default_to]);
    expect(selected.suppliers.every((supplier) => supplier.id === all.suppliers[0].id)).toBe(true);
    expect(await fetchOperations({ ...defaults, supplier_id: "unknown" })).toMatchObject({
      count: 0,
      suppliers: [],
    });
    expect(fetch).not.toHaveBeenCalled();
  });

  it("paginates unique return lines, searches and opens matching detail histories", async () => {
    const first = await fetchReturns(defaults, 1);
    const second = await fetchReturns(defaults, 2);
    expect(first.items).toHaveLength(25);
    expect(new Set([...first.items, ...second.items].map((row) => row.id)).size).toBe(50);
    expect(second.total).toBe(first.total);
    const row = first.items.find((row) => row.status === "accepted")!;
    const match = await fetchReturns({ ...defaults, q: "  " + row.id.toLowerCase() + "  " }, 1);
    expect(match).toEqual({ total: 1, items: [row] });
    const detail = await fetchReturn(row.id);
    expect(detail.row).toEqual(row);
    expect(detail.events[0].allocation_id).toBe(detail.shipments[0].lines[0].id);
    expect(detail.financial_entries.length).toBeGreaterThan(0);
    await expect(fetchReturn("missing")).rejects.toThrow("Возврат не найден");
  });

  it("reconciles stock, age and priorities on the requested day", async () => {
    const { manifest } = await fetchMeta();
    const report = await fetchReturnsReport({ ...defaults, to: "2026-10-01" });
    expect(report.as_of).toBe(manifest.as_of);
    expect(report.days.at(-1)?.value_minor).toBe(report.stock.value_minor);
    expect(report.stock.line_count).toBeGreaterThan(25);
    expect(report.stock.aged_value_minor).toBeGreaterThan(0);
    expect(report.stock.value_minor).toBe(
      report.priorities.reduce((sum, row) => sum + row.stock_value_minor, 0),
    );
    expect(new Set(report.priorities.map((row) => row.priority))).toEqual(
      new Set(["urgent", "clarify", "review"]),
    );
    const urgent = report.priorities[0];
    expect(urgent.priority).toBe("urgent");
    expect((await fetchReturn(urgent.id)).row.stock_value_minor).toBe(urgent.stock_value_minor);
    const earlier = await fetchReturnsReport({
      ...defaults,
      to: "2026-09-01",
      supplier_id: urgent.supplier_id,
    });
    expect(earlier.days).toHaveLength(1);
    expect(earlier.suppliers.map((supplier) => supplier.id)).toEqual([urgent.supplier_id]);
    expect(earlier.priorities.every((row) => row.supplier_id === urgent.supplier_id)).toBe(true);
    expect(earlier.stock.line_count).toBeLessThan(report.stock.line_count);
    await expect(
      fetchReturnsReport({ ...defaults, from: "2026-10-01", to: "2026-10-02" }),
    ).rejects.toThrow("после доступного среза");
    await expect(fetchReturns(defaults, 1, AbortSignal.abort())).rejects.toMatchObject({
      name: "AbortError",
    });
  });

  it("uses the original HTTP endpoint outside demo mode", async () => {
    vi.stubEnv("VITE_API_MODE", "");
    const request = vi.fn().mockResolvedValue({ ok: true, json: () => ({ manifest: {} }) });
    vi.stubGlobal("fetch", request);
    await fetchMeta();
    expect(request).toHaveBeenCalledWith("/api/meta", expect.anything());
  });
});
