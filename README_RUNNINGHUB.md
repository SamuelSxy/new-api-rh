POST /runninghub/v1/image         → action = "imageGenerate"（图片生成）
POST /runninghub/v1/video         → action 自动检测：有图→"generate"，无图→"textGenerate"
POST /runninghub/v1/text          → action = "textOutput"（文本工作流）
GET  /runninghub/v1/task/:task_id → 任务查询（复用现有 videoFetchByIDRespBodyBuilder）