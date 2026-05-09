import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { SSE } from 'sse.js'
import { getCommonHeaders } from '@/lib/api'
import { useEffect } from 'react'
import { StudioPromptInput } from './studio-prompt-input'
import type { ModelOption } from '../types'
import { API_ENDPOINTS } from '../constants'

interface ScriptTabProps {
  models: ModelOption[]
}

export function ScriptTab({ models }: ScriptTabProps) {
  const { t } = useTranslation()
  const [model, setModel] = useState(models[0]?.value ?? '')
  const [prompt, setPrompt] = useState('')
  const [output, setOutput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const sseRef = useRef<SSE | null>(null)

  useEffect(() => {
    if (!model && models.length > 0) {
      setModel(models[0].value)
    }
  }, [model, models])

  const stop = useCallback(() => {
    sseRef.current?.close()
    sseRef.current = null
    setIsStreaming(false)
  }, [])

  const generate = useCallback(() => {
    if (!prompt.trim() || !model) return
    setOutput('')
    setIsStreaming(true)

    const source = new SSE(API_ENDPOINTS.CHAT_COMPLETIONS, {
      headers: getCommonHeaders(),
      method: 'POST',
      payload: JSON.stringify({
        model,
        messages: [{ role: 'user', content: prompt }],
        stream: true,
      }),
    })
    sseRef.current = source

    source.addEventListener('message', (e: MessageEvent) => {
      if (e.data === '[DONE]') {
        setIsStreaming(false)
        sseRef.current = null
        return
      }
      try {
        const chunk = JSON.parse(e.data)
        const content = chunk.choices?.[0]?.delta?.content
        if (content) setOutput((prev) => prev + content)
      } catch {
        // ignore parse errors
      }
    })

    source.addEventListener('error', () => {
      setIsStreaming(false)
      sseRef.current = null
    })

    source.stream()
  }, [model, prompt])

  return (
    <div className='space-y-4'>
      {output && (
        <div className='bg-muted/60 border rounded-2xl p-4 text-sm whitespace-pre-wrap'>
          {output}
        </div>
      )}

      <StudioPromptInput
        text={prompt}
        onTextChange={setPrompt}
        onSubmit={generate}
        isGenerating={isStreaming}
        onStop={stop}
        placeholder={t('Enter your prompt...')}
        submitLabel={t('Generate Script')}
        stopLabel={t('Stop')}
        models={models}
        model={model}
        onModelChange={setModel}
      />
    </div>
  )
}
