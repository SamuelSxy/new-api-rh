import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { fetchUserTaskById, generateImage, getStudioFormSchema } from '../api'
import type { ModelOption, StudioFormSchema, StudioFormValue } from '../types'
import { buildInitialFormValues, getDefaultStudioSchema } from '../schema'
import { StudioFormFields } from './studio-form-fields'
import { StudioPromptInput } from './studio-prompt-input'

interface ImageTabProps {
  models: ModelOption[]
}

export function ImageTab({ models }: ImageTabProps) {
  const { t } = useTranslation()
  const modelOptions = useMemo(() => {
    return models
  }, [models])

  const [model, setModel] = useState(
    modelOptions[0]?.value || ''
  )
  const [prompt, setPrompt] = useState('')
  const [imageUrl, setImageUrl] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [schema, setSchema] = useState<StudioFormSchema>(() =>
    getDefaultStudioSchema('image')
  )
  const [formValues, setFormValues] = useState<Record<string, StudioFormValue>>(
    () => buildInitialFormValues(getDefaultStudioSchema('image'))
  )
  const pollTimerRef = useRef<number | null>(null)
  const pollingInFlightRef = useRef(false)

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current !== null) {
      window.clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  useEffect(() => {
    return () => {
      stopPolling()
    }
  }, [stopPolling])

  const parseImageUrlFromResponse = (response: unknown): string => {
    if (!response || typeof response !== 'object') {
      return ''
    }
    const record = response as Record<string, unknown>
    const data = Array.isArray(record.data) ? record.data : []
    if (data.length > 0 && data[0] && typeof data[0] === 'object') {
      const first = data[0] as Record<string, unknown>
      if (typeof first.url === 'string' && first.url) {
        return first.url
      }
      if (typeof first.b64_json === 'string' && first.b64_json) {
        return `data:image/png;base64,${first.b64_json}`
      }
    }
    return ''
  }

  const parseTaskId = (response: unknown): string => {
    if (!response || typeof response !== 'object') {
      return ''
    }
    const record = response as Record<string, unknown>
    const nested =
      record.data && typeof record.data === 'object'
        ? (record.data as Record<string, unknown>)
        : null

    return (
      (typeof record.task_id === 'string' ? record.task_id : '') ||
      (typeof record.taskId === 'string' ? record.taskId : '') ||
      (typeof record.id === 'string' ? record.id : '') ||
      (nested && typeof nested.task_id === 'string' ? nested.task_id : '') ||
      (nested && typeof nested.taskId === 'string' ? nested.taskId : '') ||
      (nested && typeof nested.id === 'string' ? nested.id : '') ||
      ''
    )
  }

  const parseTaskStatus = (task: Record<string, unknown>): string => {
    return typeof task.status === 'string' ? task.status : ''
  }

  const parseImageUrlFromTask = (task: Record<string, unknown>): string => {
    if (typeof task.result_url === 'string' && task.result_url) {
      return task.result_url
    }

    const failReason = typeof task.fail_reason === 'string' ? task.fail_reason : ''
    if (failReason.startsWith('http://') || failReason.startsWith('https://') || failReason.startsWith('data:image/')) {
      return failReason
    }

    const data = task.data && typeof task.data === 'object' ? (task.data as Record<string, unknown>) : null
    const results = data && Array.isArray(data.results) ? data.results : []
    if (results.length > 0 && results[0] && typeof results[0] === 'object') {
      const first = results[0] as Record<string, unknown>
      const url = typeof first.url === 'string' ? first.url : ''
      const fileUrl = typeof first.fileUrl === 'string' ? first.fileUrl : ''
      return url || fileUrl || ''
    }
    return ''
  }

  const isTaskSuccess = (status: string) => status.toUpperCase() === 'SUCCESS'
  const isTaskFailure = (status: string) => status.toUpperCase() === 'FAILURE'

  const startPollingTask = useCallback((nextTaskId: string) => {
    stopPolling()
    pollTimerRef.current = window.setInterval(async () => {
      if (pollingInFlightRef.current) {
        return
      }
      pollingInFlightRef.current = true
      try {
        const taskResp = await fetchUserTaskById(nextTaskId)
        if (!taskResp || typeof taskResp !== 'object') {
          return
        }
        const taskRecord = taskResp as Record<string, unknown>
        const data =
          taskRecord.data && typeof taskRecord.data === 'object'
            ? (taskRecord.data as Record<string, unknown>)
            : null
        const items = data && Array.isArray(data.items) ? data.items : []
        if (items.length === 0 || !items[0] || typeof items[0] !== 'object') {
          return
        }

        const task = items[0] as Record<string, unknown>
        const status = parseTaskStatus(task)
        if (status) {
          setTaskStatus(status)
        }

        const taskImageUrl = parseImageUrlFromTask(task)
        if (taskImageUrl) {
          setImageUrl(taskImageUrl)
        }

        if (status && isTaskSuccess(status)) {
          stopPolling()
          setIsLoading(false)
          if (taskImageUrl) {
            toast.success(t('Image generated successfully'))
          } else {
            setError(t('No image returned'))
          }
          return
        }

        if (status && isTaskFailure(status)) {
          stopPolling()
          setIsLoading(false)
          setError(t('Failed to generate image'))
          toast.error(t('Failed to generate image'))
        }
      } catch {
        stopPolling()
        setIsLoading(false)
        setError(t('Failed to query image task status'))
        toast.error(t('Failed to query image task status'))
      } finally {
        pollingInFlightRef.current = false
      }
    }, 2500)
  }, [stopPolling, t])

  useEffect(() => {
    if (!model && modelOptions.length > 0) {
      setModel(modelOptions[0].value)
      return
    }

    if (model && !modelOptions.some((item) => item.value === model)) {
      setModel(modelOptions[0]?.value || '')
    }
  }, [model, modelOptions])

  useEffect(() => {
    if (!model) {
      return
    }

    let mounted = true
    getStudioFormSchema('image', model)
      .then(({ schema: nextSchema, initialValues }) => {
        if (!mounted) {
          return
        }
        setSchema(nextSchema)
        setFormValues(initialValues)
      })
      .catch(() => {
        if (!mounted) {
          return
        }
        const fallback = getDefaultStudioSchema('image')
        setSchema(fallback)
        setFormValues(buildInitialFormValues(fallback))
      })

    return () => {
      mounted = false
    }
  }, [model])

  const parseErrorMessage = (error: unknown): string => {
    if (!error || typeof error !== 'object') {
      return ''
    }
    const record = error as Record<string, unknown>
    const response =
      record.response && typeof record.response === 'object'
        ? (record.response as Record<string, unknown>)
        : null
    const responseData =
      response && response.data && typeof response.data === 'object'
        ? (response.data as Record<string, unknown>)
        : null
    const openaiError =
      responseData && responseData.error && typeof responseData.error === 'object'
        ? (responseData.error as Record<string, unknown>)
        : null

    return (
      (responseData && typeof responseData.message === 'string'
        ? responseData.message
        : '') ||
      (openaiError && typeof openaiError.message === 'string'
        ? openaiError.message
        : '') ||
      (typeof record.message === 'string' ? record.message : '')
    )
  }

  const generate = useCallback(async () => {
    if (!prompt.trim() || !model) return
    stopPolling()
    setIsLoading(true)
    setImageUrl(null)
    setError(null)
    setTaskId('')
    setTaskStatus('')
    try {
      const imageUrls = Array.isArray(formValues.image_urls)
        ? formValues.image_urls.filter((item): item is string => typeof item === 'string')
        : []

      // All models (including Gemini image models) use the task-based image_generations flow.
      const aspectRatio =
        typeof formValues.aspect_ratio === 'string' && formValues.aspect_ratio.trim()
          ? formValues.aspect_ratio
          : '9:16'
      const resolution =
        typeof formValues.resolution === 'string' && formValues.resolution.trim()
          ? formValues.resolution
          : '1k'

      const metadata: Record<string, unknown> = {
        aspectRatio,
        resolution,
      }
      if (imageUrls.length > 0) {
        metadata.imageUrls = imageUrls
      }

      const res = await generateImage({
        model,
        prompt,
        metadata,
      })
      const url = parseImageUrlFromResponse(res)
      if (url) {
        setImageUrl(url)
        setIsLoading(false)
        return
      }

      const nextTaskId = parseTaskId(res)
      if (nextTaskId) {
        setTaskId(nextTaskId)
        setTaskStatus('SUBMITTED')
        toast.success(t('Image task submitted'))
        startPollingTask(nextTaskId)
      } else {
        setIsLoading(false)
        setError(t('No image returned'))
      }
    } catch (e: unknown) {
      setIsLoading(false)
      const msg = parseErrorMessage(e) || t('Failed to generate image')
      setError(msg)
    }
  }, [formValues, model, parseImageUrlFromResponse, prompt, startPollingTask, stopPolling, t])

  const handleStop = () => {
    stopPolling()
    setIsLoading(false)
    setTaskStatus((prev) => prev || 'STOPPED')
  }

  return (
    <div className='space-y-4'>
      <StudioFormFields
        schema={schema}
        values={formValues}
        onValueChange={(key, value) => {
          setFormValues((prev) => ({ ...prev, [key]: value }))
        }}
        disabled={isLoading}
      />

      {error && <p className='text-destructive text-sm'>{error}</p>}

      {(taskId || taskStatus) && (
        <div className='bg-muted/30 border rounded-2xl p-3 text-sm'>
          {taskId && <p>{t('Task ID')}: {taskId}</p>}
          {taskStatus && <p>{t('Status')}: {taskStatus}</p>}
        </div>
      )}

      {imageUrl && (
        <div className='bg-muted/30 border rounded-2xl p-3'>
          <img
            src={imageUrl}
            alt={prompt}
            className='max-h-[480px] w-full rounded-xl object-contain'
          />
        </div>
      )}

      <StudioPromptInput
        text={prompt}
        onTextChange={setPrompt}
        onSubmit={generate}
        isGenerating={isLoading}
        onStop={handleStop}
        placeholder={t('Describe the image you want to generate...')}
        submitLabel={t('Generate Image')}
        stopLabel={t('Stop')}
        models={modelOptions}
        model={model}
        onModelChange={setModel}
      />
    </div>
  )
}
