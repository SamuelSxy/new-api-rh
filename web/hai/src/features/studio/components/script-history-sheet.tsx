import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import dayjs from '@/lib/dayjs'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Trash2, RotateCcw, Clock, ChevronDown, ChevronUp } from 'lucide-react'
import type { ScriptHistoryItem } from '../hooks/use-script-history'

interface ScriptHistorySheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  history: ScriptHistoryItem[]
  onRestore: (item: ScriptHistoryItem) => void
  onDelete: (id: string) => void
  onClearAll: () => void
}

function HistoryCard({
  item,
  onRestore,
  onDelete,
}: {
  item: ScriptHistoryItem
  onRestore: (item: ScriptHistoryItem) => void
  onDelete: (id: string) => void
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  const timeAgo = dayjs(item.timestamp).fromNow()

  return (
    <div className='rounded-xl border bg-card p-3 space-y-2'>
      <div className='flex items-start justify-between gap-2'>
        <div className='flex items-center gap-1.5 text-xs text-muted-foreground min-w-0'>
          <Clock className='size-3 shrink-0' />
          <span className='truncate'>{timeAgo}</span>
          <span className='shrink-0 text-muted-foreground/60'>·</span>
          <span className='truncate font-mono text-xs text-muted-foreground/80 max-w-[120px]'>
            {item.model}
          </span>
        </div>
        <div className='flex items-center gap-1 shrink-0'>
          <Button
            variant='ghost'
            size='icon'
            className='size-7'
            title={t('Restore')}
            onClick={() => onRestore(item)}
          >
            <RotateCcw className='size-3.5' />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            className='size-7 text-destructive hover:text-destructive'
            title={t('Delete')}
            onClick={() => onDelete(item.id)}
          >
            <Trash2 className='size-3.5' />
          </Button>
        </div>
      </div>

      <div>
        <p className='text-sm font-medium line-clamp-2'>{item.prompt}</p>
      </div>

      {item.output && (
        <div>
          <button
            type='button'
            className='flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors'
            onClick={() => setExpanded((v) => !v)}
          >
            {expanded ? (
              <>
                <ChevronUp className='size-3' />
                {t('Hide output')}
              </>
            ) : (
              <>
                <ChevronDown className='size-3' />
                {t('Show output')}
              </>
            )}
          </button>
          {expanded && (
            <div className='mt-2 rounded-lg bg-muted/60 p-2.5 text-xs whitespace-pre-wrap max-h-48 overflow-y-auto'>
              {item.output}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function ScriptHistorySheet({
  open,
  onOpenChange,
  history,
  onRestore,
  onDelete,
  onClearAll,
}: ScriptHistorySheetProps) {
  const { t } = useTranslation()

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col w-full sm:max-w-md p-0'>
        <SheetHeader className='px-4 pt-4 pb-3 border-b'>
          <div className='flex items-center justify-between'>
            <SheetTitle className='text-base'>
              {t('Script History')} ({history.length})
            </SheetTitle>
            {history.length > 0 && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant='ghost' size='sm' className='text-destructive hover:text-destructive h-7 px-2 text-xs'>
                    <Trash2 className='size-3 mr-1' />
                    {t('Clear all')}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{t('Clear all history?')}</AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('This will permanently delete all script history from your browser. This action cannot be undone.')}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={onClearAll}
                      className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                    >
                      {t('Clear all')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>
        </SheetHeader>

        <ScrollArea className='flex-1 px-4 py-3'>
          {history.length === 0 ? (
            <div className='flex flex-col items-center justify-center h-40 text-muted-foreground text-sm gap-2'>
              <Clock className='size-8 opacity-30' />
              <p>{t('No history yet')}</p>
              <p className='text-xs opacity-70'>{t('Generated scripts will be saved here automatically')}</p>
            </div>
          ) : (
            <div className='space-y-2.5 pb-4'>
              {history.map((item) => (
                <HistoryCard
                  key={item.id}
                  item={item}
                  onRestore={onRestore}
                  onDelete={onDelete}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
