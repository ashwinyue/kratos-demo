package cron

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is cron providers.
var ProviderSet = wire.NewSet(NewCronUsecase, NewCronRepo)

// Task 任务实体
type Task struct {
	ID       string
	Name     string
	Spec     string
	Status   string
	NextRun  time.Time
	LastRun  time.Time
	Created  time.Time
	Updated  time.Time
}

// TaskRepo 任务仓储接口
type TaskRepo interface {
	// AddTask 添加任务
	AddTask(spec string, cmd func()) (interface{}, error)
	// RemoveTask 移除任务
	RemoveTask(entryID interface{})
	// StartScheduler 启动调度器
	StartScheduler()
	// StopScheduler 停止调度器
	StopScheduler()
	// IsRunning 检查调度器是否运行
	IsRunning() bool
}

// CronUsecase 定时任务用例
type CronUsecase struct {
	repo   TaskRepo
	logger log.Logger
}

// NewCronUsecase 创建定时任务用例
func NewCronUsecase(repo TaskRepo, logger log.Logger) *CronUsecase {
	return &CronUsecase{
		repo:   repo,
		logger: logger,
	}
}

// StartScheduler 启动调度器
func (uc *CronUsecase) StartScheduler() {
	helper := log.NewHelper(uc.logger)
	helper.Info("starting cron scheduler")
	
	uc.repo.StartScheduler()
}

// StopScheduler 停止调度器
func (uc *CronUsecase) StopScheduler() {
	helper := log.NewHelper(uc.logger)
	helper.Info("stopping cron scheduler")
	
	uc.repo.StopScheduler()
}

// AddTask 添加任务
func (uc *CronUsecase) AddTask(spec string, cmd func()) (interface{}, error) {
	helper := log.NewHelper(uc.logger)
	helper.Infof("adding task with spec: %s", spec)
	
	return uc.repo.AddTask(spec, cmd)
}

// RemoveTask 移除任务
func (uc *CronUsecase) RemoveTask(entryID interface{}) {
	helper := log.NewHelper(uc.logger)
	helper.Infof("removing task with ID: %v", entryID)
	
	uc.repo.RemoveTask(entryID)
}

// SetupDefaultTasks 设置默认任务
func (uc *CronUsecase) SetupDefaultTasks() error {
	helper := log.NewHelper(uc.logger)
	helper.Info("setting up default tasks")
	
	// 添加默认的定时任务
	_, err := uc.AddTask("@every 1m", func() {
		helper.Info("executing default task: health check")
		// 这里可以添加健康检查逻辑
	})
	if err != nil {
		helper.Errorf("failed to add default task: %v", err)
		return err
	}
	
	// 添加数据清理任务
	_, err = uc.AddTask("0 2 * * *", func() {
		helper.Info("executing data cleanup task")
		// 这里可以添加数据清理逻辑
	})
	if err != nil {
		helper.Errorf("failed to add cleanup task: %v", err)
		return err
	}
	
	return nil
}

// IsRunning 检查调度器是否运行
func (uc *CronUsecase) IsRunning() bool {
	return uc.repo.IsRunning()
}