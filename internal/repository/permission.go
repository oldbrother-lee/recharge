package repository

import (
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Create(permission *model.Permission) error {
	return r.db.Create(permission).Error
}

func (r *PermissionRepository) GetByID(id int64) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepository) GetByCode(code string) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.Where("code = ?", code).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepository) Update(permission *model.Permission) error {
	return r.db.Save(permission).Error
}

func (r *PermissionRepository) Delete(id int64) error {
	return r.db.Delete(&model.Permission{}, id).Error
}

func (r *PermissionRepository) List(page, pageSize int) ([]model.Permission, int64, error) {
	var permissions []model.Permission
	var total int64

	err := r.db.Model(&model.Permission{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&permissions).Error
	if err != nil {
		return nil, 0, err
	}

	return permissions, total, nil
}

func (r *PermissionRepository) GetTree() ([]*model.PermissionTree, error) {
	var permissions []model.Permission
	err := r.db.Order("`order` asc").Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	// Build permission tree
	permissionMap := make(map[int64]*model.PermissionTree)
	var rootPermissions []*model.PermissionTree

	// First pass: create all nodes
	for _, p := range permissions {
		permissionTree := &model.PermissionTree{
			ID:        p.ID,
			Code:      p.Code,
			Name:      p.Name,
			Type:      p.Type,
			ParentID:  p.ParentID,
			Path:      p.Path,
			Component: p.Component,
			Icon:      p.Icon,
			Layout:    p.Layout,
			Method:    p.Method,
			Show:      p.Show,
			Enable:    p.Enable,
			Order:     p.Order,
			KeepAlive: p.KeepAlive,
			Redirect:  p.Redirect,
			Children:  make([]*model.PermissionTree, 0),
		}
		permissionMap[p.ID] = permissionTree
	}

	// Second pass: build tree structure
	for _, p := range permissions {
		permissionTree := permissionMap[p.ID]
		if p.ParentID == nil {
			rootPermissions = append(rootPermissions, permissionTree)
		} else {
			parent, exists := permissionMap[*p.ParentID]
			if exists {
				parent.Children = append(parent.Children, permissionTree)
			}
		}
	}

	return rootPermissions, nil
}

func (r *PermissionRepository) GetMenuTree() ([]*model.PermissionTree, error) {
	var permissions []model.Permission
	err := r.db.Where("type = ?", "MENU").Order("`order` asc").Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	// Build permission tree
	permissionMap := make(map[int64]*model.PermissionTree)
	var rootPermissions []*model.PermissionTree

	// First pass: create all nodes
	for _, p := range permissions {
		permissionTree := &model.PermissionTree{
			ID:        p.ID,
			Code:      p.Code,
			Name:      p.Name,
			Type:      p.Type,
			ParentID:  p.ParentID,
			Path:      p.Path,
			Component: p.Component,
			Icon:      p.Icon,
			Layout:    p.Layout,
			Method:    p.Method,
			Show:      p.Show,
			Enable:    p.Enable,
			Order:     p.Order,
			KeepAlive: p.KeepAlive,
			Redirect:  p.Redirect,
			Children:  make([]*model.PermissionTree, 0),
		}
		permissionMap[p.ID] = permissionTree
	}

	// Second pass: build tree structure
	for _, p := range permissions {
		permissionTree := permissionMap[p.ID]
		if p.ParentID == nil {
			rootPermissions = append(rootPermissions, permissionTree)
		} else {
			parent, exists := permissionMap[*p.ParentID]
			if exists {
				parent.Children = append(parent.Children, permissionTree)
			}
		}
	}

	return rootPermissions, nil
}

func (r *PermissionRepository) GetButtonPermissions() ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.Where("type = ?", "BUTTON").Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetAll retrieves all permissions
func (r *PermissionRepository) GetAll() ([]*model.Permission, error) {
	var permissions []*model.Permission
	err := r.db.Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetByType retrieves permissions by type
func (r *PermissionRepository) GetByType(permissionType string) ([]*model.Permission, error) {
	var permissions []*model.Permission
	err := r.db.Where("type = ?", permissionType).Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetByRoleID 根据角色ID获取权限
func (r *PermissionRepository) GetByRoleID(roleID int64) ([]*model.Permission, error) {
	var permissions []*model.Permission
	err := r.db.Table("permissions").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// AssignToRole 为角色分配权限
func (r *PermissionRepository) AssignToRole(roleID int64, permissionIDs []int64) error {
	// 开启事务
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 删除角色原有的权限
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 添加新的权限
	for _, permissionID := range permissionIDs {
		rolePermission := &model.RolePermission{
			RoleID:       roleID,
			PermissionID: permissionID,
		}
		if err := tx.Create(rolePermission).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}
