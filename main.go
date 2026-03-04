package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	DEFAULT_MODEL    = "gemini-3.1-flash-image-preview"
	DEFAULT_PREFIX   = "gemini_generated"
	DEFAULT_BASE_URL = "https://gateway.theturbo.ai/v1"
	LISTEN_ADDR      = ":8080"
	OUTPUTS_DIR      = "./static/outputs" // 图片统一保存在 static/outputs 下，可被 web 直接访问
)

type GenerateRequest struct {
	Prompt      string `json:"prompt"`
	Model       string `json:"model"`
	AspectRatio string `json:"aspect_ratio"`
	ImageSize   string `json:"image_size"`
	Prefix      string `json:"prefix"`
	BaseURL     string `json:"base_url"`
}

type GenerateResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	ImageURL string `json:"image_url,omitempty"` // 可直接在浏览器访问的 URL 路径
	FileName string `json:"file_name,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

// ImageItem 用于资产列表
type ImageItem struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Date     string `json:"date"`
	Size     int64  `json:"size"`
}

// generating 用原子标志防止并发生成（0=空闲，1=生成中）
var generating int32

var validModels = []string{
	"gemini-2.5-flash-image",
	"gemini-3-pro-image-preview",
	"gemini-3.1-flash-image-preview",
}

var standardRatios = []string{"1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9"}
var extendedRatios = []string{"1:4", "1:8", "4:1", "8:1"}
var validImageSizes = []string{"1K", "2K", "4K"}

func main() {
	apiKey := strings.Trim(os.Getenv("GEMINI_API_KEY"), `"' `)
	if apiKey == "" {
		log.Fatal("❌ 环境变量 GEMINI_API_KEY 未设置\n" +
			"  cmd.exe:    set GEMINI_API_KEY=your-api-key\n" +
			"  PowerShell: $env:GEMINI_API_KEY=\"your-api-key\"")
	}

	// 确保输出根目录存在
	if err := os.MkdirAll(OUTPUTS_DIR, 0755); err != nil {
		log.Fatalf("无法创建输出目录: %v", err)
	}

	// 静态文件服务（含 outputs 子目录中的图片）
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		generateHandler(w, r, apiKey)
	})
	http.HandleFunc("/api/config", configHandler)
	http.HandleFunc("/api/gallery", galleryHandler)

	fmt.Printf("🚀 Gemini Image Web App 已启动\n")
	fmt.Printf("   访问地址: http://localhost%s\n", LISTEN_ADDR)
	fmt.Printf("   图片输出: %s\n", OUTPUTS_DIR)
	fmt.Printf("   API Key 长度: %d\n\n", len(apiKey))

	server := &http.Server{
		Addr:         LISTEN_ADDR,
		ReadTimeout:  10 * time.Second,   // 读取请求头超时
		WriteTimeout: 300 * time.Second,  // 写响应超时（图片生成最多等 5 分钟）
		IdleTimeout:  120 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// galleryHandler 扫描 outputs 目录，返回所有图片列表（按日期降序）
func galleryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	type DateGroup struct {
		Date   string      `json:"date"`
		Images []ImageItem `json:"images"`
	}

	// 遍历日期子目录
	dateEntries, err := os.ReadDir(OUTPUTS_DIR)
	if err != nil {
		json.NewEncoder(w).Encode([]DateGroup{})
		return
	}

	var groups []DateGroup

	for _, de := range dateEntries {
		if !de.IsDir() {
			continue
		}
		dateDir := filepath.Join(OUTPUTS_DIR, de.Name())
		files, err := os.ReadDir(dateDir)
		if err != nil {
			continue
		}

		var items []ImageItem
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") && !strings.HasSuffix(lower, ".webp") {
				continue
			}
			info, _ := f.Info()
			var sz int64
			if info != nil {
				sz = info.Size()
			}
			items = append(items, ImageItem{
				URL:      "/outputs/" + de.Name() + "/" + name,
				FileName: name,
				Date:     de.Name(),
				Size:     sz,
			})
		}

		// 同一天内按文件名降序（时间戳在末尾，自然降序 = 最新在前）
		sort.Slice(items, func(i, j int) bool {
			return items[i].FileName > items[j].FileName
		})

		if len(items) > 0 {
			groups = append(groups, DateGroup{Date: de.Name(), Images: items})
		}
	}

	// 日期降序
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Date > groups[j].Date
	})

	json.NewEncoder(w).Encode(groups)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	config := map[string]interface{}{
		"models":           validModels,
		"default_model":    DEFAULT_MODEL,
		"default_base_url": DEFAULT_BASE_URL,
		"aspect_ratios": map[string]interface{}{
			"standard":      standardRatios,
			"extended":      extendedRatios,
			"extended_note": "扩展比例仅 gemini-3.1-flash-image-preview 支持",
		},
		"image_sizes":    validImageSizes,
		"default_prefix": DEFAULT_PREFIX,
	}
	json.NewEncoder(w).Encode(config)
}

