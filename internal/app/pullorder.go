package app

import (
    "context"
    "log"
    "time"

    "recharge-go/internal/repository"
    "recharge-go/internal/service/pullorder"
)

// PullOrderApp 常驻拉单应用
// 使用框架 Application 接口，支持优雅启动/停止
type PullOrderApp struct {
    container *Container
    scheduler *pullorder.PullOrderScheduler
    repo      *repository.PullSourceRepositoryImpl
    mgr       *pullorder.PullOrderManager
    ticker    *time.Ticker
}

// NewPullOrderApp 创建拉单应用实例
func NewPullOrderApp(container *Container) *PullOrderApp {
    return &PullOrderApp{container: container}
}

// Start 启动常驻拉单（按固定间隔执行 ProcessOnce）
func (p *PullOrderApp) Start(ctx context.Context) error {
    log.Println("正在启动拉单常驻应用...")

    // 依赖初始化
    p.repo = repository.NewPullSourceRepository(p.container.GetDB())
    orderSvc := p.container.GetServices().Order
    p.mgr = pullorder.NewPullOrderManager(p.repo, orderSvc)
    p.scheduler = pullorder.NewPullOrderScheduler(p.mgr, p.repo)

    // 使用配置中的任务间隔（与 Task.Interval 保持一致），默认为 30s
    interval := time.Duration(p.container.GetConfig().Task.Interval) * time.Second
    if interval <= 0 {
        interval = 30 * time.Second
    }
    p.ticker = time.NewTicker(interval)

    // 启动循环
    go func() {
        for {
            select {
            case <-ctx.Done():
                log.Println("收到退出信号，停止拉单循环...")
                return
            case <-p.ticker.C:
                if err := p.scheduler.ProcessOnce(ctx); err != nil {
                    log.Printf("拉单周期执行失败: %v", err)
                }
            }
        }
    }()

    log.Printf("拉单常驻应用已启动，间隔: %s", interval)
    return nil
}

// Stop 停止常驻拉单
func (p *PullOrderApp) Stop(ctx context.Context) error {
    log.Println("正在停止拉单常驻应用...")
    if p.ticker != nil {
        p.ticker.Stop()
    }
    return p.container.Close()
}