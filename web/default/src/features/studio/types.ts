export interface ModelOption {
  label: string
  value: string
}

// Script tab
export interface ScriptRequest {
  model: string
  messages: Array<{ role: 'user' | 'assistant' | 'system'; content: string }>
  stream: boolean
}

// Image tab
export interface ImageRequest {
  model: string
  prompt: string
  n?: number
  size?: string
}

export interface ImageResponse {
  data: Array<{ url?: string; b64_json?: string }>
}

// Voice tab
export interface VoiceRequest {
  model: string
  input: string
  voice?: string
}
