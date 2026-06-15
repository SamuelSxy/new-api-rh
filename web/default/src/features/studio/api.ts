import { api } from '@/lib/api'
import { API_ENDPOINTS } from './constants'
import {
  buildInitialFormValues,
  getDefaultStudioSchema,
  normalizeSchemaPayload,
} from './schema'
import type {
  ModelOption,
  ImageRequest,
  ImageResponse,
  VoiceRequest,
  StudioFormSchema,
  StudioFormSchemaRecord,
  StudioModelConfig,
  StudioModelType,
  StudioFormValue,
  UserAsset,
  UserAssetListResponse,
} from './types'

export async function getUserModels(): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS)
  const { data } = res
  if (!data.success || !Array.isArray(data.data)) {
    return []
  }
  return data.data.map((model: string) => ({ label: model, value: model }))
}

export async function getStudioModels(modelType: StudioModelType): Promise<ModelOption[]> {
  try {
    const res = await api.get(API_ENDPOINTS.STUDIO_MODELS, {
      params: {
        model_type: modelType,
      },
    })
    const { data } = res
    if (!data.success || !Array.isArray(data.data)) {
      return []
    }
    const configs = data.data as StudioModelConfig[]
    return configs
      .filter((item) => item.model_name)
      .map((item) => ({ label: item.name || item.model_name, value: item.model_name }))
  } catch {
    return []
  }
}

export async function getStudioFormSchema(
  modelType: StudioModelType,
  modelName: string
): Promise<{ schema: StudioFormSchema; initialValues: Record<string, StudioFormValue> }> {
  const fallbackSchema = getDefaultStudioSchema(modelType)
  try {
    const res = await api.get(API_ENDPOINTS.STUDIO_FORM_SCHEMAS, {
      params: {
        model_type: modelType,
        model_name: modelName,
      },
    })
    const { data } = res
    if (!data.success || !Array.isArray(data.data) || data.data.length === 0) {
      return {
        schema: fallbackSchema,
        initialValues: buildInitialFormValues(fallbackSchema),
      }
    }

    const firstSchema = data.data[0] as { schema?: string }
    const parsed = firstSchema.schema ? JSON.parse(firstSchema.schema) : null
    const normalizedSchema = normalizeSchemaPayload(parsed, fallbackSchema)
    return {
      schema: normalizedSchema,
      initialValues: buildInitialFormValues(normalizedSchema),
    }
  } catch {
    return {
      schema: fallbackSchema,
      initialValues: buildInitialFormValues(fallbackSchema),
    }
  }
}

export async function getStudioAdminModels(
  modelType: StudioModelType
): Promise<StudioModelConfig[]> {
  const res = await api.get(API_ENDPOINTS.STUDIO_ADMIN_MODELS, {
    params: {
      model_type: modelType,
    },
  })
  const { data } = res
  if (!data.success || !Array.isArray(data.data)) {
    return []
  }
  return data.data as StudioModelConfig[]
}

export async function createStudioAdminModel(
  payload: Omit<StudioModelConfig, 'id'>
): Promise<StudioModelConfig | null> {
  const res = await api.post(API_ENDPOINTS.STUDIO_ADMIN_MODELS, payload)
  const { data } = res
  if (!data.success || !data.data) {
    return null
  }
  return data.data as StudioModelConfig
}

export async function updateStudioAdminModel(
  payload: StudioModelConfig
): Promise<StudioModelConfig | null> {
  const res = await api.put(API_ENDPOINTS.STUDIO_ADMIN_MODELS, payload)
  const { data } = res
  if (!data.success || !data.data) {
    return null
  }
  return data.data as StudioModelConfig
}

export async function deleteStudioAdminModel(id: number): Promise<boolean> {
  const res = await api.delete(`${API_ENDPOINTS.STUDIO_ADMIN_MODELS}/${id}`)
  const { data } = res
  return Boolean(data.success)
}

export async function getStudioAdminFormSchemas(
  modelType: StudioModelType
): Promise<StudioFormSchemaRecord[]> {
  const res = await api.get(API_ENDPOINTS.STUDIO_ADMIN_FORM_SCHEMAS, {
    params: {
      model_type: modelType,
    },
  })
  const { data } = res
  if (!data.success || !Array.isArray(data.data)) {
    return []
  }
  return data.data as StudioFormSchemaRecord[]
}

