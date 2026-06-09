# RunningHub 模型调用接口文档

本文档面向终端用户，介绍如何通过 new-api 网关调用 RunningHub AI 工作流。

---

## 目录

- [快速开始](#快速开始)
- [认证](#认证)
- [模型名称配置说明](#模型名称配置说明)
- [通用请求结构](#通用请求结构)
- [接口一览](#接口一览)
- [RunningHub 专属接口](#runninghub-专属接口)
  - [图片生成](#1-图片生成)
  - [视频生成](#2-视频生成)
  - [文本工作流](#3-文本工作流)
  - [音频合成](#4-音频合成)
  - [任务状态查询](#5-任务状态查询)
- [OpenAI 兼容接口](#openai-兼容接口)
  - [图片生成 / 图片编辑](#1-图片生成--图片编辑)
  - [视频生成](#2-视频生成-1)
  - [视频任务查询](#3-视频任务查询)
  - [语音合成 TTS](#4-语音合成-tts)
- [任务响应与轮询](#任务响应与轮询)
- [参数化计费说明](#参数化计费说明)

---

## 快速开始

```bash
# 提交一个图片生成任务
curl -X POST https://{your-new-api-host}/runninghub/v1/image \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-image-model",
    "prompt": "一只可爱的猫咪在阳光下玩耍",
    "metadata": {
      "resolution": "1024x1024"
    }
  }'

# 查询任务结果
curl https://{your-new-api-host}/runninghub/v1/task/{task_id} \
  -H "Authorization: Bearer sk-xxxxxxxx"
```

---

## 认证

所有接口均需在请求头中携带 API Key：

```
Authorization: Bearer <your-api-key>
```

---

## 模型名称配置说明

RunningHub 的"模型名称"对应 RunningHub 平台上的**工作流路径（Resource Path）**。

- **配置方式**：在 new-api 管理后台的渠道中，将你的模型名称映射到 RunningHub 工作流路径。
- **路径规则**：网关会自动将工作流路径拼接为 `https://{runninghub-host}/openapi/v2/{resource-path}`。
- **动态覆盖**：可在请求体的 `metadata.api_path` 字段中临时覆盖工作流路径，无需修改后台配置。

**示例**：

| 你的模型名 | RunningHub 工作流路径 | 实际请求 URL |
|---|---|---|
| `rh-image-gen` | `youchuan/text-to-image-v61` | `.../openapi/v2/youchuan/text-to-image-v61` |
| `rh-video-gen` | `minimax/video-01` | `.../openapi/v2/minimax/video-01` |

---

## 通用请求结构

所有任务提交接口（专属接口 + OpenAI 兼容接口）均支持以下字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `model` | string | 是 | 模型名称（对应工作流路径，OpenAI 兼容接口使用） |
| `prompt` | string | 是 | 文本提示词 |
| `image` | string | 否 | 单张参考图，URL 或 base64 编码 |
| `images` | string[] | 否 | 多张参考图列表 |
| `size` | string | 否 | 输出尺寸，如 `1024x1024`、`1280x720` |
| `duration` | int | 否 | 视频时长（秒） |
| `input_reference` | string | 否 | 参考素材 URL（等效于 `images[0]`） |
| `metadata` | object | 否 | 工作流扩展参数，所有字段均会透传至上游 |

> **`metadata` 说明**：`metadata` 中的字段会被**展开到请求体顶层**后发送给 RunningHub。你可以在此传入工作流所需的任意自定义参数（如 `steps`、`cfg_scale`、`voice_id` 等）。

### metadata 保留字段

| 字段 | 说明 |
|---|---|
| `api_path` | 动态覆盖当前请求使用的工作流路径，**不会**发送给上游 |
| `resolution` | 参数化计费用，同时透传至上游 |
| `quality` | 参数化计费用，同时透传至上游 |
| `billing_tier` | 参数化计费用，同时透传至上游 |

---

## 接口一览

| 接口 | 方法 | 路径 | 说明 |
|---|---|---|---|
| 图片生成 | POST | `/runninghub/v1/image` | RunningHub 专属图片工作流 |
| 视频生成 | POST | `/runninghub/v1/video` | RunningHub 专属视频工作流 |
| 文本工作流 | POST | `/runninghub/v1/text` | RunningHub 专属文本输出工作流 |
| 音频合成 | POST | `/runninghub/v1/audio` | RunningHub 专属音频工作流 |
| 任务查询 | GET | `/runninghub/v1/task/:task_id` | 查询任意任务状态 |
| 图片生成（OpenAI） | POST | `/v1/images/generations` | OpenAI 兼容格式 |
| 图片编辑（OpenAI） | POST | `/v1/images/edits` | OpenAI 兼容格式 |
| 视频生成（OpenAI） | POST | `/v1/video/generations` | OpenAI 兼容格式 |
| 视频任务查询（OpenAI） | GET | `/v1/video/generations/:task_id` | OpenAI 兼容格式 |
| 视频任务查询（OpenAI）| GET | `/v1/videos/:task_id` | OpenAI 兼容格式（别名） |
| 图片任务查询（OpenAI）| GET | `/v1/images/generations/:task_id` | OpenAI 兼容格式 |
| 语音合成 TTS（OpenAI）| POST | `/v1/audio/speech` | OpenAI 兼容格式（同步返回音频流） |

---

## RunningHub 专属接口

### 1. 图片生成

```
POST /runninghub/v1/image
```

**请求体**：

```json
{
  "model": "rh-image-model",
  "prompt": "一幅中国风山水画，烟雾缭绕",
  "size": "1024x1024",
  "metadata": {
    "steps": 30,
    "cfg_scale": 7.5,
    "resolution": "2k"
  }
}
```

**带参考图**：

```json
{
  "model": "rh-image-edit-model",
  "prompt": "将背景替换为星空",
  "image": "https://example.com/photo.jpg",
  "metadata": {
    "api_path": "your/custom-workflow-path"
  }
}
```

**响应**（任务提交成功）：

```json
{
  "taskId": "task-abc123",
  "status": "PENDING"
}
```

---

### 2. 视频生成

```
POST /runninghub/v1/video
```

`action` 自动检测：请求中有图片（`image`/`images`）→ 图生视频；无图片 → 文生视频。

**文生视频**：

```json
{
  "model": "rh-video-model",
  "prompt": "海浪拍打礁石，阳光折射出七彩光芒",
  "duration": 5,
  "size": "1280x720"
}
```

**图生视频**：

```json
{
  "model": "rh-video-model",
  "prompt": "让画面中的人物轻轻微笑",
  "image": "https://example.com/portrait.jpg",
  "duration": 4,
  "metadata": {
    "motion_level": "normal"
  }
}
```

**响应**（任务提交成功）：

```json
{
  "taskId": "task-xyz789",
  "status": "PENDING"
}
```

---

### 3. 文本工作流

```
POST /runninghub/v1/text
```

适用于纯文本输出类工作流（如文案生成、摘要、翻译等）。任务完成后结果在 `results[0].text` 中返回。

**请求体**：

```json
{
  "model": "rh-text-model",
  "prompt": "请为一款健康饮料写一段广告文案，风格活泼",
  "metadata": {
    "language": "zh",
    "max_tokens": 200
  }
}
```

**任务完成响应**（查询时）：

```json
{
  "taskId": "task-text001",
  "status": "SUCCESS",
  "results": [
    {
      "text": "清晨第一口，唤醒你的活力！..."
    }
  ]
}
```

---

### 4. 音频合成

```
POST /runninghub/v1/audio
```

提交后为异步任务，需通过 `/task/:task_id` 轮询结果，结果为音频文件 URL。

> 如需同步返回音频流，请使用 OpenAI 兼容接口 `POST /v1/audio/speech`。

**请求体**：

```json
{
  "model": "rh-tts-model",
  "prompt": "你好，欢迎使用我们的智能语音服务",
  "metadata": {
    "voice_id": "zh-CN-XiaoxiaoNeural",
    "speed": 1.0,
    "response_format": "mp3",
    "enable_base64_output": false
  }
}
```

**任务完成响应**（查询时）：

```json
{
  "taskId": "task-audio001",
  "status": "SUCCESS",
  "results": [
    {
      "url": "https://cdn.runninghub.cn/output/audio/xxx.mp3"
    }
  ]
}
```

---

### 5. 任务状态查询

```
GET /runninghub/v1/task/:task_id
```

查询任意已提交任务的状态和结果。

**请求示例**：

```bash
curl https://{your-new-api-host}/runninghub/v1/task/task-abc123 \
  -H "Authorization: Bearer sk-xxxxxxxx"
```

**响应示例（进行中）**：

```json
{
  "taskId": "task-abc123",
  "status": "RUNNING"
}
```

**响应示例（成功）**：

```json
{
  "taskId": "task-abc123",
  "status": "SUCCESS",
  "results": [
    {
      "url": "https://cdn.runninghub.cn/output/image/xxx.png",
      "fileUrl": "https://cdn.runninghub.cn/output/image/xxx.png",
      "text": ""
    }
  ]
}
```

**响应示例（失败）**：

```json
{
  "taskId": "task-abc123",
  "status": "FAIL",
  "errorCode": "WORKFLOW_ERROR",
  "errorMessage": "工作流执行失败：缺少必填节点参数"
}
```

---

## OpenAI 兼容接口

适合已使用 OpenAI SDK 的用户，网关会自动将请求转换为 RunningHub 格式。

### 1. 图片生成 / 图片编辑

```
POST /v1/images/generations
POST /v1/images/edits
```

**请求体（图片生成）**：

```json
{
  "model": "rh-image-model",
  "prompt": "赛博朋克风格的城市夜景",
  "size": "1024x1024",
  "quality": "hd",
  "extra": {
    "api_path": "your/workflow-path",
    "steps": 30,
    "resolution": "2k"
  }
}
```

**请求体（图片编辑）**：

```json
{
  "model": "rh-edit-model",
  "prompt": "将图中人物的衣服改为红色",
  "image": "https://example.com/photo.jpg",
  "extra": {
    "api_path": "your/edit-workflow-path"
  }
}
```

> **字段映射**：`quality` → `metadata.quality`，`style` → `metadata.style`，`extra` 中的字段直接透传至 `metadata`。

**响应**（任务提交）：

```json
{
  "taskId": "task-img001",
  "status": "PENDING"
}
```

使用 `GET /v1/images/generations/:task_id` 查询结果。

---

### 2. 视频生成

```
POST /v1/video/generations
POST /v1/videos/generations
POST /v1/videos
```

**请求体**：

```json
{
  "model": "rh-video-model",
  "prompt": "一只老鹰在山谷间翱翔",
  "duration": 5,
  "size": "1280x720"
}
```

---

### 3. 视频任务查询

```
GET /v1/video/generations/:task_id
GET /v1/videos/generations/:task_id
GET /v1/videos/:task_id
GET /v1/images/generations/:task_id
```

---

### 4. 语音合成 TTS

```
POST /v1/audio/speech
```

**同步接口**：网关内部提交任务并自动轮询（超时 120 秒），完成后**直接将音频流**返回给调用方，无需手动查询任务。

**请求体**：

```json
{
  "model": "rh-tts-model",
  "input": "你好，欢迎使用我们的语音合成服务",
  "voice": "zh-CN-XiaoxiaoNeural",
  "speed": 1.0,
  "response_format": "mp3"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `model` | string | 模型名称（对应 TTS 工作流路径）|
| `input` | string | 要合成的文字内容（必填） |
| `voice` | string | 音色 ID（可选），映射为上游 `voice_id` |
| `speed` | float | 语速，默认 1.0（可选） |
| `response_format` | string | 音频格式，如 `mp3`、`wav`（可选） |
| `metadata` | object | 额外工作流参数，展开后透传（可选） |

**响应**：直接返回音频二进制流，`Content-Type` 为 `audio/mpeg` 或工作流返回的类型。

---

## 任务响应与轮询

### 任务状态说明

| RunningHub 状态 | 含义 |
|---|---|
| `PENDING` | 任务已提交，排队等待 |
| `RUNNING` | 任务执行中 |
| `SUCCESS` | 任务成功，`results` 中包含结果 |
| `FAIL` / `FAILED` / `ERROR` | 任务失败，`errorMessage` 包含原因 |

### 轮询建议

```javascript
async function pollTask(taskId, apiKey, interval = 3000, maxWait = 300000) {
  const start = Date.now();
  while (Date.now() - start < maxWait) {
    const res = await fetch(`/runninghub/v1/task/${taskId}`, {
      headers: { 'Authorization': `Bearer ${apiKey}` }
    });
    const data = await res.json();
    if (data.status === 'SUCCESS') return data.results;
    if (['FAIL', 'FAILED', 'ERROR'].includes(data.status)) {
      throw new Error(data.errorMessage || '任务失败');
    }
    await new Promise(r => setTimeout(r, interval));
  }
  throw new Error('任务超时');
}
```

### 结果字段说明

| 字段 | 说明 |
|---|---|
| `results[].url` | 生成结果的 URL（图片/视频/音频）|
| `results[].fileUrl` | 备用 URL，与 `url` 二选一 |
| `results[].text` | 文本工作流的输出内容 |

> 优先级：`url` > `fileUrl` > `text`

---

## 参数化计费说明

RunningHub 工作流支持按参数档位计费，管理员可在后台配置不同档位的价格。

调用时，在 `metadata` 中传入对应参数即可自动匹配价格：

```json
{
  "model": "rh-image-model",
  "prompt": "生成一张超高清图片",
  "metadata": {
    "resolution": "4k"
  }
}
```

**支持的计费参数字段**（按优先级）：

`resolution` > `quality` > `billing_tier` > `size` > `style` > `duration`

如管理员未配置对应档位，则使用基础价格计费。