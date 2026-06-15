import { useRef, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ImageIcon, VideoIcon, UploadIcon, CheckIcon, Loader2Icon, LinkIcon, ChevronLeftIcon, ChevronRightIcon, TrashIcon } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { uploadUserAsset, uploadUserAssetByUrl, listUserAssets, deleteUserAsset, syncAssetArkStatus } from '../api'
import type { UserAsset } from '../types'

interface AssetLibraryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Which tab to show first */
  defaultAssetType?: 'Image' | 'Video'
  /** Called when the user clicks "Use" on an asset */
  onSelect: (asset: UserAsset) => void
}

export function AssetLibraryDialog({
  open,
  onOpenChange,
  defaultAssetType = 'Image',
  onSelect,
}: AssetLibraryDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<'Image' | 'Video'>(defaultAssetType)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const [uploading, setUploading] = useState(false)
  const [showUrlInput, setShowUrlInput] = useState(false)
  const [urlInput, setUrlInput] = useState('')
  const [page, setPage] = useState(1)
  const PAGE_SIZE = 20

  // Reset page when switching tabs
  useEffect(() => {
    setPage(1)
  }, [tab])

  const { data, isLoading } = useQuery({
    queryKey: ['studio-assets', tab, page],
    queryFn: () => listUserAssets(tab, page, PAGE_SIZE),
    enabled: open,
  })

  const handleDelete = async (id: number) => {
    const ok = await deleteUserAsset(id)
    if (ok) {
      void queryClient.invalidateQueries({ queryKey: ['studio-assets', tab] })
      toast.success(t('Asset deleted'))
    } else {
      toast.error(t('Failed to delete asset'))
    }
  }

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files
    if (!files || files.length === 0) return
    const file = files[0]
    event.target.value = ''

    setUploading(true)
    try {
      const result = await uploadUserAsset(file, file.name, tab)
      if (result) {
        void queryClient.invalidateQueries({ queryKey: ['studio-assets', tab] })
        toast.success(t('Asset uploaded successfully'))
      } else {
        toast.error(t('Upload failed'))
      }
    } catch {
      toast.error(t('Upload failed'))
    } finally {
      setUploading(false)
    }
  }

  const handleUrlUpload = async () => {
    const url = urlInput.trim()
    if (!url) return
    setUploading(true)
    try {
      const result = await uploadUserAssetByUrl(url, '', tab)
      if (result) {
        void queryClient.invalidateQueries({ queryKey: ['studio-assets', tab] })
        toast.success(t('Asset uploaded successfully'))
        setUrlInput('')
        setShowUrlInput(false)
      } else {
        toast.error(t('Upload failed'))
      }
    } catch {
      toast.error(t('Upload failed'))
    } finally {
      setUploading(false)
    }
  }

  const accept = tab === 'Image' ? 'image/*' : 'video/*'
  const assets = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[85vh] max-w-4xl overflow-hidden flex flex-col gap-0 p-0'>
        <DialogHeader className='px-6 pt-6 pb-4 border-b'>
          <DialogTitle>{t('Asset Library')}</DialogTitle>
        </DialogHeader>

        <Tabs
          value={tab}
          onValueChange={(v) => setTab(v as 'Image' | 'Video')}
          className='flex flex-col flex-1 overflow-hidden'
        >
          <div className='flex flex-col gap-2 px-6 py-3 border-b'>
            <div className='flex items-center justify-between'>
              <TabsList>
                <TabsTrigger value='Image'>
                  <ImageIcon className='mr-1 size-4' />
                  {t('Images')}
                </TabsTrigger>
                <TabsTrigger value='Video'>
                  <VideoIcon className='mr-1 size-4' />
                  {t('Videos')}
                </TabsTrigger>
              </TabsList>

              <div className='flex gap-2'>
                <input
                  ref={fileInputRef}
                  type='file'
                  accept={accept}
                  className='hidden'
                  onChange={handleFileChange}
                />
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => setShowUrlInput((v) => !v)}
                  disabled={uploading}
                >
                  <LinkIcon className='mr-1 size-4' />
                  {t('URL')}
                </Button>
                <Button
                  size='sm'
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploading}
                >
                  <UploadIcon className='mr-1 size-4' />
                  {uploading ? t('Uploading...') : t('Upload')}
                </Button>
              </div>
            </div>

            {showUrlInput && (
              <div className='flex gap-2'>
                <Input
                  value={urlInput}
                  placeholder='https://example.com/image.jpg'
                  className='h-8 text-xs'
                  onChange={(e) => setUrlInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void handleUrlUpload()
                    }
                  }}
                />
                <Button
                  size='sm'
                  disabled={uploading || !urlInput.trim()}
                  onClick={() => void handleUrlUpload()}
                >
                  {uploading ? <Loader2Icon className='size-4 animate-spin' /> : t('Save to Library')}
                </Button>
              </div>
            )}
          </div>

          <TabsContent value='Image' className='flex-1 overflow-y-auto p-4 mt-0'>
            <AssetGrid
              assets={assets}
              isLoading={isLoading}
              assetType='Image'
              page={page}
              totalPages={totalPages}
              total={total}
              onPageChange={setPage}
              onSelect={(asset) => {
                onSelect(asset)
                onOpenChange(false)
              }}
              onDelete={(id) => void handleDelete(id)}
            />
          </TabsContent>

          <TabsContent value='Video' className='flex-1 overflow-y-auto p-4 mt-0'>
            <AssetGrid
              assets={assets}
              isLoading={isLoading}
              assetType='Video'
              page={page}
              totalPages={totalPages}
              total={total}
              onPageChange={setPage}
              onSelect={(asset) => {
                onSelect(asset)
                onOpenChange(false)
              }}
              onDelete={(id) => void handleDelete(id)}
            />
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

