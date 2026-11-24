<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from "vue";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import Panel from "../components/Panel.vue";
import FilterBar from "../components/FilterBar.vue";
import StatCard from "../components/StatCard.vue";
import BaseChart from "../components/BaseChart.vue";
import DataTable from "../components/DataTable.vue";
import StatusBadge from "../components/StatusBadge.vue";
import StateMessage from "../components/StateMessage.vue";
import BaseButton from "../components/BaseButton.vue";
import BaseSelect from "../components/BaseSelect.vue";
import { useMeta } from "../composables/useMeta";
import { fetchReturnsReport } from "../api/workspace";
import type { Filters, PriorityCase, ReturnsReport } from "../api/types";
import { date, money, number, shortDate } from "../utils/format";

const { meta, load } = useMeta();
const filters = ref<Filters>({ from: "", to: "", supplier_id: "" });
const appliedFilters = ref<Filters | null>(null);
const report = ref<ReturnsReport | null>(null);
const busy = ref(true);
const error = ref("");
const page = ref(1);
const pageSize = 25;
const priorityFilter = ref("");
const analysis = useTemplateRef<HTMLDivElement>("analysis");
watch(priorityFilter, () => { page.value = 1; });
let controller: AbortController | undefined;
let disposed = false;

async function refresh() {
  controller?.abort();
  const request = new AbortController();
  controller = request;
  const requestedFilters = {
    ...filters.value,
    from: filters.value.from || meta.value?.manifest.default_from || "",
    to: filters.value.to || meta.value?.manifest.default_to || "",
  };
  filters.value = { ...requestedFilters };
  busy.value = true;
  error.value = "";
  report.value = null;
  page.value = 1;
  try {
    const result = await fetchReturnsReport(requestedFilters, request.signal);
    if (!request.signal.aborted && !disposed) {
      report.value = result;
      appliedFilters.value = requestedFilters;
    }
  } catch (cause) {
    if (!request.signal.aborted && !disposed) {
      error.value = cause instanceof Error ? cause.message : "Не удалось загрузить возвраты";
    }
  } finally {
    if (!request.signal.aborted && !disposed) busy.value = false;
  }
}

async function initialize() {
  busy.value = true;
  error.value = "";
  try {
    const data = await load();
    if (disposed) return;
    filters.value = {
      from: data.manifest.default_from,
      to: data.manifest.default_to,
      supplier_id: "",
    };
    await refresh();
  } catch (cause) {
    if (!disposed) {
      error.value = cause instanceof Error ? cause.message : "Не удалось загрузить данные";
      busy.value = false;
    }
  }
}

onMounted(initialize);
onBeforeUnmount(() => {
  disposed = true;
  controller?.abort();
});

const changedFilters = computed(() => appliedFilters.value !== null && (
  filters.value.from !== appliedFilters.value.from ||
  filters.value.to !== appliedFilters.value.to ||
  filters.value.supplier_id !== appliedFilters.value.supplier_id
));
const supplierLabel = computed(() => {
  const id = appliedFilters.value?.supplier_id;
  return id ? (meta.value?.suppliers.find((supplier) => supplier.id === id)?.name ?? id) : "Все поставщики";
});
const clampedPeriod = computed(() => report.value && appliedFilters.value && appliedFilters.value.to > report.value.as_of.slice(0, 10));
const dateTime = (value: string) => new Intl.DateTimeFormat("ru-RU", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Europe/Moscow",
}).format(new Date(value));

const dailySeries = computed(() => [
  { label: "Общий остаток", values: report.value?.days.map((day) => day.value_minor / 100) ?? [], color: "#900000" },
  { label: "Из него старше 30 дней", values: report.value?.days.map((day) => day.aged_value_minor / 100) ?? [], color: "#527a91" },
]);
const supplierSeries = computed(() => [
  { label: "0–7 дней", values: report.value?.suppliers.map((supplier) => supplier.age_0_7_minor / 100) ?? [], color: "#6c8a72" },
  { label: "8–30 дней", values: report.value?.suppliers.map((supplier) => supplier.age_8_30_minor / 100) ?? [], color: "#527a91" },
  { label: "31–60 дней", values: report.value?.suppliers.map((supplier) => supplier.age_31_60_minor / 100) ?? [], color: "#b89c69" },
  { label: "61+ дней", values: report.value?.suppliers.map((supplier) => supplier.age_61_plus_minor / 100) ?? [], color: "#900000" },
]);

