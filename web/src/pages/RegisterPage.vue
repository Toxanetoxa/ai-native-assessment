<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import Panel from "../components/Panel.vue";
import FilterBar from "../components/FilterBar.vue";
import DataTable from "../components/DataTable.vue";
import StateMessage from "../components/StateMessage.vue";
import StatusBadge from "../components/StatusBadge.vue";
import BaseButton from "../components/BaseButton.vue";
import ReturnDialog from "../components/ReturnDialog.vue";
import { useMeta } from "../composables/useMeta";
import { fetchReturns, fetchReturn } from "../api/workspace";
import type { Filters, ReturnDetail, ReturnRow } from "../api/types";
import { date, money, number, statuses } from "../utils/format";

const route = useRoute();
const { meta, load } = useMeta();
const filters = ref<Filters>({ from: "", to: "", supplier_id: "", q: "" });
const rows = ref<ReturnRow[]>([]),
  total = ref(0),
  page = ref(1);
const busy = ref(true),
  error = ref("");
const selected = ref(""),
  detail = ref<ReturnDetail | null>(null),
  detailError = ref("");
let controller: AbortController | undefined;
async function refresh(resetPage = true) {
  if (resetPage) page.value = 1;
  controller?.abort();
  const request = new AbortController();
  controller = request;
  busy.value = true;
  error.value = "";
  try {
    const result = await fetchReturns(
      filters.value,
      page.value,
      request.signal,
    );
    rows.value = result.items;
    total.value = result.total;
  } catch (cause) {
    if (!request.signal.aborted)
      error.value =
        cause instanceof Error
          ? cause.message
          : "Не удалось загрузить возвраты";
  } finally {
    if (!request.signal.aborted) busy.value = false;
  }
}
async function initialize() {
  try {
    busy.value = true;
    error.value = "";
    const data = await load();
    filters.value = {
      from: data.manifest.default_from,
      to: data.manifest.default_to,
      supplier_id: "",
      q: String(route.query.q ?? ""),
    };
    await refresh();
  } catch (cause) {
    busy.value = false;
    error.value =
      cause instanceof Error ? cause.message : "Не удалось загрузить данные";
  }
}
async function showDetail(id: string) {
  selected.value = id;
  detail.value = null;
  detailError.value = "";
  try {
    const result = await fetchReturn(id);
    if (selected.value === id) detail.value = result;
  } catch (cause) {
    if (selected.value === id)
      detailError.value =
        cause instanceof Error ? cause.message : "Ошибка загрузки";
  }
}
function nextPage(delta: number) {
  page.value += delta;
  void refresh(false);
}
onMounted(initialize);
onBeforeUnmount(() => controller?.abort());
const columns = [
  { key: "return_id", label: "Возврат" },
  { key: "product", label: "Товар" },
  { key: "supplier_name", label: "Поставщик" },
  { key: "warehouse_received_at", label: "Принят на складе" },
  { key: "quantity", label: "Кол-во", numeric: true },
  { key: "book_value_minor", label: "Учётная сумма", numeric: true },
  { key: "status", label: "Состояние" },
];
</script>
<template>
  <DashboardLayout
    title="Реестр возвратов"
    description="Клиентские возвраты и история работы с поставщиками."
    ><Panel
      ><FilterBar
        v-model="filters"
        :suppliers="meta?.suppliers ?? []"
        :busy="busy"
        search
        @apply="refresh()" /></Panel
    ><StateMessage
      v-if="error"
      title="Не удалось загрузить реестр"
      :detail="error"
      error
      @retry="meta ? refresh() : initialize()" /><StateMessage
      v-else-if="busy"
      title="Загружаем возвраты"
      loading /><Panel
      v-else
      title="Клиентские возвраты"
      :subtitle="`Найдено строк: ${number(total)}. Период — по дате приёма на складе.`"
      ><DataTable
        :rows="rows"
        :columns="columns"
        caption="Реестр клиентских возвратов"
        ><template #return_id="{ row }"
          ><button class="text-link document-link" @click="showDetail(row.id)">
            {{ row.return_id }}</button
          ><span class="row-id">{{ row.id }}</span></template
        ><template #product="{ row }"
          ><span class="product-name">{{ row.product }}</span></template
        ><template #warehouse_received_at="{ row }">{{
          date(row.warehouse_received_at)
        }}</template
        ><template #quantity="{ row }">{{ row.quantity }} шт.</template
        ><template #book_value_minor="{ row }">{{
          money(row.book_value_minor)
        }}</template
        ><template #status="{ row }"
          ><StatusBadge :tone="statuses[row.status]?.tone">{{
            statuses[row.status]?.label
          }}</StatusBadge></template
        ></DataTable
      ><template #footer
        ><div class="pagination">
          <span class="small muted"
            >{{ total ? (page - 1) * 25 + 1 : 0 }}–{{
              Math.min(page * 25, total)
            }}
            из {{ number(total) }}</span
          >
          <div class="pagination__buttons">
            <BaseButton :disabled="page === 1 || busy" @click="nextPage(-1)"
              >Назад</BaseButton
            ><BaseButton
              :disabled="page * 25 >= total || busy"
              @click="nextPage(1)"
              >Далее</BaseButton
            >
          </div>
        </div></template
      ></Panel
    ><ReturnDialog
      :open="Boolean(selected)"
      :detail="detail"
      :error="detailError"
      @close="selected = ''"
      @retry="showDetail(selected)"
  /></DashboardLayout>
</template>
