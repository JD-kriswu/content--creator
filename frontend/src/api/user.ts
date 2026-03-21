import request from './request'

export interface UserStyle {
  language_style: string
  emotion_tone: string
  opening_style: string
  closing_style: string
  catchphrases: string
}

export function getProfile() {
  return request.get<{ style: UserStyle | null }>('/user/profile')
}

export function updateStyle(style: UserStyle) {
  return request.put('/user/style', style)
}
