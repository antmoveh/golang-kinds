package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type Options struct {
	Args    []string      `json:"args"`
	Url     string        `json:"url"`
	Token   string        `json:"token"`
	Output  string        `json:"output"`
	Number  int           `json:"number"`
	TimeOut time.Duration `json:"timeOut""`
}

// sync 实现从gitlab获取所有项目
func (o *Options) sync() {
	projects := []Project{}
	var err error

	if !isExists(o.Output) {
		err = os.MkdirAll(o.Output, 0644)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	projectJson := path.Join(o.Output, "projects.json")
	if isExists(projectJson) {
		// 读取json文件,序列化到projects
		data, err := os.ReadFile(projectJson)
		if err != nil {
			fmt.Println("Error reading projects from file:", err)
			return
		}
		// 反序列化
		err = json.Unmarshal(data, &projects)
		if err != nil {
			fmt.Println("Error unmarshalling projects from JSON:", err)
			return
		}
	} else {
		// 获取gitlab仓库列表
		projects, err = getProjects(o.Url, o.Token, 1000)
		if err != nil {
			fmt.Println("Error fetching projects:", err)
			return
		}
		// 将projects结果写入本地json文件
		data, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			fmt.Println("Error marshalling projects to JSON:", err)
			return
		}

		err = os.WriteFile(projectJson, data, 0644)
		if err != nil {
			fmt.Println("Error writing projects to file:", err)
			return
		}
	}

	// 下载仓库
	downloadRepo(projects, o.Output, o.Number, o.TimeOut)
}

// Project 代表GitLab项目的简化结构
type Project struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	WebURL        string `json:"web_url"`
	SSHURLToRepo  string `json:"ssh_url_to_repo"`
	HTTPURLToRepo string `json:"http_url_to_repo"`
}

func getProjects(url, token string, perPage int) ([]Project, error) {

	projectURL := fmt.Sprintf("%s/api/v4/projects?page=1&per_page=%d", url, perPage)
	req, err := http.NewRequest("GET", projectURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("PRIVATE-TOKEN", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var projects []Project
	err = json.Unmarshal(body, &projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func downloadRepo(projects []Project, output string, number int, timeout time.Duration) {

	// 获取output目录下前两层的文件
	files, err := os.ReadDir(output)
	if err != nil {
		fmt.Println("Error reading output directory:", err)
		return
	}

	existProjectNumber := 0
	// 遍历文件
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		// 获取目录下的文件
		subFiles, err := os.ReadDir(path.Join(output, file.Name()))
		if err != nil {
			fmt.Println("Error reading subdirectory:", err)
			return
		}
		// 遍历文件
		for _, subFile := range subFiles {
			if subFile.IsDir() {
				existProjectNumber++
			}
		}
	}
	fmt.Println(fmt.Sprintf("project number: %d, exist project number: %d", len(projects), existProjectNumber))

	// 保证必须有number个仓库更新，优先更新未有仓库
	start := 0
	if existProjectNumber+number <= len(projects) {
		start = existProjectNumber
	}
	if existProjectNumber+number > len(projects) {
		if existProjectNumber < len(projects) {
			start = len(projects) - number
		} else {
			// 保证必须有number个仓库更新
			if number >= len(projects) {
				start = 0
			} else {
				start = rand.Int() % (len(projects) - number)
			}
		}
	}
	if start < 0 {
		start = 0
	}

	// 遍历projects
	for i := start; i < start+number && i < len(projects); i++ {
		time.Sleep(time.Duration(rand.Int()%10) * time.Second)
		project := projects[i]
		// 解析url, http://gitlab.com/ope/batman.git , ope为目录，batman为项目名称
		// projectURL is the GitLab project URL
		projectURL := project.HTTPURLToRepo
		// Trim the .git suffix
		trimmedURL := strings.TrimSuffix(projectURL, ".git")
		// Get the directory path which includes the project name
		dirPath := path.Dir(trimmedURL)
		// Extract the project name
		projectName := path.Base(trimmedURL)
		// Split the directory path to get the directory name
		parts := strings.Split(dirPath, "/")
		directoryName := parts[len(parts)-1]
		fmt.Printf("Directory: %s, Project Name: %s\n", directoryName, projectName)

		if !isExists(path.Join(output, directoryName)) {
			err = os.MkdirAll(path.Join(output, directoryName), 0644)
			if err != nil {
				fmt.Println("Error creating directory:", err)
				return
			}
		}

		repoName := path.Join(output, directoryName, projectName)
		if isExists(repoName) || number > 999 {
			fmt.Println(fmt.Sprintf("repoName: %s exists", repoName))
			// Navigate into the repository directory
			err = os.Chdir(repoName)
			if err != nil {
				fmt.Println("Error changing directory:", err)
				continue
			}
			// Pull the latest changes for the current branch
			err = runCmd("git pull", timeout)
			if err != nil {
				fmt.Println("Error pulling repository:", err)
				continue
			}
			// Fetch all branches
			err = runCmd("git fetch --all", timeout)
			if err != nil {
				fmt.Println("Error fetching branches:", err)
				continue
			}
			// Navigate back to the original directory
			err = os.Chdir("..")
			if err != nil {
				fmt.Println("Error changing directory:", err)
				continue
			}
		} else {
			// Clone the repository
			cmd := fmt.Sprintf("git clone %s %s", project.HTTPURLToRepo, repoName)
			fmt.Println(cmd)
			if err = runCmd(cmd, timeout); err != nil {
				fmt.Println("Error cloning repository:", err)
				continue
			}
			// Navigate into the cloned repository directory
			err = os.Chdir(repoName)
			if err != nil {
				fmt.Println("Error changing directory:", err)
				continue
			}
			// Fetch all branches
			err = runCmd("git fetch --all", timeout)
			if err != nil {
				fmt.Println("Error fetching branches:", err)
				continue
			}
			// Navigate back to the original directory
			err = os.Chdir("..")
			if err != nil {
				fmt.Println("Error changing directory:", err)
				continue
			}
		}
	}
}

func runCmd(cmd string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	parts := strings.Fields(cmd)
	out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("command timed out")
		}
		return err
	}
	fmt.Println(string(out))
	return nil
}

// isExists判断文件是否存在
//func isExists(path string) bool {
//	_, err := os.Stat(path)
//	if err != nil {
//		fmt.Println(err.Error())
//	}
//	return err == nil || os.IsExist(err)
//}

// win 判断文件夹是否存在
func isExists(path string) bool {
	// 判断目录是否存在
	_, err := os.Stat(path)
	if err != nil {
		fmt.Println(err.Error())
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
