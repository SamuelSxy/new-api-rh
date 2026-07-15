# gpt-image-2 接入文档

## 一、基本信息

| 项目 | 说明 |
|---|---|
| 模型名 | `gpt-image-2` |
| 类型 | 图像生成 / 图像编辑(同步返回) |
| 兼容接口 | `POST /v1/images/generations`、`POST /v1/images/edits` |
| Channel | OpenAI(走现有 `gpt-image-*` 同步链路,包装为 task 写入任务日志) |

## 二、请求参数

### 1. 通用参数(generations + edits)

| 字段 | 类型 | 必填 | 默认 | 允许值 / 说明 |
|---|---|---|---|---|
| `model` | string | ✅ | - | 固定为 `gpt-image-2` |
| `prompt` | string | ✅ | - | 图像描述文本 |
| `n` | number | - | `1` | 生成数量,范围 `1 ~ 10`(仅 generations) |
| `size` | string | - | `auto` | 见下方「size 取值与约束」 |
| `quality` | string | - | `high` | `high` / `medium` / `low` |
| `background` | string | - | `auto` | `transparent` / `opaque` / `auto`;选 `transparent` 时 `output_format` 必须为 `png` |
| `output_format` | string | - | `png` | `png` / `jpeg` |
| `output_compression` | number | - | `100` | 仅 `jpeg` 生效,`0 ~ 100` |
| `moderation` | string | - | `auto` | `auto`(标准)/ `low`(更宽松) |

### 2. 编辑专用参数(edits)

| 字段 | 类型 | 说明 |
|---|---|---|
| `image` | file | 单张参考图 |
| `image[]` | file[] | 多张参考图,**最多 5 张** |
| `mask` | file | 蒙版图(可通过 URL 引用) |
| `input_fidelity` | string | `high` / `low`,保留原图细节程度 |

### 2.1 通过 generations 端点带 URL 参考图（网关扩展能力）

OpenAI 官方协议中，参考图必须以 `multipart/form-data` 二进制走 `/v1/images/edits`。
本网关在 `/v1/images/generations` 上额外兼容了 JSON 形式的参考图输入，方便创意控制台等前端
直接传 TOS URL：

| 字段 | 类型 | 说明 |
|---|---|---|
| `image` | string \| string[] | 参考图，支持 `http(s)://` URL、`data:image/...;base64,...` 或纯 base64 |
| `images` | string \| string[] | 同上，多张时与 `image` 合并 |
| `metadata.imageUrls` | string[] | 与 Gemini 图生图对齐，前端创意控制台默认走此字段 |
| `mask` | string | 蒙版图，单张，同样支持 URL / data URI / base64 |

行为：
- 网关检测到上述字段任一非空时，**自动改路由到 `/v1/images/edits`**，URL 内部 fetch 后转 multipart 上传，对客户端透明。
- 单张图片 fetch 失败立即返回 `502 fetch_reference_image_failed`，mask 失败返回 `502 fetch_mask_failed`，与 Gemini 图生图行为一致。
- 总数超过 `5` 张返回 `400 invalid_request`。
- mask 必须是字符串；若需要多张参考图 + mask，仍可走原生 `/v1/images/edits` + multipart 上传二进制。

### 3. `size` 取值与约束

允许值:`1024x1024`、`1536x1024`、`1024x1536`、`2048x2048`、`2048x1152`、`3840x2160`、`2160x3840`、`auto`

自定义尺寸需同时满足:

- 单边长 ≤ `3840px`
- 两边均为 `16px` 的倍数
- 长边 : 短边 ≤ `3:1`
- 总像素数 ∈ `[655,360, 8,294,400]`

## 三、约束与默认值汇总(实现时校验)

| 场景 | 校验规则 |
|---|---|
| `background=transparent` | 强制要求 `output_format=png`,否则报错 |
| `output_compression` | 仅在 `output_format=jpeg` 时生效,其它格式应忽略或拒绝 |
| `n` | 不在 `[1,10]` 报错;为空/0 时回填 `1` |
| `quality` 缺省 | 回填 `high`(与 `gpt-image-1` 的 `auto` 不同) |
| `size` 缺省 | 回填 `auto` |
| `image[]` | 数量 > 5 报错 |

## 四、响应格式

接口同步返回，结构与 OpenAI `/v1/images/generations` 一致。

**成功响应（HTTP 200）：**

