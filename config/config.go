package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Config is loaded once from config.json next to the working directory.
type Config struct {
	Port      int              `json:"port"`
	Mode      string           `json:"mode"`
	MySQLDsn  string           `json:"mysqlDsn"`
	Scheduler SchedulerConfig  `json:"scheduler"`
	Browser   BrowserConfig    `json:"browser"`
	LLM       LLMConfig        `json:"llm"`
	Prompts   PromptsConfig    `json:"prompts"`
	SeedTasks []SeedTaskConfig `json:"seedTasks"`
}

type SchedulerConfig struct {
	Enabled bool `json:"enabled"`
	TickSec int  `json:"tickSec"`
}

type BrowserConfig struct {
	Bin             string `json:"bin"`
	TimeoutSec      int    `json:"timeoutSec"`
	WaitNetworkIdle bool   `json:"waitNetworkIdle"`
}

type LLMConfig struct {
	BaseURL    string `json:"baseUrl"`
	APIKey     string `json:"apiKey"`
	Model      string `json:"model"`
	TimeoutSec int    `json:"timeoutSec"`
}

type PromptsConfig struct {
	DiffPath string `json:"diffPath"`
}

type SeedTaskConfig struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Interval string `json:"interval"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`
}

var (
	cfg  Config
	once sync.Once
)

func init() {
	once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("getwd: %v", err)
		}
		path := filepath.Join(dir, "config.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("read config %s: %v", path, err)
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			log.Fatalf("parse config: %v", err)
		}
	})
}

// GetConfig returns the loaded application config.
func GetConfig() Config {
	return cfg
}