interface AssetGridProps {
  assets: UserAsset[]
  isLoading: boolean
  assetType: 'Image' | 'Video'
  page: number
  totalPages: number
  total: number
  onPageChange: (page: number) => void
  onSelect: (asset: UserAsset) => void
  onDelete: (id: number) => void
}

function AssetGrid({ assets, isLoading, assetType, page, totalPages, total, onPageChange, onSelect, onDelete }: AssetGridProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  // Poll every 5 s for assets that are still Processing.
  useEffect(() => {
    const processing = assets.filter(
      (a) => a.ark_asset_id && a.ark_status === 'Processing'
    )
    if (processing.length === 0) return

    let attempts = 0
    const MAX_ATTEMPTS = 120

    const timer = setInterval(async () => {
      attempts++
      const stillProcessing: number[] = []
      await Promise.all(
        processing.map(async (asset) => {
          const updated = await syncAssetArkStatus(asset.id)
          if (updated && updated.ark_status !== asset.ark_status) {
            queryClient.setQueryData(
              ['studio-assets', assetType, page],
              (old: { items: UserAsset[]; total: number; page: number; page_size: number } | undefined) => {
                if (!old) return old
                return {
                  ...old,
                  items: old.items.map((item) =>
                    item.id === asset.id ? { ...item, ...updated } : item
                  ),
                }
              }
            )
          }
          if (!updated || updated.ark_status === 'Processing') {
            stillProcessing.push(asset.id)
          }
        })
      )
      if (stillProcessing.length === 0 || attempts >= MAX_ATTEMPTS) {
        clearInterval(timer)
      }
    }, 5000)

    return () => clearInterval(timer)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(assets.filter((a) => a.ark_asset_id && a.ark_status === 'Processing').map((a) => a.id))])

  if (isLoading) {
    return (
      <div className='py-12 text-center text-muted-foreground text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  if (assets.length === 0) {
    return (
      <div className='py-12 text-center text-muted-foreground text-sm'>
        {assetType === 'Image' ? t('No images uploaded yet') : t('No videos uploaded yet')}
      </div>
    )
  }

  return (
    <div className='flex flex-col gap-4'>
      <div className='grid grid-cols-3 gap-3'>
        {assets.map((asset) => (
          <AssetCard
            key={asset.id}
            asset={asset}
            onSelect={() => onSelect(asset)}
            onDelete={() => onDelete(asset.id)}
          />
        ))}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className='flex items-center justify-center gap-2 pt-1'>
          <Button
            size='sm'
            variant='outline'
            className='h-7 w-7 p-0'
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            <ChevronLeftIcon className='size-4' />
          </Button>
          <span className='text-xs text-muted-foreground'>
            {page} / {totalPages}
            <span className='ml-1 text-[10px]'>({total})</span>
          </span>
          <Button
            size='sm'
            variant='outline'
            className='h-7 w-7 p-0'
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            <ChevronRightIcon className='size-4' />
          </Button>
        </div>
      )}
    </div>
  )
}