export async function createStudioAdminFormSchema(payload: {
  name: string
  model_type: StudioModelType
  model_name?: string
  version: number
  schema: string
  description?: string
  status: number
}): Promise<StudioFormSchemaRecord | null> {
  const res = await api.post(API_ENDPOINTS.STUDIO_ADMIN_FORM_SCHEMAS, payload)
  const { data } = res
  if (!data.success || !data.data) {
    return null
  }
  return data.data as StudioFormSchemaRecord
}

export async function updateStudioAdminFormSchema(payload: {
  id: number
  name: string
  model_type: StudioModelType
  model_name?: string
  version: number
  schema: string
  description?: string
  status: number
}): Promise<StudioFormSchemaRecord | null> {
  const res = await api.put(API_ENDPOINTS.STUDIO_ADMIN_FORM_SCHEMAS, payload)
  const { data } = res
  if (!data.success || !data.data) {
    return null
  }
  return data.data as StudioFormSchemaRecord
}

export async function deleteStudioAdminFormSchema(id: number): Promise<boolean> {
  const res = await api.delete(`${API_ENDPOINTS.STUDIO_ADMIN_FORM_SCHEMAS}/${id}`)
  const { data } = res
  return Boolean(data.success)
}

export async function generateImage(payload: ImageRequest): Promise<ImageResponse> {
  const res = await api.post(API_ENDPOINTS.IMAGES_GENERATIONS, payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function generateVoice(
  payload: VoiceRequest
): Promise<Blob | Record<string, unknown>> {
  const res = await api.post(API_ENDPOINTS.AUDIO_SPEECH, payload, {
    responseType: 'blob',
    skipErrorHandler: true,
  } as Record<string, unknown>)

  const blob = res.data as Blob
  const rawContentType =
    ((res.headers as Record<string, unknown>)?.['content-type'] as string | undefined) ||
    blob.type ||
    ''
  const contentType = rawContentType.toLowerCase()

  if (contentType.includes('application/json')) {
    const text = await blob.text()
    try {
      const parsed = JSON.parse(text)
      if (parsed && typeof parsed === 'object') {
        return parsed as Record<string, unknown>
      }
    } catch {
      // Fall back to blob for non-standard upstream payloads.
    }
  }

  return blob
}

export async function submitVideoTask(
  payload: Record<string, unknown>
): Promise<unknown> {
  const res = await api.post(API_ENDPOINTS.VIDEO_GENERATIONS, payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function fetchVideoTask(taskId: string): Promise<unknown> {
  const res = await api.get(`${API_ENDPOINTS.VIDEO_GENERATIONS}/${taskId}`, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function fetchUserTaskById(taskId: string): Promise<unknown> {
  const res = await api.get(API_ENDPOINTS.TASK_SELF, {
    params: {
      task_id: taskId,
      page_size: 1,
    },
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function uploadUserAsset(
  file: File,
  name: string,
  assetType: 'Image' | 'Video'
): Promise<UserAsset | null> {
  const form = new FormData()
  form.append('file', file)
  form.append('name', name)
  form.append('asset_type', assetType)
  const res = await api.post(API_ENDPOINTS.STUDIO_ASSETS, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  const { data } = res
  if (!data.success || !data.data) return null
  return data.data as UserAsset
}

export async function uploadUserAssetByUrl(
  url: string,
  name: string,
  assetType: 'Image' | 'Video'
): Promise<UserAsset | null> {
  const form = new FormData()
  form.append('url', url)
  form.append('name', name || url.split('/').pop() || 'asset')
  form.append('asset_type', assetType)
  const res = await api.post(API_ENDPOINTS.STUDIO_ASSETS, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  const { data } = res
  if (!data.success || !data.data) return null
  return data.data as UserAsset
}

export async function listUserAssets(
  assetType?: 'Image' | 'Video',
  page = 1,
  pageSize = 40
): Promise<UserAssetListResponse> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (assetType) params['asset_type'] = assetType
  const res = await api.get(API_ENDPOINTS.STUDIO_ASSETS, { params })
  const { data } = res
  if (!data.success || !data.data) return { items: [], total: 0, page: 1, page_size: pageSize }
  return data.data as UserAssetListResponse
}

export async function deleteUserAsset(id: number): Promise<boolean> {
  const res = await api.delete(`${API_ENDPOINTS.STUDIO_ASSETS}/${id}`)
  const { data } = res
  return Boolean(data.success)
}

export async function syncAssetArkStatus(id: number): Promise<UserAsset | null> {
  try {
    const res = await api.get(`${API_ENDPOINTS.STUDIO_ASSETS}/${id}/ark-status`)
    const { data } = res
    if (data.success && data.data) {
      return data.data as UserAsset
    }
  } catch {
    // ignore
  }
  return null
}
