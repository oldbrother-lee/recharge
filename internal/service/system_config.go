package service

import (
	"context"
	"encoding/json"
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type SystemConfigService struct {
	systemConfigRepo *repository.SystemConfigRepository
}

func NewSystemConfigService(systemConfigRepo *repository.SystemConfigRepository) *SystemConfigService {
	return &SystemConfigService{
		systemConfigRepo: systemConfigRepo,
	}
}

// Create 创建系统配置
func (s *SystemConfigService) Create(ctx context.Context, req *model.SystemConfigRequest) error {
	// 检查配置键是否已存在
	existing, err := s.systemConfigRepo.GetByKey(req.ConfigKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		return errors.New("配置键已存在")
	}

	// 验证配置类型和值
	if err := s.validateConfigValue(req.ConfigType, req.ConfigValue); err != nil {
		return err
	}

	config := &model.SystemConfig{
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigDesc:  req.ConfigDesc,
		ConfigType:  req.ConfigType,
		IsSystem:    0, // 用户创建的配置默认不是系统配置
		Status:      1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.systemConfigRepo.Create(config)
}

// Update 更新系统配置
func (s *SystemConfigService) Update(ctx context.Context, id int64, req *model.SystemConfigRequest) error {
	config, err := s.systemConfigRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 验证配置类型和值
	if err := s.validateConfigValue(req.ConfigType, req.ConfigValue); err != nil {
		return err
	}

	config.ConfigValue = req.ConfigValue
	config.ConfigDesc = req.ConfigDesc
	config.ConfigType = req.ConfigType
	config.UpdatedAt = time.Now()

	return s.systemConfigRepo.Update(config)
}

// Delete 删除系统配置
func (s *SystemConfigService) Delete(ctx context.Context, id int64) error {
	config, err := s.systemConfigRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 系统配置不允许删除
	if config.IsSystem == 1 {
		return errors.New("系统配置不允许删除")
	}

	return s.systemConfigRepo.Delete(id)
}

// GetByID 根据ID获取系统配置
func (s *SystemConfigService) GetByID(ctx context.Context, id int64) (*model.SystemConfigResponse, error) {
	config, err := s.systemConfigRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(config), nil
}

// GetByKey 根据配置键获取系统配置
func (s *SystemConfigService) GetByKey(ctx context.Context, key string) (*model.SystemConfigResponse, error) {
	config, err := s.systemConfigRepo.GetByKey(key)
	if err != nil {
		return nil, err
	}

	return s.toResponse(config), nil
}

// GetList 获取系统配置列表
func (s *SystemConfigService) GetList(ctx context.Context, page, pageSize int, configKey string) ([]model.SystemConfigResponse, int64, error) {
	configs, total, err := s.systemConfigRepo.GetList(page, pageSize, configKey)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]model.SystemConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = *s.toResponse(&config)
	}

	return responses, total, nil
}

// UpdateSystemName 更新系统名称
func (s *SystemConfigService) UpdateSystemName(ctx context.Context, systemName string) error {
	return s.systemConfigRepo.UpdateByKey("system_name", systemName)
}

// GetSystemName 获取系统名称
func (s *SystemConfigService) GetSystemName(ctx context.Context) (string, error) {
	config, err := s.systemConfigRepo.GetByKey("system_name")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "充值系统", nil // 默认系统名称
		}
		return "", err
	}
	return config.ConfigValue, nil
}

// BatchUpdate 批量更新配置
func (s *SystemConfigService) BatchUpdate(ctx context.Context, configs map[string]string) error {
	// 验证所有配置值
	for key, value := range configs {
		existing, err := s.systemConfigRepo.GetByKey(key)
		if err != nil {
			return err
		}
		if err := s.validateConfigValue(existing.ConfigType, value); err != nil {
			return err
		}
	}

	return s.systemConfigRepo.BatchUpdateByKeys(configs)
}

// InitSystemConfigs 初始化系统配置
func (s *SystemConfigService) InitSystemConfigs(ctx context.Context) error {
	defaultConfigs := []model.SystemConfig{
		{
			ConfigKey:   "system_name",
			ConfigValue: "充值系统",
			ConfigDesc:  "系统名称",
			ConfigType:  "string",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "system_logo",
			ConfigValue: "/logo.png",
			ConfigDesc:  "系统Logo",
			ConfigType:  "string",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "system_version",
			ConfigValue: "1.0.0",
			ConfigDesc:  "系统版本",
			ConfigType:  "string",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "system_description",
			ConfigValue: "一个功能强大的充值管理系统",
			ConfigDesc:  "系统描述",
			ConfigType:  "string",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "maintenance_mode",
			ConfigValue: "false",
			ConfigDesc:  "维护模式",
			ConfigType:  "boolean",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "session_timeout",
			ConfigValue: "3600",
			ConfigDesc:  "会话超时时间（秒）",
			ConfigType:  "number",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ConfigKey:   "max_upload_size",
			ConfigValue: "10485760",
			ConfigDesc:  "最大上传文件大小（字节）",
			ConfigType:  "number",
			IsSystem:    1,
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, config := range defaultConfigs {
		// 检查配置是否已存在
		existing, err := s.systemConfigRepo.GetByKey(config.ConfigKey)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing == nil {
			if err := s.systemConfigRepo.Create(&config); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateConfigValue 验证配置值
func (s *SystemConfigService) validateConfigValue(configType, value string) error {
	switch configType {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return errors.New("配置值必须是数字")
		}
	case "boolean":
		if _, err := strconv.ParseBool(value); err != nil {
			return errors.New("配置值必须是布尔值")
		}
	case "json":
		var js json.RawMessage
		if err := json.Unmarshal([]byte(value), &js); err != nil {
			return errors.New("配置值必须是有效的JSON")
		}
	}
	return nil
}

// GetAllAsMap 获取所有配置并以map形式返回
func (s *SystemConfigService) GetAllAsMap(ctx context.Context) (map[string]string, error) {
	configs, err := s.systemConfigRepo.GetAllEnabled()
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.ConfigKey] = config.ConfigValue
	}

	return configMap, nil
}

// toResponse 转换为响应结构
func (s *SystemConfigService) toResponse(config *model.SystemConfig) *model.SystemConfigResponse {
	return &model.SystemConfigResponse{
		ID:          config.ID,
		ConfigKey:   config.ConfigKey,
		ConfigValue: config.ConfigValue,
		ConfigDesc:  config.ConfigDesc,
		ConfigType:  config.ConfigType,
		IsSystem:    config.IsSystem,
		Status:      config.Status,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}
}
