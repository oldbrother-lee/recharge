package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepository) GetByID(id int64) (*model.Role, error) {
	var role model.Role
	err := r.db.First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByCode(code string) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Update(role *model.Role) error {
	return r.db.Save(role).Error
}

func (r *RoleRepository) Delete(id int64) error {
	return r.db.Delete(&model.Role{}, id).Error
}

func (r *RoleRepository) List(page, pageSize int) ([]model.Role, int64, error) {
	var roles []model.Role
	var total int64

	err := r.db.Model(&model.Role{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&roles).Error
	if err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (r *RoleRepository) GetAll() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) GetRolePermissions(roleID int64) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *RoleRepository) AddRolePermission(roleID, permissionID int64) error {
	return r.db.Create(&model.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}).Error
}

func (r *RoleRepository) RemoveRolePermission(roleID, permissionID int64) error {
	return r.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&model.RolePermission{}).Error
}

func (r *RoleRepository) RemoveAllRolePermissions(roleID int64) error {
	return r.db.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error
}
