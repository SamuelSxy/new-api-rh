import { SendIcon, SquareIcon } from 'lucide-react'
import {
  PromptInput,
  PromptInputButton,
  PromptInputFooter,
  PromptInputTextarea,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { ModelSelector } from '@/components/model-group-selector'
import type { ModelOption } from '../types'

interface StudioPromptInputProps {
  text: string
  onTextChange: (value: string) => void
  onSubmit: () => void
  isGenerating: boolean
  onStop?: () => void
  placeholder: string
  submitLabel: string
  stopLabel: string
  models: ModelOption[]
  model: string
  onModelChange: (value: string) => void
  disabled?: boolean
}

export function StudioPromptInput({
  text,
  onTextChange,
  onSubmit,
  isGenerating,
  onStop,
  placeholder,
  submitLabel,
  stopLabel,
  models,
  model,
  onModelChange,
  disabled = false,
}: StudioPromptInputProps) {
  const handleSubmit = (message: PromptInputMessage) => {
    if (!message.text?.trim() || disabled) return
    onSubmit()
  }

  const isModelSelectDisabled = disabled || models.length === 0

  return (
    <PromptInput
      groupClassName='rounded-[20px] [--radius:20px]'
      onSubmit={handleSubmit}
    >
      <PromptInputTextarea
        autoComplete='off'
        autoCorrect='off'
        autoCapitalize='off'
        spellCheck={false}
        className='px-5 md:text-base'
        disabled={disabled}
        onChange={(event) => onTextChange(event.target.value)}
        placeholder={placeholder}
        value={text}
      />

      <PromptInputFooter className='p-2.5'>
        <div className='flex items-center gap-1.5 md:gap-2'>
          <ModelSelector
            selectedModel={model}
            models={models}
            onModelChange={onModelChange}
            disabled={isModelSelectDisabled}
          />

          {isGenerating && onStop ? (
            <PromptInputButton
              className='text-foreground rounded-full font-medium'
              onClick={onStop}
              variant='secondary'
              type='button'
            >
              <SquareIcon className='fill-current' size={16} />
              <span className='hidden sm:inline'>{stopLabel}</span>
              <span className='sr-only sm:hidden'>{stopLabel}</span>
            </PromptInputButton>
          ) : (
            <PromptInputButton
              className='text-foreground rounded-full font-medium'
              disabled={disabled || !text.trim()}
              type='submit'
              variant='secondary'
            >
              <SendIcon size={16} />
              <span className='hidden sm:inline'>{submitLabel}</span>
              <span className='sr-only sm:hidden'>{submitLabel}</span>
            </PromptInputButton>
          )}
        </div>
      </PromptInputFooter>
    </PromptInput>
  )
}
