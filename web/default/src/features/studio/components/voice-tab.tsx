import { useState, useCallback, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { generateVoice } from '../api'
import type { ModelOption } from '../types'
import { StudioPromptInput } from './studio-prompt-input'

interface VoiceTabProps {
  models: ModelOption[]
}

export function VoiceTab({ models }: VoiceTabProps) {
  const { t } = useTranslation()
  const [model, setModel] = useState(models[0]?.value ?? '')
  const [text, setText] = useState('')
  const [audioUrl, setAudioUrl] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const prevUrlRef = useRef<string | null>(null)

  useEffect(() => {
    if (!model && models.length > 0) {
      setModel(models[0].value)
    }
  }, [model, models])

  useEffect(() => {
    return () => {
      if (prevUrlRef.current) URL.revokeObjectURL(prevUrlRef.current)
    }
  }, [])

  const generate = useCallback(async () => {
    if (!text.trim() || !model) return
    setIsLoading(true)
    setError(null)
    try {
      const blob = await generateVoice({ model, input: text })
      if (prevUrlRef.current) URL.revokeObjectURL(prevUrlRef.current)
      const url = URL.createObjectURL(blob)
      prevUrlRef.current = url
      setAudioUrl(url)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : t('Failed to generate voiceover')
      setError(msg)
    } finally {
      setIsLoading(false)
    }
  }, [model, text, t])

  return (
    <div className='space-y-4'>
      {error && <p className='text-destructive text-sm'>{error}</p>}

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
        placeholder={t('Enter the text to convert to speech...')}
        submitLabel={t('Generate Voiceover')}
        stopLabel={t('Stop')}
        models={models}
        model={model}
        onModelChange={setModel}
      />
    </div>
  )
}
