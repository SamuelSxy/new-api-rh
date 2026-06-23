import { useCallback, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { LinkIcon } from 'lucide-react'
import type { StudioFormField, StudioFormSchema, StudioFormValue, UserAsset } from '../types'
import { AssetLibraryDialog } from './asset-library-dialog'

/** Prefix used to mark video URLs stored inside a media_urls string array. */
const VIDEO_PREFIX = 'video:'

/** Returns true if the stored URL represents a video entry. */
function isVideoEntry(url: string): boolean {
  return url.startsWith(VIDEO_PREFIX) || url.startsWith('data:video')
}

/** Strips VIDEO_PREFIX so the URL is suitable for <video src> or API submission. */
function stripVideoPrefix(url: string): string {
  return url.startsWith(VIDEO_PREFIX) ? url.slice(VIDEO_PREFIX.length) : url
}

/** Auto-detect media type from a plain URL string. */
function detectMediaTypeFromUrl(url: string): 'image' | 'video' {
  const lower = url.toLowerCase().split('?')[0]
  if (['.mp4', '.mov', '.webm', '.avi', '.mpeg', '.m4v', '.mkv'].some((ext) => lower.endsWith(ext))) {
    return 'video'
  }
  if (lower.startsWith('data:video')) return 'video'
  return 'image'
}

interface StudioFormFieldsProps {
  schema: StudioFormSchema
  values: Record<string, StudioFormValue>
  onValueChange: (key: string, value: StudioFormValue) => void
  disabled?: boolean
  showAssetLibrary?: boolean
}

export function StudioFormFields({
  schema,
  values,
  onValueChange,
  disabled = false,
  showAssetLibrary = false,
}: StudioFormFieldsProps) {
  const { t } = useTranslation()
  const [assetLibraryOpen, setAssetLibraryOpen] = useState(false)
  const [assetLibraryField, setAssetLibraryField] = useState<{
    key: string
    assetType: 'Image' | 'Video'
  } | null>(null)
  // Maps submit URL → preview URL for library-selected images.
  // Base64 uploads use the same string for both, so no entry needed.
  const [imagePreviewMap, setImagePreviewMap] = useState<Map<string, string>>(new Map())

  if (schema.fields.length === 0) {
    return null
  }

  const openAssetLibrary = (key: string, assetType: 'Image' | 'Video') => {
    setAssetLibraryField({ key, assetType })
    setAssetLibraryOpen(true)
  }

  const handleAssetSelect = (asset: UserAsset) => {
    if (!assetLibraryField) return
    const submitUrl = asset.ark_asset_id ? `asset://${asset.ark_asset_id}` : asset.source_url
    const previewUrl = asset.source_url
    const fieldDef = schema.fields.find((f) => f.key === assetLibraryField.key)
    if (fieldDef?.type === 'image_upload' || fieldDef?.type === 'media_upload') {
      // Append to the existing array
      const existing = Array.isArray(values[assetLibraryField.key])
        ? (values[assetLibraryField.key] as string[])
        : []
      // For video assets, prefix the stored URL so we can distinguish type later
      const storeUrl = asset.asset_type === 'Video' ? `${VIDEO_PREFIX}${submitUrl}` : submitUrl
      onValueChange(assetLibraryField.key, [...existing, storeUrl])
      // Track preview URL separately so the thumbnail can display correctly
      if (storeUrl !== previewUrl) {
        setImagePreviewMap((prev) => new Map([...prev, [storeUrl, previewUrl]]))
      }
    } else {
      onValueChange(assetLibraryField.key, submitUrl)
    }
  }

  return (
    <>
      {showAssetLibrary && assetLibraryField && (
        <AssetLibraryDialog
          open={assetLibraryOpen}
          onOpenChange={setAssetLibraryOpen}
          defaultAssetType={assetLibraryField.assetType}
          onSelect={handleAssetSelect}
        />
      )}
      <div className='rounded-2xl border bg-muted/20 p-4'>
      <div className='mb-3 text-sm font-medium'>{t('Parameters')}</div>
      <div className='grid gap-3 md:grid-cols-2'>
        {schema.fields.map((field) => {
          const rawValue = values[field.key]
          const value = rawValue ?? ''

          return (
            <div key={field.key} className='space-y-1.5'>
              <Label htmlFor={field.key} className='text-xs'>
                {t(field.label)}
                {field.required ? ' *' : ''}
              </Label>

              {field.type === 'text' && (
                <Input
                  id={field.key}
                  value={String(value)}
                  placeholder={field.placeholder ? t(field.placeholder) : undefined}
                  disabled={disabled}
                  onChange={(event) => onValueChange(field.key, event.target.value)}
                />
              )}

              {field.type === 'textarea' && (
                <Textarea
                  id={field.key}
                  value={String(value)}
                  placeholder={field.placeholder ? t(field.placeholder) : undefined}
                  disabled={disabled}
                  className='min-h-20'
                  onChange={(event) => onValueChange(field.key, event.target.value)}
                />
              )}

              {field.type === 'number' && (
                <Input
                  id={field.key}
                  type='number'
                  min={field.min}
                  max={field.max}
                  step={field.step}
                  value={String(value)}
                  disabled={disabled}
                  onChange={(event) => {
                    const next = event.target.value
                    if (next === '') {
                      onValueChange(field.key, '')
                      return
                    }
                    onValueChange(field.key, Number(next))
                  }}
                />
              )}

              {field.type === 'select' && (
                <Select
                  value={String(value)}
                  onValueChange={(nextValue) => onValueChange(field.key, nextValue)}
                  disabled={disabled}
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(field.options || []).map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {t(option.label)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}

              {field.type === 'switch' && (
                <div className='flex h-9 items-center'>
                  <Switch
                    id={field.key}
                    checked={Boolean(value)}
                    disabled={disabled}
                    onCheckedChange={(checked) => onValueChange(field.key, checked)}
                  />
                </div>
              )}

              {field.type === 'image_upload' && (
                <div className='space-y-1.5'>
                  <ImageUploadField
                    field={field}
                    value={value}
                    disabled={disabled}
                    previewMap={imagePreviewMap}
                    onValueChange={(nextValue) => onValueChange(field.key, nextValue)}
                  />
                  {showAssetLibrary && (
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={disabled}
                      onClick={() => openAssetLibrary(field.key, 'Image')}
                    >
                      {t('From Library')}
                    </Button>
                  )}
                </div>
              )}

              {field.type === 'media_upload' && (
                <MediaUploadField
                  field={field}
                  value={value}
                  disabled={disabled}
                  previewMap={imagePreviewMap}
                  showAssetLibrary={showAssetLibrary}
                  onValueChange={(nextValue) => onValueChange(field.key, nextValue)}
                  onFromLibrary={() => openAssetLibrary(field.key, 'Image')}
                  onPreviewMapUpdate={(entries) =>
                    setImagePreviewMap((prev) => new Map([...prev, ...entries]))
                  }
                />
              )}

              {field.type === 'asset_uri' && (
                <div className='flex gap-2'>
                  <Input
                    id={field.key}
                    value={String(value)}
                    placeholder={field.placeholder ? t(field.placeholder) : undefined}
                    disabled={disabled}
                    onChange={(event) => onValueChange(field.key, event.target.value)}
                  />
                  {showAssetLibrary && (
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={disabled}
                      onClick={() => openAssetLibrary(field.key, 'Video')}
                    >
                      {t('From Library')}
                    </Button>
                  )}
                </div>
              )}

              {field.helpText && (
                <p className='text-muted-foreground text-xs'>{t(field.helpText)}</p>
              )}
            </div>
          )
        })}
      </div>
    </div>
    </>
  )
}

interface ImageUploadFieldProps {
  field: StudioFormField
  value: StudioFormValue
  disabled: boolean
  /** Maps submit URL → preview URL for library-selected assets */
  previewMap?: Map<string, string>
  onValueChange: (value: string[]) => void
}

function ImageUploadField({
  field,
  value,
  disabled,
  previewMap,
  onValueChange,
}: ImageUploadFieldProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [isDragging, setIsDragging] = useState(false)

  const images = Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []

  const maxCount =
    typeof field.max === 'number' && field.max > 0 ? field.max : undefined

  const appendFiles = useCallback(
    async (files: FileList | File[]) => {
      if (disabled) {
        return
      }

      const incoming = Array.from(files).filter((file) =>
        file.type.startsWith('image/')
      )

      if (incoming.length === 0) {
        toast.error(t('Please upload image files only'))
        return
      }

      if (maxCount && images.length >= maxCount) {
        toast.warning(
          t('Maximum upload count reached: {{count}}', {
            count: maxCount,
          })
        )
        return
      }

      const allowedCount = maxCount
        ? Math.max(0, maxCount - images.length)
        : incoming.length
      const selectedFiles = incoming.slice(0, allowedCount)

      if (incoming.length > selectedFiles.length) {
        toast.warning(
          t('Only {{count}} more images can be added', {
            count: allowedCount,
          })
        )
      }

      if (selectedFiles.length === 0) {
        return
      }

      const valuesAsBase64 = await Promise.all(selectedFiles.map(readFileAsDataUrl))
      onValueChange([...images, ...valuesAsBase64])
    },
    [disabled, images, maxCount, onValueChange, t]
  )

  return (
    <div className='space-y-2'>
      <input
        ref={inputRef}
        type='file'
        accept='image/*'
        multiple
        disabled={disabled}
        className='hidden'
        onChange={async (event) => {
          const files = event.target.files
          if (!files || files.length === 0) {
            return
          }
          await appendFiles(files)
          event.target.value = ''
        }}
      />

      <div
        className={`rounded-md border border-dashed p-3 text-xs ${
          isDragging ? 'border-primary bg-primary/5' : 'border-border'
        }`}
        onDragOver={(event) => {
          event.preventDefault()
          if (!disabled) {
            setIsDragging(true)
          }
        }}
        onDragLeave={(event) => {
          event.preventDefault()
          setIsDragging(false)
        }}
        onDrop={async (event) => {
          event.preventDefault()
          setIsDragging(false)
          if (disabled) {
            return
          }
          await appendFiles(event.dataTransfer.files)
        }}
      >
        <div className='text-muted-foreground'>
          {t('Drag and drop images here, or click Add Images')}
        </div>
        <div className='mt-2 flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            {t('Add Images')}
          </Button>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={disabled || images.length === 0}
            onClick={() => onValueChange([])}
          >
            {t('Clear All')}
          </Button>
        </div>
      </div>

      {typeof maxCount === 'number' && (
        <p className='text-muted-foreground text-xs'>
          {t('Maximum upload count')}: {maxCount}
        </p>
      )}

      {images.length > 0 && (
        <>
          <div className='text-muted-foreground text-xs'>
            {t('Uploaded images')}: {images.length}
          </div>
          <div className='grid grid-cols-3 gap-2'>
            {images.map((item, index) => (
              <div key={`${field.key}-${index}`} className='relative'>
                <img
                  src={previewMap?.get(item) ?? item}
                  alt={`${field.key}-${index}`}
                  className='h-16 w-full rounded-md border object-cover'
                />
                <Button
                  type='button'
                  size='sm'
                  variant='destructive'
                  className='absolute top-1 right-1 h-6 px-2 text-[10px]'
                  disabled={disabled}
                  onClick={() => {
                    onValueChange(images.filter((_, imageIndex) => imageIndex !== index))
                  }}
                >
                  {t('Delete')}
                </Button>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
        return
      }
      reject(new Error('Failed to convert image to base64 data URL'))
    }
    reader.onerror = () => {
      reject(new Error('Failed to read image file'))
    }
    reader.readAsDataURL(file)
  })
}

// ──────────────────────────────────────────────────────────────────────────────
// MediaUploadField — unified image + video upload / URL input / library picker
// ──────────────────────────────────────────────────────────────────────────────

interface MediaUploadFieldProps {
  field: StudioFormField
  value: StudioFormValue
  disabled: boolean
  previewMap?: Map<string, string>
  showAssetLibrary?: boolean
  onValueChange: (value: string[]) => void
  onFromLibrary: () => void
  /** Called with new [storeUrl, previewUrl] pairs to bubble up to the parent map. */
  onPreviewMapUpdate: (entries: [string, string][]) => void
}

function MediaUploadField({
  field,
  value,
  disabled,
  previewMap,
  showAssetLibrary,
  onValueChange,
  onFromLibrary,
  onPreviewMapUpdate,
}: MediaUploadFieldProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [showUrlInput, setShowUrlInput] = useState(false)
  const [urlInput, setUrlInput] = useState('')

  const items = Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []

  const maxCount =
    typeof field.max === 'number' && field.max > 0 ? field.max : undefined

  const appendFiles = useCallback(
    async (files: FileList | File[]) => {
      if (disabled) return
      const incoming = Array.from(files).filter(
        (f) => f.type.startsWith('image/') || f.type.startsWith('video/')
      )
      if (incoming.length === 0) {
        toast.error(t('Please upload image or video files only'))
        return
      }
      if (maxCount && items.length >= maxCount) {
        toast.warning(t('Maximum upload count reached: {{count}}', { count: maxCount }))
        return
      }
      const allowedCount = maxCount ? Math.max(0, maxCount - items.length) : incoming.length
      const selected = incoming.slice(0, allowedCount)
      if (incoming.length > selected.length) {
        toast.warning(t('Only {{count}} more files can be added', { count: allowedCount }))
      }
      if (selected.length === 0) return

      const encoded = await Promise.all(
        selected.map(async (file) => {
          const dataUrl = await readFileAsDataUrl(file)
          // Prefix videos so we can later distinguish them from images
          return file.type.startsWith('video/') ? `${VIDEO_PREFIX}${dataUrl}` : dataUrl
        })
      )
      onValueChange([...items, ...encoded])
    },
    [disabled, items, maxCount, onValueChange, t]
  )

  const handleAddUrl = () => {
    const url = urlInput.trim()
    if (!url) return
    const mediaType = detectMediaTypeFromUrl(url)
    const storeUrl = mediaType === 'video' ? `${VIDEO_PREFIX}${url}` : url
    onValueChange([...items, storeUrl])
    // For external URLs, preview URL is the same as display URL
    if (storeUrl !== url) {
      onPreviewMapUpdate([[storeUrl, url]])
    }
    setUrlInput('')
    setShowUrlInput(false)
  }

  return (
    <div className='space-y-2'>
      <input
        ref={inputRef}
        type='file'
        accept='image/*,video/*'
        multiple
        disabled={disabled}
        className='hidden'
        onChange={async (event) => {
          const files = event.target.files
          if (!files || files.length === 0) return
          await appendFiles(files)
          event.target.value = ''
        }}
      />

      <div
        className={`rounded-md border border-dashed p-3 text-xs ${
          isDragging ? 'border-primary bg-primary/5' : 'border-border'
        }`}
        onDragOver={(event) => {
          event.preventDefault()
          if (!disabled) setIsDragging(true)
        }}
        onDragLeave={(event) => {
          event.preventDefault()
          setIsDragging(false)
        }}
        onDrop={async (event) => {
          event.preventDefault()
          setIsDragging(false)
          if (disabled) return
          await appendFiles(event.dataTransfer.files)
        }}
      >
        <div className='text-muted-foreground'>
          {t('Drag and drop images or videos here, or use the buttons below')}
        </div>
        <div className='mt-2 flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            {t('Upload Files')}
          </Button>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={disabled}
            onClick={() => setShowUrlInput((v) => !v)}
          >
            <LinkIcon className='mr-1 size-3' />
            {t('Add URL')}
          </Button>
          {showAssetLibrary && (
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={disabled}
              onClick={onFromLibrary}
            >
              {t('From Library')}
            </Button>
          )}
          <Button
            type='button'
            variant='ghost'
            size='sm'
            disabled={disabled || items.length === 0}
            onClick={() => onValueChange([])}
          >
            {t('Clear All')}
          </Button>
        </div>

        {showUrlInput && (
          <div className='mt-2 flex gap-2'>
            <Input
              value={urlInput}
              placeholder='https://example.com/image.jpg or video.mp4'
              disabled={disabled}
              className='h-8 text-xs'
              onChange={(e) => setUrlInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  handleAddUrl()
                }
              }}
            />
            <Button
              type='button'
              size='sm'
              disabled={disabled || !urlInput.trim()}
              onClick={handleAddUrl}
            >
              {t('Add')}
            </Button>
          </div>
        )}
      </div>

      {typeof maxCount === 'number' && (
        <p className='text-muted-foreground text-xs'>
          {t('Maximum upload count')}: {maxCount}
        </p>
      )}

      {items.length > 0 && (
        <>
          <div className='text-muted-foreground text-xs'>
            {t('Media items')}: {items.length}
          </div>
          <div className='grid grid-cols-3 gap-2'>
            {items.map((item, index) => {
              const isVideo = isVideoEntry(item)
              const displaySrc = previewMap?.get(item) ?? (isVideo ? stripVideoPrefix(item) : item)
              return (
                <div key={`${field.key}-${index}`} className='relative'>
                  {isVideo ? (
                    <video
                      src={displaySrc}
                      className='h-16 w-full rounded-md border object-cover bg-muted/30'
                      muted
                      preload='metadata'
                    />
                  ) : (
                    <img
                      src={displaySrc}
                      alt={`${field.key}-${index}`}
                      className='h-16 w-full rounded-md border object-cover'
                    />
                  )}
                  {isVideo && (
                    <div className='absolute bottom-1 left-1 rounded bg-black/50 px-1 text-[9px] text-white'>
                      {t('Video')}
                    </div>
                  )}
                  <Button
                    type='button'
                    size='sm'
                    variant='destructive'
                    className='absolute top-1 right-1 h-6 px-2 text-[10px]'
                    disabled={disabled}
                    onClick={() => {
                      onValueChange(items.filter((_, i) => i !== index))
                    }}
                  >
                    {t('Delete')}
                  </Button>
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}
