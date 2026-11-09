const currency = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  maximumFractionDigits: 2,
});
const integer = new Intl.NumberFormat("ru-RU");
export const money = (minor: number | null | undefined) =>
  minor == null ? "Нет данных" : currency.format(minor / 100);
export const number = (value: number) => integer.format(value);
export const date = (value: string | null | undefined) =>
  !value
    ? "Нет данных"
    : new Intl.DateTimeFormat("ru-RU", { timeZone: "Europe/Moscow" }).format(
        new Date(value.length === 10 ? `${value}T12:00:00+03:00` : value),
      );
export const shortDate = (value: string) => date(value).slice(0, 5);
export const statuses: Record<
  string,
  { label: string; tone: "neutral" | "warning" | "success" | "danger" }
> = {
  on_warehouse: { label: "На складе", tone: "neutral" },
  pending: { label: "Ожидает решения", tone: "warning" },
  accepted: { label: "Принят поставщиком", tone: "success" },
  rejected: { label: "Отказ", tone: "danger" },
  partial: { label: "Частичное решение", tone: "warning" },
};
export const reasons: Record<string, string> = {
  deadline: "Нарушение срока",
  packaging: "Комплектность / упаковка",
  policy: "Условия поставщика",
  other: "Прочая причина",
};
