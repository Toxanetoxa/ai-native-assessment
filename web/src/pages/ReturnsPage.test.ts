import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import type { Meta, PriorityCase, ReturnsReport } from "../api/types";
import ReturnsPage from "./ReturnsPage.vue";

const api = vi.hoisted(() => ({ report: vi.fn(), load: vi.fn() }));
vi.mock("../api/workspace", () => ({ fetchReturnsReport: api.report }));
vi.mock("../composables/useMeta", async () => {
  const { ref } = await import("vue");
  return { useMeta: () => ({ meta: ref(metadata), load: api.load }) };
});

const metadata: Meta = {
  manifest: {
    version: "test", as_of: "2026-09-30T23:59:59+03:00", timezone: "Europe/Moscow",
    currency: "RUB", default_from: "2026-09-01", default_to: "2026-09-30",
  },
  suppliers: [{ id: "SUP-A", name: "Поставщик A" }, { id: "SUP-B", name: "Поставщик B" }],
};

function makeReport(count = 2, supplierID = "SUP-A"): ReturnsReport {
  const priorities: PriorityCase[] = Array.from({ length: count }, (_, index) => ({
    id: `RL-${String(index + 1).padStart(3, "0")}`,
    return_id: `R-${index + 1}`,
    supplier_id: supplierID,
    supplier_name: supplierID === "SUP-A" ? "Поставщик A" : "Поставщик B",
    product: "Тестовая деталь",
    stock_quantity: 1,
    stock_value_minor: index === 0 ? 12345 : 10000,
    age_days: index === 1 ? 40 : 30,
    priority: index === 0 ? "urgent" : index === 1 ? "clarify" : "review",
    reason: index === 1 ? "Условия поставщика не подтверждены." : "Требуется разбор складского остатка.",
    next_action: "Уточнить условия и согласовать следующее действие.",
    deadline_date: index === 1 ? null : index === 0 ? "2026-09-30" : "2026-10-04",
    days_remaining: index === 1 ? null : index === 0 ? 0 : 4,
    rejected_quantity: 0,
  }));
  const value = priorities.reduce((sum, row) => sum + row.stock_value_minor, 0);
  const aged = priorities.filter((row) => row.age_days > 30).reduce((sum, row) => sum + row.stock_value_minor, 0);
  const stock = { value_minor: value, aged_value_minor: aged, line_count: count, quantity: count };
  return {
    as_of: metadata.manifest.as_of,
    stock,
    days: [
      { date: "2026-09-01", value_minor: value, aged_value_minor: 0 },
      { date: "2026-09-30", value_minor: value, aged_value_minor: aged },
    ],
    suppliers: count ? [{
      id: supplierID, name: priorities[0].supplier_name, ...stock,
      age_0_7_minor: 0, age_8_30_minor: value - aged,
      age_31_60_minor: aged, age_61_plus_minor: 0,
    }] : [],
    priorities,
    total_priority_cases: count,
    priority_rules: { urgent_window_days: 3, order: "Сначала срочные случаи, затем сумма остатка." },
    limitations: ["Время доставки неизвестно."],
  };
}

const ChartStub = defineComponent({
  name: "BaseChart",
  props: {
    type: String,
    labels: Array,
    series: Array,
    unit: String,
    stacked: Boolean,
    stepped: Boolean,
    label: String,
  },
  template: '<div class="chart-stub" role="img" :aria-label="label" />',
});
let wrapper: VueWrapper | undefined;
async function renderPage() {
  wrapper = mount(ReturnsPage, { global: { stubs: { BaseChart: ChartStub } } });
  await flushPromises();
  return wrapper;
}

beforeEach(() => {
  vi.resetAllMocks();
  api.load.mockResolvedValue(metadata);
  api.report.mockResolvedValue(makeReport());
});
afterEach(() => {
  wrapper?.unmount();
  wrapper = undefined;
});

