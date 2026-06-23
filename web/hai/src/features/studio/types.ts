export interface ModelOption {
  label: string
  value: string
}

export type StudioModelType = 'script' | 'image' | 'voice' | 'video'

export interface StudioModelConfig {
  id: number
  name: string
  model_name: string
  model_type: StudioModelType
  description?: string
  capability?: string
  default_params?: string
  visible_groups?: string
  status: number
}

export interface StudioFormSchemaRecord {
  id: number
  name: string
  model_type: StudioModelType
  model_name?: string
  version: number
  schema: string
  description?: string
  status: number
}

export interface StudioFormFieldOption {
  label: string
  value: string
}

export interface StudioFormField {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'switch' | 'image_upload' | 'asset_uri' | 'media_upload'
  required?: boolean
  placeholder?: string
  helpText?: string
  min?: number
  max?: number
  step?: number
  options?: StudioFormFieldOption[]
  defaultValue?: string | number | boolean | string[]
}

export type StudioFormValue = string | number | boolean | string[]

export interface StudioFormSchema {
  name: string
  modelType: StudioModelType
  modelName?: string
  /**
   * Controls which API endpoint the image tab uses for generation.
   * - "image_generations" (default): POST /v1/images/generations  — OpenAI image format
   * - "chat_completions":           POST /v1/chat/completions     — for Gemini Flash / models
   *                                   that output images via generateContent
   */
  apiType?: 'image_generations' | 'chat_completions'
  fields: StudioFormField[]
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
  metadata?: {
    imageUrls?: string[]
    aspectRatio?: string
    resolution?: string
    [key: string]: unknown
  }
}

export interface ImageResponse {
  data: Array<{ url?: string; b64_json?: string }>
}

// Voice tab
export interface VoiceRequest {
  model: string
  input: string
  voice?: string
  speed?: number
  response_format?: string
  metadata?: {
    voice_id?: string
    speed?: number
    volume?: number
    pitch?: number
    emotion?: string
    enable_base64_output?: boolean
    english_normalization?: boolean
    [key: string]: unknown
  }
}

export interface UserAsset {
  id: number
  user_id: number
  name: string
  asset_type: 'Image' | 'Video'
  file_name: string
  file_size: number
  content_type: string
  source_url: string
  ark_asset_id?: string
  ark_asset_uri?: string
  ark_status?: string // Processing | Active | Failed | ''
  created_time: number
}

export interface UserAssetListResponse {
  items: UserAsset[]
  total: number
  page: number
  page_size: number
}
