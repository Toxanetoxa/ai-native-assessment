<script setup lang="ts">
import { computed } from "vue";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Tooltip,
  Legend,
  Filler,
  type TooltipItem,
} from "chart.js";
import { Bar, Line } from "vue-chartjs";
import StateMessage from "./StateMessage.vue";

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Tooltip,
  Legend,
  Filler,
);
const props = withDefaults(
  defineProps<{
    type?: "line" | "bar";
    labels: string[];
    series: { label: string; values: number[]; color?: string }[];
    unit?: "number" | "rubles";
    label: string;
  }>(),
  { type: "bar", unit: "number" },
);
const palette = ["#900000", "#527a91", "#b89c69", "#6c8a72"];
const data = computed(() => ({
  labels: props.labels,
  datasets: props.series.map((series, index) => ({
    label: series.label,
    data: series.values,
    borderColor: series.color ?? palette[index % palette.length],
    backgroundColor:
      props.type === "line"
        ? "#8a000010"
        : (series.color ?? palette[index % palette.length]),
    borderWidth: props.type === "line" ? 2 : 0,
    borderRadius: 3,
    maxBarThickness: 46,
    pointRadius: 2,
    pointHoverRadius: 5,
    tension: 0.25,
    fill: props.type === "line",
  })),
}));
const formatter = (value: number) =>
  props.unit === "rubles"
    ? `${new Intl.NumberFormat("ru-RU", { notation: "compact", maximumFractionDigits: 1 }).format(value)} ₽`
    : new Intl.NumberFormat("ru-RU").format(value);
const options = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false as const,
  interaction: { intersect: false, mode: "index" as const },
  plugins: {
    legend: { display: props.series.length > 1, position: "bottom" as const },
    tooltip: {
      callbacks: {
        label: (context: TooltipItem<"line"> | TooltipItem<"bar">) =>
          `${context.dataset.label}: ${formatter(Number(context.parsed.y ?? 0))}`,
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      border: { display: false },
      ticks: {
        color: "#64748b",
        font: { family: "system-ui", size: 11 },
        maxTicksLimit: 8,
      },
    },
    y: {
      beginAtZero: true,
      border: { display: false },
      grid: { color: "#f0f2f4" },
      ticks: {
        color: "#64748b",
        font: { family: "system-ui", size: 11 },
        callback: (value: string | number) => formatter(Number(value)),
      },
    },
  },
}));
</script>
<template>
  <div class="chart">
    <StateMessage
      v-if="!labels.length"
      title="Нет данных для графика"
    /><component
      :is="type === 'line' ? Line : Bar"
      v-else
      :data="data"
      :options="options"
      role="img"
      :aria-label="label"
    />
    <table class="sr-only">
      <caption>
        {{
          label
        }}
      </caption>
      <thead>
        <tr>
          <th>Показатель</th>
          <th v-for="seriesItem in series" :key="seriesItem.label">
            {{ seriesItem.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(item, index) in labels" :key="`${item}-${index}`">
          <th>{{ item }}</th>
          <td v-for="seriesItem in series" :key="seriesItem.label">
            {{ formatter(seriesItem.values[index] ?? 0) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
