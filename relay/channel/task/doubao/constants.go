package doubao

import "github.com/QuantumNous/new-api/setting/billing_setting"

var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
	"doubao-seedance-2-0-fall",
	"doubao-seedance-2-0-fast-fall",
}

var ChannelName = "doubao-video"

// videoInputRatioMap 视频输入折扣比率（含视频单价 / 不含视频单价）。
// 管理员应将 ModelRatio 设置为"不含视频"的较高费率，
// 系统在检测到视频输入时自动乘以此折扣。
var videoInputRatioMap = map[string]float64{
	"doubao-seedance-2-0-260128":      28.0 / 46.0, // ~0.6087
	"doubao-seedance-2-0-fast-260128": 22.0 / 37.0, // ~0.5946
}

// mediakitEnhanceModels 指定需要走 480p 生成 + Mediakit 超分路径的模型集合。
// 当 MediakitEnabled=true 且请求分辨率为 720p/1080p 时，系统会将实际请求替换为 480p，
// 生成完成后由 Mediakit enhance-video API 超分到目标分辨率。
var mediakitEnhanceModels = map[string]bool{
	"doubao-seedance-2-0-fall":      true,
	"doubao-seedance-2-0-fast-fall": true,
}

// durationBillingModels 指定按生成视频时长（秒）计费的模型，值为请求未指定时长时使用的默认秒数。
// 计费公式：ModelRatio × QuotaPerUnit × GroupRatio × 实际秒数。
// 管理员应将 ModelRatio 设置为每秒对应的 Dollar 价格（如 $0.014/s 填 0.014）。
var durationBillingModels = map[string]int{
	"doubao-seedance-2-0-fall":      5,
	"doubao-seedance-2-0-fast-fall": 5,
}

func init() {
	billing_setting.RegisterDurationBillingFallbacks(durationBillingModels)
}

func GetVideoInputRatio(modelName string) (float64, bool) {
	r, ok := videoInputRatioMap[modelName]
	return r, ok
}
