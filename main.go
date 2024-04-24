package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"

	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type ServerConfig struct {
	Servers      []Server `json:"servers"`
	LocalPort    int      `json:"local_port"`
	LocalAddress string   `json:"local_address"`
}

type Server struct {
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Plugin     string `json:"plugin"`
	PluginOpts string `json:"plugin_opts"`
}
type Proxy struct {
	Name       string
	Type       string
	Server     string
	Port       string
	Cipher     string
	Password   string
	Plugin     string
	PluginOpts map[string]string
}

type TemplateData struct {
	Proxies []Proxy
}

// 定义全局变量 portMap
var portMap = map[string]bool{
	"80":   true,
	"8080": true,
	"8880": true,
	"2052": true,
	"2082": true,
	"2086": true,
	"2095": true,
}

func main() {
	var directoryPath string
	fmt.Print("输入shadowsock配置文件所在文件夹: ")
	fmt.Scanln(&directoryPath)

	proxies, err := LoadProxiesFromConfig(directoryPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	finalProxies := make([]Proxy, 0)

	namePrefix := "ss节点"
	var size int
	vpsIP := ""
	for index := range proxies {
		if proxies[index].Server == "0.0.0.0" {
			if vpsIP == "" {
				fmt.Print("请输入 VPS IP 地址: ")
				fmt.Scanln(&vpsIP)
			}
			proxies[index].Server = vpsIP

		}

		port := proxies[index].Port

		//可能是cdn节点
		if portMap[port] && len(proxies[index].PluginOpts) == 3 {
			fmt.Println(proxies[index])
			fmt.Println("检测到当前节点可能是一个cdn节点,需要导入优选ip吗")
			if getUserInput() == "y" {
				csvFile := getCsvFilePath()
				speedFastIps, err := GetIPAddressesFromCSV(csvFile)
				speedFastIps = append(speedFastIps, "csgo.com", "www.visa.com.sg")
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				for _, ip := range speedFastIps {
					proxies[index].Server = ip
					if len(finalProxies) == 0 {
						size = 1
					} else {
						size = len(finalProxies) + 1
					}
					proxies[index].Name = namePrefix + strconv.Itoa(size)
					finalProxies = append(finalProxies, proxies[index])

				}
			} else {
				if len(finalProxies) == 0 {
					size = 1
				} else {
					size = len(finalProxies) + 1
				}
				proxies[index].Name = namePrefix + strconv.Itoa(size)
				finalProxies = append(finalProxies, proxies[index])
			}

		} else {
			if len(finalProxies) == 0 {
				size = 1
			} else {
				size = len(finalProxies) + 1
			}
			proxies[index].Name = namePrefix + strconv.Itoa(size)
			finalProxies = append(finalProxies, proxies[index])
		}

	}
	createYaml(finalProxies)
	fmt.Print("按回车退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// 生成yaml文件
func createYaml(proxies []Proxy) {
	// 创建模板数据
	data := TemplateData{
		Proxies: proxies,
	}

	// 读取模板文件的内容
	templateData, err := ioutil.ReadFile("template.yaml")
	if err != nil {
		fmt.Println("Error reading template file:", err)
		return
	}

	// 解析模板内容
	tmpl, err := template.New("yamlTemplate").Parse(string(templateData))
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	// 创建输出文件
	outputFile, err := os.Create("output.yaml")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	// 应用模板并将结果写入输出文件
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		return
	}

	fmt.Println("YAML 文件 'output.yaml' 创建成功!")
}

// 获取输入
func getUserInput() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("输入 'y' 或者 'n': ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "y" || input == "n" {
			return input
		}

		fmt.Println("输入错误. 请输入'y' 或者 'n'.")
	}
}

// GetIPAddressesFromCSV 从 CSV 文件中获取符合条件的优选IP 地址
func GetIPAddressesFromCSV(csvFilePath string) ([]string, error) {
	// 打开 CSV 文件
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// 创建 CSV Reader
	reader := csv.NewReader(file)

	// 读取 CSV 文件内容
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	// 存储符合条件的第一列数据
	var ipAddresses []string

	// 遍历每行数据，跳过第一行
	for idx, record := range records {
		if idx == 0 {
			continue // 跳过标题行
		}

		// 解析最后一列数据
		speedStr := record[len(record)-1]
		speed, err := strconv.ParseFloat(speedStr, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing speed: %w", err)
		}

		// 检查最后一列数据是否大于 0.00，如果是，则将第一列数据存入数组
		if speed > 0.00 {
			ip := record[0]
			ipAddresses = append(ipAddresses, ip)
		}
	}

	return ipAddresses, nil
}
func getCsvFilePath() string {
	// 获取用户输入的目录路径
	var directoryPath string
	fmt.Print("输入优选ip csv文件所在文件夹: ")
	fmt.Scanln(&directoryPath)

	// 构建 CSV 文件路径
	csvFilePath := filepath.Join(directoryPath, "result.csv")
	return csvFilePath
}

// 从json文件读取节点信息到切片中
func LoadProxiesFromConfig(directoryPath string) ([]Proxy, error) {
	// 构建 JSON 文件路径
	jsonFilePath := filepath.Join(directoryPath, "config.json")

	// 读取 JSON 文件
	jsonFile, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening JSON file: %w", err)
	}
	defer jsonFile.Close()

	// 解析 JSON 数据
	var config ServerConfig
	err = json.NewDecoder(jsonFile).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	proxies := make([]Proxy, 0, len(config.Servers))

	// 替换生成链接中的地址和 path 参数
	for _, server := range config.Servers {
		address := server.Address
		port := strconv.Itoa(server.Port)
		cipher := server.Method
		password := server.Password
		plugin := server.Plugin
		pluginOpts := make(map[string]string)

		if server.PluginOpts == "server" {
			pluginOpts["mode"] = "websocket"
		} else {
			// 提取 path、mode 和 host 的值
			opts := strings.Split(server.PluginOpts, ";")
			for _, opt := range opts {
				parts := strings.SplitN(opt, "=", 2)
				if len(parts) == 2 {
					pluginOpts[parts[0]] = parts[1]
				}
			}
		}

		// 添加结构体到切片
		proxies = append(proxies, Proxy{
			Name:       "ss节点",
			Type:       "ss",
			Server:     address,
			Port:       port,
			Cipher:     cipher,
			Password:   password,
			Plugin:     plugin,
			PluginOpts: pluginOpts,
		})
	}

	return proxies, nil
}