describe("ReturnsPage", () => {
  it("shows honest metrics, ruble chart values and different unknown/last-day deadlines", async () => {
    const page = await renderPage();
    const cards = page.findAll(".stat-card");
    expect(cards).toHaveLength(3);
    expect(cards[0].text()).toContain("223,45");
    expect(cards[1].text()).toContain("100,00");
    expect(cards[2].text()).toContain("Нет данных");
    expect(cards[2].text()).toContain("полные расходы");
    expect(page.find(".returns-metric-note").text()).toContain("не доказывает улучшения");

    const charts = page.findAllComponents({ name: "BaseChart" });
    expect(charts).toHaveLength(2);
    expect(charts[0].props("type")).toBe("line");
    expect(charts[0].props("stepped")).toBe(true);
    expect(charts[0].props("unit")).toBe("rubles");
    expect(charts[0].props("series").map((series: { values: number[] }) => series.values))
      .toEqual([[223.45, 223.45], [0, 100]]);
    expect(charts[1].props("stacked")).toBe(true);
    expect(charts[1].props("series").map((series: { values: number[] }) => series.values))
      .toEqual([[0], [123.45], [100], [0]]);

    const rows = page.findAll(".data-table tbody tr");
    expect(rows[0].find(".returns-deadline").text()).toContain("Последний день окна");
    expect(rows[1].find(".returns-deadline").text()).toBe("Срок не определён");
    expect(rows[1].text()).toContain("Условия поставщика не подтверждены.");
    expect(rows[1].text()).toContain("Уточнить условия и согласовать следующее действие.");
  });

  it("paginates, keeps applied labels until submit and resets the page for new filters", async () => {
    api.report.mockResolvedValueOnce(makeReport(26));
    const page = await renderPage();
    expect(page.findAll(".data-table tbody tr")).toHaveLength(25);
    const next = page.findAll("button").find((button) => button.text() === "Далее");
    expect(next).toBeDefined();
    await next!.trigger("click");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(1);
    expect(page.find(".data-table").text()).toContain("RL-026");

    const dates = page.findAll('input[type="date"]');
    await dates[0].setValue("2026-09-10");
    await dates[1].setValue("2026-10-15");
    await page.find("select").setValue("SUP-B");
    expect(api.report).toHaveBeenCalledTimes(1);
    expect(page.find(".returns-context").text()).toContain("Все поставщики");
    expect(page.find(".returns-context").text()).toContain("01.09.2026 — 30.09.2026");
    expect(page.find(".returns-context").text()).toContain("Фильтры изменены");

    const filtered = makeReport(2, "SUP-B");
    filtered.days[0].date = "2026-09-10";
    api.report.mockResolvedValueOnce(filtered);
    await page.find("form").trigger("submit");
    await flushPromises();
    expect(api.report.mock.lastCall?.[0]).toEqual({ from: "2026-09-10", to: "2026-10-15", supplier_id: "SUP-B" });
    expect(page.findAll(".data-table tbody tr")).toHaveLength(2);
    expect(page.find(".returns-pagination").text()).toContain("страница 1 из 1");
    expect(page.find(".returns-context").text()).toContain("Поставщик B");
    expect(page.find(".returns-context").text()).toContain("Конец выбранного периода позже доступных данных");
    expect(page.find(".returns-context").text()).not.toContain("Фильтры изменены");
  });

  it("filters the whole priority queue before pagination without changing the report metrics", async () => {
    api.report.mockResolvedValueOnce(makeReport(26));
    const page = await renderPage();
    const metrics = page.find(".returns-metrics").text();
    const series = page.findAllComponents({ name: "BaseChart" }).map((chart) => chart.props("series"));
    await page.findAll("button").find((button) => button.text() === "Далее")!.trigger("click");
    expect(page.find(".data-table").text()).toContain("RL-026");

    const priority = page.find(".returns-priority-filter select");
    await priority.setValue("urgent");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(1);
    expect(page.find(".data-table").text()).toContain("RL-001");
    expect(page.find(".returns-queue .panel__header").text()).toContain("Найдено 1 из 26");
    expect(page.find(".returns-pagination").text()).toContain("страница 1 из 1");

    await priority.setValue("clarify");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(1);
    expect(page.find(".data-table").text()).toContain("RL-002");

    await priority.setValue("review");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(24);
    expect(page.find(".data-table").text()).toContain("RL-026");
    expect(page.find(".data-table").text()).not.toContain("RL-001");

    await priority.setValue("");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(25);
    expect(page.findAll(".data-table tbody tr")[0].text()).toContain("RL-001");
    expect(page.find(".returns-metrics").text()).toBe(metrics);
    expect(page.findAllComponents({ name: "BaseChart" }).map((chart) => chart.props("series"))).toEqual(series);
    expect(api.report).toHaveBeenCalledTimes(1);
  });

  it("distinguishes no matching priority from no stock and keeps the priority when applying a supplier", async () => {
    const page = await renderPage();
    await page.find(".returns-priority-filter select").setValue("review");
    expect(page.find(".data-table").exists()).toBe(false);
    expect(page.text()).toContain("Нет случаев с выбранным приоритетом");
    expect(page.text()).not.toContain("На выбранном срезе нет складского остатка");
    expect(page.find(".returns-queue .panel__header").text()).toContain("Найдено 0 из 2");

    const filtered = makeReport(1, "SUP-B");
    filtered.priorities[0].priority = "review";
    filtered.priorities[0].days_remaining = 4;
    filtered.priorities[0].deadline_date = "2026-10-04";
    api.report.mockResolvedValueOnce(filtered);
    await page.find(".filter-bar select").setValue("SUP-B");
    await page.find("form").trigger("submit");
    await flushPromises();
    expect(page.find<HTMLSelectElement>(".returns-priority-filter select").element.value).toBe("review");
    expect(page.findAll(".data-table tbody tr")).toHaveLength(1);
    expect(page.find(".data-table").text()).toContain("Поставщик B");
    expect(api.report.mock.lastCall?.[0]).toEqual({ from: "2026-09-01", to: "2026-09-30", supplier_id: "SUP-B" });
  });

  it("shows the default period used by the API when date inputs are cleared", async () => {
    const page = await renderPage();
    for (const input of page.findAll('input[type="date"]')) await input.setValue("");
    await page.find("form").trigger("submit");
    await flushPromises();
    expect(api.report.mock.lastCall?.[0]).toEqual({ from: "2026-09-01", to: "2026-09-30", supplier_id: "" });
    expect(page.find(".returns-context").text()).toContain("01.09.2026 — 30.09.2026");
    expect(page.find(".returns-context").text()).not.toContain("Нет данных");
  });

  it("hides stale results on failure and distinguishes zero stock from missing financial data", async () => {
    const page = await renderPage();
    api.report.mockRejectedValueOnce(new Error("База временно недоступна"));
    await page.find("form").trigger("submit");
    await flushPromises();
    expect(page.find('[role="alert"]').text()).toContain("База временно недоступна");
    expect(page.findAll(".stat-card")).toHaveLength(0);
    expect(page.find(".data-table").exists()).toBe(false);

    api.report.mockResolvedValueOnce(makeReport(0));
    await page.find('[role="alert"] button').trigger("click");
    await flushPromises();
    expect(page.findAll(".stat-card")[0].text()).toContain("0,00");
    expect(page.findAll(".stat-card")[2].text()).toContain("Нет данных");
    expect(page.text()).toContain("На выбранном срезе нет складского остатка");
    expect(page.findAllComponents({ name: "BaseChart" })[1].props("labels")).toEqual([]);
  });

  it("aborts the report request when leaving the page", async () => {
    let resolve: (value: ReturnsReport) => void = () => {};
    api.report.mockReturnValueOnce(new Promise<ReturnsReport>((done) => { resolve = done; }));
    const page = await renderPage();
    const signal = api.report.mock.lastCall?.[1] as AbortSignal;
    expect(signal.aborted).toBe(false);
    expect(page.text()).toContain("Рассчитываем остатки и приоритеты");
    page.unmount();
    wrapper = undefined;
    expect(signal.aborted).toBe(true);
    resolve(makeReport());
    await flushPromises();
  });
});
