export const STUDIO_TABS = {
  SCRIPT: 'script',
  IMAGE: 'image',
  VOICE: 'voice',
  VIDEO: 'video',
} as const

export type StudioTab = (typeof STUDIO_TABS)[keyof typeof STUDIO_TABS]

export const API_ENDPOINTS = {
  CHAT_COMPLETIONS: '/pg/chat/completions',
  IMAGES_GENERATIONS: '/v1/images/generations',
  AUDIO_SPEECH: '/v1/audio/speech',
  USER_MODELS: '/api/user/models',
} as const
