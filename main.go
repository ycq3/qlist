package main

import (
	"log"
	"net/http"
	"qlist/api"
	"qlist/config"
	"qlist/docs"
	"qlist/handlers"
	"qlist/middleware"
	"strconv"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title 积分管理系统API
// @version 1.0
// @description 提供用户积分管理、积分配置和积分日志查询等功能
// @BasePath /api

func main() {
	// 注册静态文件处理
	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/dist/download.html")
	})

	// 加载配置文件
	err := config.LoadConfig("config.json")
	if err != nil {
		// 使用默认配置
		config.Instance.Port = 8080
	} else {
		// 初始化数据库连接
		if err := api.InitDB(); err != nil {
			log.Fatal(err)
		}
	}

	// 初始化 Swagger 文档
	docs.SwaggerInfo.Title = "Qlist积分管理系统 API"
	docs.SwaggerInfo.Description = "提供用户积分管理、积分配置和积分日志查询等功能"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api"

	// 初始化静态文件处理器
	staticHandler := &handlers.StaticHandler{}

	// 初始化认证中间件
	authMiddleware := &middleware.AuthMiddleware{}

	// 注册静态文件处理器，处理所有静态文件和需要权限控制的页面
	http.Handle("/", staticHandler)

	// 普通API路由（无需认证）
	http.HandleFunc("/api/getUserPoints", api.GetUserPoints)
	http.HandleFunc("/api/getPointsLog", api.GetPointsLog)
	http.HandleFunc("/api/getPointsList", api.GetPointsList)
	http.HandleFunc("/api/getUserInfo", api.GetUserInfo)
	http.HandleFunc("/api/downloadFile", api.DownloadFile)
	http.HandleFunc("/api/getFileInfo", api.GetFileInfo)
	// 三方登录跳转接口
	http.HandleFunc("/api/login/google", api.LoginGoogle)
	http.HandleFunc("/api/login/github", api.LoginGitHub)
	http.HandleFunc("/api/login/wechat", api.LoginWechat)
	http.HandleFunc("/api/login/google/callback", api.GoogleCallback)
	http.HandleFunc("/api/login/github/callback", api.GitHubCallback)
	http.HandleFunc("/api/login/wechat/callback", api.WechatCallback)

	// 敏感API路由（需要认证）
	http.HandleFunc("/api/configurePoints", authMiddleware.RequireAuth(api.ConfigurePoints))
	http.HandleFunc("/api/getUsersList", authMiddleware.RequireAuth(api.GetUsersList))
	http.HandleFunc("/api/adminGrantPoints", authMiddleware.RequireAuth(api.AdminGrantPoints))
	handler := &handlers.ConfigHandler{}
	http.HandleFunc("/api/generateApiKey", authMiddleware.RequireAuth(handler.GenerateAPIKey))
	http.HandleFunc("/api/getApiKey", authMiddleware.RequireAuth(handler.GetApiKey))

	// Swagger API 文档
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("doc.json"), // URL指向API文档
	))

	// 启动服务器
	addr := ":" + strconv.Itoa(config.Instance.Port)
	log.Printf("Server started on port %d", config.Instance.Port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
