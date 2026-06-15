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

// EnsureDefaultStudioData inserts built-in model configs and form schemas that may
// be missing from the database (e.g. after an upgrade that introduced new models).
// For fall models, it also upgrades v1 schemas to v2 (unified media_upload field).
func EnsureDefaultStudioData() {
	now := common.GetTimestamp()

	type modelEntry struct {
		Name        string
		ModelName   string
		Description string
	}

	modelEntries := []modelEntry{
		{"Seedance 2.0 Fall", "doubao-seedance-2-0-fall", "High quality video generation with Mediakit upscaling"},
		{"Seedance 2.0 Fast Fall", "doubao-seedance-2-0-fast-fall", "Fast video generation with Mediakit upscaling"},
	}

	for _, e := range modelEntries {
		var count int64
		DB.Model(&StudioModelConfig{}).Where("model_name = ?", e.ModelName).Count(&count)
		if count == 0 {
			cfg := &StudioModelConfig{
				Name:        e.Name,
				ModelName:   e.ModelName,
				ModelType:   StudioModelTypeVideo,
				Description: e.Description,
				Status:      1,
				CreatedTime: now,
				UpdatedTime: now,
			}
			if err := DB.Create(cfg).Error; err != nil {
				common.SysError("EnsureDefaultStudioData: insert model config " + e.ModelName + ": " + err.Error())
			}
		}
	}

	type schemaEntry struct {
		Name      string
		ModelName string
		Version   int
		Schema    string
	}

	fallSchema := `{
  "name": "Seedance 2.0 Fall Video Form",
  "modelType": "video",
  "modelName": "doubao-seedance-2-0-fall",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"},
        {"label": "1080p", "value": "1080"}
      ],
      "defaultValue": "1080"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 10,
      "step": 1,
      "defaultValue": 5
    }
  ]
}`

	fastFallSchema := `{
  "name": "Seedance 2.0 Fast Fall Video Form",
  "modelType": "video",
  "modelName": "doubao-seedance-2-0-fast-fall",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"}
      ],
      "defaultValue": "720"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 10,
      "step": 1,
      "defaultValue": 5
    }
  ]
}`

	seedanceFastSchema := `{
  "name": "Seedance 2.0 Fast Video Form",
  "modelType": "video",
  "modelName": "seedance2.0-fast",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"}
      ],
      "defaultValue": "720"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}`

	seedanceSchema := `{
  "name": "Seedance 2.0 Video Form",
  "modelType": "video",
  "modelName": "seedance2.0",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"},
        {"label": "1080p", "value": "1080"}
      ],
      "defaultValue": "1080"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}`

	doubaoFast260128Schema := `{
  "name": "Seedance 2.0 Fast Video Form",
  "modelType": "video",
  "modelName": "doubao-seedance-2-0-fast-260128",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"}
      ],
      "defaultValue": "720"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}`

	doubao260128Schema := `{
  "name": "Seedance 2.0 Video Form",
  "modelType": "video",
  "modelName": "doubao-seedance-2-0-260128",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {"label": "480p", "value": "480"},
        {"label": "720p", "value": "720"},
        {"label": "1080p", "value": "1080"}
      ],
      "defaultValue": "1080"
    },
    {
      "key": "media_urls",
      "label": "Images / Videos",
      "type": "media_upload",
      "required": true,
      "max": 10,
      "helpText": "Upload images or videos, paste a URL, or pick from your asset library"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}`

	schemaEntries := []schemaEntry{
		{
			Name:      "doubao-seedance-2-0-fall Video Form",
			ModelName: "doubao-seedance-2-0-fall",
			Version:   3,
			Schema:    fallSchema,
		},
		{
			Name:      "doubao-seedance-2-0-fast-fall Video Form",
			ModelName: "doubao-seedance-2-0-fast-fall",
			Version:   3,
			Schema:    fastFallSchema,
		},
		{
			Name:      "Seedance 2.0 Fast Video Form",
			ModelName: "seedance2.0-fast",
			Version:   3,
			Schema:    seedanceFastSchema,
		},
		{
			Name:      "Seedance 2.0 Video Form",
			ModelName: "seedance2.0",
			Version:   3,
			Schema:    seedanceSchema,
		},
		{
			Name:      "doubao-seedance-2-0-fast-260128 Video Form",
			ModelName: "doubao-seedance-2-0-fast-260128",
			Version:   3,
			Schema:    doubaoFast260128Schema,
		},
		{
			Name:      "doubao-seedance-2-0-260128 Video Form",
			ModelName: "doubao-seedance-2-0-260128",
			Version:   3,
			Schema:    doubao260128Schema,
		},
	}

	for _, e := range schemaEntries {
		var existing StudioFormSchema
		err := DB.Where("model_name = ?", e.ModelName).Order("version DESC").First(&existing).Error
		if err != nil {
			// No existing record — create it
			s := &StudioFormSchema{
				Name:        e.Name,
				ModelType:   StudioModelTypeVideo,
				ModelName:   e.ModelName,
				Version:     e.Version,
				Schema:      e.Schema,
				Status:      1,
				CreatedTime: now,
				UpdatedTime: now,
			}
			if err := DB.Create(s).Error; err != nil {
				common.SysError("EnsureDefaultStudioData: insert form schema " + e.ModelName + ": " + err.Error())
			}
		} else if existing.Version < e.Version {
			// Upgrade older version to the latest schema
			if err := DB.Model(&existing).Updates(map[string]any{
				"version":      e.Version,
				"schema":       e.Schema,
				"name":         e.Name,
				"updated_time": now,
			}).Error; err != nil {
				common.SysError("EnsureDefaultStudioData: upgrade form schema " + e.ModelName + ": " + err.Error())
			}
		}
	}
}
