export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: unknown }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: unknown }
  | { type: 'final_draft'; content: string }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }
  | { type: 'stage_start'; stage_id: string; stage_name: string; stage_type: 'parallel' | 'serial' | 'human' }
  | { type: 'stage_done'; stage_id: string }
  | { type: 'worker_start'; stage_id: string; worker_name: string; worker_display: string }
  | { type: 'worker_token'; worker_name: string; content: string }
  | { type: 'worker_done'; worker_name: string }
  | { type: 'synth_start'; stage_id: string }
  | { type: 'synth_token'; content: string }
  | { type: 'synth_done'; stage_id: string }

export function parseSSELine(line: string): SSEEvent | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as SSEEvent
  } catch {
    return null
  }
}
