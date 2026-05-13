import { useState, useCallback, useRef, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { fetchUserTaskById, generateVoice, getStudioFormSchema } from '../api'
import type { ModelOption, StudioFormSchema, StudioFormValue } from '../types'
import { buildInitialFormValues, getDefaultStudioSchema } from '../schema'
import { StudioFormFields } from './studio-form-fields'
import { StudioPromptInput } from './studio-prompt-input'

interface VoiceTabProps {
  models: ModelOption[]
}

const DEFAULT_VOICE_MODEL = 'rhart-audio/text-to-audio/speech-2.8-turbo'
const DEFAULT_VOICE_ID = 'Elegant_Man'

export function VoiceTab({ models }: VoiceTabProps) {
  const { t } = useTranslation()
  const modelOptions = useMemo(() => {
    if (models.some((item) => item.value === DEFAULT_VOICE_MODEL)) {
      return models
    }
    return [{ label: DEFAULT_VOICE_MODEL, value: DEFAULT_VOICE_MODEL }, ...models]
  }, [models])

  const [model, setModel] = useState(
    modelOptions.find((item) => item.value === DEFAULT_VOICE_MODEL)?.value ||
      modelOptions[0]?.value ||
      ''
  )
  const [text, setText] = useState('')
  const [audioUrl, setAudioUrl] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [schema, setSchema] = useState<StudioFormSchema>(() =>
    getDefaultStudioSchema('voice')
  )
  const [formValues, setFormValues] = useState<Record<string, StudioFormValue>>(
    () => buildInitialFormValues(getDefaultStudioSchema('voice'))
  )
  const prevUrlRef = useRef<string | null>(null)
  const pollTimerRef = useRef<number | null>(null)
  const pollingInFlightRef = useRef(false)

  const revokeIfBlobUrl = useCallback((url: string | null) => {
    if (url && url.startsWith('blob:')) {
      URL.revokeObjectURL(url)
    }
  }, [])

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current !== null) {
      window.clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  useEffect(() => {
    if (!model && modelOptions.length > 0) {
      const preferred = modelOptions.find((item) => item.value === DEFAULT_VOICE_MODEL)
      setModel(preferred?.value || modelOptions[0].value)
      return
    }

    if (model && !modelOptions.some((item) => item.value === model)) {
      const preferred = modelOptions.find((item) => item.value === DEFAULT_VOICE_MODEL)
      setModel(preferred?.value || modelOptions[0]?.value || '')
    }
  }, [model, modelOptions])

  useEffect(() => {
    if (!model) {
      return
    }

    let mounted = true
    getStudioFormSchema('voice', model)
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
        const fallback = getDefaultStudioSchema('voice')
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
      revokeIfBlobUrl(prevUrlRef.current)
    }
  }, [revokeIfBlobUrl, stopPolling])

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

  const parseAudioUrlFromResponse = (response: unknown): string => {
    if (!response || typeof response !== 'object') {
      return ''
    }
    const record = response as Record<string, unknown>
    if (typeof record.result_url === 'string' && record.result_url) {
      return record.result_url
    }

    const data = record.data
    if (Array.isArray(data) && data.length > 0 && data[0] && typeof data[0] === 'object') {
      const first = data[0] as Record<string, unknown>
      const url = typeof first.url === 'string' ? first.url : ''
      const fileUrl = typeof first.fileUrl === 'string' ? first.fileUrl : ''
      return url || fileUrl || ''
    }
    if (data && typeof data === 'object') {
      const nested = data as Record<string, unknown>
      if (typeof nested.url === 'string' && nested.url) {
        return nested.url
      }
      if (typeof nested.fileUrl === 'string' && nested.fileUrl) {
        return nested.fileUrl
      }
    }
    return ''
  }

  const parseTaskStatus = (task: Record<string, unknown>): string => {
    return typeof task.status === 'string' ? task.status : ''
  }

  const parseAudioUrlFromTask = (task: Record<string, unknown>): string => {
    if (typeof task.result_url === 'string' && task.result_url) {
      return task.result_url
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

  const startPollingTask = useCallback(
    (nextTaskId: string) => {
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

          const taskAudioUrl = parseAudioUrlFromTask(task)
          if (taskAudioUrl) {
            revokeIfBlobUrl(prevUrlRef.current)
            prevUrlRef.current = taskAudioUrl
            setAudioUrl(taskAudioUrl)
          }

          if (status && isTaskSuccess(status)) {
            stopPolling()
            setIsLoading(false)
            if (taskAudioUrl) {
              toast.success(t('Voiceover generated successfully'))
            } else {
              setError(t('No audio returned'))
            }
            return
          }

          if (status && isTaskFailure(status)) {
            stopPolling()
            setIsLoading(false)
            setError(t('Failed to generate voiceover'))
            toast.error(t('Failed to generate voiceover'))
          }
        } catch {
          stopPolling()
          setIsLoading(false)
          setError(t('Failed to query voice task status'))
          toast.error(t('Failed to query voice task status'))
        } finally {
          pollingInFlightRef.current = false
        }
      }, 2500)
    },
    [revokeIfBlobUrl, stopPolling, t]
  )

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
    if (!text.trim() || !model) return
    stopPolling()
    setIsLoading(true)
    setError(null)
    setTaskId('')
    setTaskStatus('')
    try {
      const voiceId =
        typeof formValues.voice === 'string' && formValues.voice.trim()
          ? formValues.voice
          : DEFAULT_VOICE_ID
      const speed =
        typeof formValues.speed === 'number' ? formValues.speed : 1
      const volume =
        typeof formValues.volume === 'number' ? formValues.volume : 1
      const pitch =
        typeof formValues.pitch === 'number' ? formValues.pitch : 0
      const emotion =
        typeof formValues.emotion === 'string' && formValues.emotion.trim()
          ? formValues.emotion
          : 'happy'
      const responseFormat =
        typeof formValues.response_format === 'string' && formValues.response_format.trim()
          ? formValues.response_format
          : 'mp3'
      const enableBase64Output =
        typeof formValues.enable_base64_output === 'boolean'
          ? formValues.enable_base64_output
          : false
      const englishNormalization =
        typeof formValues.english_normalization === 'boolean'
          ? formValues.english_normalization
          : false

      const voiceResp = await generateVoice({
        model,
        input: text,
        voice: voiceId,
        response_format: responseFormat,
        metadata: {
          voice_id: voiceId,
          speed,
          volume,
          pitch,
          emotion,
          enable_base64_output: enableBase64Output,
          english_normalization: englishNormalization,
        },
      })

      if (voiceResp instanceof Blob) {
        revokeIfBlobUrl(prevUrlRef.current)
        const url = URL.createObjectURL(voiceResp)
        prevUrlRef.current = url
        setAudioUrl(url)
        setIsLoading(false)
        return
      }

      const syncUrl = parseAudioUrlFromResponse(voiceResp)
      if (syncUrl) {
        revokeIfBlobUrl(prevUrlRef.current)
        prevUrlRef.current = syncUrl
        setAudioUrl(syncUrl)
        setIsLoading(false)
        return
      }

      const nextTaskId = parseTaskId(voiceResp)
      if (nextTaskId) {
        setTaskId(nextTaskId)
        setTaskStatus('SUBMITTED')
        toast.success(t('Voice task submitted'))
        startPollingTask(nextTaskId)
      } else {
        setIsLoading(false)
        setError(t('No audio returned'))
      }
    } catch (e: unknown) {
      const msg = parseErrorMessage(e) || t('Failed to generate voiceover')
      setError(msg)
    } finally {
      if (!pollTimerRef.current) {
        setIsLoading(false)
      }
    }
  }, [
    formValues,
    model,
    parseErrorMessage,
    revokeIfBlobUrl,
    startPollingTask,
    stopPolling,
    t,
    text,
  ])

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

      {audioUrl && (
        <div className='bg-muted/30 border rounded-2xl p-4'>
          {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
          <audio controls src={audioUrl} className='w-full' />
        </div>
      )}

      <StudioPromptInput
        text={text}
        onTextChange={setText}
        onSubmit={generate}
        isGenerating={isLoading}
        onStop={handleStop}
        placeholder={t('Enter the text to convert to speech...')}
        submitLabel={t('Generate Voiceover')}
        stopLabel={t('Stop')}
        models={modelOptions}
        model={model}
        onModelChange={setModel}
      />
    </div>
  )
}