const priorityLabels: Record<PriorityCase["priority"], {
  label: string;
  tone: "neutral" | "warning" | "danger";
}> = {
  urgent: { label: "Срочно", tone: "danger" },
  clarify: { label: "Уточнить", tone: "warning" },
  review: { label: "К разбору", tone: "neutral" },
};
const columns = [
  { key: "id", label: "Случай" },
  { key: "stock_value_minor", label: "Остаток", numeric: true },
  { key: "priority", label: "Приоритет и срок" },
  { key: "reason", label: "Основание" },
  { key: "next_action", label: "Следующее действие" },
];
const filteredPriorities = computed(() => {
  const priorities = report.value?.priorities ?? [];
  return priorityFilter.value
    ? priorities.filter((row) => row.priority === priorityFilter.value)
    : priorities;
});
const total = computed(() => filteredPriorities.value.length);
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));
const visibleRows = computed(() => filteredPriorities.value.slice((page.value - 1) * pageSize, page.value * pageSize));
const firstRow = computed(() => total.value ? (page.value - 1) * pageSize + 1 : 0);
const lastRow = computed(() => Math.min(page.value * pageSize, total.value));

function nextPage(delta: number) {
  page.value = Math.min(pageCount.value, Math.max(1, page.value + delta));
}

function remainingLabel(days: number | null) {
  if (days === null) return "Срок не определён";
  if (days < 0) return `Окно истекло ${number(-days)} дн. назад`;
  if (days === 0) return "Последний день окна";
  return `Осталось ${number(days)} календ. дн.`;
}
</script>

