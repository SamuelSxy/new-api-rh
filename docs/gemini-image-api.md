# Gemini 图像生成 API 文档

## 接口地址

```
POST https://ai-gc.net/v1/images/generations
```

## 认证

```
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json
```

## 支持模型

| 模型名称 | 说明 |
|---|---|
| `gemini-3.1-flash-image` | Gemini 3.1 Flash 图像生成 |
| `gemini-3.1-flash-image-preview` | Gemini 3.1 Flash 图像生成（预览版）|
| `gemini-3-pro-image` | Gemini 3 Pro 图像生成 |
| `gemini-2.5-flash-image` | Gemini 2.5 Flash 图像生成 |
| `gemini-2.0-flash-preview-image-generation` | Gemini 2.0 Flash 图像生成（预览版）|

---

## 请求参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `model` | string | 是 | — | 模型名称 |
| `prompt` | string | 是 | — | 图像描述文本 |
| `size` | string | 否 | `1024x1024` | 图像尺寸，用于映射宽高比，见下表 |
| `quality` | string | 否 | `standard` | `standard`（1K）/ `hd`（2K）|
| `n` | integer | 否 | `1` | 生成数量（目前固定为 1）|
| `image` | string/array | 否 | — | 参考图片（图生图），Base64 Data URI 或 URL |
| `seed` | integer | 否 | — | 随机种子（`0-4294967295`），相同种子可复现结果。仅 Pro 模型支持 |
| `enhance_prompt` | boolean | 否 | `true` | 是否启用提示词优化。仅 Pro 模型支持 |
| `negative_prompt` | string | 否 | — | 负面提示词（描述不希望出现的元素）。仅 Pro 模型支持 |

### size 与宽高比映射

| size 值 | 宽高比 |
|---|---|
| `1024x1024` | 1:1 |
| `1792x1024` | 16:9 |
| `1024x1792` | 9:16 |
| `1024x1280` | 4:5 |
| `1280x1024` | 5:4 |

### 各模型参数支持情况

| 参数 | Flash 系列 | Pro 系列 |
|---|---|---|
| `prompt` / `size` / `quality` / `image` | ✅ | ✅ |
| `seed` | ❌ | ✅ |
| `enhance_prompt` | ❌ | ✅ |
| `negative_prompt` | ❌ | ✅ |
| `quality: hd`（2K）| ✅ | ✅ |
| `quality: standard`（1K）| ✅ | ✅ |
| `size: 512`（512px）| ✅ | ❌ |

---

## 模式一：文生图 (Text-to-Image)

**请求示例：**

```bash
curl -X POST https://ai-gc.net/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3.1-flash-image-preview",
    "prompt": "一只橘猫坐在窗台上看夕阳，写实风格",
    "size": "1792x1024",
    "quality": "standard"
  }'
```

**请求体：**

```json
{
  "model": "gemini-3.1-flash-image-preview",
  "prompt": "一只橘猫坐在窗台上看夕阳，写实风格",
  "size": "1792x1024",
  "quality": "standard"
}
```

---

## 模式二：图片编辑 (Image Editing)

将参考图片通过 `image` 字段传入，模型将根据 `prompt` 对图片进行编辑。

> 图片编辑模式下，若未指定 `size`，Pro 模型会自动匹配输入图像的宽高比。

**请求示例（Base64 Data URI）：**

```bash
curl -X POST https://ai-gc.net/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3.1-flash-image-preview",
    "prompt": "把这张图片改成水彩画风格",
    "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
  }'
```

**请求体：**

```json
{
  "model": "gemini-3.1-flash-image-preview",
  "prompt": "把这张图片改成水彩画风格",
  "image": "data:image/png;base64,<Base64数据>"
}
```

> `image` 字段支持单张（字符串）或多张（字符串数组）。支持 `data:image/jpeg`、`data:image/png`、`data:image/webp` 格式。

---

## 模式三：Pro 模型高级参数示例

仅适用于 `gemini-3-pro-image` / `gemini-3-pro-image-preview`。

**文生图（含负面提示词与种子）：**

```bash
curl -X POST https://ai-gc.net/v1/images/generations \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3-pro-image",
    "prompt": "a majestic snow mountain at golden hour, photorealistic, 8K",
    "size": "1792x1024",
    "quality": "hd",
    "negative_prompt": "blurry, low quality, cartoon",
    "seed": 42,
    "enhance_prompt": true
  }'
```

```json
{
  "model": "gemini-3-pro-image",
  "prompt": "a majestic snow mountain at golden hour, photorealistic, 8K",
  "size": "1792x1024",
  "quality": "hd",
  "negative_prompt": "blurry, low quality, cartoon",
  "seed": 42,
  "enhance_prompt": true
}
```

---

## 响应格式

**成功响应（HTTP 200）：**

```json
{
  "created": 1750034962,
  "data": [
    {
      "url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
    }
  ]
}
```

| 字段 | 说明 |
|---|---|
| `created` | Unix 时间戳 |
| `data[0].url` | 生成的图片，`data:image/...;base64,...` 格式 |

**失败响应（示例）：**

```json
{
  "error": {
    "message": "no image returned from Gemini",
    "type": "no_image_in_response",
    "code": 502
  }
}
```

---

## 任务日志

通过本接口生成的图片会自动记录到任务日志，可在管理后台 **日志 → 任务日志** 中查看，状态为 `SUCCESS`，支持图片预览。

---

## 与 Gemini 原生接口对比

| 维度 | 本接口（`/v1/images/generations`）| Gemini 原生（`/v1beta/models/...:generateContent`）|
|---|---|---|
| 格式 | OpenAI 标准 | Gemini 原生 |
| 认证 | Bearer Token | Bearer Token |
| 响应图片位置 | `data[0].url` | `candidates[0].content.parts[].inlineData` |
| 任务日志 | ✅ 有 | ❌ 无 |
| Studio 页面 | ✅ 支持 | ❌ 不支持 |
| 参数丰富度 | 简洁（prompt/size/quality）| 完整（aspectRatio/imageSize/thinkingConfig 等）|
