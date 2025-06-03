package service

import (
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

var (
	ErrPhoneNumberExists = errors.New("手机号已存在")
	ErrRecordNotFound    = errors.New("记录不存在")
)

type PhoneLocationService struct {
	repo *repository.PhoneLocationRepository
}

func NewPhoneLocationService(repo *repository.PhoneLocationRepository) *PhoneLocationService {
	return &PhoneLocationService{
		repo: repo,
	}
}

// Create 创建手机归属地
func (s *PhoneLocationService) Create(phoneLocation *model.PhoneLocation) error {
	// 检查手机号是否已存在
	existing, err := s.repo.GetByPhoneNumber(phoneLocation.PhoneNumber)
	if err == nil && existing != nil {
		return ErrPhoneNumberExists
	}
	return s.repo.Create(phoneLocation)
}

// Update 更新手机归属地
func (s *PhoneLocationService) Update(phoneLocation *model.PhoneLocation) error {
	// 检查记录是否存在
	existing, err := s.repo.GetByID(phoneLocation.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRecordNotFound
	}
	return s.repo.Update(phoneLocation)
}

// Delete 删除手机归属地
func (s *PhoneLocationService) Delete(id int64) error {
	// 检查记录是否存在
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRecordNotFound
	}
	return s.repo.Delete(id)
}

// GetByID 根据ID获取手机归属地
func (s *PhoneLocationService) GetByID(id int64) (*model.PhoneLocation, error) {
	return s.repo.GetByID(id)
}

// GetByPhoneNumber 根据手机号获取归属地
func (s *PhoneLocationService) GetByPhoneNumber(phoneNumber string) (*model.PhoneLocation, error) {
	return s.repo.GetByPhoneNumber(phoneNumber)
}

// List 获取手机归属地列表
func (s *PhoneLocationService) List(req *model.PhoneLocationListRequest) (*model.PhoneLocationListResponse, error) {
	return s.repo.List(req)
}
