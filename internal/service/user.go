package service

import (
	"context"
	stderr "errors"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"

	//"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/utils"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserService 用户服务层
type UserService struct {
	userRepo         repository.UserRepository
	codeRepo         repository.CodeRepository
	postgresUserRepo *repository.PostgresUserRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, codeRepo repository.CodeRepository, postgresUserRepo *repository.PostgresUserRepository) *UserService {
	return &UserService{
		userRepo:         userRepo,
		codeRepo:         codeRepo,
		postgresUserRepo: postgresUserRepo,
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	// 1. 检查邮箱是否已存在 - 通过 Repository 操作
	exists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "查询用户失败", "internal/service/user.go/Register")
	}
	if exists {
		logger.Error("邮箱已被注册", zap.String("email", req.Email))
		return nil, stderr.New("邮箱已被注册")
	}

	// 2. 验证验证码 - 通过 Repository 操作
	code, err := s.codeRepo.GetRegisterCode(ctx, req.Email)
	if err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "获取验证码失败", "internal/service/user.go/Register")
	}
	if code != req.Code {
		logger.Error("验证码错误", zap.String("email", req.Email))
		return nil, stderr.New("验证码错误")
	}

	// 3. 删除已使用的验证码
	if err = s.codeRepo.DeleteRegisterCode(ctx, req.Email); err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "删除验证码失败", "internal/service/user.go/Register")
	}

	// 4. 密码哈希
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "密码哈希失败", "internal/service/user.go/Register")
	}

	// 5. 创建用户 - 通过 Repository 操作
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "创建用户失败", "internal/service/user.go/Register")
	}

	// 6. 在PostgreSQL中创建用户记录
	if s.postgresUserRepo != nil {
		if err := s.postgresUserRepo.Create(ctx, user.ID); err != nil {
			logger.Error("创建PostgreSQL用户记录失败", zap.Uint("user_id", user.ID), zap.Error(err))
			// PostgreSQL用户记录创建失败不影响用户注册
		}
	}

	return user, nil
}

// Login 用户登录（密码方式）
func (s *UserService) Login(ctx context.Context, req model.LoginRequest) (*model.User, error) {
	// 通过 Repository 查询用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if stderr.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("邮箱不存在", zap.String("email", req.Email))
			return nil, stderr.New("邮箱不存在")
		}
		return nil, errors.WrapError(err, errors.DatabaseError, "查询失败", "internal/service/user.go/Login")
	}

	// 验证密码
	if !utils.CheckPassword(req.Password, user.Password) {
		logger.Error("密码错误", zap.String("email", req.Email))
		return nil, stderr.New("密码错误")
	}

	return user, nil
}

// LoginByCode 用户登录（验证码方式）
func (s *UserService) LoginByCode(ctx context.Context, req model.LoginByCodeRequest) (*model.User, error) {
	// 通过 Repository 查询用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if stderr.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("邮箱不存在", zap.String("email", req.Email))
			return nil, stderr.New("邮箱不存在")
		}
		return nil, errors.WrapError(err, errors.DatabaseError, "查询失败", "internal/service/user.go/LoginByCode")
	}

	// 验证验证码
	code, err := s.codeRepo.GetLoginCode(ctx, req.Email)
	if err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "获取验证码失败", "internal/service/user.go/LoginByCode")
	}
	if code != req.Code {
		logger.Error("验证码错误", zap.String("email", req.Email))
		return nil, stderr.New("验证码错误")
	}

	// 删除已使用的验证码
	if err := s.codeRepo.DeleteLoginCode(ctx, req.Email); err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "删除验证码失败", "internal/service/user.go/LoginByCode")
	}

	return user, nil
}

// LSendCode 发送登录验证码
func (s *UserService) LSendCode(ctx context.Context, req model.SendCodeRequest) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if stderr.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("邮箱不存在", zap.String("email", req.Email))
			return stderr.New("邮箱不存在")
		}
		return errors.WrapError(err, errors.DatabaseError, "查询失败", "internal/service/user.go/LSendCode")
	}

	// 检查发送频率限制
	canSend, err := s.codeRepo.CheckRateLimit(ctx, req.Email)
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "检查频率限制失败", "internal/service/user.go/LSendCode")
	}
	if !canSend {
		logger.Error("验证码发送太频繁", zap.String("email", req.Email))
		return stderr.New("验证码发送太频繁")
	}

	// 生成验证码
	code := utils.GenerateCode()
	logger.Info("生成验证码", zap.String("code", code))

	// 发送验证码邮件
	if err := SendEmail(ctx, req.Email, "验证码", code); err != nil {
		return errors.WrapError(err, errors.UtilsError, "发送验证码失败", "internal/service/user.go/LSendCode")
	}

	// 存储验证码到 Redis
	logger.Info("准备存储验证码到Redis", zap.String("email", req.Email), zap.String("code", code))
	if err := s.codeRepo.SetLoginCode(ctx, req.Email, code, 5*time.Minute); err != nil {
		logger.Error("存储验证码到Redis失败", zap.String("email", req.Email), zap.Error(err))
		return errors.WrapError(err, errors.UtilsError, "存储验证码失败", "internal/service/user.go/LSendCode")
	}

	// 设置频率限制
	if err := s.codeRepo.SetRateLimit(ctx, req.Email, 1*time.Minute); err != nil {
		logger.Error("设置频率限制失败", zap.String("email", req.Email), zap.Error(err))
	}

	logger.Info("验证码已存储到Redis", zap.String("email", req.Email), zap.Int("ttl", 5*60))
	return nil
}

