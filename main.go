package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

var (
	alistUrl       string
	alistToken     string
	alistTyporaDir string
	alistImageExt  string
	client         = req.C()
	currentDate    = time.Now()
)

func main() {
	// 检测alist目录是否存在，不存在则创建
	if !CheckAlistDir() {
		CreateAlistDir()
	}
	UploadTyporaImage(os.Args[1:]...)

}

// 初始化配置
func init() {
	alistUrl = os.Getenv("alist_url")
	alistToken = os.Getenv("alist_token")
	alistTyporaDir = os.Getenv("alist_typora_dir")
	alistImageExt = os.Getenv("alist_image_ext")
	if alistImageExt == "" {
		alistImageExt = ".png,.jpg,.jpeg,.gif,.webp,.bmp,.ico,.svg"
	}
	if alistUrl == "" || alistToken == "" || alistTyporaDir == "" {
		panic("请设置环境变量")
	}

	client.SetBaseURL(alistUrl).SetCommonHeader("Authorization", alistToken)
}

// GetCurrentDatePath 获取当前日期路径
func GetCurrentDatePath() string {
	return fmt.Sprintf("%v/%v/%v/%v", alistTyporaDir, currentDate.Year(), int(currentDate.Month()), currentDate.Day())
}

// ToJsonString 生成json字符串
func ToJsonString(data map[string]string) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// CheckAlistDir 检测alist是否存在指定文件夹
func CheckAlistDir() bool {
	body := ToJsonString(map[string]string{"path": GetCurrentDatePath()})
	resp := client.R().SetBodyJsonString(body).MustPost(alistUrl + "/api/fs/dirs")
	return gjson.Get(resp.String(), "code").Int() == 200
}

// CreateAlistDir 创建alist目录
func CreateAlistDir() bool {
	body := ToJsonString(map[string]string{"path": GetCurrentDatePath()})
	resp := client.R().SetBodyJsonString(body).MustPost(alistUrl + "/api/fs/mkdir")
	return gjson.Get(resp.String(), "code").Int() == 200
}

// UploadAlistFile 上传文件到alist里面去
func UploadAlistFile(filePath string) error {
	if !strings.Contains(alistImageExt, filepath.Ext(filePath)) {
		return errors.New("文件类型不支持")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New("读取文件失败")
	}

	resp := client.R().SetHeader("Content-Type", "application/octet-stream").
		SetHeader("File-Path", url.QueryEscape(fmt.Sprintf(`%v/%v`, GetCurrentDatePath(), filepath.Base(filePath)))).
		SetBodyBytes(data).MustPut(alistUrl + "/api/fs/put")
	if gjson.Get(resp.String(), "code").Int() != 200 {
		return errors.New("上传失败")
	}
	return nil
}

// GetAlistFileUrl 获取alist的文件url
func GetAlistFileUrl(filePath string) string {
	body := ToJsonString(map[string]string{"path": fmt.Sprintf("%v/%v", GetCurrentDatePath(), filepath.Base(filePath))})
	resp := client.R().SetBodyJsonString(body).MustPut(alistUrl + "/api/fs/get")
	return gjson.Get(resp.String(), "data.raw_url").String()
}

// IsNetworkPath 判断typora要上传的图片是本地图片还是网络图片
func IsNetworkPath(filePath string) bool {
	u, err := url.Parse(filePath)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// UploadTyporaImage 上传typora图片
func UploadTyporaImage(imagePaths ...string) {
	fmt.Println("Upload Success:")
	for _, imagePath := range imagePaths {
		// 如果是本地图片并且上传成功，那么就打印图片地址
		if !IsNetworkPath(imagePath) && UploadAlistFile(imagePath) == nil {
			for {
				imageurl := GetAlistFileUrl(imagePath)
				if imageurl != "" {
					fmt.Println(imageurl)
					break
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
		// 如果是网络图片，那么就直接打印
		if IsNetworkPath(imagePath) {
			fmt.Println(imagePath)
		}
	}
}
