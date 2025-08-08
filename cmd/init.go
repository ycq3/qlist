package cmd

import (
	"flag"
	"fmt"
	"log"
	"qlist/api"
	"qlist/models"

	"golang.org/x/crypto/bcrypt"
)

// InitFlags 保存从命令行传入的参数
type InitFlags struct {
	SiteName   string
	SiteDomain string
	AdminUser  string
	AdminPass  string
}

// RegisterInitFlags 注册 `init` 命令的命令行参数
func RegisterInitFlags() *InitFlags {
	flags := &InitFlags{}
	flag.StringVar(&flags.SiteName, "site-name", "", "The name of the site to create")
	flag.StringVar(&flags.SiteDomain, "site-domain", "", "The domain of the site to create")
	flag.StringVar(&flags.AdminUser, "admin-user", "", "The username of the admin user to create")
	flag.StringVar(&flags.AdminPass, "admin-pass", "", "The password of the admin user to create")
	return flags
}

// HandleInitCommand 处理 `init` 命令，创建站点和管理员
func HandleInitCommand(flags *InitFlags) {
	if flags.SiteName == "" || flags.SiteDomain == "" || flags.AdminUser == "" || flags.AdminPass == "" {
		log.Fatal("All flags (--site-name, --site-domain, --admin-user, --admin-pass) are required for init command")
	}

	// 初始化数据库
	api.InitDB()
	db := api.GetDB()

	// 1. 创建站点
	site := models.Site{
		Name:   flags.SiteName,
		Domain: flags.SiteDomain,
	}
	if err := db.Create(&site).Error; err != nil {
		log.Fatalf("Failed to create site: %v", err)
	}
	fmt.Printf("Site '%s' created successfully.\n", site.Name)

	// 2. 创建管理员用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(flags.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := models.User{
		Username: flags.AdminUser,
		Password: string(hashedPassword),
		Provider: "local",
		IsAdmin:  true,
		SiteID:   site.ID,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("Admin user '%s' created successfully for site '%s'.\n", adminUser.Username, site.Name)
}