<template>
  <DashboardLayout
    class="returns-page"
    title="Возвраты"
    description="Складские остатки и очередь случаев для первоочередного разбора."
  >
    <Panel>
      <FilterBar
        v-model="filters"
        :suppliers="meta?.suppliers ?? []"
        :busy="busy"
        @apply="refresh"
      />
    </Panel>
    <p class="small muted returns-period-note">
      Начало периода задаёт диапазон динамики. Остаток и очередь рассчитываются на конец периода,
      включая поступления прошлых месяцев. Все даты — по Москве.
    </p>

    <StateMessage
      v-if="error"
      title="Не удалось загрузить возвраты"
      :detail="error"
      error
      @retry="meta ? refresh() : initialize()"
    />
    <StateMessage v-else-if="busy" title="Рассчитываем остатки и приоритеты" loading />
    <template v-else-if="report && appliedFilters">
      <div class="returns-context" aria-live="polite">
        <p>
          <strong>Срез: {{ dateTime(report.as_of) }} МСК</strong>
          <span class="muted"> · {{ supplierLabel }}</span>
        </p>
        <p class="small muted">
          Период динамики: {{ date(appliedFilters.from) }} — {{ date(appliedFilters.to) }}.
          <template v-if="clampedPeriod">Динамика заканчивается на доступном срезе.</template>
        </p>
        <p v-if="clampedPeriod" class="returns-notice small" role="status">
          Конец выбранного периода позже доступных данных. Показан остаток на {{ date(report.as_of) }}.
        </p>
        <p v-if="changedFilters" class="returns-notice small" role="status">
          Фильтры изменены. Нажмите «Применить», чтобы обновить показанные данные.
        </p>
      </div>

      <div class="stats-grid returns-metrics">
        <StatCard
          label="Остаток по учётной стоимости"
          :value="money(report.stock.value_minor)"
          :note="`${number(report.stock.line_count)} строк · ${number(report.stock.quantity)} шт. Стоимость положительных остатков на дату среза.`"
        />
        <StatCard
          label="Из него старше 30 дней"
          :value="money(report.stock.aged_value_minor)"
          note="Часть остатка с возрастом строго больше 30 календарных дней от складской приёмки."
        />
        <StatCard
          class="returns-finance-card"
          label="Чистое финансовое возмещение"
          value="Нет данных"
          note="Нужны возмещения и зачёты поставщика, выручка от повторных продаж и полные расходы по выбранной группе возвратов."
        />
      </div>
      <p class="small muted returns-metric-note">
        Учётный остаток показывает замороженную стоимость. Экономический эффект нужно подтверждать
        возмещениями и расходами: уменьшение остатка после отправки или списания само по себе не доказывает улучшения.
      </p>

      <div class="returns-workflow">
        <p><strong>Выберите случай → согласуйте действие и ответственного → зафиксируйте результат.</strong></p>
        <p class="small muted">
          Очередь предлагает приоритет и следующий шаг. Ответственного, согласованное действие и результат
          фиксируйте в отдельной рабочей таблице пилота.
        </p>
        <button type="button" class="text-link small" @click="analysis?.focus()">Перейти к графикам</button>
      </div>

      <Panel
        class="returns-queue"
        title="Приоритетные случаи"
        :subtitle="`Найдено ${number(total)} из ${number(report.total_priority_cases)} строк с положительным складским остатком.`"
      >
        <template #actions>
          <BaseSelect
            v-model="priorityFilter"
            class="returns-priority-filter"
            label="Приоритет в таблице"
            :options="[
              { value: '', label: 'Все приоритеты' },
              { value: 'urgent', label: 'Срочно' },
              { value: 'clarify', label: 'Уточнить' },
              { value: 'review', label: 'К разбору' },
            ]"
          />
        </template>
        <div class="returns-rules small">
          <p>{{ report.priority_rules.order }}</p>
          <p class="muted">
            Срочный разбор — до конца подтверждённого окна получения поставщиком осталось
            от 0 до {{ report.priority_rules.urgent_window_days }} календарных дней.
            Порог срочности — правило пилота. Перед действием нужно проверить препятствия и согласование поставщика.
          </p>
        </div>
        <DataTable
          v-if="total > 0"
          class="returns-table"
          :rows="visibleRows"
          :columns="columns"
          caption="Приоритетные случаи возвратов с основаниями выбора и следующими действиями"
        >
          <template #id="{ row }">
            <div class="returns-case">
              <strong>{{ row.return_id || row.id }}</strong>
              <span class="row-id">{{ row.id }}</span>
              <span>{{ row.product }}</span>
              <span class="small muted">{{ row.supplier_name }}</span>
            </div>
          </template>
          <template #stock_value_minor="{ row }">
            <strong class="returns-amount">{{ money(row.stock_value_minor) }}</strong>
            <span class="small muted returns-quantity">{{ number(row.stock_quantity) }} шт.</span>
          </template>
          <template #priority="{ row }">
            <StatusBadge :tone="priorityLabels[row.priority].tone">{{ priorityLabels[row.priority].label }}</StatusBadge>
            <div class="returns-deadline small" :class="{ 'returns-deadline--expired': row.days_remaining !== null && row.days_remaining < 0 }">
              <template v-if="row.deadline_date !== null">
                <span>Получение до {{ date(row.deadline_date) }} включительно</span>
                <span>{{ remainingLabel(row.days_remaining) }}</span>
              </template>
              <span v-else>Срок не определён</span>
            </div>
          </template>
          <template #reason="{ row }"><p>{{ row.reason }}</p></template>
          <template #next_action="{ row }"><p>{{ row.next_action }}</p></template>
        </DataTable>
        <StateMessage
          v-else-if="report.priorities.length"
          title="Нет случаев с выбранным приоритетом"
          detail="Выберите другой приоритет или все приоритеты. Метрики и графики рассчитаны по периоду и поставщику."
        />
        <StateMessage
          v-else
          title="На выбранном срезе нет складского остатка"
          detail="Положительные остатки по выбранному поставщику не найдены."
        />
        <template v-if="total > 0" #footer>
          <div class="pagination returns-pagination">
            <span class="small muted">{{ number(firstRow) }}–{{ number(lastRow) }} из {{ number(total) }} · страница {{ page }} из {{ pageCount }}</span>
            <div class="pagination__buttons">
              <BaseButton :disabled="page <= 1" @click="nextPage(-1)">Назад</BaseButton>
              <BaseButton :disabled="page >= pageCount" @click="nextPage(1)">Далее</BaseButton>
            </div>
          </div>
        </template>
      </Panel>

      <div id="returns-analysis" ref="analysis" tabindex="-1" class="charts-grid returns-analysis">
        <Panel title="Динамика остатков" subtitle="На конец каждого дня · рубли">
          <BaseChart
            type="line"
            stepped
            :labels="report.days.map((day) => shortDate(day.date))"
            :series="dailySeries"
            unit="rubles"
            label="Общий остаток возвратов и его часть старше 30 дней по дням, рубли"
          />
          <template #footer>
            <p class="small muted">
              Каждая точка учитывает сведения, известные на тот день. Последняя ограничена временем среза.
            </p>
          </template>
        </Panel>
        <Panel title="Остатки по поставщикам" :subtitle="`Возрастная структура на ${date(report.as_of)} · рубли`">
          <BaseChart
            stacked
            :labels="report.suppliers.map((supplier) => supplier.name)"
            :series="supplierSeries"
            unit="rubles"
            label="Остатки возвратов по поставщикам с разбивкой по календарному возрасту, рубли"
          />
          <template #footer>
            <p class="small muted">
              Возраст — от приёмки на нашем складе. Группы не пересекаются; высота столбца — общий остаток поставщика.
            </p>
          </template>
        </Panel>
      </div>

      <Panel>
        <details class="returns-methods">
          <summary>Данные для расчётов и ограничения</summary>
          <ul>
            <li>Для остатка нужны проведённые строки возвратов и движения количества и стоимости с датами операции и регистрации.</li>
            <li>Для возраста нужна дата складской приёмки. Порог 30 дней — допущение для анализа залежавшихся товаров.</li>
            <li>Для оценки экономического эффекта нужно отслеживать исходную группу возвратов, подтверждённые возмещения и полные расходы.</li>
            <li v-for="limitation in report.limitations" :key="limitation">{{ limitation }}</li>
          </ul>
        </details>
      </Panel>
    </template>
  </DashboardLayout>
