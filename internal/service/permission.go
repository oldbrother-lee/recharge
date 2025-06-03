package service

import (
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type PermissionService struct {
	permissionRepo *repository.PermissionRepository
}

func NewPermissionService(permissionRepo *repository.PermissionRepository) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
	}
}

func (s *PermissionService) Create(req *model.PermissionRequest) error {
	// Check if code exists
	_, err := s.permissionRepo.GetByCode(req.Code)
	if err == nil {
		return errors.New("permission code already exists")
	}

	permission := &model.Permission{
		Code:        req.Code,
		Name:        req.Name,
		Type:        req.Type,
		ParentID:    req.ParentID,
		Path:        req.Path,
		Component:   req.Component,
		Icon:        req.Icon,
		Layout:      req.Layout,
		Method:      req.Method,
		Description: req.Description,
		Show:        req.Show,
		Enable:      req.Enable,
		Order:       req.Order,
		KeepAlive:   req.KeepAlive,
		Redirect:    req.Redirect,
	}

	return s.permissionRepo.Create(permission)
}

func (s *PermissionService) Update(id int64, req *model.PermissionRequest) error {
	permission, err := s.permissionRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if code exists for other permissions
	if req.Code != permission.Code {
		existing, err := s.permissionRepo.GetByCode(req.Code)
		if err == nil && existing.ID != id {
			return errors.New("permission code already exists")
		}
	}

	permission.Code = req.Code
	permission.Name = req.Name
	permission.Type = req.Type
	permission.ParentID = req.ParentID
	permission.Path = req.Path
	permission.Component = req.Component
	permission.Icon = req.Icon
	permission.Layout = req.Layout
	permission.Method = req.Method
	permission.Description = req.Description
	permission.Show = req.Show
	permission.Enable = req.Enable
	permission.Order = req.Order
	permission.KeepAlive = req.KeepAlive
	permission.Redirect = req.Redirect

	return s.permissionRepo.Update(permission)
}

func (s *PermissionService) Delete(id int64) error {
	return s.permissionRepo.Delete(id)
}

func (s *PermissionService) GetByID(id int64) (*model.Permission, error) {
	return s.permissionRepo.GetByID(id)
}

func (s *PermissionService) List(page, pageSize int) ([]model.Permission, int64, error) {
	return s.permissionRepo.List(page, pageSize)
}

func (s *PermissionService) GetPermissionTree() ([]*model.PermissionTree, error) {
	return s.permissionRepo.GetTree()
}

func (s *PermissionService) GetMenuPermissions() ([]*model.Permission, error) {
	return s.permissionRepo.GetByType("MENU")
}

func (s *PermissionService) GetAllPermissions() ([]*model.Permission, error) {
	return s.permissionRepo.GetAll()
}

func (s *PermissionService) GetButtonPermissions() ([]*model.Permission, error) {
	return s.permissionRepo.GetByType("BUTTON")
}

func (s *PermissionService) DeletePermission(id int64) error {
	return s.permissionRepo.Delete(id)
}

func (s *PermissionService) CreatePermission(req *model.PermissionRequest) error {
	permission := &model.Permission{
		Code:        req.Code,
		Name:        req.Name,
		Type:        req.Type,
		ParentID:    req.ParentID,
		Path:        req.Path,
		Component:   req.Component,
		Icon:        req.Icon,
		Layout:      req.Layout,
		Method:      req.Method,
		Description: req.Description,
		Show:        req.Show,
		Enable:      req.Enable,
		Order:       req.Order,
		KeepAlive:   req.KeepAlive,
		Redirect:    req.Redirect,
	}

	return s.permissionRepo.Create(permission)
}

func (s *PermissionService) UpdatePermission(permission *model.Permission) (*model.Permission, error) {
	err := s.permissionRepo.Update(permission)
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// GetByRoleID 根据角色ID获取权限
func (s *PermissionService) GetByRoleID(roleID int64) ([]*model.Permission, error) {
	return s.permissionRepo.GetByRoleID(roleID)
}

// AssignToRole 为角色分配权限
func (s *PermissionService) AssignToRole(roleID int64, permissionIDs []int64) error {
	return s.permissionRepo.AssignToRole(roleID, permissionIDs)
}
