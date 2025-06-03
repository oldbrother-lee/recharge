package repository

import (
	"context"
	"recharge-go/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// DB 返回数据库连接
func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据ID获取用户
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// List 获取用户列表
func (r *UserRepository) List(ctx context.Context, req *model.UserListRequest) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := r.db.WithContext(ctx).Model(&model.User{})

	if req.Username != "" {
		query = query.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+req.Phone+"%")
	}
	if req.Email != "" {
		query = query.Where("email LIKE ?", "%"+req.Email+"%")
	}
	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((req.Current - 1) * req.Size).Limit(req.Size).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// AddUserRole adds a user to a role
func (r *UserRepository) AddUserRole(userID, roleID int64) error {
	return r.db.Create(&model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}).Error
}

// RemoveUserRole removes a user from a role
func (r *UserRepository) RemoveUserRole(userID, roleID int64) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

// GetUserRoles gets all roles for a user
func (r *UserRepository) GetUserRoles(userID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserWithRoles gets a user with their roles
func (r *UserRepository) GetUserWithRoles(ctx context.Context, userID int64) (*model.UserWithRoles, error) {
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles, err := r.GetUserRoles(userID)
	if err != nil {
		return nil, err
	}

	return &model.UserWithRoles{
		User:  *user,
		Roles: roles,
	}, nil
}

// RemoveAllUserRoles removes all roles from a user
func (r *UserRepository) RemoveAllUserRoles(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error
}

// UserGradeRepository 用户等级仓储
type UserGradeRepository struct {
	db *gorm.DB
}

func NewUserGradeRepository(db *gorm.DB) *UserGradeRepository {
	return &UserGradeRepository{db: db}
}

// Create 创建用户等级
func (r *UserGradeRepository) Create(ctx context.Context, grade *model.UserGrade) error {
	return r.db.WithContext(ctx).Create(grade).Error
}

// GetByID 根据ID获取用户等级
func (r *UserGradeRepository) GetByID(ctx context.Context, id int64) (*model.UserGrade, error) {
	var grade model.UserGrade
	err := r.db.WithContext(ctx).First(&grade, id).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

// Update 更新用户等级
func (r *UserGradeRepository) Update(ctx context.Context, grade *model.UserGrade) error {
	return r.db.WithContext(ctx).Save(grade).Error
}

// Delete 删除用户等级
func (r *UserGradeRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.UserGrade{}, id).Error
}

// List 获取用户等级列表
func (r *UserGradeRepository) List(ctx context.Context) ([]model.UserGrade, error) {
	var grades []model.UserGrade
	err := r.db.WithContext(ctx).Find(&grades).Error
	if err != nil {
		return nil, err
	}
	return grades, nil
}

// UserTagRepository 用户标签仓储
type UserTagRepository struct {
	db *gorm.DB
}

func NewUserTagRepository(db *gorm.DB) *UserTagRepository {
	return &UserTagRepository{db: db}
}

// Create 创建用户标签
func (r *UserTagRepository) Create(ctx context.Context, tag *model.UserTag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

// GetByID 根据ID获取用户标签
func (r *UserTagRepository) GetByID(ctx context.Context, id int64) (*model.UserTag, error) {
	var tag model.UserTag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// Update 更新用户标签
func (r *UserTagRepository) Update(ctx context.Context, tag *model.UserTag) error {
	return r.db.WithContext(ctx).Save(tag).Error
}

// Delete 删除用户标签
func (r *UserTagRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.UserTag{}, id).Error
}

// List 获取用户标签列表
func (r *UserTagRepository) List(ctx context.Context) ([]model.UserTag, error) {
	var tags []model.UserTag
	err := r.db.WithContext(ctx).Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// UserTagRelationRepository 用户标签关系仓储
type UserTagRelationRepository struct {
	db *gorm.DB
}

func NewUserTagRelationRepository(db *gorm.DB) *UserTagRelationRepository {
	return &UserTagRelationRepository{db: db}
}

// Create 创建用户标签关系
func (r *UserTagRelationRepository) Create(ctx context.Context, relation *model.UserTagRelation) error {
	return r.db.WithContext(ctx).Create(relation).Error
}

// Delete 删除用户标签关系
func (r *UserTagRelationRepository) Delete(ctx context.Context, userID, tagID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND tag_id = ?", userID, tagID).Delete(&model.UserTagRelation{}).Error
}

// GetUserTags 获取用户的所有标签
func (r *UserTagRelationRepository) GetUserTags(ctx context.Context, userID int64) ([]model.UserTag, error) {
	var tags []model.UserTag
	err := r.db.WithContext(ctx).Joins("JOIN user_tag_relations ON user_tags.id = user_tag_relations.tag_id").
		Where("user_tag_relations.user_id = ?", userID).
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// Exists 检查用户标签关系是否存在
func (r *UserTagRelationRepository) Exists(ctx context.Context, userID, tagID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserTagRelation{}).
		Where("user_id = ? AND tag_id = ?", userID, tagID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UserGradeRelationRepository 用户等级关系仓储
type UserGradeRelationRepository struct {
	db *gorm.DB
}

func NewUserGradeRelationRepository(db *gorm.DB) *UserGradeRelationRepository {
	return &UserGradeRelationRepository{db: db}
}

// Create 创建用户等级关系
func (r *UserGradeRelationRepository) Create(ctx context.Context, relation *model.UserGradeRelation) error {
	return r.db.WithContext(ctx).Create(relation).Error
}

// Delete 删除用户等级关系
func (r *UserGradeRelationRepository) Delete(ctx context.Context, userID, gradeID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND grade_id = ?", userID, gradeID).Delete(&model.UserGradeRelation{}).Error
}

// GetUserGrade 获取用户的等级
func (r *UserGradeRelationRepository) GetUserGrade(ctx context.Context, userID int64) (*model.UserGrade, error) {
	var grade model.UserGrade
	err := r.db.WithContext(ctx).Joins("JOIN user_grade_relations ON user_grades.id = user_grade_relations.grade_id").
		Where("user_grade_relations.user_id = ?", userID).
		First(&grade).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

// AssignRoles 为用户分配多个角色
func (r *UserRepository) AssignRoles(userID int64, roleIDs []int64) error {
	// 开启事务
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 批量创建用户角色关系
	for _, roleID := range roleIDs {
		if err := tx.Create(&model.UserRole{
			UserID: userID,
			RoleID: roleID,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}
