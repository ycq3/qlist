package main

import (
	"log"
	"os"
	"qlist/api"
	"qlist/cmd"
	"qlist/config"
	"qlist/db"
	"qlist/docs"
	"qlist/middleware"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title 积分管理系统API
// @version 1.0
// @description 提供用户积分管理、积分配置和积分日志查询等功能
// @BasePath /api
func main() {
	// 加载配置文件
	if err := config.LoadConfig("config.json"); err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}

	// 初始化数据库连接
	if err := db.InitDB(); err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}

	// 如果是 `init` 命令，则执行初始化并退出
	if len(os.Args) > 1 && os.Args[1] == "init" {
		initCmd, initFlags := cmd.NewInitCommand()
		if err := initCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Error parsing init flags: %v", err)
		}
		cmd.HandleInitCommand(initFlags)
		return
	}

	

	// 初始化 Gin 引擎
	router := gin.Default()

	// 初始化 Swagger 文档
	docs.SwaggerInfo.Title = "Qlist积分管理系统 API"
	docs.SwaggerInfo.Description = "提供用户积分管理、积分配置和积分日志查询等功能"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 应用站点中间件
	router.Use(middleware.SiteMiddleware())

	// API 路由
	apiGroup := router.Group("/api")
	{
		// 积分相关
		pointsGroup := apiGroup.Group("/points")
		{
			pointsGroup.GET("", api.GetPointsList)
			pointsGroup.POST("/configure", api.ConfigurePoints)
			pointsGroup.GET("/log", api.GetPointsLog)
		}

		// 用户相关
		usersGroup := apiGroup.Group("/users")
		{
			usersGroup.GET("", api.GetUsersList)
			usersGroup.POST("/grant", api.AdminGrantPoints)
			usersGroup.GET("/points", api.GetUserPoints)
		}

		// 文件相关
		apiGroup.GET("/download", api.DownloadFile)
		apiGroup.GET("/fileinfo", api.GetFileInfo)
	}

	// 启动服务器
	addr := ":" + strconv.Itoa(config.Instance.Port)
	log.Printf("Server started on port %d", config.Instance.Port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("无法启动服务器: %v", err)
	}
}
