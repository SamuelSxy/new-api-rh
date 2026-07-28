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
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  deleteUserModelRatio,
  getUserModelRatiosByUserId,
  type UserModelRatio,
} from '../api'
import { UserModelRatioMutateDialog } from './user-model-ratio-mutate-dialog'

interface UserModelRatiosTableProps {
  userId: number
}

export function UserModelRatiosTable({ userId }: UserModelRatiosTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [currentRow, setCurrentRow] = useState<UserModelRatio | undefined>()

  const { data, isLoading } = useQuery({
    queryKey: ['user-model-ratios', userId],
    queryFn: () => getUserModelRatiosByUserId(userId),
    enabled: userId > 0,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUserModelRatio(id),
    onSuccess: () => {
      toast.success(t('User model ratio override deleted'))
      queryClient.invalidateQueries({
        queryKey: ['user-model-ratios', userId],
      })
    },
    onError: (err: unknown) => {
      const msg =
        err instanceof Error ? err.message : 'Failed to delete override'
      toast.error(msg)
    },
  })

  const handleAdd = () => {
    setCurrentRow(undefined)
    setDialogOpen(true)
  }

  const handleEdit = (row: UserModelRatio) => {
    setCurrentRow(row)
    setDialogOpen(true)
  }

  const handleDelete = (row: UserModelRatio) => {
    deleteMutation.mutate(row.id)
  }

  const items = data?.data ?? []

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {t(
            'Configure per-model multiplier overrides for this user. Final ratio = global model ratio × group ratio × user multiplier.'
          )}
        </p>
        <Button size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t('Add Override')}
        </Button>
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Model Name')}</TableHead>
              <TableHead className="w-[140px]">{t('Multiplier')}</TableHead>
              <TableHead className="w-[120px] text-right">
                {t('Actions')}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell
                  colSpan={3}
                  className="py-6 text-center text-muted-foreground"
                >
                  {t('Loading...')}
                </TableCell>
              </TableRow>
            ) : items.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={3}
                  className="py-6 text-center text-muted-foreground"
                >
                  {t('No overrides configured')}
                </TableCell>
              </TableRow>
            ) : (
              items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-medium">
                    {row.model_name}
                  </TableCell>
                  <TableCell>{row.multiplier}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleEdit(row)}
                        title={t('Edit')}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(row)}
                        title={t('Delete')}
                        disabled={deleteMutation.isPending}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <UserModelRatioMutateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        userId={userId}
        currentRow={currentRow}
      />
    </div>
  )
}
