import { ref } from 'vue'
import { listSources } from '@/api/source'
import type { DataSource } from '@/types/source'

const sourceOptions = ref<DataSource[]>([])
const loaded = ref(false)
const loading = ref(false)

export function useSourceOptions() {
  async function loadSourceOptions(force = false) {
    if (loaded.value && !force) return
    if (loading.value) return
    loading.value = true
    try {
      const result = await listSources(1, 1000)
      sourceOptions.value = result?.list || []
      loaded.value = true
    } catch {
      sourceOptions.value = []
    } finally {
      loading.value = false
    }
  }

  function invalidateSourceOptions() {
    loaded.value = false
  }

  function getSourceName(sourceId: number): string {
    const s = sourceOptions.value.find((src) => src.id === sourceId)
    return s ? s.name : `ID: ${sourceId}`
  }

  return {
    sourceOptions,
    loadSourceOptions,
    invalidateSourceOptions,
    getSourceName,
  }
}
