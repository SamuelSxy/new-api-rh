import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/ui/tabs'
import { Loader2 } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { getStudioModels, getUserModels } from '../api'
import { STUDIO_TABS, type StudioTab } from '../constants'
import type { ModelOption, StudioModelType } from '../types'
import { ScriptTab } from './script-tab'
import { ImageTab } from './image-tab'
import { VoiceTab } from './voice-tab'
import { VideoTab } from './video-tab'
import { StudioAdminPanel } from './studio-admin-panel'

export function Studio() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const [models, setModels] = useState<ModelOption[]>([])
  const [activeTab, setActiveTab] = useState<StudioTab>(STUDIO_TABS.SCRIPT)
  const [isLoadingModels, setIsLoadingModels] = useState(true)
  const canManageStudio = (user?.role ?? 0) >= ROLE.ADMIN

  useEffect(() => {
    setIsLoadingModels(true)
    getStudioModels(activeTab as StudioModelType)
      .then(async (studioModels) => {
        if (studioModels.length > 0) {
          setModels(studioModels)
          return
        }
        const fallbackModels = await getUserModels()
        setModels(fallbackModels)
      })
      .finally(() => setIsLoadingModels(false))
  }, [activeTab])

  if (isLoadingModels) {
    return (
      <div className='flex h-40 items-center justify-center'>
        <Loader2 className='size-6 animate-spin' />
      </div>
    )
  }

  return (
    <div className='relative flex size-full flex-col overflow-hidden'>
      <div className='mx-auto flex w-full max-w-4xl flex-1 flex-col gap-5 px-2 py-4 md:px-4'>
      <div>
        <h1 className='text-2xl font-semibold'>{t('Studio')}</h1>
        <p className='text-muted-foreground text-sm mt-1'>
          {t('Create scripts, images, voiceovers, and videos with AI')}
        </p>
      </div>

      {canManageStudio && <StudioAdminPanel />}

      <Tabs defaultValue={STUDIO_TABS.SCRIPT} value={activeTab} onValueChange={(value) => setActiveTab(value as StudioTab)}>
        <TabsList className='w-full justify-start overflow-x-auto'>
          <TabsTrigger value={STUDIO_TABS.SCRIPT}>{t('Generate Script')}</TabsTrigger>
          <TabsTrigger value={STUDIO_TABS.IMAGE}>{t('Generate Image')}</TabsTrigger>
          <TabsTrigger value={STUDIO_TABS.VOICE}>{t('Generate Voiceover')}</TabsTrigger>
          <TabsTrigger value={STUDIO_TABS.VIDEO}>{t('Generate Video')}</TabsTrigger>
        </TabsList>

        <div className='mt-6'>
          <TabsContent value={STUDIO_TABS.SCRIPT}>
            <ScriptTab models={models} />
          </TabsContent>
          <TabsContent value={STUDIO_TABS.IMAGE}>
            <ImageTab models={models} />
          </TabsContent>
          <TabsContent value={STUDIO_TABS.VOICE}>
            <VoiceTab models={models} />
          </TabsContent>
          <TabsContent value={STUDIO_TABS.VIDEO}>
            <VideoTab models={models} />
          </TabsContent>
        </div>
      </Tabs>
      </div>
    </div>
  )
}
