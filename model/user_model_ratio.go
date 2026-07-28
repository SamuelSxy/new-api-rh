package model

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"gorm.io/gorm"
)

const userModelRatioCacheNamespace = "new-api:user_model_ratio:v1"

var (
	userModelRatioCacheOnce sync.Once
	userModelRatioCache     *cachex.HybridCache[userModelRatioCacheValue]
)

type userModelRatioCacheValue struct {
	Multiplier float64 `json:"multiplier"`
	Found      bool    `json:"found"`
}

func userModelRatioCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("USER_MODEL_RATIO_CACHE_TTL", 300)
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return time.Duration(ttlSeconds) * time.Second
}

func userModelRatioNegativeCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("USER_MODEL_RATIO_NEGATIVE_CACHE_TTL", 60)
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	return time.Duration(ttlSeconds) * time.Second
}

func userModelRatioCacheCapacity() int {
	capacity := common.GetEnvOrDefault("USER_MODEL_RATIO_CACHE_CAP", 20000)
	if capacity <= 0 {
		capacity = 20000
	}
	return capacity
}

func getUserModelRatioCache() *cachex.HybridCache[userModelRatioCacheValue] {
	userModelRatioCacheOnce.Do(func() {
		ttl := userModelRatioCacheTTL()
		userModelRatioCache = cachex.NewHybridCache[userModelRatioCacheValue](cachex.HybridCacheConfig[userModelRatioCacheValue]{
			Namespace: cachex.Namespace(userModelRatioCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[userModelRatioCacheValue]{},
			Memory: func() *hot.HotCache[string, userModelRatioCacheValue] {
				return hot.NewHotCache[string, userModelRatioCacheValue](hot.LRU, userModelRatioCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return userModelRatioCache
}

func userModelRatioCacheKey(userId int, modelName string) string {
	if userId <= 0 || modelName == "" {
		return ""
	}
	return fmt.Sprintf("%d:%s", userId, modelName)
}

type UserModelRatio struct {
	Id         int     `json:"id" gorm:"primaryKey"`
	UserId     int     `json:"user_id" gorm:"index:idx_user_model,unique;not null"`
	ModelName  string  `json:"model_name" gorm:"type:varchar(64);index:idx_user_model,unique;not null"`
	Multiplier float64 `json:"multiplier" gorm:"default:1.0"`
	CreatedAt  int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64   `json:"updated_at" gorm:"bigint"`
}

func (r *UserModelRatio) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *UserModelRatio) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (r *UserModelRatio) Insert() error {
	return DB.Create(r).Error
}

func (r *UserModelRatio) Update() error {
	return DB.Save(r).Error
}

func (r *UserModelRatio) Delete() error {
	return DB.Delete(r).Error
}

func GetUserModelRatioById(id int) (*UserModelRatio, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	var r UserModelRatio
	if err := DB.Where("id = ?", id).First(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func GetUserModelRatiosByUserId(userId int) ([]UserModelRatio, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	var rs []UserModelRatio
	if err := DB.Where("user_id = ?", userId).Find(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

type UserModelRatioQueryParams struct {
	UserId    int
	ModelName string
	Page      int
	PageSize  int
}

func SearchUserModelRatios(params UserModelRatioQueryParams) ([]UserModelRatio, int64, error) {
	var rs []UserModelRatio
	var total int64
	query := DB.Model(&UserModelRatio{})
	if params.UserId > 0 {
		query = query.Where("user_id = ?", params.UserId)
	}
	if params.ModelName != "" {
		query = query.Where("model_name = ?", params.ModelName)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if err := query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rs).Error; err != nil {
		return nil, 0, err
	}
	return rs, total, nil
}

// GetUserModelMultiplier 返回 (multiplier, found)。
// found=false 表示该用户该模型未配置,调用方按 1.0 处理。
// 走缓存(DB 未命中时负缓存 60s,命中时 5min)。
func GetUserModelMultiplier(userId int, modelName string) (float64, bool) {
	if userId <= 0 || modelName == "" {
		return 1.0, false
	}
	key := userModelRatioCacheKey(userId, modelName)
	if key != "" {
		if cached, found, err := getUserModelRatioCache().Get(key); err == nil && found {
			if !cached.Found {
				return 1.0, false
			}
			return cached.Multiplier, true
		}
	}

	var r UserModelRatio
	err := DB.Select("multiplier").
		Where("user_id = ? AND model_name = ?", userId, modelName).
		First(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = getUserModelRatioCache().SetWithTTL(key, userModelRatioCacheValue{Found: false}, userModelRatioNegativeCacheTTL())
			return 1.0, false
		}
		common.SysError(fmt.Sprintf("query user_model_ratio failed: user_id=%d model=%s err=%v", userId, modelName, err))
		return 1.0, false
	}

	_ = getUserModelRatioCache().SetWithTTL(key, userModelRatioCacheValue{Multiplier: r.Multiplier, Found: true}, userModelRatioCacheTTL())
	return r.Multiplier, true
}

func InvalidateUserModelMultiplierCache(userId int, modelName string) {
	if userId <= 0 {
		return
	}
	if modelName != "" {
		_, _ = getUserModelRatioCache().DeleteMany([]string{userModelRatioCacheKey(userId, modelName)})
		return
	}
	_, _ = getUserModelRatioCache().DeleteByPrefix(strconv.Itoa(userId) + ":")
}

func InvalidateAllUserModelRatioCache() {
	_ = getUserModelRatioCache().Purge()
}

func DeleteUserModelRatioByUserId(userId int) (int64, error) {
	if userId <= 0 {
		return 0, errors.New("invalid user id")
	}
	result := DB.Where("user_id = ?", userId).Delete(&UserModelRatio{})
	if result.Error != nil {
		return 0, result.Error
	}
	InvalidateUserModelMultiplierCache(userId, "")
	return result.RowsAffected, nil
}