</template>

<style scoped>
.returns-period-note,
.returns-metric-note,
.returns-workflow {
  max-width: 100ch;
}
.returns-context {
  padding: 12px 16px;
  border-left: 3px solid var(--primary);
  background: var(--subtle);
}
.returns-workflow p + p,
.returns-context p + p {
  margin-top: 5px;
}
.returns-workflow > button {
  display: inline-block;
  margin-top: 5px;
}
.returns-analysis {
  scroll-margin-top: 16px;
}
.returns-notice {
  color: #956b1e;
}
.returns-finance-card :deep(.stat-card__value) {
  font-size: 23px;
}
.returns-rules {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 16px;
}
.returns-queue :deep(.panel__header) {
  flex-wrap: wrap;
  align-items: flex-start;
}
.returns-priority-filter {
  min-width: 180px;
  flex: 0 1 220px;
}
.returns-table :deep(.data-table) {
  table-layout: fixed;
  min-width: 960px;
}
.returns-table :deep(.data-table th:nth-child(1)) { width: 20%; }
.returns-table :deep(.data-table th:nth-child(2)) { width: 12%; }
.returns-table :deep(.data-table th:nth-child(3)) { width: 18%; }
.returns-table :deep(.data-table th:nth-child(4)),
.returns-table :deep(.data-table th:nth-child(5)) { width: 25%; }
.returns-table :deep(.data-table td) {
  vertical-align: top;
  padding-top: 12px;
  padding-bottom: 12px;
  font-size: 13px;
  overflow-wrap: anywhere;
}
.returns-table :deep(.data-table th) {
  font-size: 13px;
}
.returns-case {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.returns-case .row-id {
  margin-top: -3px;
}
.returns-amount {
  white-space: nowrap;
}
.returns-quantity {
  display: block;
  margin-top: 5px;
}
.returns-deadline {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
  color: var(--muted);
}
.returns-deadline--expired {
  color: #a04c4c;
}
.returns-methods summary {
  cursor: pointer;
  font-weight: 600;
}
.returns-methods ul {
  padding-left: 20px;
  margin: 12px 0 0;
  color: var(--muted);
  font-size: 12px;
}
.returns-methods li + li {
  margin-top: 7px;
}
@media (max-width: 1200px) {
  .returns-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .returns-finance-card { grid-column: 1 / -1; }
}
@media (max-width: 600px) {
  .returns-priority-filter { min-width: 0; flex-basis: 100%; }
  .returns-metrics { grid-template-columns: 1fr; }
  .returns-finance-card { grid-column: auto; }
  .returns-finance-card :deep(.stat-card__value) { font-size: 20px; }
  .returns-pagination { align-items: flex-start; flex-direction: column; gap: 12px; }
}
</style>
