import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { VideoIcon } from 'lucide-react'
import { toast } from 'sonner'
import { fetchVideoTask, getStudioFormSchema, submitVideoTask } from '../api'
import { buildInitialFormValues, getDefaultStudioSchema } from '../schema'
import type { ModelOption, StudioFormSchema, StudioFormValue } from '../types'
import { StudioFormFields } from './studio-form-fields'
import { StudioPromptInput } from './studio-prompt-input'
import { Progress } from '@/components/ui/progress'

interface VideoTabProps {
  models: ModelOption[]
}

function isSeedanceModel(modelName: string): boolean {
  return modelName.toLowerCase().includes('seedance')
}

const DEFAULT_VIDEO_MODEL = 'doubao-seedance-2-0-fall'

export function VideoTab({ models }: VideoTabProps) {
  const { t } = useTranslation()
  const preferred = models.find((m) => m.value === DEFAULT_VIDEO_MODEL)
  const [model, setModel] = useState(preferred?.value ?? models[0]?.value ?? '')
  const [prompt, setPrompt] = useState('')
  const [isGenerating, setIsGenerating] = useState(false)
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [videoUrl, setVideoUrl] = useState('')
  const [error, setError] = useState('')
  const [progress, setProgress] = useState(0)
  const [progressIsExact, setProgressIsExact] = useState(false)
  const [schema, setSchema] = useState<StudioFormSchema>(() =>
    getDefaultStudioSchema('video')
  )
  const [formValues, setFormValues] = useState<Record<string, StudioFormValue>>(
    () => buildInitialFormValues(getDefaultStudioSchema('video'))
  )
  const pollTimerRef = useRef<number | null>(null)
  const pollingInFlightRef = useRef(false)
  const progressTimerRef = useRef<number | null>(null)
  const taskStartRef = useRef<number>(0)
  const progressIsExactRef = useRef(false)

  const stopPolling = () => {
    if (pollTimerRef.current !== null) {
      window.clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }

  const stopProgressTicker = () => {
    if (progressTimerRef.current !== null) {
      window.clearInterval(progressTimerRef.current)
      progressTimerRef.current = null
    }
  }

  // Asymptotic estimator: grows toward ~90% in roughly 120s, never reaches 100
  // until a terminal state explicitly sets it.
  const estimateProgress = (elapsedMs: number) => {
    const tau = 60_000
    const target = 92
    const value = target * (1 - Math.exp(-elapsedMs / tau))
    return Math.max(5, Math.min(target, value))
  }

  const startProgressTicker = () => {
    stopProgressTicker()
    progressTimerRef.current = window.setInterval(() => {
      if (progressIsExactRef.current) {
        return
      }
      const elapsed = Date.now() - taskStartRef.current
      setProgress((prev) => Math.max(prev, estimateProgress(elapsed)))
    }, 500)
  }

  useEffect(() => {
    if (!model && models.length > 0) {
      const preferred = models.find((m) => m.value === DEFAULT_VIDEO_MODEL)
      setModel(preferred?.value ?? models[0].value)
    }
  }, [model, models])

  useEffect(() => {
    if (!model) {
      return
    }

    let mounted = true
    getStudioFormSchema('video', model)
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
        const fallback = getDefaultStudioSchema('video')
        setSchema(fallback)
        setFormValues(buildInitialFormValues(fallback))
      })

    return () => {
      mounted = false
    }
  }, [model])

  useEffect(() => {
    return () => {
      stopPolling()
      stopProgressTicker()
    }
  }, [])

  const parseProgress = (response: unknown): number | null => {
    if (!response || typeof response !== 'object') {
      return null
    }
    const record = response as Record<string, unknown>
    const nested =
      record.data && typeof record.data === 'object'
        ? (record.data as Record<string, unknown>)
        : null

    const candidates: unknown[] = [
      record.progress,
      record.percent,
      record.percentage,
      nested?.progress,
      nested?.percent,
      nested?.percentage,
      nested?.task_progress,
    ]
    for (const raw of candidates) {
      if (raw === undefined || raw === null) continue
      let n: number | null = null
      if (typeof raw === 'number' && Number.isFinite(raw)) {
        n = raw
      } else if (typeof raw === 'string') {
        const trimmed = raw.trim().replace(/%$/, '')
        const parsed = Number(trimmed)
        if (Number.isFinite(parsed)) {
          n = parsed
        }
      }
      if (n === null) continue
      if (n > 0 && n <= 1) {
        n = n * 100
      }
      if (n < 0) n = 0
      if (n > 100) n = 100
      return n
    }
    return null
  }

  const parseStatus = (response: unknown): string => {
    if (!response || typeof response !== 'object') {
      return ''
    }
    const record = response as Record<string, unknown>
    const nested =
      record.data && typeof record.data === 'object'
        ? (record.data as Record<string, unknown>)
        : null

    return (
      (typeof record.status === 'string' ? record.status : '') ||
      (typeof record.taskStatus === 'string' ? record.taskStatus : '') ||
      (nested && typeof nested.status === 'string' ? nested.status : '') ||
      (nested && typeof nested.taskStatus === 'string' ? nested.taskStatus : '') ||
      ''
    )
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

  const parseVideoUrl = (response: unknown): string => {
    if (!response || typeof response !== 'object') {
      return ''
    }
    const record = response as Record<string, unknown>
    const nested =
      record.data && typeof record.data === 'object'
        ? (record.data as Record<string, unknown>)
        : null

    const fromRecord =
      (typeof record.url === 'string' ? record.url : '') ||
      (typeof record.result_url === 'string' ? record.result_url : '')
    if (fromRecord) {
      return fromRecord
    }

    const fromNested =
      (nested && typeof nested.url === 'string' ? nested.url : '') ||
      (nested && typeof nested.result_url === 'string' ? nested.result_url : '')
    if (fromNested) {
      return fromNested
    }

    if (nested && nested.data && typeof nested.data === 'object') {
      const deep = nested.data as Record<string, unknown>
      return (
        (typeof deep.url === 'string' ? deep.url : '') ||
        (typeof deep.result_url === 'string' ? deep.result_url : '') ||
        ''
      )
    }
    return ''
  }

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

    const status =
      response && typeof response.status === 'number' ? response.status : undefined

    const directMessage =
      (responseData && typeof responseData.message === 'string'
        ? responseData.message
        : '') ||
      (openaiError && typeof openaiError.message === 'string'
        ? openaiError.message
        : '') ||
      (typeof record.message === 'string' ? record.message : '')

    if (directMessage) {
      return directMessage
    }

    if (status === 503) {
      return t(
        'Service is temporarily unavailable. Please check whether this model has an available channel and retry shortly'
      )
    }

    return ''
  }

  const isTerminalSuccess = (status: string) => {
    const normalized = status.toLowerCase()
    return ['completed', 'succeeded', 'success', 'done'].includes(normalized)
  }

  const isTerminalFailed = (status: string) => {
    const normalized = status.toLowerCase()
    return ['failed', 'error', 'cancelled', 'canceled'].includes(normalized)
  }

  const buildVideoPayload = () => {
    const metadata: Record<string, unknown> = {}
    Object.entries(formValues).forEach(([key, value]) => {
      if (value === '' || value === undefined || value === null) {
        return
      }
      metadata[key] = value
    })

    // Resolve media from the unified media_urls field (new schemas) or
    // fallback to legacy image_urls (old schemas).
    const allMediaUrls: string[] = (
      Array.isArray(formValues.media_urls)
        ? formValues.media_urls
        : Array.isArray(formValues.image_urls)
          ? formValues.image_urls
          : []
    ).filter((item): item is string => typeof item === 'string')

    // Split into images and videos based on the VIDEO_PREFIX stored by MediaUploadField
    // and data URL MIME prefix for file-uploaded videos.
    const videoUrlsRaw = allMediaUrls.filter(
      (u) => u.startsWith('video:') || u.startsWith('data:video')
    )
    const imageUrls = allMediaUrls.filter(
      (u) => !u.startsWith('video:') && !u.startsWith('data:video')
    )
    // Strip the 'video:' prefix before sending
    const videoUrls = videoUrlsRaw.map((u) => (u.startsWith('video:') ? u.slice(6) : u))

    // Also read the legacy text-field video_url for backward compat
    const legacyVideoUrl =
      typeof formValues.video_url === 'string' && formValues.video_url.trim()
        ? formValues.video_url.trim()
        : null

    const payload: Record<string, unknown> = {
      model,
      prompt,
      metadata,
    }

    // Images: send first as top-level `image`, all as `images`
    if (imageUrls.length > 0) {
      payload.image = imageUrls[0]
      if (imageUrls.length > 1) {
        payload.images = imageUrls
      }
    }

    // Videos: send as `content` array items with type "video_url"
    const allVideos = [...videoUrls, ...(legacyVideoUrl ? [legacyVideoUrl] : [])]
    if (allVideos.length > 0) {
      payload.content = allVideos.map((url) => ({
        type: 'video_url',
        video_url: { url },
      }))
    }

    if (typeof formValues.duration === 'number') {
      payload.duration = formValues.duration
    }
    if (typeof formValues.seed === 'number') {
      payload.seed = formValues.seed
    }
    if (typeof formValues.n === 'number') {
      payload.n = formValues.n
    }

    return payload
  }

  const applyProgress = (response: unknown, status: string) => {
    const exact = parseProgress(response)
    if (exact !== null) {
      progressIsExactRef.current = true
      setProgressIsExact(true)
      setProgress((prev) => Math.max(prev, exact))
      return
    }
    if (status && /running|in[_-]?progress|processing/i.test(status)) {
      const elapsed = Date.now() - taskStartRef.current
      setProgress((prev) => Math.max(prev, estimateProgress(elapsed)))
    }
  }

  const startPollingTask = (nextTaskId: string) => {
    stopPolling()
    pollTimerRef.current = window.setInterval(async () => {
      if (pollingInFlightRef.current) {
        return
      }
      pollingInFlightRef.current = true
      try {
        const result = await fetchVideoTask(nextTaskId)
        const nextStatus = parseStatus(result)
        if (nextStatus) {
          setTaskStatus(nextStatus)
        }

        const resultUrl = parseVideoUrl(result)
        if (resultUrl) {
          setVideoUrl(resultUrl)
        }

        applyProgress(result, nextStatus)

        if (isTerminalSuccess(nextStatus)) {
          stopPolling()
          stopProgressTicker()
          progressIsExactRef.current = true
          setProgressIsExact(true)
          setProgress(100)
          setIsGenerating(false)
          if (!resultUrl) {
            setVideoUrl(`/v1/videos/${nextTaskId}/content`)
          }
          toast.success(t('Video task completed'))
          return
        }

        if (isTerminalFailed(nextStatus)) {
          stopPolling()
          stopProgressTicker()
          setIsGenerating(false)
          setError(t('Video task failed'))
          toast.error(t('Video task failed'))
        }
      } catch {
        stopPolling()
        stopProgressTicker()
        setIsGenerating(false)
        setError(t('Failed to query video task status'))
        toast.error(t('Failed to query video task status'))
      } finally {
        pollingInFlightRef.current = false
      }
    }, 3000)
  }

  const handleGenerate = async () => {
    if (!prompt.trim() || !model) {
      return
    }

    setError('')
    setVideoUrl('')
    setTaskId('')
    setTaskStatus('')
    setIsGenerating(true)
    progressIsExactRef.current = false
    setProgressIsExact(false)
    setProgress(0)
    taskStartRef.current = Date.now()
    startProgressTicker()

    try {
      const submitResult = await submitVideoTask(buildVideoPayload())
      const nextTaskId = parseTaskId(submitResult)
      const nextStatus = parseStatus(submitResult)
      const initialUrl = parseVideoUrl(submitResult)

      if (nextStatus) {
        setTaskStatus(nextStatus)
      }
      if (initialUrl) {
        setVideoUrl(initialUrl)
      }

      applyProgress(submitResult, nextStatus)

      if (!nextTaskId) {
        if (initialUrl) {
          stopProgressTicker()
          progressIsExactRef.current = true
          setProgressIsExact(true)
          setProgress(100)
          setIsGenerating(false)
          toast.success(t('Video generated successfully'))
          return
        }
        stopProgressTicker()
        setIsGenerating(false)
        setError(t('Failed to get task id from video response'))
        toast.error(t('Failed to get task id from video response'))
        return
      }

      setTaskId(nextTaskId)

      if (nextStatus && isTerminalSuccess(nextStatus)) {
        stopProgressTicker()
        progressIsExactRef.current = true
        setProgressIsExact(true)
        setProgress(100)
        setIsGenerating(false)
        if (!initialUrl) {
          setVideoUrl(`/v1/videos/${nextTaskId}/content`)
        }
        toast.success(t('Video task completed'))
        return
      }

      if (nextStatus && isTerminalFailed(nextStatus)) {
        stopProgressTicker()
        setIsGenerating(false)
        setError(t('Video task failed'))
        toast.error(t('Video task failed'))
        return
      }

      toast.success(t('Video task submitted'))
      startPollingTask(nextTaskId)
    } catch (err) {
      stopProgressTicker()
      setIsGenerating(false)
      const message = parseErrorMessage(err)
      const finalMessage = message || t('Failed to submit video task')
      setError(finalMessage)
      toast.error(finalMessage)
    }
  }

  const handleStop = () => {
    stopPolling()
    stopProgressTicker()
    setIsGenerating(false)
    setTaskStatus((prev) => prev || 'stopped')
    toast.info(t('Stopped polling video task status'))
  }

  return (
    <div className='space-y-4'>
      <StudioFormFields
        schema={schema}
        values={formValues}
        onValueChange={(key, value) => {
          setFormValues((prev) => ({ ...prev, [key]: value }))
        }}
        showAssetLibrary={isSeedanceModel(model)}
      />

      <div className='flex items-start gap-3 rounded-2xl border bg-muted/20 p-4'>
        <VideoIcon className='text-muted-foreground mt-0.5 size-5 shrink-0' />
        <div className='flex-1 space-y-2 text-sm'>
          <p className='font-medium'>{t('Video task is connected')}</p>
          <p className='text-muted-foreground'>
            {taskId
              ? `${t('Task ID')}: ${taskId}`
              : t('Submit task to generate and poll video status')}
          </p>
          {taskStatus && (
            <p className='text-muted-foreground'>
              {t('Status')}: {taskStatus}
            </p>
          )}
          {(isGenerating || progress > 0) && (
            <div className='space-y-1'>
              <Progress
                value={progress}
                className={error ? 'opacity-60' : undefined}
              />
              <div className='text-muted-foreground flex justify-between text-xs'>
                <span>
                  {progressIsExact ? t('Progress') : t('Estimated progress')}
                </span>
                <span>{Math.round(progress)}%</span>
              </div>
            </div>
          )}
          {error && <p className='text-destructive'>{error}</p>}
        </div>
      </div>

      {videoUrl && (
        <div className='space-y-2 rounded-2xl border bg-background p-3'>
          <video className='max-h-96 w-full rounded-md border bg-black' controls src={videoUrl} />
          <a
            className='text-primary text-sm underline-offset-4 hover:underline'
            href={videoUrl}
            target='_blank'
            rel='noreferrer'
          >
            {t('Open video in new tab')}
          </a>
        </div>
      )}

      <StudioPromptInput
        text={prompt}
        onTextChange={setPrompt}
        onSubmit={handleGenerate}
        isGenerating={isGenerating}
        onStop={handleStop}
        placeholder={t('Describe the video you want to generate...')}
        submitLabel={t('Generate Video')}
        stopLabel={t('Stop')}
        models={models}
        model={model}
        onModelChange={setModel}
        disabled={isGenerating}
      />
    </div>
  )
}
