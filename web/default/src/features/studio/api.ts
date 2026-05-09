import { api } from '@/lib/api'
import { API_ENDPOINTS } from './constants'
import type { ModelOption, ImageRequest, ImageResponse, VoiceRequest } from './types'

export async function getUserModels(): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS)
  const { data } = res
  if (!data.success || !Array.isArray(data.data)) {
    return []
  }
  return data.data.map((model: string) => ({ label: model, value: model }))
}

export async function generateImage(payload: ImageRequest): Promise<ImageResponse> {
  const res = await api.post(API_ENDPOINTS.IMAGES_GENERATIONS, payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function generateVoice(payload: VoiceRequest): Promise<Blob> {
  const res = await api.post(API_ENDPOINTS.AUDIO_SPEECH, payload, {
    responseType: 'blob',
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data as Blob
}
