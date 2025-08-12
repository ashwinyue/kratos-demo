package service

import (
	"kratos-demo/internal/cron"

	"github.com/go-kratos/kratos/v2/log"
)

// TaskService 定时任务服务
type TaskService struct {
	cronUc *cron.CronUsecase
	logger log.Logger
}

// NewTaskService 创建定时任务服务
func NewTaskService(cronUc *cron.CronUsecase, logger log.Logger) *TaskService {
	return &TaskService{
		cronUc: cronUc,
		logger: logger,
	}
}

// Start 启动定时任务服务
func (s *TaskService) Start() {
	helper := log.NewHelper(s.logger)

	// 设置默认任务
	if err := s.cronUc.SetupDefaultTasks(); err != nil {
		helper.Errorf("failed to setup default tasks: %v", err)
		return
	}

	// 启动调度器
	s.cronUc.StartScheduler()
	helper.Info("task service started")
}

// Stop 停止定时任务服务
func (s *TaskService) Stop() {
	helper := log.NewHelper(s.logger)
	s.cronUc.StopScheduler()
	helper.Info("task service stopped")
}



// AddTask 动态添加任务
func (s *TaskService) AddTask(spec string, cmd func()) (interface{}, error) {
	return s.cronUc.AddTask(spec, cmd)
}

// RemoveTask 移除任务
func (s *TaskService) RemoveTask(entryID interface{}) {
	s.cronUc.RemoveTask(entryID)
}

// IsRunning 检查调度器是否运行
func (s *TaskService) IsRunning() bool {
	return s.cronUc.IsRunning()
}