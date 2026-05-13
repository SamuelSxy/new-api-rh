import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getModels } from '@/features/models/api'
import { STUDIO_TABS } from '../constants'
import {
  createStudioAdminFormSchema,
  createStudioAdminModel,
  deleteStudioAdminFormSchema,
  deleteStudioAdminModel,
  getStudioAdminFormSchemas,
  getStudioAdminModels,
  updateStudioAdminFormSchema,
  updateStudioAdminModel,
} from '../api'
import type {
  ModelOption,
  StudioFormSchemaRecord,
  StudioModelConfig,
  StudioModelType,
} from '../types'

type ModelDraft = {
  id?: number
  name: string
  model_name: string
  description: string
  status: number
}

type StudioPreset = {
  name: string
  modelName: string
  description: string
  schemaName: string
  schema: {
    name: string
    modelType: StudioModelType
    modelName: string
    fields: Array<Record<string, unknown>>
  }
}

const imagePresets: StudioPreset[] = [
  {
    name: 'RunningHub Image Edit',
    modelName: 'rhart-image-n-pro-official/edit',
    description: 'RunningHub image-to-image editing workflow',
    schemaName: 'RunningHub Image Edit Form',
    schema: {
      name: 'RunningHub Image Edit Form',
      modelType: 'image',
      modelName: 'rhart-image-n-pro-official/edit',
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
  },
]

const voicePresets: StudioPreset[] = [
  {
    name: 'RunningHub Speech 2.8 Turbo',
    modelName: 'rhart-audio/text-to-audio/speech-2.8-turbo',
    description: 'RunningHub text-to-audio speech workflow',
    schemaName: 'RunningHub Speech 2.8 Turbo Form',
    schema: {
      name: 'RunningHub Speech 2.8 Turbo Form',
      modelType: 'voice',
      modelName: 'rhart-audio/text-to-audio/speech-2.8-turbo',
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
  },
]

const videoPresets: StudioPreset[] = [
  {
    name: 'Seedance 2.0 Fast',
    modelName: 'seedance2.0-fast',
    description: 'Fast video generation',
    schemaName: 'Seedance 2.0 Fast Video Form',
    schema: {
      name: 'Seedance 2.0 Fast Video Form',
      modelType: 'video',
      modelName: 'seedance2.0-fast',
      fields: [
        {
          key: 'resolution',
          label: 'Resolution',
          type: 'select',
          required: true,
          options: [
            { label: '480p', value: '480' },
            { label: '720p', value: '720' },
          ],
          defaultValue: '720',
        },
        {
          key: 'image_urls',
          label: 'Images (Local upload, multiple)',
          type: 'image_upload',
          required: true,
          max: 10,
          helpText: 'Images will be converted to base64 data URLs automatically',
        },
        {
          key: 'video_url',
          label: 'Reference Video URL',
          type: 'text',
          placeholder: 'https://example.com/reference.mp4',
        },
        {
          key: 'duration',
          label: 'Duration (s)',
          type: 'number',
          required: true,
          min: 4,
          max: 15,
          step: 1,
          defaultValue: 8,
        },
      ],
    },
  },
  {
    name: 'Seedance 2.0',
    modelName: 'seedance2.0',
    description: 'High quality video generation',
    schemaName: 'Seedance 2.0 Video Form',
    schema: {
      name: 'Seedance 2.0 Video Form',
      modelType: 'video',
      modelName: 'seedance2.0',
      fields: [
        {
          key: 'resolution',
          label: 'Resolution',
          type: 'select',
          required: true,
          options: [
            { label: '480p', value: '480' },
            { label: '720p', value: '720' },
            { label: '1080p', value: '1080' },
          ],
          defaultValue: '1080',
        },
        {
          key: 'image_urls',
          label: 'Images (Local upload, multiple)',
          type: 'image_upload',
          required: true,
          max: 10,
          helpText: 'Images will be converted to base64 data URLs automatically',
        },
        {
          key: 'video_url',
          label: 'Reference Video URL',
          type: 'text',
          placeholder: 'https://example.com/reference.mp4',
        },
        {
          key: 'duration',
          label: 'Duration (s)',
          type: 'number',
          required: true,
          min: 4,
          max: 15,
          step: 1,
          defaultValue: 8,
        },
      ],
    },
  },
  {
    name: 'Happyhorse',
    modelName: 'happyhorse',
    description: 'Creative short video generation',
    schemaName: 'Happyhorse Video Form',
    schema: {
      name: 'Happyhorse Video Form',
      modelType: 'video',
      modelName: 'happyhorse',
      fields: [
        {
          key: 'resolution',
          label: 'Resolution',
          type: 'select',
          required: true,
          options: [
            { label: '480p', value: '480' },
            { label: '720p', value: '720' },
            { label: '1080p', value: '1080' },
          ],
          defaultValue: '720',
        },
        {
          key: 'image_urls',
          label: 'Images (Local upload, multiple)',
          type: 'image_upload',
          required: true,
          max: 8,
          helpText: 'Images will be converted to base64 data URLs automatically',
        },
        {
          key: 'duration',
          label: 'Duration (s)',
          type: 'number',
          required: true,
          min: 3,
          max: 15,
          step: 1,
          defaultValue: 6,
        },
      ],
    },
  },
] as const

export function StudioAdminPanel() {
  const { t } = useTranslation()
  const [managedType, setManagedType] = useState<StudioModelType>(STUDIO_TABS.SCRIPT)
  const [loading, setLoading] = useState(false)
  const [modelList, setModelList] = useState<StudioModelConfig[]>([])
  const [schemaList, setSchemaList] = useState<StudioFormSchemaRecord[]>([])
  const [configuredModels, setConfiguredModels] = useState<ModelOption[]>([])
  const [selectedModelName, setSelectedModelName] = useState('')
  const [modelDraft, setModelDraft] = useState<ModelDraft>({
    name: '',
    model_name: '',
    description: '',
    status: 1,
  })
  const [schemaId, setSchemaId] = useState<number | undefined>(undefined)
  const [schemaDraft, setSchemaDraft] = useState('')

  const activeSchema = useMemo(() => {
    return schemaList.find((item) => item.model_name === modelDraft.model_name)
  }, [schemaList, modelDraft.model_name])

  const configuredModelSet = useMemo(() => {
    return new Set(configuredModels.map((item) => item.value))
  }, [configuredModels])

  const resetDraftState = useCallback(() => {
    setSelectedModelName('')
    setModelDraft({
      name: '',
      model_name: '',
      description: '',
      status: 1,
    })
    setSchemaId(undefined)
    setSchemaDraft('')
  }, [])

  const hydrateDraftByModelName = useCallback(
    (
      modelName: string,
      models: StudioModelConfig[],
      schemas: StudioFormSchemaRecord[]
    ) => {
      setSelectedModelName(modelName)
      const matchedModel = models.find((item) => item.model_name === modelName)
      if (matchedModel) {
        setModelDraft({
          id: matchedModel.id,
          name: matchedModel.name,
          model_name: matchedModel.model_name,
          description: matchedModel.description || '',
          status: matchedModel.status,
        })
      } else {
        setModelDraft((prev) => ({
          ...prev,
          id: undefined,
          model_name: modelName,
        }))
      }

      const matchedSchema = schemas.find((item) => item.model_name === modelName)
      if (matchedSchema) {
        setSchemaId(matchedSchema.id)
        setSchemaDraft(matchedSchema.schema)
        return
      }
      setSchemaId(undefined)
      setSchemaDraft('')
    },
    []
  )

  const refreshData = useCallback(
    async (preferredModelName?: string) => {
    setLoading(true)
    try {
      const [models, schemas, modelResponse] = await Promise.all([
        getStudioAdminModels(managedType),
        getStudioAdminFormSchemas(managedType),
        getModels({ page_size: 1000 }),
      ])

      const configured = Array.from(
        new Set(
          (modelResponse.data?.items || [])
            .map((item) => item.model_name)
            .filter((item): item is string => Boolean(item))
        )
      )
        .sort((a, b) => a.localeCompare(b))
        .map((name) => ({ label: name, value: name }))

      setConfiguredModels(configured)
      setModelList(models)
      setSchemaList(schemas)

      const nextName =
        preferredModelName &&
        models.some((item) => item.model_name === preferredModelName)
          ? preferredModelName
          : models[0]?.model_name

      if (nextName) {
        hydrateDraftByModelName(nextName, models, schemas)
        return
      }

      resetDraftState()
    } finally {
      setLoading(false)
    }
    },
    [hydrateDraftByModelName, managedType, resetDraftState]
  )

  const handleSelectModel = (modelName: string) => {
    hydrateDraftByModelName(modelName, modelList, schemaList)
  }

  useEffect(() => {
    let mounted = true
    resetDraftState()
    void refreshData().catch(() => {
      if (!mounted) {
        return
      }
      toast.error(t('Failed to load Studio management data'))
    })
    return () => {
      mounted = false
    }
  }, [managedType, refreshData, resetDraftState, t])

  const handleSaveModel = async () => {
    if (!modelDraft.name || !modelDraft.model_name) {
      toast.error(t('Model name and model key are required'))
      return
    }
    if (!configuredModelSet.has(modelDraft.model_name)) {
      toast.error(t('Model key must be selected from configured models'))
      return
    }
    setLoading(true)
    try {
      if (modelDraft.id) {
        await updateStudioAdminModel({
          id: modelDraft.id,
          name: modelDraft.name,
          model_name: modelDraft.model_name,
          model_type: managedType,
          description: modelDraft.description,
          capability: '',
          default_params: '',
          visible_groups: '',
          status: modelDraft.status,
        })
      } else {
        await createStudioAdminModel({
          name: modelDraft.name,
          model_name: modelDraft.model_name,
          model_type: managedType,
          description: modelDraft.description,
          capability: '',
          default_params: '',
          visible_groups: '',
          status: modelDraft.status,
        })
      }
      toast.success(t('Model saved'))
      await refreshData(modelDraft.model_name)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveSchema = async () => {
    if (!modelDraft.model_name || !schemaDraft.trim()) {
      toast.error(t('Please select a model and provide schema JSON'))
      return
    }
    if (!configuredModelSet.has(modelDraft.model_name)) {
      toast.error(t('Model key must be selected from configured models'))
      return
    }

    try {
      JSON.parse(schemaDraft)
    } catch {
      toast.error(t('Schema JSON is invalid'))
      return
    }

    setLoading(true)
    try {
      const typeLabel =
        managedType.charAt(0).toUpperCase() + managedType.slice(1)
      const payload = {
        name: `${modelDraft.model_name} ${typeLabel} Form`,
        model_type: managedType,
        model_name: modelDraft.model_name,
        version: activeSchema?.version || 1,
        schema: schemaDraft,
        description: `${typeLabel} generation form for ${modelDraft.model_name}`,
        status: 1,
      }

      if (schemaId) {
        await updateStudioAdminFormSchema({
          id: schemaId,
          ...payload,
        })
      } else {
        await createStudioAdminFormSchema(payload)
      }
      toast.success(t('Form schema saved'))
      await refreshData(modelDraft.model_name)
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteModel = async () => {
    if (!modelDraft.id) {
      toast.error(t('Please select an existing model first'))
      return
    }

    const confirmed = window.confirm(
      t('Delete current model configuration? This only removes Studio config.')
    )
    if (!confirmed) {
      return
    }

    setLoading(true)
    try {
      const ok = await deleteStudioAdminModel(modelDraft.id)
      if (!ok) {
        toast.error(t('Delete model failed'))
        return
      }
      toast.success(t('Model deleted'))
      await refreshData()
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteSchema = async () => {
    if (!schemaId) {
      toast.error(t('Please select an existing schema first'))
      return
    }

    const confirmed = window.confirm(
      t('Delete current form schema configuration?')
    )
    if (!confirmed) {
      return
    }

    setLoading(true)
    try {
      const ok = await deleteStudioAdminFormSchema(schemaId)
      if (!ok) {
        toast.error(t('Delete form schema failed'))
        return
      }
      toast.success(t('Form schema deleted'))
      await refreshData(modelDraft.model_name)
    } finally {
      setLoading(false)
    }
  }

    const getPresetsByType = (type: StudioModelType): StudioPreset[] => {
      switch (type) {
        case STUDIO_TABS.IMAGE:
          return imagePresets
        case STUDIO_TABS.VOICE:
          return voicePresets
        case STUDIO_TABS.VIDEO:
          return videoPresets
        default:
          return []
      }
    }

  const handleInitPresets = async () => {
      const presets = getPresetsByType(managedType)
      if (presets.length === 0) {
        toast.info(t('No presets for this type'))
        return
      }

    setLoading(true)
    try {
      const [models, schemas] = await Promise.all([
          getStudioAdminModels(managedType),
          getStudioAdminFormSchemas(managedType),
      ])

        for (const preset of presets) {
        const existingModel = models.find(
          (item) => item.model_name === preset.modelName
        )

        if (existingModel) {
          await updateStudioAdminModel({
            ...existingModel,
            name: preset.name,
            model_name: preset.modelName,
            model_type: managedType,
            description: preset.description,
            capability: existingModel.capability || '',
            default_params: existingModel.default_params || '',
            visible_groups: existingModel.visible_groups || '',
            status: 1,
          })
        } else {
          await createStudioAdminModel({
            name: preset.name,
            model_name: preset.modelName,
            model_type: managedType,
            description: preset.description,
            capability: '',
            default_params: '',
            visible_groups: '',
            status: 1,
          })
        }

        const existingSchema = schemas.find(
          (item) => item.model_name === preset.modelName
        )
        const schemaText = JSON.stringify(preset.schema, null, 2)

        if (existingSchema) {
          await updateStudioAdminFormSchema({
            id: existingSchema.id,
            name: preset.schemaName,
            model_type: managedType,
            model_name: preset.modelName,
            version: existingSchema.version || 1,
            schema: schemaText,
            description: `${typeLabel} generation form for ${preset.modelName}`,
            status: 1,
          })
        } else {
          await createStudioAdminFormSchema({
            name: preset.schemaName,
            model_type: managedType,
            model_name: preset.modelName,
            version: 1,
            schema: schemaText,
            description: `${typeLabel} generation form for ${preset.modelName}`,
            status: 1,
          })
        }
      }

      toast.success(
        t('{{type}} model presets initialized', {
          type: typeLabel,
        })
      )
      await refreshData(modelDraft.model_name)
    } finally {
      setLoading(false)
    }
  }

  const typeLabel =
    managedType.charAt(0).toUpperCase() + managedType.slice(1)
  const modelDraftInvalid =
    Boolean(modelDraft.model_name) &&
    !configuredModelSet.has(modelDraft.model_name)

  return (
    <div className='rounded-2xl border bg-muted/10 p-4'>
      <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
        <div>
          <h2 className='text-base font-semibold'>
            {t('Studio Model Form Management')}
          </h2>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Configure script/image/voice/video models and dynamic form schema for Studio generation'
            )}
          </p>
        </div>
        <div className='flex flex-wrap gap-2'>
          <Select
            value={managedType}
            onValueChange={(value) => setManagedType(value as StudioModelType)}
          >
            <SelectTrigger className='w-36'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={STUDIO_TABS.SCRIPT}>{t('Script')}</SelectItem>
              <SelectItem value={STUDIO_TABS.IMAGE}>{t('Image')}</SelectItem>
              <SelectItem value={STUDIO_TABS.VOICE}>{t('Voice')}</SelectItem>
              <SelectItem value={STUDIO_TABS.VIDEO}>{t('Video')}</SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant='outline'
            onClick={() => {
              void refreshData()
            }}
            disabled={loading}
          >
            {t('Refresh')}
          </Button>
          {managedType !== STUDIO_TABS.SCRIPT && (
            <Button onClick={() => void handleInitPresets()} disabled={loading}>
              {t('Initialize {{type}} Presets', { type: typeLabel })}
            </Button>
          )}
        </div>
      </div>

      <div className='grid gap-4 md:grid-cols-2'>
        <div className='space-y-3 rounded-xl border bg-background p-3'>
          <div className='flex items-center justify-between gap-2'>
            <Label>{t('Configured Studio Model')}</Label>
            <Button
              variant='ghost'
              size='sm'
              onClick={resetDraftState}
              disabled={loading}
            >
              {t('New')}
            </Button>
          </div>
          <Select value={selectedModelName} onValueChange={handleSelectModel}>
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('Select configured studio model')} />
            </SelectTrigger>
            <SelectContent>
              {modelList.map((item) => (
                <SelectItem key={item.id} value={item.model_name}>
                  {item.name} ({item.model_name})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {modelList.length === 0 && (
            <p className='text-muted-foreground text-xs'>
              {t('No studio model for current type yet. Create one below.')}
            </p>
          )}

          <div className='grid gap-2'>
            <Label>{t('Display Name')}</Label>
            <Input
              value={modelDraft.name}
              onChange={(event) =>
                setModelDraft((prev) => ({ ...prev, name: event.target.value }))
              }
              placeholder='Seedance 2.0 Fast'
            />
          </div>

          <div className='grid gap-2'>
            <Label>{t('Model Key')}</Label>
            <Select
              value={modelDraft.model_name}
              onValueChange={(value) => {
                setModelDraft((prev) => ({ ...prev, model_name: value }))
                const linkedSchema = schemaList.find((item) => item.model_name === value)
                if (linkedSchema) {
                  setSchemaId(linkedSchema.id)
                  setSchemaDraft(linkedSchema.schema)
                  return
                }
                setSchemaId(undefined)
                setSchemaDraft('')
              }}
            >
              <SelectTrigger className='w-full'>
                <SelectValue
                  placeholder={
                    configuredModels.length > 0
                      ? t('Select model key from configured models')
                      : t('No configured models, please configure models first')
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {configuredModels.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {modelDraftInvalid && (
              <p className='text-destructive text-xs'>
                {t('Model key must be selected from configured models')}
              </p>
            )}
            {configuredModels.length === 0 && (
              <p className='text-muted-foreground text-xs'>
                {t('No configured models, please configure models first')}
              </p>
            )}
          </div>

          <div className='grid gap-2'>
            <Label>{t('Description')}</Label>
            <Input
              value={modelDraft.description}
              onChange={(event) =>
                setModelDraft((prev) => ({ ...prev, description: event.target.value }))
              }
            />
          </div>

          <div className='flex flex-wrap gap-2'>
            <Button
              onClick={() => void handleSaveModel()}
              disabled={loading || configuredModels.length === 0}
            >
              {t('Save {{type}} Model', { type: typeLabel })}
            </Button>
            <Button
              variant='destructive'
              onClick={() => void handleDeleteModel()}
              disabled={loading || !modelDraft.id}
            >
              {t('Delete Model')}
            </Button>
          </div>
        </div>

        <div className='space-y-3 rounded-xl border bg-background p-3'>
          <div className='flex items-center justify-between'>
            <Label>{t('Form Schema JSON')}</Label>
            <span className='text-muted-foreground text-xs'>
              {t('Model')}: {selectedModelName || '-'}
            </span>
          </div>
          <Textarea
            className='h-80 resize-none overflow-auto font-mono text-xs'
            value={schemaDraft}
            onChange={(event) => setSchemaDraft(event.target.value)}
            placeholder='{"fields":[]}'
          />
          <div className='flex flex-wrap gap-2'>
            <Button
              onClick={() => void handleSaveSchema()}
              disabled={loading || configuredModels.length === 0}
            >
              {t('Save Form Schema')}
            </Button>
            <Button
              variant='destructive'
              onClick={() => void handleDeleteSchema()}
              disabled={loading || !schemaId}
            >
              {t('Delete Form Schema')}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
