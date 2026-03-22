export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: unknown }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: unknown }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }

export function parseSSELine(line: string): SSEEvent | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as SSEEvent
  } catch {
    return null
  }
}
