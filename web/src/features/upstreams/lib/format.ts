export function formatUpstreamMoney(value: number | null): string {
  return value === null
    ? '--'
    : new Intl.NumberFormat(undefined, {
        style: 'currency',
        currency: 'USD',
      }).format(value)
}

export function formatUpstreamTime(value: number, timezone?: string): string {
  if (!value) return '--'
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'short',
    timeStyle: 'short',
    timeZone: timezone,
  }).format(value * 1000)
}
