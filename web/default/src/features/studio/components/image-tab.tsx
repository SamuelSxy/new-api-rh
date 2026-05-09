import { useState, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { generateImage } from '../api'
import type { ModelOption } from '../types'
import { StudioPromptInput } from './studio-prompt-input'

interface ImageTabProps {
  models: ModelOption[]
}

export function ImageTab({ models }: ImageTabProps) {
  const { t } = useTranslation()
  const [model, setModel] = useState(models[0]?.value ?? '')
  const [prompt, setPrompt] = useState('')
  const [imageUrl, setImageUrl] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!model && models.length > 0) {
      setModel(models[0].value)
    }
  }, [model, models])

  const generate = useCallback(async () => {
    if (!prompt.trim() || !model) return
    setIsLoading(true)
    setImageUrl(null)
    setError(null)
    try {
      const res = await generateImage({ model, prompt })
      const url = res.data?.[0]?.url ?? (res.data?.[0]?.b64_json ? `data:image/png;base64,${res.data[0].b64_json}` : null)
      if (url) {
        setImageUrl(url)
      } else {
        setError(t('No image returned'))
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : t('Failed to generate image')
      setError(msg)
    } finally {
      setIsLoading(false)
    }
  }, [model, prompt, t])

  return (
    <div className='space-y-4'>
      {error && <p className='text-destructive text-sm'>{error}</p>}

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
        placeholder={t('Describe the image you want to generate...')}
        submitLabel={t('Generate Image')}
        stopLabel={t('Stop')}
        models={models}
        model={model}
        onModelChange={setModel}
      />
    </div>
  )
}
