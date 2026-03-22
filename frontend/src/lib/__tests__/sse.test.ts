import { describe, it, expect } from 'vitest'
import { parseSSELine } from '../sse'

describe('parseSSELine', () => {
  it('parses token event', () => {
    const result = parseSSELine('data: {"type":"token","content":"hello"}')
    expect(result).toEqual({ type: 'token', content: 'hello' })
  })

  it('parses step event', () => {
    const result = parseSSELine('data: {"type":"step","step":1,"name":"分析"}')
    expect(result).toEqual({ type: 'step', step: 1, name: '分析' })
  })

  it('parses info event', () => {
    const result = parseSSELine('data: {"type":"info","content":"已提取500字"}')
    expect(result).toEqual({ type: 'info', content: '已提取500字' })
  })

  it('parses outline event', () => {
    const result = parseSSELine('data: {"type":"outline","data":{"title":"标题","sections":[]}}')
    expect(result).toEqual({ type: 'outline', data: { title: '标题', sections: [] } })
  })

  it('parses action event', () => {
    const result = parseSSELine('data: {"type":"action","options":["方案1","方案2","方案3","方案4"]}')
    expect(result).toEqual({ type: 'action', options: ['方案1', '方案2', '方案3', '方案4'] })
  })

  it('parses similarity event', () => {
    const result = parseSSELine('data: {"type":"similarity","data":{"score":15}}')
    expect(result).toEqual({ type: 'similarity', data: { score: 15 } })
  })

  it('parses complete event', () => {
    const result = parseSSELine('data: {"type":"complete","scriptId":42}')
    expect(result).toEqual({ type: 'complete', scriptId: 42 })
  })

  it('parses error event', () => {
    const result = parseSSELine('data: {"type":"error","message":"处理失败"}')
    expect(result).toEqual({ type: 'error', message: '处理失败' })
  })

  it('returns null for empty line', () => {
    expect(parseSSELine('')).toBeNull()
  })

  it('returns null for comment line', () => {
    expect(parseSSELine(': keep-alive')).toBeNull()
  })

  it('returns null for malformed JSON', () => {
    expect(parseSSELine('data: {invalid}')).toBeNull()
  })
})
