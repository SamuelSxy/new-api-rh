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
import { EXCLUDED_GROUPS, QUOTA_TYPE_VALUES } from '../constants'
import type { PricingModel } from '../types'

// ----------------------------------------------------------------------------
// Model Helper Utilities
// ----------------------------------------------------------------------------

/**
 * Get available groups for a model
 */
export function getAvailableGroups(
  model: PricingModel,
  usableGroup: Record<string, { desc: string; ratio: number }>
): string[] {
  const modelEnableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []

  return Object.keys(usableGroup)
    .filter((g) => !EXCLUDED_GROUPS.includes(g))
    .filter((g) => modelEnableGroups.includes(g))
}

/**
 * Replace model placeholder in endpoint path
 */
export function replaceModelInPath(path: string, modelName: string): string {
  return path.replace(/\{model\}/g, modelName)
}

/**
 * Check if model is token-based pricing
 */
export function isTokenBasedModel(model: PricingModel): boolean {
  return model.quota_type === QUOTA_TYPE_VALUES.TOKEN
}

/**
 * Check if model uses per-second (duration-based) billing
 */
export function isDurationBillingModel(model: PricingModel): boolean {
  return model.billing_mode === 'per-second'
}

/**
 * Check if a model is a named variant (model_name contains '@')
 */
export function isVariantModel(model: PricingModel): boolean {
  return (model.model_name ?? '').includes('@')
}

/**
 * Get the base model name (strips @suffix, e.g. "model@720p" → "model")
 */
export function getBaseModelName(modelName: string): string {
  const at = modelName.indexOf('@')
  return at === -1 ? modelName : modelName.slice(0, at)
}

/**
 * Extract the variant label (part after '@', e.g. "model@720p" → "720p")
 */
export function extractVariantLabel(modelName: string): string {
  const at = modelName.indexOf('@')
  return at === -1 ? '' : modelName.slice(at + 1)
}

/**
 * Group @variant models under their base model's `variants` field.
 * Variant models (those containing '@' whose base exists) are removed from the
 * top-level list and attached to their parent. Preserves original ordering.
 */
export function groupModelVariants(models: PricingModel[]): PricingModel[] {
  const baseMap = new Map<string, PricingModel>()
  for (const model of models) {
    if (!isVariantModel(model)) {
      baseMap.set(model.model_name ?? '', { ...model })
    }
  }
  for (const model of models) {
    if (isVariantModel(model)) {
      const baseName = getBaseModelName(model.model_name ?? '')
      const base = baseMap.get(baseName)
      if (base) {
        base.variants = [...(base.variants ?? []), model]
      }
    }
  }
  return models
    .filter((m) => !isVariantModel(m))
    .map((m) => baseMap.get(m.model_name ?? '') ?? m)
}
