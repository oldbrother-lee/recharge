package handler

import (
	"context"
	"fmt"
	"recharge-go/internal/config"
	"recharge-go/internal/middleware"
	"recharge-go/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器
type Server struct {
	engine *gin.Engine
	cfg    *config.Config
}

// NewServer 创建HTTP服务器
func NewServer(rechargeHandler *RechargeHandler, cfg *config.Config) *Server {
	// 设置gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建gin引擎
	engine := gin.New()

	// 使用中间件
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recovery())
	engine.Use(middleware.CORS())

	// 创建服务器实例
	server := &Server{
		engine: engine,
		cfg:    cfg,
	}

	// 注册路由
	server.registerRoutes(rechargeHandler)

	return server
}

// registerRoutes 注册路由
func (s *Server) registerRoutes(rechargeHandler *RechargeHandler) {
	// API路由组
	api := s.engine.Group("/api/v1")
	{
		// 充值相关路由
		recharge := api.Group("/recharge")
		{
			recharge.POST("/callback/:platform", rechargeHandler.HandleCallback)
		}
	}
}

// Engine 获取gin引擎
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Start 启动服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	logger.Info("HTTP服务器启动，监听地址: %s", addr)
	return s.engine.Run(addr)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	// TODO: 实现优雅关闭
	return nil
}
