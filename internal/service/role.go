package service

import (
	"errors"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
)

type RoleService struct {
	roleRepo *repository.RoleRepository
}

func NewRoleService(roleRepo *repository.RoleRepository) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
	}
}

func (s *RoleService) Create(req *model.RoleRequest) (*model.Role, error) {
	// Check if role code already exists
	_, err := s.roleRepo.GetByCode(req.Code)
	if err == nil {
		return nil, errors.New("role code already exists")
	}

	role := &model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      req.Status,
	}

	err = s.roleRepo.Create(role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RoleService) Update(id int64, req *model.RoleRequest) (*model.Role, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if new code conflicts with existing roles
	if req.Code != role.Code {
		existing, err := s.roleRepo.GetByCode(req.Code)
		if err == nil && existing.ID != id {
			return nil, errors.New("role code already exists")
		}
	}

	role.Name = req.Name
	role.Code = req.Code
	role.Description = req.Description
	role.Status = req.Status

	err = s.roleRepo.Update(role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RoleService) Delete(id int64) error {
	return s.roleRepo.Delete(id)
}

func (s *RoleService) GetByID(id int64) (*model.RoleWithPermissions, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	permissions, err := s.roleRepo.GetRolePermissions(id)
	if err != nil {
		return nil, err
	}

	return &model.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (s *RoleService) List(page, pageSize int) ([]model.Role, int64, error) {
	return s.roleRepo.List(page, pageSize)
}

func (s *RoleService) GetAll() ([]model.Role, error) {
	return s.roleRepo.GetAll()
}

func (s *RoleService) AddPermission(roleID, permissionID int64) error {
	return s.roleRepo.AddRolePermission(roleID, permissionID)
}

func (s *RoleService) RemovePermission(roleID, permissionID int64) error {
	return s.roleRepo.RemoveRolePermission(roleID, permissionID)
}

func (s *RoleService) RemoveAllPermissions(roleID int64) error {
	return s.roleRepo.RemoveAllRolePermissions(roleID)
}
