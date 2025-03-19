package siteFavicon

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sun-panel/lib/cmn"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func IsHTTPURL(url string) bool {
	httpPattern := `^(http://|https://|//)`
	match, err := regexp.MatchString(httpPattern, url)
	if err != nil {
		return false
	}
	return match
}

func GetOneFaviconURL(urlStr string) (string, error) {
	iconURLs, err := getFaviconURL(urlStr)
	if err != nil {
		return "", err
	}

	for _, v := range iconURLs {
		// 标准的路径地址
		if IsHTTPURL(v) {
			return v, nil
		} else {
			urlInfo, _ := url.Parse(urlStr)
			fullUrl := urlInfo.Scheme + "://" + urlInfo.Host + "/" + strings.TrimPrefix(v, "/")
			return fullUrl, nil
		}
	}
	return "", fmt.Errorf("not found ico")
}

// 获取远程文件的大小
func GetRemoteFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP request failed, status code: %d", resp.StatusCode)
	}

	// 获取Content-Length字段，即文件大小
	size := resp.ContentLength
	return size, nil
}

// 下载图片
func DownloadImage(url, savePath string, maxSize int64) (*os.File, error) {
	// 获取远程文件大小
	fileSize, err := GetRemoteFileSize(url)
	if err != nil {
		// 有些服务器可能不返回 Content-Length，这种情况下继续下载
		if fileSize <= 0 {
			fileSize = maxSize / 2 // 设置一个默认值，继续尝试下载
		} else {
			return nil, err
		}
	}

	// 判断文件大小是否在阈值内
	if fileSize > maxSize {
		return nil, fmt.Errorf("文件太大，不下载。大小：%d字节", fileSize)
	}

	// 发送HTTP GET请求获取图片数据
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头，更好地处理跨域和可能的反爬虫限制
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	request.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	request.Header.Set("Referer", url)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// 检查HTTP响应状态
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed, status code: %d", response.StatusCode)
	}

	// 获取文件名和扩展名
	urlFileName := path.Base(url)
	fileExt := path.Ext(url)

	// 如果没有扩展名，则尝试从Content-Type中推断
	if fileExt == "" {
		contentType := response.Header.Get("Content-Type")
		switch contentType {
		case "image/png":
			fileExt = ".png"
		case "image/jpeg", "image/jpg":
			fileExt = ".jpg"
		case "image/gif":
			fileExt = ".gif"
		case "image/svg+xml":
			fileExt = ".svg"
		case "image/webp":
			fileExt = ".webp"
		case "image/x-icon", "image/vnd.microsoft.icon":
			fileExt = ".ico"
		default:
			fileExt = ".ico" // 默认为.ico
		}
	}

	// 生成唯一文件名
	fileName := cmn.Md5(fmt.Sprintf("%s%s", urlFileName, time.Now().String())) + fileExt

	destination := savePath + "/" + fileName

	// 创建本地文件用于保存图片
	file, err := os.Create(destination)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 将图片数据写入本地文件
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return nil, err
	}

	// 重新打开文件以返回文件句柄
	return os.Open(destination)
}

func GetOneFaviconURLAndUpload(urlStr string) (string, bool) {
	iconURLs, err := getFaviconURL(urlStr)
	if err != nil {
		return "", false
	}

	for _, v := range iconURLs {
		// 标准的路径地址
		if IsHTTPURL(v) {
			return v, true
		} else {
			urlInfo, _ := url.Parse(urlStr)
			fullUrl := urlInfo.Scheme + "://" + urlInfo.Host + "/" + strings.TrimPrefix(v, "/")
			return fullUrl, true
		}
	}
	return "", false
}

func getFaviconURL(urlStr string) ([]string, error) {
	var icons []string
	icons = make([]string, 0)
	client := &http.Client{
		Timeout: 10 * time.Second, // 添加超时设置
	}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return icons, err
	}

	// 设置User-Agent头字段，模拟现代浏览器请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return icons, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return icons, errors.New("HTTP request failed with status code " + strconv.Itoa(resp.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return icons, err
	}

	// 查找所有link标签，按照优先级收集图标链接
	// 1. 首先查找apple-touch-icon，这通常是高质量图标
	doc.Find("link[rel='apple-touch-icon'], link[rel='apple-touch-icon-precomposed']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			icons = append(icons, href)
		}
	})

	// 2. 查找标准的icon和shortcut icon
	doc.Find("link[rel='icon'], link[rel='shortcut icon']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			icons = append(icons, href)
		}
	})

	// 3. 查找任何包含"icon"关键词的link标签
	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		rel, exists := s.Attr("rel")
		if !exists {
			return
		}

		if strings.Contains(strings.ToLower(rel), "icon") {
			if href, exists := s.Attr("href"); exists && href != "" {
				// 检查是否已经添加过这个图标，避免重复
				isDuplicate := false
				for _, icon := range icons {
					if icon == href {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					icons = append(icons, href)
				}
			}
		}
	})

	// 4. 查找网站根目录下的favicon.ico
	urlParsed, _ := url.Parse(urlStr)
	defaultFavicon := urlParsed.Scheme + "://" + urlParsed.Host + "/favicon.ico"
	icons = append(icons, defaultFavicon)

	if len(icons) == 0 {
		return icons, errors.New("favicon not found on the page")
	}

	return icons, nil
}

// GetWebTitleAndDescription 获取网页标题和描述
func GetWebTitleAndDescription(urlStr string) (title string, description string, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", "", err
	}

	// 设置User-Agent头字段，模拟浏览器请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New("HTTP request failed with status code " + strconv.Itoa(resp.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", err
	}

	// 获取网页标题
	title = doc.Find("title").Text()

	// 获取网页描述
	description = ""
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("name"); strings.ToLower(name) == "description" {
			if content, exists := s.Attr("content"); exists {
				description = content
			}
		}
	})

	return title, description, nil
}
