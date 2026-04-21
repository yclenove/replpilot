package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	baseDirName        = ".replpilot"
	taskStateFileName  = "tasks.json"
	checkStateFileName = "preflight_checks.json"
)

type Task struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	ReplicaID string    `json:"replica_id"`
	Mode      string    `json:"mode"`
	DryRun    bool      `json:"dry_run"`
	Status    string    `json:"status"`
	Steps     []string  `json:"steps"`
	Message   string    `json:"message"`
	PreStatus string    `json:"pre_status,omitempty"`
	PostStatus string   `json:"post_status,omitempty"`
	RollbackHint string `json:"rollback_hint,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PreflightCheck struct {
	SourceID  string    `json:"source_id"`
	ReplicaID string    `json:"replica_id"`
	Name      string    `json:"name"`
	OK        bool      `json:"ok"`
	Detail    string    `json:"detail"`
	CheckedAt time.Time `json:"checked_at"`
}

func stateFilePath(fileName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, baseDirName, fileName), nil
}

func LoadTasks() ([]Task, error) {
	path, err := stateFilePath(taskStateFileName)
	if err != nil {
		return nil, err
	}
	return loadTaskFile(path)
}

func SaveTasks(tasks []Task) error {
	path, err := stateFilePath(taskStateFileName)
	if err != nil {
		return err
	}
	return saveJSON(path, tasks)
}

func AppendTask(task Task) error {
	tasks, err := LoadTasks()
	if err != nil {
		return err
	}
	tasks = append(tasks, task)
	return SaveTasks(tasks)
}

func LatestTask(sourceID, replicaID string) (*Task, error) {
	tasks, err := LoadTasks()
	if err != nil {
		return nil, err
	}
	matched := make([]Task, 0)
	for _, t := range tasks {
		if t.SourceID != sourceID {
			continue
		}
		if replicaID != "" && t.ReplicaID != replicaID {
			continue
		}
		matched = append(matched, t)
	}
	if len(matched) == 0 {
		return nil, nil
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].UpdatedAt.After(matched[j].UpdatedAt)
	})
	latest := matched[0]
	return &latest, nil
}

func SavePreflightChecks(checks []PreflightCheck) error {
	path, err := stateFilePath(checkStateFileName)
	if err != nil {
		return err
	}
	return saveJSON(path, checks)
}

func LoadPreflightChecks() ([]PreflightCheck, error) {
	path, err := stateFilePath(checkStateFileName)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []PreflightCheck{}, nil
	}
	var checks []PreflightCheck
	if err := loadJSON(path, &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

func UpsertPreflightChecks(sourceID, replicaID string, items []PreflightCheck) error {
	existed, err := LoadPreflightChecks()
	if err != nil {
		return err
	}
	filtered := make([]PreflightCheck, 0, len(existed))
	for _, item := range existed {
		if item.SourceID == sourceID && item.ReplicaID == replicaID {
			continue
		}
		filtered = append(filtered, item)
	}
	filtered = append(filtered, items...)
	return SavePreflightChecks(filtered)
}

func LatestPreflightChecks(sourceID, replicaID string) ([]PreflightCheck, error) {
	checks, err := LoadPreflightChecks()
	if err != nil {
		return nil, err
	}
	result := make([]PreflightCheck, 0)
	for _, item := range checks {
		if item.SourceID == sourceID && item.ReplicaID == replicaID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CheckedAt.After(result[j].CheckedAt)
	})
	return result, nil
}

func loadTaskFile(path string) ([]Task, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Task{}, nil
	}
	var tasks []Task
	if err := loadJSON(path, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func loadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取状态文件失败: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析状态文件失败: %w", err)
	}
	return nil
}

func saveJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建状态目录失败: %w", err)
	}
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}
	return nil
}
