package service

import (
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

// TestService 测试服务
type TestService struct {
	testRepo *repository.TestRepository
}

// NewTestService 创建测试服务实例
func NewTestService(testRepo *repository.TestRepository) *TestService {
	return &TestService{testRepo: testRepo}
}

// CreateTest 创建测试数据
func (s *TestService) CreateTest(inputString string) (*model.Test, error) {
	test := &model.Test{
		InputString: inputString,
	}
	
	err := s.testRepo.Create(test)
	if err != nil {
		return nil, err
	}
	
	return test, nil
}
