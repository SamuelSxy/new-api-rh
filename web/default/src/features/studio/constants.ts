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
  VIDEO_GENERATIONS: '/v1/video/generations',
  USER_MODELS: '/api/user/models',
  STUDIO_MODELS: '/api/studio/models',
  STUDIO_FORM_SCHEMAS: '/api/studio/form-schemas',
  STUDIO_ADMIN_MODELS: '/api/studio/admin/models',
  STUDIO_ADMIN_FORM_SCHEMAS: '/api/studio/admin/form-schemas',
  TASK_SELF: '/api/task/self',
} as const
