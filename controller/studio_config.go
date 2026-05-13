package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetStudioModels(c *gin.Context) {
	modelType := c.Query("model_type")
	list, err := model.ListStudioModelConfigs(modelType, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func AdminGetStudioModels(c *gin.Context) {
	modelType := c.Query("model_type")
	list, err := model.ListStudioModelConfigs(modelType, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func AdminCreateStudioModel(c *gin.Context) {
	var item model.StudioModelConfig
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Name == "" || item.ModelName == "" || item.ModelType == "" {
		common.ApiErrorMsg(c, "name、model_name、model_type 不能为空")
		return
	}
	if err := item.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func AdminUpdateStudioModel(c *gin.Context) {
	var item model.StudioModelConfig
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Id == 0 {
		common.ApiErrorMsg(c, "缺少模型 ID")
		return
	}
	if item.Name == "" || item.ModelName == "" || item.ModelType == "" {
		common.ApiErrorMsg(c, "name、model_name、model_type 不能为空")
		return
	}
	if err := item.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func AdminDeleteStudioModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteStudioModelConfigByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetStudioFormSchemas(c *gin.Context) {
	modelType := c.Query("model_type")
	modelName := c.Query("model_name")
	list, err := model.ListStudioFormSchemas(modelType, modelName, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func AdminGetStudioFormSchemas(c *gin.Context) {
	modelType := c.Query("model_type")
	modelName := c.Query("model_name")
	list, err := model.ListStudioFormSchemas(modelType, modelName, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func AdminCreateStudioFormSchema(c *gin.Context) {
	var item model.StudioFormSchema
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Name == "" || item.ModelType == "" || item.Schema == "" {
		common.ApiErrorMsg(c, "name、model_type、schema 不能为空")
		return
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	if err := item.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func AdminUpdateStudioFormSchema(c *gin.Context) {
	var item model.StudioFormSchema
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Id == 0 {
		common.ApiErrorMsg(c, "缺少 schema ID")
		return
	}
	if item.Name == "" || item.ModelType == "" || item.Schema == "" {
		common.ApiErrorMsg(c, "name、model_type、schema 不能为空")
		return
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	if err := item.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func AdminDeleteStudioFormSchema(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteStudioFormSchemaByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
