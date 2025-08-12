package data

import (
	"kratos-demo/internal/biz"
	"kratos-demo/internal/conf"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

// cronRepo cron仓储实现
type cronRepo struct {
	cron   *cron.Cron
	logger log.Logger
}

// NewCronRepo 创建cron仓储
func NewCronRepo(c *conf.Bootstrap, logger log.Logger) (biz.TaskRepo, error) {
	helper := log.NewHelper(logger)

	if !c.Cron.Enabled {
		helper.Info("cron scheduler is disabled")
		return &cronRepo{
			cron:   nil,
			logger: logger,
		}, nil
	}

	location, err := time.LoadLocation(c.Cron.Timezone)
	if err != nil {
		helper.Errorf("failed to load timezone %s: %v", c.Cron.Timezone, err)
		location = time.UTC
	}

	cronScheduler := cron.New(cron.WithLocation(location))

	helper.Infof("cron scheduler created with timezone: %s", location.String())
	return &cronRepo{
		cron:   cronScheduler,
		logger: logger,
	}, nil
}

// AddTask 添加任务
func (r *cronRepo) AddTask(spec string, cmd func()) (interface{}, error) {
	helper := log.NewHelper(r.logger)

	if r.cron == nil {
		helper.Error("cron scheduler is not available")
		return nil, nil
	}

	entryID, err := r.cron.AddFunc(spec, cmd)
	if err != nil {
		helper.Errorf("failed to add task: %v", err)
		return nil, err
	}

	helper.Infof("task added with ID: %d, spec: %s", entryID, spec)
	return entryID, nil
}

// RemoveTask 移除任务
func (r *cronRepo) RemoveTask(entryID interface{}) {
	helper := log.NewHelper(r.logger)

	if r.cron == nil {
		helper.Error("cron scheduler is not available")
		return
	}

	if id, ok := entryID.(cron.EntryID); ok {
		r.cron.Remove(id)
		helper.Infof("task removed with ID: %d", id)
	} else {
		helper.Errorf("invalid entry ID type: %T", entryID)
	}
}

// StartScheduler 启动调度器
func (r *cronRepo) StartScheduler() {
	helper := log.NewHelper(r.logger)

	if r.cron == nil {
		helper.Error("cron scheduler is not available")
		return
	}

	r.cron.Start()
	helper.Info("cron scheduler started")
}

// StopScheduler 停止调度器
func (r *cronRepo) StopScheduler() {
	helper := log.NewHelper(r.logger)

	if r.cron == nil {
		helper.Error("cron scheduler is not available")
		return
	}

	r.cron.Stop()
	helper.Info("cron scheduler stopped")
}

// IsRunning 检查调度器是否运行
func (r *cronRepo) IsRunning() bool {
	if r.cron == nil {
		return false
	}
	
	// cron v3没有直接的IsRunning方法，我们通过检查entries来判断
	entries := r.cron.Entries()
	return len(entries) > 0
}