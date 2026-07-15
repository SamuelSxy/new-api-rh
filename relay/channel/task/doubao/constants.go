package doubao

import (
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

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

// mediakitEnhanceModels 指定需要走 Mediakit 超分路径的模型集合。
// 当 MediakitEnabled=true 且请求分辨率为 720p/1080p/4k 时，系统将实际请求替换为中间分辨率：
// 超分720/1080：上游生成 480p；超分 4k：上游生成 1080p。
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

// GetVideoInputRatio 返回模型的视频输入折扣比率（含视频单价 / 不含视频单价）。
// 数据由 ratio_setting.VideoInputRatio 管理，支持管理员后台配置。
func GetVideoInputRatio(modelName string) (float64, bool) {
	return ratio_setting.GetVideoInputRatio(modelName)
}
