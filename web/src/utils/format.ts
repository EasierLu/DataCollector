export function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

export function formatDateOnly(dateStr: string | null | undefined): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

export function truncate(text: string | null | undefined, maxLength = 50): string {
  if (!text) return '-'
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

export function formatJSON(obj: any): string {
  try {
    if (typeof obj === 'string') {
      obj = JSON.parse(obj)
    }
    return JSON.stringify(obj, null, 2)
  } catch {
    return String(obj)
  }
}

export function getDataSummary(data: any): string {
  if (!data) return '-'
  try {
    const obj = typeof data === 'string' ? JSON.parse(data) : data
    const str = JSON.stringify(obj)
    return str.length > 60 ? str.substring(0, 60) + '...' : str
  } catch {
    return String(data).substring(0, 60)
  }
}