```json
{
  "created": 1750034962,
  "data": [
    {
      "url": "https://tos.example.com/openai-image/3f1c....png",
      "b64_json": "",
      "revised_prompt": "a calico cat astronaut on Mars, cinematic, dramatic lighting"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `created` | int64 | Unix 时间戳；上游缺省时由网关回填 |
| `data` | array | 图片数组，长度 = 请求的 `n` |
| `data[].url` | string | 图片地址；TOS 启用时为对象存储 https URL，否则回退为 `data:image/png;base64,...` 形式的 Data URI |
| `data[].b64_json` | string | 当 TOS 上传成功并已转 URL 后为空字符串；TOS 未启用且上游直接给出 base64 时可能保留 |
| `data[].revised_prompt` | string | 上游可能返回的改写后的提示词，原样透传 |
| `metadata` | object | 上游附加的元数据（如 token 用量），原样透传 |

> **图片落盘策略**：上游返回 `b64_json` 时，若开启了 TOS（对象存储）会自动上传并把 `data[].url` 替换为 https URL、清空 `b64_json`，避免大体积 base64 进入数据库与日志；TOS 未启用或上传失败时回退为 Data URI，仍可被前端直接渲染。

**失败响应（示例）：**

```json
{
  "error": {
    "message": "no image in response: ...",
    "type": "no_image_in_response",
    "code": 502
  }
}
```

| 常见 `type` | HTTP | 触发条件 |
|---|---|---|
| `invalid_request` | 400 | `prompt` 为空、参数校验未通过（`n` 越界、`background=transparent` 但 `output_format!=png` 等）|
| `no_image_in_response` | 502 | 上游返回 `data` 为空，或全部条目既无 `url` 也无 `b64_json` |
| `read_response_body_failed` / `unmarshal_response_failed` | 500 | 上游响应读取或 JSON 解析失败 |

## 五、任务日志

通过本接口生成的图片会自动记录到任务日志，可在管理后台 **日志 → 任务日志** 中查看，状态为 `SUCCESS`，`result_url` 取 `data[0].url`，支持图片预览。

## 六、本仓库改动清单

> 参考现有 `gpt-image-1.5` 的接入位置即可对齐。

1. **`relay/channel/openai/constant.go:68`** — 模型清单加入 `gpt-image-2`:

   ```go
   "gpt-image-1", "gpt-image-1-mini", "gpt-image-1.5", "gpt-image-2",
   ```

2. **`relay/relay_adaptor.go:202`** — `strings.HasPrefix(m, "gpt-image-")` 已自动覆盖,无需改动;新模型会复用 `openai_image` 的 task 适配器写入任务日志。

3. **`relay/helper/valid_request.go:161 / 211`** — `GetAndValidOpenAIImageRequest` 中新增 `gpt-image-2` 分支:
   - 默认 `quality = "high"`(而非 `gpt-image-1` 的 `auto`)
   - 默认 `size = "auto"`
   - 校验 `background=transparent` ⇒ `output_format=png`
   - 校验 `n ∈ [1,10]`、`image[]` ≤ 5

4. **`dto`(图像请求 DTO)** — 若现有 `ImageRequest` 缺少以下字段需补充(遵循 **Rule 6**,可选标量用指针 + `omitempty`):
   - `Background *string`
   - `OutputFormat *string`
   - `OutputCompression *int`
   - `InputFidelity *string`
   - `Moderation *string`
   - `Mask`(已有则复用)

5. **`setting/ratio_setting/model_ratio.go`** — 三处定价(待补具体单价,先按 `gpt-image-1` 占位):
   - `ModelRatio["gpt-image-2"]`(输入 token 倍率)
   - `CompletionRatio["gpt-image-2"]`(输出/图像 token 倍率,当前 `gpt-image-1` 为 `8`)
   - `ModelPrice["gpt-image-2"]`(按次计价兜底,当前 `gpt-image-1` 为 `2`)

6. **`common/model.go:15`** — 若该列表是默认开放模型清单,按需追加 `gpt-image-2`。

## 七、请求示例

### 生成

```bash
curl https://your-gateway/v1/images/generations \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "a calico cat astronaut on Mars, cinematic",
    "n": 1,
    "size": "2048x1152",
    "quality": "high",
    "background": "transparent",
    "output_format": "png",
    "moderation": "auto"
  }'
```

### 编辑(多图 + mask)

```bash
curl https://your-gateway/v1/images/edits \
  -H "Authorization: Bearer $KEY" \
  -F model="gpt-image-2" \
  -F prompt="replace the sky with aurora" \
  -F 'image[]=@a.png' -F 'image[]=@b.png' \
  -F mask=@mask.png \
  -F input_fidelity=high \
  -F size=1536x1024 \
  -F quality=high \
  -F output_format=jpeg \
  -F output_compression=85
```