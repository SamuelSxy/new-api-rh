package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	StudioModelTypeScript = "script"
	StudioModelTypeImage  = "image"
	StudioModelTypeVoice  = "voice"
	StudioModelTypeVideo  = "video"
)

type StudioModelConfig struct {
	Id            int            `json:"id"`
	Name          string         `json:"name" gorm:"size:128;not null"`
	ModelName     string         `json:"model_name" gorm:"size:128;not null;index:idx_studio_model_type_name"`
	ModelType     string         `json:"model_type" gorm:"size:32;not null;index:idx_studio_model_type_name"`
	Description   string         `json:"description,omitempty" gorm:"type:text"`
	Capability    string         `json:"capability,omitempty" gorm:"type:text"`
	DefaultParams string         `json:"default_params,omitempty" gorm:"type:text"`
	VisibleGroups string         `json:"visible_groups,omitempty" gorm:"type:text"`
	Status        int            `json:"status" gorm:"default:1;index"`
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type StudioFormSchema struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:128;not null"`
	ModelType   string         `json:"model_type" gorm:"size:32;not null;index:idx_studio_schema_type_model_status"`
	ModelName   string         `json:"model_name,omitempty" gorm:"size:128;index:idx_studio_schema_type_model_status"`
	Version     int            `json:"version" gorm:"default:1"`
	Schema      string         `json:"schema" gorm:"type:text;not null"`
	Description string         `json:"description,omitempty" gorm:"type:text"`
	Status      int            `json:"status" gorm:"default:1;index:idx_studio_schema_type_model_status"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (m *StudioModelConfig) Insert() error {
	now := common.GetTimestamp()
	m.CreatedTime = now
	m.UpdatedTime = now
	return DB.Create(m).Error
}

func (m *StudioModelConfig) Update() error {
	m.UpdatedTime = common.GetTimestamp()
	return DB.Model(&StudioModelConfig{}).Where("id = ?", m.Id).
		Select("name", "model_name", "model_type", "description", "capability", "default_params", "visible_groups", "status", "updated_time").
		Updates(m).Error
}

func DeleteStudioModelConfigByID(id int) error {
	return DB.Delete(&StudioModelConfig{}, id).Error
}

func ListStudioModelConfigs(modelType string, onlyEnabled bool) ([]*StudioModelConfig, error) {
	var list []*StudioModelConfig
	query := DB.Model(&StudioModelConfig{})
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	if onlyEnabled {
		query = query.Where("status = ?", 1)
	}
	err := query.Order("updated_time DESC").Find(&list).Error
	return list, err
}

func (s *StudioFormSchema) Insert() error {
	now := common.GetTimestamp()
	s.CreatedTime = now
	s.UpdatedTime = now
	return DB.Create(s).Error
}

func (s *StudioFormSchema) Update() error {
	s.UpdatedTime = common.GetTimestamp()
	return DB.Model(&StudioFormSchema{}).Where("id = ?", s.Id).
		Select("name", "model_type", "model_name", "version", "schema", "description", "status", "updated_time").
		Updates(s).Error
}

func DeleteStudioFormSchemaByID(id int) error {
	return DB.Delete(&StudioFormSchema{}, id).Error
}

func ListStudioFormSchemas(modelType string, modelName string, onlyEnabled bool) ([]*StudioFormSchema, error) {
	var list []*StudioFormSchema
	query := DB.Model(&StudioFormSchema{})
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	if modelName != "" {
		query = query.Where("model_name = ? OR model_name = ''", modelName)
	}
	if onlyEnabled {
		query = query.Where("status = ?", 1)
	}
	err := query.Order("model_name DESC, version DESC, updated_time DESC").Find(&list).Error
	return list, err
}
