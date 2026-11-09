<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import Panel from "../components/Panel.vue";
import FilterBar from "../components/FilterBar.vue";
import StatCard from "../components/StatCard.vue";
import BaseChart from "../components/BaseChart.vue";
import DataTable from "../components/DataTable.vue";
import StateMessage from "../components/StateMessage.vue";
import { useMeta } from "../composables/useMeta";
import { fetchOperations } from "../api/workspace";
import type { Filters, OperationsReport } from "../api/types";
import { money, number, shortDate } from "../utils/format";

const { meta, load } = useMeta();
const filters = ref<Filters>({ from: "", to: "", supplier_id: "" });
const report = ref<OperationsReport | null>(null);
const busy = ref(true),
  error = ref("");
let controller: AbortController | undefined;
async function refresh() {
  controller?.abort();
  const request = new AbortController();
  controller = request;
  busy.value = true;
  error.value = "";
  try {
    report.value = await fetchOperations(filters.value, request.signal);
  } catch (cause) {
    if (!request.signal.aborted)
      error.value =
        cause instanceof Error ? cause.message : "Не удалось загрузить отчёт";
  } finally {
    if (!request.signal.aborted) busy.value = false;
  }
}
async function initialize() {
  try {
    error.value = "";
    busy.value = true;
    const data = await load();
    filters.value = {
      from: data.manifest.default_from,
      to: data.manifest.default_to,
      supplier_id: "",
    };
    await refresh();
  } catch (cause) {
    busy.value = false;
    error.value =
      cause instanceof Error ? cause.message : "Не удалось загрузить данные";
  }
}
onMounted(initialize);
onBeforeUnmount(() => controller?.abort());
const columns = [
  { key: "name", label: "Поставщик" },
  { key: "count", label: "Поступления", numeric: true },
  { key: "quantity", label: "Товары, шт.", numeric: true },
  { key: "amount_minor", label: "Сумма", numeric: true },
];
const daily = computed(() => [
  {
    label: "Сумма поступлений",
    values: report.value?.days.map((day) => day.amount_minor / 100) ?? [],
  },
]);
const bySupplier = computed(() => [
  {
    label: "Сумма поступлений",
    values:
      report.value?.suppliers.map((supplier) => supplier.amount_minor / 100) ??
      [],
  },
]);
</script>
<template>
  <DashboardLayout
    title="Операционные показатели"
    description="Поступления товаров от поставщиков за выбранный период."
    ><Panel
      ><FilterBar
        v-model="filters"
        :suppliers="meta?.suppliers ?? []"
        :busy="busy"
        @apply="refresh" /></Panel
    ><StateMessage
      v-if="error"
      title="Не удалось загрузить отчёт"
      :detail="error"
      error
      @retry="meta ? refresh() : initialize()"
    /><StateMessage
      v-else-if="busy"
      title="Загружаем показатели"
      loading
    /><template v-else-if="report"
      ><div class="stats-grid">
        <StatCard
          label="Поступления"
          :value="number(report.count)"
          note="Документов за период"
        /><StatCard
          label="Сумма поступлений"
          :value="money(report.amount_minor)"
          note="По учётной стоимости"
        /><StatCard
          label="Поставщики"
          :value="number(report.suppliers.length)"
          note="С поступлениями за период"
        />
      </div>
      <div class="charts-grid">
        <Panel title="Динамика поступлений" subtitle="По дням · рубли"
          ><BaseChart
            :labels="
              report.count ? report.days.map((day) => shortDate(day.date)) : []
            "
            :series="daily"
            unit="rubles"
            label="Сумма поступлений по дням" /></Panel
        ><Panel
          title="Поступления по поставщикам"
          subtitle="Сумма за выбранный период · рубли"
          ><BaseChart
            :labels="report.suppliers.map((supplier) => supplier.name)"
            :series="bySupplier"
            unit="rubles"
            label="Сумма поступлений по поставщикам"
        /></Panel>
      </div>
      <Panel title="Поставщики" subtitle="Детализация поступлений за период"
        ><DataTable
          :rows="report.suppliers"
          :columns="columns"
          caption="Поступления по поставщикам"
          ><template #count="{ row }">{{ number(row.count) }}</template
          ><template #quantity="{ row }">{{ number(row.quantity) }}</template
          ><template #amount_minor="{ row }"
            ><strong>{{ money(row.amount_minor) }}</strong></template
          ></DataTable
        ><template #footer
          ><span class="small muted"
            >Период определяется датой поступления. Часовой пояс — Москва.</span
          ></template
        ></Panel
      ></template
    ></DashboardLayout
  >
</template>
