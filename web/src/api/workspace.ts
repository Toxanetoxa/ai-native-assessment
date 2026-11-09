import { get } from "./client";
import type {
  Filters,
  Meta,
  OperationsReport,
  ReturnDetail,
  ReturnRow,
  ReturnsReport,
} from "./types";

export const fetchMeta = () => get<Meta>("/meta");
export const fetchOperations = (filters: Filters, signal?: AbortSignal) =>
  get<OperationsReport>("/operations", { ...filters }, signal);
export const fetchReturnsReport = (filters: Filters, signal?: AbortSignal) =>
  get<ReturnsReport>("/returns-report", { ...filters }, signal);
export const fetchReturns = (
  filters: Filters,
  page: number,
  signal?: AbortSignal,
) =>
  get<{ items: ReturnRow[]; total: number }>(
    "/returns",
    { ...filters, page, page_size: 25 },
    signal,
  );
export const fetchReturn = (id: string) =>
  get<ReturnDetail>(`/returns/${encodeURIComponent(id)}`);
