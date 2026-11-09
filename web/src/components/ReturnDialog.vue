<script setup lang="ts">
import { nextTick, ref, watch } from "vue";
import type { ReturnDetail } from "../api/types";
import BaseButton from "./BaseButton.vue";
import BaseIcon from "./BaseIcon.vue";
import StateMessage from "./StateMessage.vue";
import StatusBadge from "./StatusBadge.vue";
import { date, money, reasons, statuses } from "../utils/format";

const props = defineProps<{
  open: boolean;
  detail: ReturnDetail | null;
  error: string;
}>();
defineEmits<{ close: []; retry: [] }>();
const dialog = ref<HTMLDialogElement>();
watch(
  () => props.open,
  async (open) => {
    await nextTick();
    if (open && !dialog.value?.open) dialog.value?.showModal();
    else if (!open && dialog.value?.open) dialog.value.close();
  },
  { immediate: true },
);
const eventNames: Record<string, string> = {
  received: "Получение поставщиком",
  accepted: "Принято",
  rejected: "Отказ",
  reversal: "Отмена события",
};
</script>
<template>
  <dialog
    ref="dialog"
    class="detail-dialog"
    aria-labelledby="return-dialog-title"
    @cancel.prevent="$emit('close')"
    @click="$event.target === $event.currentTarget && $emit('close')"
  >
    <header class="detail-dialog__header">
      <div>
        <p class="small muted">Карточка возврата</p>
        <h2 id="return-dialog-title">
          {{ detail?.row.return_id ?? "Загрузка" }}
        </h2>
      </div>
      <BaseButton
        variant="ghost"
        aria-label="Закрыть карточку"
        @click="$emit('close')"
        ><BaseIcon name="close"
      /></BaseButton>
    </header>
    <div class="detail-dialog__body">
      <StateMessage
        v-if="error"
        title="Не удалось загрузить карточку"
        :detail="error"
        error
        @retry="$emit('retry')"
      /><StateMessage
        v-else-if="!detail"
        title="Загружаем историю"
        loading
      /><template v-else
        ><div class="detail-product">
          <strong>{{ detail.row.product }}</strong
          ><StatusBadge :tone="statuses[detail.row.status]?.tone">{{
            statuses[detail.row.status]?.label
          }}</StatusBadge>
        </div>
        <dl class="details-list">
          <div>
            <dt>Строка документа</dt>
            <dd>{{ detail.row.id }}</dd>
          </div>
          <div>
            <dt>Поставщик</dt>
            <dd>{{ detail.row.supplier_name }}</dd>
          </div>
          <div>
            <dt>Количество</dt>
            <dd>{{ detail.row.quantity }} шт.</dd>
          </div>
          <div>
            <dt>Учётная сумма</dt>
            <dd>{{ money(detail.row.book_value_minor) }}</dd>
          </div>
          <div>
            <dt>Отправка товара поставщиком</dt>
            <dd>{{ date(detail.row.supplier_shipped_at) }}</dd>
          </div>
          <div>
            <dt>Обращение клиента</dt>
            <dd>{{ date(detail.row.requested_at) }}</dd>
          </div>
          <div>
            <dt>Приём возврата от клиента</dt>
            <dd>{{ date(detail.row.warehouse_received_at) }}</dd>
          </div>
          <div>
            <dt>Остаток на складе</dt>
            <dd>
              {{ detail.row.stock_quantity }} шт. ·
              {{ money(detail.row.stock_value_minor) }}
            </dd>
          </div>
          <div>
            <dt>Комплектность</dt>
            <dd>
              {{
                detail.row.packaging_complete == null
                  ? "Нет данных"
                  : detail.row.packaging_complete
                    ? "Полная"
                    : "Нарушена"
              }}
            </dd>
          </div>
          <div>
            <dt>Окно возврата</dt>
            <dd>
              {{
                detail.policy?.confirmed && detail.policy.window_days != null
                  ? `${detail.policy.window_days} календарных дней`
                  : "Требует проверки"
              }}
            </dd>
          </div>
        </dl>
        <h3>Отправки возврата поставщику</h3>
        <p v-if="!detail.shipments.length" class="small muted">
          Отправки не зарегистрированы.
        </p>
        <ul v-else class="event-list">
          <li v-for="shipment in detail.shipments" :key="shipment.id">
            <strong>{{ shipment.id }}</strong
            ><span class="small muted"
              >{{ date(shipment.dispatched_at) }} ·
              {{ shipment.posted ? "Проведён" : "Отменён" }}</span
            >
          </li>
        </ul>
        <h3>Сообщения учётной системы</h3>
        <p v-if="!detail.events.length" class="small muted">
          Сообщений пока нет.
        </p>
        <ul v-else class="event-list">
          <li
            v-for="(event, index) in detail.events"
            :key="`${event.event_id}-${index}`"
          >
            <div>
              <strong>{{ eventNames[event.type] ?? event.type }}</strong
              ><span class="small muted"
                >{{ date(event.occurred_at) }} · {{ event.quantity }} шт.{{
                  event.reason
                    ? ` · ${reasons[event.reason] ?? event.reason}`
                    : ""
                }}</span
              >
              <span class="small muted"
                >Внесено {{ date(event.recorded_at)
                }}{{
                  event.reverses_event_id
                    ? ` · отменяет ${event.reverses_event_id}`
                    : ""
                }}</span
              >
            </div>
            <span class="event-id">{{ event.event_id }}</span>
          </li>
        </ul>
        <h3>Финансовые записи</h3>
        <p v-if="!detail.financial_entries.length" class="small muted">
          Финансовые записи отсутствуют.
        </p>
        <ul v-else class="event-list">
          <li v-for="entry in detail.financial_entries" :key="entry.id">
            <span>{{
              entry.type === "resale"
                ? "Повторная продажа"
                : entry.type === "handling_cost"
                  ? "Расходы обработки"
                  : entry.type
            }}</span
            ><strong>{{ money(entry.amount_minor) }}</strong>
          </li>
        </ul></template
      >
    </div>
    <footer class="detail-dialog__footer">
      <BaseButton @click="$emit('close')">Закрыть</BaseButton>
    </footer>
  </dialog>
</template>