// RSendCode 发送注册验证码
func (s *UserService) RSendCode(ctx context.Context, req model.SendCodeRequest) error {
	// 检查邮箱是否已存在
	exists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "查询用户失败", "internal/service/user.go/RSendCode")
	}
	if exists {
		logger.Error("邮箱已被注册", zap.String("email", req.Email))
		return stderr.New("邮箱已被注册")
	}

	// 检查发送频率限制
	canSend, err := s.codeRepo.CheckRateLimit(ctx, req.Email)
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "检查频率限制失败", "internal/service/user.go/RSendCode")
	}
	if !canSend {
		logger.Error("验证码发送太频繁", zap.String("email", req.Email))
		return stderr.New("验证码发送太频繁")
	}

	// 生成验证码
	code := utils.GenerateCode()
	logger.Info("生成验证码", zap.String("code", code))

	// 发送验证码邮件
	if err := SendEmail(ctx, req.Email, "验证码", code); err != nil {
		return errors.WrapError(err, errors.UtilsError, "发送验证码失败", "internal/service/user.go/RSendCode")
	}

	// 存储验证码到 Redis
	logger.Info("准备存储验证码到Redis", zap.String("email", req.Email), zap.String("code", code))
	if err := s.codeRepo.SetRegisterCode(ctx, req.Email, code, 5*time.Minute); err != nil {
		logger.Error("存储验证码到Redis失败", zap.String("email", req.Email), zap.Error(err))
		return errors.WrapError(err, errors.UtilsError, "存储验证码失败", "internal/service/user.go/RSendCode")
	}

	// 设置频率限制
	if err := s.codeRepo.SetRateLimit(ctx, req.Email, 1*time.Minute); err != nil {
		logger.Error("设置频率限制失败", zap.String("email", req.Email), zap.Error(err))
	}

	logger.Info("验证码已存储到Redis", zap.String("email", req.Email), zap.Int("ttl", 5*60))
	return nil
}

// SendResetPasswordCode 发送找回密码验证码
func (s *UserService) SendResetPasswordCode(ctx context.Context, req model.SendResetPasswordCodeRequest) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if stderr.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("邮箱不存在", zap.String("email", req.Email))
			return stderr.New("邮箱不存在")
		}
		return errors.WrapError(err, errors.DatabaseError, "查询用户失败", "internal/service/user.go/SendResetPasswordCode")
	}

	// 检查发送频率限制
	canSend, err := s.codeRepo.CheckRateLimit(ctx, req.Email)
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "检查频率限制失败", "internal/service/user.go/SendResetPasswordCode")
	}
	if !canSend {
		logger.Error("验证码发送太频繁", zap.String("email", req.Email))
		return stderr.New("验证码发送太频繁")
	}

	// 生成验证码
	code := utils.GenerateCode()
	logger.Info("生成找回密码验证码", zap.String("email", req.Email), zap.String("code", code))

	// 发送验证码邮件
	if err := SendEmail(ctx, req.Email, "找回密码验证码", code); err != nil {
		return errors.WrapError(err, errors.UtilsError, "发送验证码失败", "internal/service/user.go/SendResetPasswordCode")
	}

	// 存储验证码到 Redis
	logger.Info("准备存储找回密码验证码到Redis", zap.String("email", req.Email))
	if err := s.codeRepo.SetCode(ctx, "reset_password:code:"+req.Email, code, 5*time.Minute); err != nil {
		logger.Error("存储找回密码验证码到Redis失败", zap.String("email", req.Email), zap.Error(err))
		return errors.WrapError(err, errors.UtilsError, "存储验证码失败", "internal/service/user.go/SendResetPasswordCode")
	}

	// 设置频率限制
	if err := s.codeRepo.SetRateLimit(ctx, req.Email, 1*time.Minute); err != nil {
		logger.Error("设置频率限制失败", zap.String("email", req.Email), zap.Error(err))
	}

	logger.Info("找回密码验证码已存储到Redis", zap.String("email", req.Email))
	return nil
}

// ResetPassword 重置密码
func (s *UserService) ResetPassword(ctx context.Context, req model.ResetPasswordRequest) error {
	// 查询用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if stderr.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("邮箱不存在", zap.String("email", req.Email))
			return stderr.New("邮箱不存在")
		}
		return errors.WrapError(err, errors.DatabaseError, "查询用户失败", "internal/service/user.go/ResetPassword")
	}

	// 验证验证码
	code, err := s.codeRepo.GetCode(ctx, "reset_password:code:"+req.Email)
	if err != nil {
		logger.Error("获取验证码失败", zap.String("email", req.Email))
		return errors.WrapError(err, errors.UtilsError, "获取验证码失败", "internal/service/user.go/ResetPassword")
	}
	if code != req.Code {
		logger.Error("验证码错误", zap.String("email", req.Email))
		return stderr.New("验证码错误")
	}

	// 删除已使用的验证码
	if err = s.codeRepo.DeleteCode(ctx, "reset_password:code:"+req.Email); err != nil {
		logger.Error("删除验证码失败", zap.String("email", req.Email))
	}

	// 对新密码进行哈希
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "密码哈希失败", "internal/service/user.go/ResetPassword")
	}

	// 更新用户密码
	user.Password = hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return errors.WrapError(err, errors.DatabaseError, "更新密码失败", "internal/service/user.go/ResetPassword")
	}

	logger.Info("密码重置成功", zap.String("email", req.Email))
	return nil
}
