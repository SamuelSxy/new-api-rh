import { useRef, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ImageIcon, VideoIcon, TrashIcon, UploadIcon, CheckIcon, Loader2Icon } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { uploadUserAsset, listUserAssets, deleteUserAsset, syncAssetArkStatus } from '../api'
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

  const { data, isLoading } = useQuery({
    queryKey: ['studio-assets', tab],
    queryFn: () => listUserAssets(tab),
    enabled: open,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUserAsset(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['studio-assets', tab] })
      toast.success(t('Asset deleted'))
    },
    onError: () => {
      toast.error(t('Failed to delete asset'))
    },
  })

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

  const accept = tab === 'Image' ? 'image/*' : 'video/*'
  const assets = data?.items ?? []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[80vh] max-w-3xl overflow-hidden flex flex-col gap-0 p-0'>
        <DialogHeader className='px-6 pt-6 pb-4 border-b'>
          <DialogTitle>{t('Asset Library')}</DialogTitle>
        </DialogHeader>

        <Tabs
          value={tab}
          onValueChange={(v) => setTab(v as 'Image' | 'Video')}
          className='flex flex-col flex-1 overflow-hidden'
        >
          <div className='flex items-center justify-between px-6 py-3 border-b'>
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

            <div>
              <input
                ref={fileInputRef}
                type='file'
                accept={accept}
                className='hidden'
                onChange={handleFileChange}
              />
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

          <TabsContent value='Image' className='flex-1 overflow-y-auto p-4 mt-0'>
            <AssetGrid
              assets={assets}
              isLoading={isLoading}
              assetType='Image'
              onSelect={(asset) => {
                onSelect(asset)
                onOpenChange(false)
              }}
              onDelete={(id) => deleteMutation.mutate(id)}
            />
          </TabsContent>

          <TabsContent value='Video' className='flex-1 overflow-y-auto p-4 mt-0'>
            <AssetGrid
              assets={assets}
              isLoading={isLoading}
              assetType='Video'
              onSelect={(asset) => {
                onSelect(asset)
                onOpenChange(false)
              }}
              onDelete={(id) => deleteMutation.mutate(id)}
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
  onSelect: (asset: UserAsset) => void
  onDelete: (id: number) => void
}

function AssetGrid({ assets, isLoading, assetType, onSelect, onDelete }: AssetGridProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  // Poll every 5 s for assets that are still Processing.
  // Stops automatically once all settle (Active / Failed).
  useEffect(() => {
    const processing = assets.filter(
      (a) => a.ark_asset_id && a.ark_status === 'Processing'
    )
    if (processing.length === 0) return

    let attempts = 0
    const MAX_ATTEMPTS = 120 // 10 minutes

    const timer = setInterval(async () => {
      attempts++
      const stillProcessing: number[] = []

      await Promise.all(
        processing.map(async (asset) => {
          const status = await syncAssetArkStatus(asset.id)
          if (status && status !== asset.ark_status) {
            // Patch the cached query data in-place so we avoid a full refetch.
            queryClient.setQueryData(
              ['studio-assets', assetType],
              (old: { items: UserAsset[]; total: number; page: number; page_size: number } | undefined) => {
                if (!old) return old
                return {
                  ...old,
                  items: old.items.map((item) =>
                    item.id === asset.id ? { ...item, ark_status: status } : item
                  ),
                }
              }
            )
          }
          if (!status || status === 'Processing') {
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
    <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4'>
      {assets.map((asset) => (
        <AssetCard
          key={asset.id}
          asset={asset}
          confirmDelete={confirmDelete === asset.id}
          onSelect={() => onSelect(asset)}
          onDeleteRequest={() => setConfirmDelete(asset.id)}
          onDeleteConfirm={() => {
            onDelete(asset.id)
            setConfirmDelete(null)
          }}
          onDeleteCancel={() => setConfirmDelete(null)}
        />
      ))}
    </div>
  )
}

interface AssetCardProps {
  asset: UserAsset
  confirmDelete: boolean
  onSelect: () => void
  onDeleteRequest: () => void
  onDeleteConfirm: () => void
  onDeleteCancel: () => void
}

function AssetCard({
  asset,
  confirmDelete,
  onSelect,
  onDeleteRequest,
  onDeleteConfirm,
  onDeleteCancel,
}: AssetCardProps) {
  const { t } = useTranslation()

  return (
    <div className='group relative rounded-lg border overflow-hidden bg-muted/20 hover:border-primary transition-colors'>
      {/* Thumbnail */}
      <div className='aspect-video flex items-center justify-center overflow-hidden bg-muted/40'>
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

      {/* Ark status badge — shown in top-right corner of thumbnail */}
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

      {/* Actions overlay */}
      {confirmDelete ? (
        <div className='absolute inset-0 flex flex-col items-center justify-center gap-2 bg-background/90'>
          <p className='text-xs font-medium'>{t('Delete this asset?')}</p>
          <div className='flex gap-2'>
            <Button size='sm' variant='destructive' onClick={onDeleteConfirm}>
              {t('Confirm')}
            </Button>
            <Button size='sm' variant='outline' onClick={onDeleteCancel}>
              {t('Cancel')}
            </Button>
          </div>
        </div>
      ) : (
        <div className='absolute inset-0 flex items-center justify-center gap-2 opacity-0 group-hover:opacity-100 bg-background/70 transition-opacity'>
          <Button
            size='sm'
            onClick={onSelect}
            className='h-8 px-3'
            disabled={Boolean(asset.ark_asset_id) && asset.ark_status !== 'Active'}
          >
            <CheckIcon className='mr-1 size-3' />
            {t('Use')}
          </Button>
          <Button
            size='sm'
            variant='destructive'
            onClick={onDeleteRequest}
            className='h-8 px-3'
          >
            <TrashIcon className='size-3' />
          </Button>
        </div>
      )}
    </div>
  )
}
