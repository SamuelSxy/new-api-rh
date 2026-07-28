/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  createUserModelRatio,
  updateUserModelRatio,
  type UserModelRatio,
} from '../api'

interface FormValues {
  model_name: string
  multiplier: number
}

interface UserModelRatioMutateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
  currentRow?: UserModelRatio
}

export function UserModelRatioMutateDialog({
  open,
  onOpenChange,
  userId,
  currentRow,
}: UserModelRatioMutateDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isUpdate = !!currentRow
  const [submitting, setSubmitting] = useState(false)

  const form = useForm<FormValues>({
    defaultValues: {
      model_name: '',
      multiplier: 1.0,
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        model_name: currentRow?.model_name ?? '',
        multiplier: currentRow?.multiplier ?? 1.0,
      })
    }
  }, [open, currentRow, form])

  const mutation = useMutation({
    mutationFn: async (values: FormValues) => {
      if (isUpdate && currentRow) {
        return updateUserModelRatio({
          id: currentRow.id,
          user_id: userId,
          model_name: values.model_name.trim(),
          multiplier: values.multiplier,
        })
      }
      return createUserModelRatio({
        user_id: userId,
        model_name: values.model_name.trim(),
        multiplier: values.multiplier,
      })
    },
    onSuccess: () => {
      toast.success(
        isUpdate
          ? t('User model ratio override updated')
          : t('User model ratio override created')
      )
      queryClient.invalidateQueries({
        queryKey: ['user-model-ratios', userId],
      })
      onOpenChange(false)
    },
    onError: (err: unknown) => {
      const msg =
        err instanceof Error ? err.message : 'Failed to save override'
      toast.error(msg)
    },
    onSettled: () => setSubmitting(false),
  })

  const onSubmit = (values: FormValues) => {
    setSubmitting(true)
    mutation.mutate(values)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {isUpdate ? t('Edit Override') : t('Add Override')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Set a per-model multiplier for this user. 1.0 = no change, 0.8 = 20% off, 1.2 = 20% surcharge, 0 = free.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4 py-2"
          >
            <FormField
              control={form.control}
              name="model_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Model Name')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder="gpt-4"
                      disabled={isUpdate}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="multiplier"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Multiplier')}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="0.01"
                      min="0"
                      {...field}
                      onChange={(e) =>
                        field.onChange(parseFloat(e.target.value) || 0)
                      }
                      value={field.value}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="outline" />}>
                {t('Cancel')}
              </DialogClose>
              <Button type="submit" disabled={submitting}>
                {isUpdate ? t('Save') : t('Create')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
