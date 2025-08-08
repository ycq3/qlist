package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"qlist/api"
	"qlist/cmd"
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
	// 注册 `init` 命令的命令行参数
	initFlags := cmd.RegisterInitFlags()
	flag.Parse()

	// 如果是 `init` 命令，则执行初始化并退出
	if os.Args[1] == "init" {
		cmd.HandleInitCommand(initFlags)
		return
	}

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

	// 初始化中间件
	authMiddleware := &middleware.AuthMiddleware{}
	siteMiddleware := &middleware.SiteMiddleware{}

	// 创建一个新的路由器
	mux := http.NewServeMux()

	// 注册静态文件处理
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/dist/download.html")
	})
	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/dist/login.html")
	})
	mux.HandleFunc("/register.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/dist/register.html")
	})

	// 初始化静态文件处理器
	staticHandler := &handlers.StaticHandler{}
	mux.Handle("/", staticHandler)

	// 普通API路由（无需认证）
	mux.HandleFunc("/api/getUserPoints", api.GetUserPoints)
	mux.HandleFunc("/api/getPointsLog", api.GetPointsLog)
	mux.HandleFunc("/api/getPointsList", api.GetPointsList)
	mux.HandleFunc("/api/getUserInfo", api.GetUserInfo)
	mux.HandleFunc("/api/downloadFile", api.DownloadFile)
	mux.HandleFunc("/api/getFileInfo", api.GetFileInfo)
	// 本地登录注册接口
	mux.HandleFunc("/api/login/local", api.LocalLoginHandler)
	mux.HandleFunc("/api/register/local", api.LocalRegisterHandler)
	// 三方登录跳转接口
	mux.HandleFunc("/api/login/google", api.LoginGoogle)
	mux.HandleFunc("/api/login/github", api.LoginGitHub)
	mux.HandleFunc("/api/login/wechat", api.LoginWechat)
	mux.HandleFunc("/api/login/google/callback", api.GoogleCallback)
	mux.HandleFunc("/api/login/github/callback", api.GitHubCallback)
	mux.HandleFunc("/api/login/wechat/callback", api.WechatCallback)

	// 站点管理API (需要认证)
	mux.HandleFunc("/api/sites", authMiddleware.RequireAuth(api.CreateSite))
	mux.HandleFunc("/api/sites/list", authMiddleware.RequireAuth(api.GetSites))
	mux.HandleFunc("/api/sites/get", authMiddleware.RequireAuth(api.GetSite))
	mux.HandleFunc("/api/sites/update", authMiddleware.RequireAuth(api.UpdateSite))
	mux.HandleFunc("/api/sites/delete", authMiddleware.RequireAuth(api.DeleteSite))

	// 敏感API路由（需要认证）
	mux.HandleFunc("/api/configurePoints", authMiddleware.RequireAuth(api.ConfigurePoints))
	mux.HandleFunc("/api/getUsersList", authMiddleware.RequireAuth(api.GetUsersList))
	mux.HandleFunc("/api/adminGrantPoints", authMiddleware.RequireAuth(api.AdminGrantPoints))
	handler := &handlers.ConfigHandler{}
	mux.HandleFunc("/api/generateApiKey", authMiddleware.RequireAuth(handler.GenerateAPIKey))
	mux.HandleFunc("/api/getApiKey", authMiddleware.RequireAuth(handler.GetApiKey))

	// Swagger API 文档
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("doc.json"), // URL指向API文档
	))

	// 应用站点中间件
	mainHandler := siteMiddleware.Handler(mux)

	// 启动服务器
	addr := ":" + strconv.Itoa(config.Instance.Port)
	log.Printf("Server started on port %d", config.Instance.Port)
	log.Fatal(http.ListenAndServe(addr, mainHandler))
}
