import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { SSE } from 'sse.js'
import { getCommonHeaders } from '@/lib/api'
import { useEffect } from 'react'
import { getStudioFormSchema } from '../api'
import type { StudioFormSchema, StudioFormValue } from '../types'
import { getDefaultStudioSchema, buildInitialFormValues } from '../schema'
import { StudioPromptInput } from './studio-prompt-input'
import { StudioFormFields } from './studio-form-fields'
import type { ModelOption } from '../types'
import { API_ENDPOINTS } from '../constants'
import { Button } from '@/components/ui/button'
import { History } from 'lucide-react'
import { useScriptHistory } from '../hooks/use-script-history'
import type { ScriptHistoryItem } from '../hooks/use-script-history'
import { ScriptHistorySheet } from './script-history-sheet'

interface ScriptTabProps {
  models: ModelOption[]
}

export function ScriptTab({ models }: ScriptTabProps) {
  const { t } = useTranslation()
  const [model, setModel] = useState(models[0]?.value ?? '')
  const [prompt, setPrompt] = useState('')
  const [output, setOutput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [schema, setSchema] = useState<StudioFormSchema>(() =>
    getDefaultStudioSchema('script')
  )
  const [formValues, setFormValues] = useState<Record<string, StudioFormValue>>(
    () => buildInitialFormValues(getDefaultStudioSchema('script'))
  )
  const sseRef = useRef<SSE | null>(null)
  const outputRef = useRef('')
  const { history, addRecord, deleteRecord, clearAll } = useScriptHistory()

  useEffect(() => {
    if (!model && models.length > 0) {
      setModel(models[0].value)
    }
  }, [model, models])

  useEffect(() => {
    if (!model) {
      return
    }

    let mounted = true
    getStudioFormSchema('script', model)
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
        const fallback = getDefaultStudioSchema('script')
        setSchema(fallback)
        setFormValues(buildInitialFormValues(fallback))
      })

    return () => {
      mounted = false
    }
  }, [model])

  const stop = useCallback(() => {
    sseRef.current?.close()
    sseRef.current = null
    setIsStreaming(false)
    // save to history when stopped mid-stream if there is output
    if (outputRef.current.trim() && prompt.trim()) {
      addRecord({ prompt, output: outputRef.current, model })
    }
  }, [prompt, model, addRecord])

  const generate = useCallback(() => {
    if (!prompt.trim() || !model) return
    setOutput('')
    outputRef.current = ''
    setIsStreaming(true)

    const source = new SSE(API_ENDPOINTS.CHAT_COMPLETIONS, {
      headers: getCommonHeaders(),
      method: 'POST',
      payload: JSON.stringify({
        model,
        messages: [{ role: 'user', content: prompt }],
        stream: true,
        temperature:
          typeof formValues.temperature === 'number'
            ? formValues.temperature
            : undefined,
        max_tokens:
          typeof formValues.max_tokens === 'number'
            ? formValues.max_tokens
            : undefined,
      }),
    })
    sseRef.current = source

    source.addEventListener('message', (e: MessageEvent) => {
      if (e.data === '[DONE]') {
        setIsStreaming(false)
        sseRef.current = null
        // save completed generation to history
        if (outputRef.current.trim() && prompt.trim()) {
          addRecord({ prompt, output: outputRef.current, model })
        }
        return
      }
      try {
        const chunk = JSON.parse(e.data)
        const content = chunk.choices?.[0]?.delta?.content
        if (content) {
          outputRef.current += content
          setOutput((prev) => prev + content)
        }
      } catch {
        // ignore parse errors
      }
    })

    source.addEventListener('error', () => {
      setIsStreaming(false)
      sseRef.current = null
    })

    source.stream()
  }, [model, prompt, addRecord])

  const handleRestore = useCallback((item: ScriptHistoryItem) => {
    setPrompt(item.prompt)
    setOutput(item.output)
    outputRef.current = item.output
    setModel(item.model)
    setHistoryOpen(false)
  }, [])

  return (
    <div className='space-y-4'>
      <div className='flex justify-end'>
        <Button
          variant='outline'
          size='sm'
          className='gap-1.5'
          onClick={() => setHistoryOpen(true)}
        >
          <History className='size-4' />
          {t('Script History')}
          {history.length > 0 && (
            <span className='ml-0.5 rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-medium text-primary'>
              {history.length}
            </span>
          )}
        </Button>
      </div>

      <StudioFormFields
        schema={schema}
        values={formValues}
        onValueChange={(key, value) => {
          setFormValues((prev) => ({ ...prev, [key]: value }))
        }}
        disabled={isStreaming}
      />

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

      <ScriptHistorySheet
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        history={history}
        onRestore={handleRestore}
        onDelete={deleteRecord}
        onClearAll={clearAll}
      />
    </div>
  )
}
