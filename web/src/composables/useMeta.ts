import { ref } from "vue";
import { fetchMeta } from "../api/workspace";
import type { Meta } from "../api/types";

const meta = ref<Meta | null>(null);
let request: Promise<Meta> | null = null;

export function useMeta() {
  async function load() {
    if (meta.value) return meta.value;
    request ??= fetchMeta().catch((error) => {
      request = null;
      throw error;
    });
    meta.value = await request;
    return meta.value;
  }
  return { meta, load };
}
