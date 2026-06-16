import { STUDIO_TABS } from './constants'
import type {
  StudioFormSchema,
  StudioModelType,
  StudioFormField,
  StudioFormValue,
} from './types'

const defaultStudioSchemas: Record<StudioModelType, StudioFormSchema> = {
  [STUDIO_TABS.SCRIPT]: {
    name: 'Script Default Schema',
    modelType: STUDIO_TABS.SCRIPT,
    fields: [
      {
        key: 'temperature',
        label: 'Temperature',
        type: 'number',
        min: 0,
        max: 2,
        step: 0.1,
        defaultValue: 0.7,
      },
      {
        key: 'max_tokens',
        label: 'Max Tokens',
        type: 'number',
        min: 1,
        max: 32000,
        step: 1,
        defaultValue: 1024,
      },
    ],
  },
  [STUDIO_TABS.IMAGE]: {
    name: 'Image Default Schema',
    modelType: STUDIO_TABS.IMAGE,
    fields: [
      {
        key: 'image_urls',
        label: 'Reference Images',
        type: 'image_upload',
        helpText: 'Upload one or more reference images',
      },
      {
        key: 'aspect_ratio',
        label: 'Aspect Ratio',
        type: 'select',
        options: [
          { label: '1:1', value: '1:1' },
          { label: '4:3', value: '4:3' },
          { label: '3:4', value: '3:4' },
          { label: '16:9', value: '16:9' },
          { label: '9:16', value: '9:16' },
        ],
        defaultValue: '9:16',
      },
      {
        key: 'resolution',
        label: 'Resolution',
        type: 'select',
        options: [
          { label: '1k', value: '1k' },
          { label: '2k', value: '2k' },
          { label: '4k', value: '4k' },
        ],
        defaultValue: '1k',
      },
    ],
  },
  [STUDIO_TABS.VOICE]: {
    name: 'Voice Default Schema',
    modelType: STUDIO_TABS.VOICE,
    fields: [
      {
        key: 'voice',
        label: 'Voice',
        type: 'text',
        placeholder: 'Elegant_Man',
        defaultValue: 'Elegant_Man',
      },
      {
        key: 'speed',
        label: 'Speed',
        type: 'number',
        min: 0.25,
        max: 4,
        step: 0.05,
        defaultValue: 1,
      },
      {
        key: 'volume',
        label: 'Volume',
        type: 'number',
        min: 0,
        max: 2,
        step: 0.1,
        defaultValue: 1,
      },
      {
        key: 'pitch',
        label: 'Pitch',
        type: 'number',
        min: -12,
        max: 12,
        step: 1,
        defaultValue: 0,
      },
      {
        key: 'emotion',
        label: 'Emotion',
        type: 'text',
        placeholder: 'happy',
        defaultValue: 'happy',
      },
      {
        key: 'response_format',
        label: 'Response Format',
        type: 'select',
        options: [
          { label: 'mp3', value: 'mp3' },
          { label: 'wav', value: 'wav' },
          { label: 'pcm', value: 'pcm' },
        ],
        defaultValue: 'mp3',
      },
      {
        key: 'enable_base64_output',
        label: 'Enable Base64 Output',
        type: 'switch',
        defaultValue: false,
      },
      {
        key: 'english_normalization',
        label: 'English Normalization',
        type: 'switch',
        defaultValue: false,
      },
    ],
  },
  [STUDIO_TABS.VIDEO]: {
    name: 'Video Default Schema',
    modelType: STUDIO_TABS.VIDEO,
    fields: [
      {
        key: 'duration',
        label: 'Duration (s)',
        type: 'number',
        min: 1,
        max: 60,
        step: 1,
        defaultValue: 8,
      },
      {
        key: 'resolution',
        label: 'Resolution',
        type: 'select',
        options: [
          { label: '720p', value: '1280x720' },
          { label: '1080p', value: '1920x1080' },
        ],
        defaultValue: '1280x720',
      },
    ],
  },
}

export function getDefaultStudioSchema(modelType: StudioModelType): StudioFormSchema {
  return defaultStudioSchemas[modelType]
}

export function buildInitialFormValues(schema: StudioFormSchema): Record<string, StudioFormValue> {
  return schema.fields.reduce<Record<string, StudioFormValue>>(
    (acc, field) => {
      if (field.defaultValue !== undefined) {
        acc[field.key] = field.defaultValue
        return acc
      }
      if (field.type === 'switch') {
        acc[field.key] = false
        return acc
      }
      if (field.type === 'image_upload' || field.type === 'media_upload') {
        acc[field.key] = []
        return acc
      }
      acc[field.key] = ''
      return acc
    },
    {}
  )
}

export function normalizeSchemaPayload(
  payload: unknown,
  fallback: StudioFormSchema
): StudioFormSchema {
  if (!payload || typeof payload !== 'object') {
    return fallback
  }

  const candidate = payload as Partial<StudioFormSchema>
  if (!Array.isArray(candidate.fields)) {
    return fallback
  }

  const fields = candidate.fields.filter(isValidField)
  if (fields.length === 0) {
    return fallback
  }

  return {
    name: candidate.name || fallback.name,
    modelType: candidate.modelType || fallback.modelType,
    modelName: candidate.modelName || fallback.modelName,
    ...(candidate.apiType === 'chat_completions' || candidate.apiType === 'image_generations'
      ? { apiType: candidate.apiType }
      : {}),
    fields,
  }
}

function isValidField(field: unknown): field is StudioFormField {
  if (!field || typeof field !== 'object') {
    return false
  }
  const candidate = field as Partial<StudioFormField>
  if (!candidate.key || !candidate.label || !candidate.type) {
    return false
  }
  return ['text', 'textarea', 'number', 'select', 'switch', 'image_upload', 'media_upload', 'asset_uri'].includes(candidate.type)
}
