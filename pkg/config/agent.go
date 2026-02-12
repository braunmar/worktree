package config

// ScheduledAgents is a map of agent task names to their configurations
type ScheduledAgents map[string]*AgentTask

// AgentTask represents a scheduled agent maintenance task
type AgentTask struct {
	Name          string       `yaml:"name"`
	Description   string       `yaml:"description"`
	Schedule      string       `yaml:"schedule"`
	Context       AgentContext `yaml:"context"`
	Steps         []AgentStep  `yaml:"steps,omitempty"`
	Safety        SafetyConfig `yaml:"safety"`
	Notifications NotifyConfig `yaml:"notifications"`
	GSD           *GSDConfig   `yaml:"gsd,omitempty"` // GSD framework integration
}

// AgentContext defines the execution environment for an agent task
type AgentContext struct {
	Preset   string `yaml:"preset"`   // Which preset to use (frontend, backend, fullstack)
	Branch   string `yaml:"branch"`   // Base branch to work from
	Instance int    `yaml:"instance"` // Instance number for port allocation
	Yolo     bool   `yaml:"yolo"`     // Enable YOLO mode for autonomous execution
}

// AgentStep represents a single step in an agent task
type AgentStep struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`                  // "shell" or "skill"
	Command    string `yaml:"command,omitempty"`     // For shell steps
	Skill      string `yaml:"skill,omitempty"`       // For skill steps
	Args       string `yaml:"args,omitempty"`        // Arguments for skill steps
	WorkingDir string `yaml:"working_dir,omitempty"` // Working directory for execution
}

// SafetyConfig defines safety mechanisms for agent tasks
type SafetyConfig struct {
	Gates    []SafetyGate   `yaml:"gates"`
	Git      GitConfig      `yaml:"git"`
	Rollback RollbackConfig `yaml:"rollback"`
}

// SafetyGate represents a quality gate that must pass before committing
type SafetyGate struct {
	Name     string `yaml:"name"`
	Command  string `yaml:"command"`
	Required bool   `yaml:"required"`
}

// GitConfig defines Git operations for agent tasks
type GitConfig struct {
	Branch        string     `yaml:"branch"`
	CommitMessage string     `yaml:"commit_message"`
	Push          PushConfig `yaml:"push"`
}

// PushConfig defines push and PR creation settings
type PushConfig struct {
	Enabled   bool   `yaml:"enabled"`
	CreatePR  bool   `yaml:"create_pr"`
	PRTitle   string `yaml:"pr_title"`
	PRBody    string `yaml:"pr_body"`
	AutoMerge bool   `yaml:"auto_merge"`
}

// RollbackConfig defines rollback behavior on failure
type RollbackConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Strategy string `yaml:"strategy"` // "cleanup-worktree"
}

// NotifyConfig defines notification channels for agent tasks
type NotifyConfig struct {
	OnSuccess []Notification `yaml:"on_success,omitempty"`
	OnFailure []Notification `yaml:"on_failure,omitempty"`
}

// Notification represents a single notification channel
type Notification struct {
	Type       string   `yaml:"type"`                 // "slack", "email", "gitlab_issue"
	Project    string   `yaml:"project,omitempty"`    // GitLab project (for gitlab_issue)
	Title      string   `yaml:"title,omitempty"`      // Slack channel or email subject
	Body       string   `yaml:"body,omitempty"`       // Message body
	Labels     []string `yaml:"labels,omitempty"`     // Labels (for gitlab_issue)
	Recipients []string `yaml:"recipients,omitempty"` // Webhook URL (for slack) or email addresses
}

// GSDConfig defines GSD framework integration settings
type GSDConfig struct {
	Enabled      bool   `yaml:"enabled"`                  // Enable GSD workflow
	Milestone    string `yaml:"milestone"`                // GSD milestone name
	ReadTaskFile bool   `yaml:"read_task_file,omitempty"` // Read .task.md from worktree
	AutoExecute  bool   `yaml:"auto_execute,omitempty"`   // Auto-execute after planning
}
