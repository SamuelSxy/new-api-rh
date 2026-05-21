package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type UserAsset struct {
	Id          int            `json:"id"`
	UserId      int            `json:"user_id" gorm:"index"`
	Name        string         `json:"name" gorm:"size:256;not null"`
	AssetType   string         `json:"asset_type" gorm:"size:16;not null;index"` // Image | Video
	FileName    string         `json:"file_name" gorm:"size:512"`
	FileSize    int64          `json:"file_size"`
	ContentType string         `json:"content_type" gorm:"size:128"`
	SourceUrl   string         `json:"source_url" gorm:"type:text"`
	ArkAssetId  string         `json:"ark_asset_id,omitempty" gorm:"size:256"`
	ArkAssetUri string         `json:"ark_asset_uri,omitempty" gorm:"size:512"`
	ArkStatus   string         `json:"ark_status,omitempty" gorm:"size:32"` // Processing | Active | Failed
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *UserAsset) Insert() error {
	a.CreatedTime = common.GetTimestamp()
	return DB.Create(a).Error
}

func UpdateArkStatus(id int, status string) error {
	return DB.Model(&UserAsset{}).Where("id = ?", id).Update("ark_status", status).Error
}

func DeleteUserAsset(userId, id int) error {
	return DB.Where("id = ? AND user_id = ?", id, userId).Delete(&UserAsset{}).Error
}

func GetUserAssetById(userId, id int) (*UserAsset, error) {
	var asset UserAsset
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func ListUserAssets(userId int, assetType string, page, pageSize int) ([]*UserAsset, int64, error) {
	var list []*UserAsset
	var total int64

	query := DB.Model(&UserAsset{}).Where("user_id = ?", userId)
	if assetType != "" {
		query = query.Where("asset_type = ?", assetType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_time DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}
