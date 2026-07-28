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
import { api } from '@/lib/api'

export interface UserModelRatio {
  id: number
  user_id: number
  model_name: string
  multiplier: number
  created_at: number
  updated_at: number
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface PaginatedUserModelRatios {
  items: UserModelRatio[]
  total: number
  page: number
  page_size: number
}

export interface GetUserModelRatiosParams {
  user_id?: number
  model_name?: string
  p?: number
  page_size?: number
}

export interface UserModelRatioFormData {
  user_id: number
  model_name: string
  multiplier: number
}

export async function getUserModelRatios(
  params: GetUserModelRatiosParams = {}
): Promise<ApiResponse<PaginatedUserModelRatios>> {
  const { user_id, model_name, p = 1, page_size = 20 } = params
  const queryParams = new URLSearchParams()
  if (user_id) queryParams.set('user_id', String(user_id))
  if (model_name) queryParams.set('model_name', model_name)
  queryParams.set('p', String(p))
  queryParams.set('page_size', String(page_size))
  const res = await api.get(`/api/user_model_ratio/?${queryParams.toString()}`)
  return res.data
}

export async function getUserModelRatiosByUserId(
  userId: number
): Promise<ApiResponse<UserModelRatio[]>> {
  const res = await api.get(`/api/user_model_ratio/${userId}`)
  return res.data
}

export async function createUserModelRatio(
  data: UserModelRatioFormData
): Promise<ApiResponse<UserModelRatio>> {
  const res = await api.post('/api/user_model_ratio/', data)
  return res.data
}

export async function updateUserModelRatio(
  data: UserModelRatioFormData & { id: number }
): Promise<ApiResponse<UserModelRatio>> {
  const res = await api.put('/api/user_model_ratio/', data)
  return res.data
}

export async function deleteUserModelRatio(
  id: number
): Promise<ApiResponse<UserModelRatio>> {
  const res = await api.delete(`/api/user_model_ratio/${id}`)
  return res.data
}
