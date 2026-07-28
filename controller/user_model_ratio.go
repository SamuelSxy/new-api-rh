package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type userModelRatioForm struct {
	Id         int     `json:"id"`
	UserId     int     `json:"user_id"`
	ModelName  string  `json:"model_name"`
	Multiplier float64 `json:"multiplier"`
}

// GetAllUserModelRatios 列表(管理员,支持 user_id / model_name 过滤 + 分页)
func GetAllUserModelRatios(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	modelName := strings.TrimSpace(c.Query("model_name"))

	pageInfo := common.GetPageQuery(c)
	rs, total, err := model.SearchUserModelRatios(model.UserModelRatioQueryParams{
		UserId:    userId,
		ModelName: modelName,
		Page:      pageInfo.GetPage(),
		PageSize:  pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rs)
	common.ApiSuccess(c, pageInfo)
}

// GetUserModelRatiosByUserId 某用户的所有条目
func GetUserModelRatiosByUserId(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "invalid user_id")
		return
	}
	rs, err := model.GetUserModelRatiosByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rs)
}

// CreateUserModelRatio 新建
func CreateUserModelRatio(c *gin.Context) {
	var form userModelRatioForm
	if err := c.ShouldBindJSON(&form); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateUserModelRatioForm(form); err != nil {
		common.ApiError(c, err)
		return
	}

	r := model.UserModelRatio{
		UserId:     form.UserId,
		ModelName:  form.ModelName,
		Multiplier: form.Multiplier,
	}
	if err := r.Insert(); err != nil {
		if isDuplicateEntryErr(err) {
			common.ApiErrorMsg(c, "该用户该模型的倍率已存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	model.InvalidateUserModelMultiplierCache(r.UserId, r.ModelName)
	common.ApiSuccess(c, r)
}

// UpdateUserModelRatio 更新(按 id)
func UpdateUserModelRatio(c *gin.Context) {
	var form userModelRatioForm
	if err := c.ShouldBindJSON(&form); err != nil {
		common.ApiError(c, err)
		return
	}
	if form.Id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := validateUserModelRatioForm(form); err != nil {
		common.ApiError(c, err)
		return
	}

	existing, err := model.GetUserModelRatioById(form.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "条目不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	prevUserId := existing.UserId
	prevModelName := existing.ModelName
	existing.UserId = form.UserId
	existing.ModelName = form.ModelName
	existing.Multiplier = form.Multiplier
	if err := existing.Update(); err != nil {
		if isDuplicateEntryErr(err) {
			common.ApiErrorMsg(c, "该用户该模型的倍率已存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	model.InvalidateUserModelMultiplierCache(prevUserId, prevModelName)
	if prevUserId != form.UserId || prevModelName != form.ModelName {
		model.InvalidateUserModelMultiplierCache(form.UserId, form.ModelName)
	}
	common.ApiSuccess(c, existing)
}

// DeleteUserModelRatio 删除
func DeleteUserModelRatio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	existing, err := model.GetUserModelRatioById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "条目不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	if err := existing.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateUserModelMultiplierCache(existing.UserId, existing.ModelName)
	common.ApiSuccess(c, existing)
}

func validateUserModelRatioForm(form userModelRatioForm) error {
	if form.UserId <= 0 {
		return errors.New("user_id 必须大于 0")
	}
	form.ModelName = strings.TrimSpace(form.ModelName)
	if form.ModelName == "" {
		return errors.New("model_name 不能为空")
	}
	if len(form.ModelName) > 64 {
		return errors.New("model_name 长度不能超过 64")
	}
	if form.Multiplier < 0 {
		return errors.New("multiplier 不能为负数")
	}
	if _, err := model.GetUserById(form.UserId, false); err != nil {
		return errors.New("用户不存在")
	}
	return nil
}

func isDuplicateEntryErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint") ||
		strings.Contains(msg, "duplicate key")
}
