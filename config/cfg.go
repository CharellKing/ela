package config

type TaskAction string

const (
	TaskActionCopyIndex TaskAction = "copy_index"
	TaskActionSync      TaskAction = "sync"
	TaskActionSyncDiff  TaskAction = "sync_diff"
	TaskActionCompare   TaskAction = "compare"
)

type TaskCfg struct {
	Name             string       `mapstructure:"name"`
	IndexPattern     *string      `mapstructure:"index_pattern"`
	SourceES         string       `mapstructure:"source_es"`
	TargetES         string       `mapstructure:"target_es"`
	IndexPairs       []*IndexPair `mapstructure:"index_pairs"`
	TaskAction       TaskAction   `mapstructure:"action"`
	Force            bool         `mapstructure:"force"`
	ScrollTime       uint         `mapstructure:"scroll_time"`
	Parallelism      uint         `mapstructure:"parallelism"`
	SliceSize        uint         `mapstructure:"slice_size"`
	BufferCount      uint         `mapstructure:"buffer_count"`
	WriteParallelism uint         `mapstructure:"write_parallelism"`
	WriteSize        uint         `mapstructure:"write_size"`
}

type IndexPair struct {
	SourceIndex string `mapstructure:"source_index"`
	TargetIndex string `mapstructure:"target_index"`
}

type ESConfig struct {
	Addresses []string `mapstructure:"addresses"`
	User      string   `mapstructure:"user"`
	Password  string   `mapstructure:"password"`
}

type Config struct {
	ESConfigs map[string]*ESConfig `mapstructure:"elastics"`
	Tasks     []*TaskCfg           `mapstructure:"tasks"`
	Level     string               `mapstructure:"level"`
}
