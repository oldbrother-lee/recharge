package service

import (
	"context"
	"errors"
	"fmt"
	"recharge-go/internal/model"
	"recharge-go/internal/repository"
	"recharge-go/internal/utils"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo              *repository.UserRepository
	userGradeRepo         *repository.UserGradeRepository
	userTagRepo           *repository.UserTagRepository
	userTagRelationRepo   *repository.UserTagRelationRepository
	userGradeRelationRepo *repository.UserGradeRelationRepository
	userLogRepo           *repository.UserLogRepository
}

type UserLoginResponse struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refreshToken"`
	UserInfo     UserInfo `json:"userInfo"`
}

type UserInfo struct {
	UserId   string   `json:"userId"`
	UserName string   `json:"userName"`
	Roles    []string `json:"roles"`
	Buttons  []string `json:"buttons"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func NewUserService(
	userRepo *repository.UserRepository,
	userGradeRepo *repository.UserGradeRepository,
	userTagRepo *repository.UserTagRepository,
	userTagRelationRepo *repository.UserTagRelationRepository,
	userGradeRelationRepo *repository.UserGradeRelationRepository,
	userLogRepo *repository.UserLogRepository,
) *UserService {
	return &UserService{
		userRepo:              userRepo,
		userGradeRepo:         userGradeRepo,
		userTagRepo:           userTagRepo,
		userTagRelationRepo:   userTagRelationRepo,
		userGradeRelationRepo: userGradeRelationRepo,
		userLogRepo:           userLogRepo,
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *model.UserRegisterRequest) error {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return errors.New("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 创建用户
	user := &model.User{
		Username:  req.Username,
		Password:  string(hashedPassword),
		Status:    1, // 默认启用状态
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 设置可选字段
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}

	// 创建用户
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return err
	}

	// 记录用户注册日志
	log := &model.UserLog{
		UserID:    user.ID,
		Action:    model.UserLogActionCreate,
		Content:   "用户注册",
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *model.UserLoginRequest) (*UserLoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	roles, err := s.userRepo.GetUserRoles(user.ID) // 返回 []model.Role
	if err != nil {
		return nil, err
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Code) // 或 r.Name
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("密码错误")
	}

	if user.Status != 1 { // 1 表示正常状态
		return nil, errors.New("账号已被禁用")
	}

	// 生成访问令牌和刷新令牌
	token, refreshToken, err := utils.GenerateJWT(user.ID, user.Username, roleNames)
	if err != nil {
		return nil, err
	}
	userInfo := UserInfo{
		UserId:   fmt.Sprintf("%d", user.ID),
		UserName: user.Username,
		Roles:    roleNames,
		Buttons:  []string{}, // 可根据权限系统填充
	}

	return &UserLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		UserInfo:     userInfo,
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 验证刷新令牌
	claims, err := utils.ValidateJWT(refreshToken, true)
	if err != nil {
		return nil, errors.New("无效的刷新令牌")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if user.Status != 1 { // 1 表示正常状态
		return nil, errors.New("账号已被禁用")
	}

	// 生成新的访问令牌和刷新令牌
	newToken, newRefreshToken, err := utils.GenerateJWT(user.ID, user.Username, claims.Roles)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, userID int64, req *model.UserUpdateRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 记录原始信息
	oldInfo := *user
	changes := make([]string, 0)

	if req.Phone != nil && *req.Phone != oldInfo.Phone {
		changes = append(changes, fmt.Sprintf("手机: %s -> %s", oldInfo.Phone, *req.Phone))
		user.Phone = *req.Phone
	}

	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 记录用户信息更新日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionUpdate,
		Content:   fmt.Sprintf("更新用户信息: %s", strings.Join(changes, ", ")),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(ctx context.Context, userID int64, req *model.UserChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 验证旧密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword))
	if err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 记录密码修改日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionPassword,
		Content:   "修改密码",
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// GetUserList 获取用户列表
func (s *UserService) GetUserList(ctx context.Context, req *model.UserListRequest) (*model.UserListResponse, error) {
	users, total, err := s.userRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var userResponses []model.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, model.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Phone:     user.Phone,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Status:    user.Status,
			Credit:    user.Credit,
			Balance:   user.Balance,
			LastLogin: user.LastLogin,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return &model.UserListResponse{
		List:  userResponses,
		Total: total,
	}, nil
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, userID int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(ctx context.Context, userID int64, status int) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	oldStatus := user.Status
	user.Status = status
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 记录状态修改日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionStatus,
		Content:   fmt.Sprintf("修改用户状态: %d -> %d", oldStatus, status),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// SetUserGrade 设置用户等级
func (s *UserService) SetUserGrade(ctx context.Context, userID, gradeID int64) error {
	// 检查等级是否存在
	grade, err := s.userGradeRepo.GetByID(ctx, gradeID)
	if err != nil {
		return errors.New("等级不存在")
	}

	// 获取用户当前等级
	oldGrade, _ := s.userGradeRelationRepo.GetUserGrade(ctx, userID)

	// 删除旧的等级关系
	err = s.userGradeRelationRepo.Delete(ctx, userID, 0) // 0 表示删除该用户的所有等级关系
	if err != nil {
		return err
	}

	// 创建新的等级关系
	relation := &model.UserGradeRelation{
		UserID:    userID,
		GradeID:   gradeID,
		CreatedAt: time.Now(),
	}

	if err := s.userGradeRelationRepo.Create(ctx, relation); err != nil {
		return err
	}

	// 记录等级修改日志
	content := "设置用户等级"
	if oldGrade != nil {
		content = fmt.Sprintf("修改用户等级: %s -> %s", oldGrade.Name, grade.Name)
	} else {
		content = fmt.Sprintf("设置用户等级: %s", grade.Name)
	}

	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionGrade,
		Content:   content,
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// GetUserGrade 获取用户等级
func (s *UserService) GetUserGrade(ctx context.Context, userID int64) (*model.UserGrade, error) {
	return s.userGradeRelationRepo.GetUserGrade(ctx, userID)
}

// GetUserWithRoles 获取用户及其角色信息
func (s *UserService) GetUserWithRoles(ctx context.Context, userID int64) (*model.UserWithRoles, error) {
	return s.userRepo.GetUserWithRoles(ctx, userID)
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	// 检查用户是否存在
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 删除用户的所有标签关系
	err = s.userTagRelationRepo.Delete(ctx, userID, 0)
	if err != nil {
		return err
	}

	// 删除用户的所有等级关系
	err = s.userGradeRelationRepo.Delete(ctx, userID, 0)
	if err != nil {
		return err
	}

	// 删除用户
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return err
	}

	// 记录用户删除日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionDelete,
		Content:   fmt.Sprintf("删除用户: %s", user.Username),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// AddUserTag 添加用户标签
func (s *UserService) AddUserTag(ctx context.Context, userID, tagID int64) error {
	// 检查标签是否存在
	tag, err := s.userTagRepo.GetByID(ctx, tagID)
	if err != nil {
		return errors.New("标签不存在")
	}

	// 检查是否已存在该标签
	exists, err := s.userTagRelationRepo.Exists(ctx, userID, tagID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("用户已拥有该标签")
	}

	relation := &model.UserTagRelation{
		UserID:    userID,
		TagID:     tagID,
		CreatedAt: time.Now(),
	}

	if err := s.userTagRelationRepo.Create(ctx, relation); err != nil {
		return err
	}

	// 记录标签添加日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionTag,
		Content:   fmt.Sprintf("添加用户标签: %s", tag.Name),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// RemoveUserTag 移除用户标签
func (s *UserService) RemoveUserTag(ctx context.Context, userID, tagID int64) error {
	// 检查标签是否存在
	tag, err := s.userTagRepo.GetByID(ctx, tagID)
	if err != nil {
		return errors.New("标签不存在")
	}

	// 检查是否拥有该标签
	exists, err := s.userTagRelationRepo.Exists(ctx, userID, tagID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("用户未拥有该标签")
	}

	if err := s.userTagRelationRepo.Delete(ctx, userID, tagID); err != nil {
		return err
	}

	// 记录标签移除日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionTag,
		Content:   fmt.Sprintf("移除用户标签: %s", tag.Name),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// GetUserTags 获取用户的所有标签
func (s *UserService) GetUserTags(ctx context.Context, userID int64) ([]model.UserTag, error) {
	return s.userTagRelationRepo.GetUserTags(ctx, userID)
}

func (s *UserService) GetUserRoles(userID int64) ([]model.Role, error) {
	return s.userRepo.GetUserRoles(userID)
}

// ResetPassword 管理员重置用户密码
func (s *UserService) ResetPassword(ctx context.Context, userID int64, newPassword string) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", errors.New("用户不存在")
	}
	if newPassword == "" {
		newPassword = "123456"
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return "", err
	}
	// 记录密码重置日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionPassword,
		Content:   "管理员重置密码",
		CreatedAt: time.Now(),
	}
	_ = s.userLogRepo.Create(ctx, log)
	return newPassword, nil
}

// AssignRoles 为用户分配角色
func (s *UserService) AssignRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	// 先删除用户现有的所有角色
	if err := s.userRepo.RemoveAllUserRoles(userID); err != nil {
		return fmt.Errorf("移除用户现有角色失败: %v", err)
	}

	// 添加新的角色
	if err := s.userRepo.AssignRoles(userID, roleIDs); err != nil {
		return fmt.Errorf("分配角色失败: %v", err)
	}

	// 记录用户角色分配日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionUpdate,
		Content:   fmt.Sprintf("分配角色: %v", roleIDs),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}

// RemoveRole 移除用户角色
func (s *UserService) RemoveRole(ctx context.Context, userID int64, roleID int64) error {
	if err := s.userRepo.RemoveUserRole(userID, roleID); err != nil {
		return fmt.Errorf("移除角色失败: %v", err)
	}

	// 记录用户角色移除日志
	log := &model.UserLog{
		UserID:    userID,
		Action:    model.UserLogActionUpdate,
		Content:   fmt.Sprintf("移除角色: %d", roleID),
		CreatedAt: time.Now(),
	}
	return s.userLogRepo.Create(ctx, log)
}
