package cmd

import (
	"flag"
	"fmt"
	"log"
	"qlist/db"
	"qlist/models"

	"golang.org/x/crypto/bcrypt"
)

// InitFlags 保存从命令行传入的参数
type InitFlags struct {
	SiteName   *string
	SiteDomain *string
	AdminUser  *string
	AdminPass  *string
}

// NewInitCommand 创建并返回一个用于 'init' 子命令的标志集和关联的标志变量
func NewInitCommand() (*flag.FlagSet, *InitFlags) {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	flags := &InitFlags{
		SiteName:   initCmd.String("site-name", "", "The name of the site to create"),
		SiteDomain: initCmd.String("site-domain", "", "The domain of the site to create"),
		AdminUser:  initCmd.String("admin-user", "", "The username of the admin user to create"),
		AdminPass:  initCmd.String("admin-pass", "", "The password of the admin user to create"),
	}
	return initCmd, flags
}

// HandleInitCommand 处理 `init` 命令，创建站点和管理员
func HandleInitCommand(flags *InitFlags) {
	if *flags.SiteName == "" || *flags.SiteDomain == "" || *flags.AdminUser == "" || *flags.AdminPass == "" {
		log.Fatal("All flags (--site-name, --site-domain, --admin-user, --admin-pass) are required for init command")
	}

	// 初始化数据库
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	database := db.GetDB()

	// 1. 创建站点
	site := models.Site{
		Name:   *flags.SiteName,
		Domain: *flags.SiteDomain,
	}
	if err := database.Create(&site).Error; err != nil {
		log.Fatalf("Failed to create site: %v", err)
	}
	fmt.Printf("Site '%s' created successfully.\\n", site.Name)

	// 2. 创建管理员用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*flags.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := models.User{
		Username: *flags.AdminUser,
		Password: string(hashedPassword),
		Provider: "local",
		IsAdmin:  true,
		SiteID:   site.ID,
	}
	if err := database.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("Admin user '%s' created successfully for site '%s'.\\n", adminUser.Username, site.Name)
}