interface AssetCardProps {
  asset: UserAsset
  onSelect: () => void
  onDelete: () => void
}

function AssetCard({ asset, onSelect, onDelete }: AssetCardProps) {
  const { t } = useTranslation()
  const [confirmDelete, setConfirmDelete] = useState(false)

  return (
    <div className='group relative rounded-lg border overflow-hidden bg-muted/20 hover:border-primary transition-colors'>
      {/* Thumbnail */}
      <div className='aspect-square flex items-center justify-center overflow-hidden bg-muted/40'>
        {asset.asset_type === 'Image' ? (
          <img
            src={asset.source_url}
            alt={asset.name}
            className='h-full w-full object-cover'
          />
        ) : (
          <video
            src={asset.source_url}
            className='h-full w-full object-cover'
            muted
            preload='metadata'
          />
        )}
      </div>

      {/* Name */}
      <div className='px-2 py-1.5'>
        <p className='truncate text-xs font-medium' title={asset.name}>
          {asset.name}
        </p>
        {asset.ark_asset_uri && (
          <p className='truncate text-[10px] text-muted-foreground' title={asset.ark_asset_uri}>
            {asset.ark_asset_uri}
          </p>
        )}
      </div>

      {/* Ark status badge */}
      {asset.ark_asset_id && (
        <div className='absolute top-1 right-1'>
          {asset.ark_status === 'Active' ? (
            <span className='rounded px-1.5 py-0.5 text-[10px] font-medium bg-green-500/80 text-white'>
              {t('Active')}
            </span>
          ) : asset.ark_status === 'Failed' ? (
            <span className='rounded px-1.5 py-0.5 text-[10px] font-medium bg-destructive text-destructive-foreground'>
              {t('Review failed')}
            </span>
          ) : (
            <span className='flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium bg-black/60 text-white'>
              <Loader2Icon className='size-3 animate-spin' />
              {t('Processing')}
            </span>
          )}
        </div>
      )}

      {/* Hover overlay */}
      {confirmDelete ? (
        <div className='absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/90'>
          <p className='text-xs font-medium'>{t('Delete this asset?')}</p>
          <div className='flex gap-2'>
            <Button size='sm' variant='destructive' onClick={() => { onDelete(); setConfirmDelete(false) }}>
              {t('Confirm')}
            </Button>
            <Button size='sm' variant='outline' onClick={() => setConfirmDelete(false)}>
              {t('Cancel')}
            </Button>
          </div>
        </div>
      ) : (
        <div className='absolute inset-0 flex flex-col items-center justify-center gap-2 opacity-0 group-hover:opacity-100 bg-background/70 transition-opacity'>
          <Button
            size='sm'
            onClick={onSelect}
            disabled={Boolean(asset.ark_asset_id) && asset.ark_status !== 'Active'}
          >
            <CheckIcon className='mr-1 size-3' />
            {t('Use')}
          </Button>
          <Button size='sm' variant='destructive' onClick={() => setConfirmDelete(true)}>
            <TrashIcon className='mr-1 size-3' />
            {t('Delete')}
          </Button>
        </div>
      )}
    </div>
  )
}