func generateHandler(w http.ResponseWriter, r *http.Request, apiKey string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "仅支持 POST 请求")
		return
	}

	// 原子 CAS：若已有任务在运行则直接拒绝
	if !atomic.CompareAndSwapInt32(&generating, 0, 1) {
		sendError(w, http.StatusTooManyRequests, "正在生成图片，请等待当前任务完成后再试")
		return
	}
	defer atomic.StoreInt32(&generating, 0)

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		sendError(w, http.StatusBadRequest, "prompt 不能为空")
		return
	}
	if req.Model == "" {
		req.Model = DEFAULT_MODEL
	}
	if req.Prefix == "" {
		req.Prefix = DEFAULT_PREFIX
	}
	if err := validateParameters(req.Model, req.AspectRatio, req.ImageSize); err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 按日期建子目录
	today := time.Now().Format("20060102")
	saveDir := filepath.Join(OUTPUTS_DIR, today)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "创建日期目录失败: "+err.Error())
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	aspectRatioSafe := strings.ReplaceAll(req.AspectRatio, ":", "x")
	modelShort := getModelShortName(req.Model)
	fileName := fmt.Sprintf("%s_%s_%s_%s_%s.png", req.Prefix, modelShort, aspectRatioSafe, req.ImageSize, timestamp)
	filePath := filepath.Join(saveDir, fileName)
	// 浏览器可访问的 URL
	imageURL := "/outputs/" + today + "/" + fileName

	// 使用前端传入的 BaseURL，若为空则使用默认值
	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = DEFAULT_BASE_URL
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	log.Printf("🎨 生成: model=%s ratio=%s size=%s base_url=%s prompt=%q",
		req.Model, req.AspectRatio, req.ImageSize, baseURL, truncate(req.Prompt, 60))

	resp, err := client.Images.Generate(
		context.Background(),
		openai.ImageGenerateParams{
			Model:          openai.ImageModel(req.Model),
			Prompt:         req.Prompt,
			ResponseFormat: openai.ImageGenerateParamsResponseFormatB64JSON,
		},
		option.WithJSONSet("aspect_ratio", req.AspectRatio),
		option.WithJSONSet("image_size", req.ImageSize),
	)
	if err != nil {
		log.Printf("❌ API 失败: %v", err)
		sendError(w, http.StatusInternalServerError, "API 请求失败: "+err.Error())
		return
	}
	if len(resp.Data) == 0 {
		sendError(w, http.StatusInternalServerError, "API 返回空数据")
		return
	}

	item := resp.Data[0]
	log.Printf("📦 响应: B64JSON长度=%d URL=%q", len(item.B64JSON), item.URL)

	var imageBytes []byte

	if item.B64JSON != "" {
		imageBytes, err = base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Base64 解码失败: "+err.Error())
			return
		}
	} else if item.URL != "" {
		log.Printf("🌐 下载 URL: %s", item.URL)
		imageBytes, err = downloadURL(item.URL)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "下载图片失败: "+err.Error())
			return
		}
	} else {
		sendError(w, http.StatusInternalServerError, "API 返回数据中无图片内容")
		return
	}

	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		log.Printf("⚠️ 保存失败: %v", err)
		sendError(w, http.StatusInternalServerError, "图片保存失败: "+err.Error())
		return
	}

	log.Printf("✅ 已保存: %s (%d bytes)", filePath, len(imageBytes))

	json.NewEncoder(w).Encode(GenerateResponse{
		Success:  true,
		Message:  "图片生成并保存成功",
		ImageURL: imageURL,
		FileName: fileName,
		FilePath: filePath,
	})
}

func downloadURL(url string) ([]byte, error) {
	c := &http.Client{Timeout: 120 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func sendError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(GenerateResponse{Success: false, Message: msg})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func validateParameters(model, aspectRatio, imageSize string) error {
	validModel := false
	for _, m := range validModels {
		if model == m {
			validModel = true
			break
		}
	}
	if !validModel {
		return fmt.Errorf("无效模型 '%s'", model)
	}

	validAR := false
	for _, r := range standardRatios {
		if aspectRatio == r {
			validAR = true
			break
		}
	}
	if !validAR {
		for _, r := range extendedRatios {
			if aspectRatio == r {
				if model == "gemini-3.1-flash-image-preview" {
					validAR = true
				} else {
					return fmt.Errorf("宽高比 '%s' 仅 gemini-3.1-flash-image-preview 支持", aspectRatio)
				}
				break
			}
		}
	}
	if !validAR {
		return fmt.Errorf("无效宽高比 '%s'", aspectRatio)
	}

	validSize := false
	for _, s := range validImageSizes {
		if strings.ToUpper(imageSize) == s {
			validSize = true
			break
		}
	}
	if !validSize {
		return fmt.Errorf("无效图片尺寸 '%s'", imageSize)
	}
	return nil
}

func getModelShortName(model string) string {
	switch model {
	case "gemini-2.5-flash-image":
		return "g25f"
	case "gemini-3-pro-image-preview":
		return "g3p"
	case "gemini-3.1-flash-image-preview":
		return "g31f"
	default:
		return "gemini"
	}
}
