package main

import (
	"log"
	"net/http"
	"qlist/api"
	"qlist/config"
	"qlist/docs"
	"qlist/handlers"
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

	// 注册静态文件处理器，处理所有静态文件和需要权限控制的页面
	http.Handle("/", staticHandler)

	// API routes
	http.HandleFunc("/api/getUserPoints", api.GetUserPoints)
	http.HandleFunc("/api/configurePoints", api.ConfigurePoints)
	http.HandleFunc("/api/getPointsLog", api.GetPointsLog)
	http.HandleFunc("/api/getPointsList", api.GetPointsList)
	http.HandleFunc("/api/getUsersList", api.GetUsersList)
	http.HandleFunc("/api/adminGrantPoints", api.AdminGrantPoints)
	http.HandleFunc("/api/getUserInfo", api.GetUserInfo)
	http.HandleFunc("/api/downloadFile", api.DownloadFile)
	http.HandleFunc("/api/getFileInfo", api.GetFileInfo)

	// Swagger API 文档
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("doc.json"), // URL指向API文档
	))

	// 启动服务器
	addr := ":" + strconv.Itoa(config.Instance.Port)
	log.Printf("Server started on port %d", config.Instance.Port)
	log.Fatal(http.ListenAndServe(addr, nil))
}
