package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"mcp-task-manager-go/internal/task"
)

// AutoEvaluationConfig controls automatic task evaluation behavior
type AutoEvaluationConfig struct {
	Enabled           bool          `json:"enabled"`
	CacheTimeout      time.Duration `json:"cache_timeout"`
	MaxConcurrent     int           `json:"max_concurrent"`
	SkipReadOnlyTools bool          `json:"skip_read_only_tools"`
	VerboseLogging    bool          `json:"verbose_logging"`
}

// DefaultAutoEvaluationConfig returns sensible defaults
func DefaultAutoEvaluationConfig() AutoEvaluationConfig {
	return AutoEvaluationConfig{
		Enabled:           true,
		CacheTimeout:      5 * time.Minute,
		MaxConcurrent:     3,
		SkipReadOnlyTools: true,
		VerboseLogging:    false,
	}
}

// EvaluationResult contains the results of automatic task evaluation
type EvaluationResult struct {
	ProjectName     string                 `json:"project_name"`
	UpdatesApplied  []string              `json:"updates_applied"`
	AttentionItems  []task.TaskAttention  `json:"attention_items"`
	EvaluationTime  time.Time             `json:"evaluation_time"`
	ProcessingTime  time.Duration         `json:"processing_time"`
	CacheHit        bool                  `json:"cache_hit"`
}

// AutoEvaluationMiddleware handles automatic task evaluation before tool execution
type AutoEvaluationMiddleware struct {
	taskManager    *task.Manager
	config         AutoEvaluationConfig
	cache          map[string]*EvaluationResult
	cacheMutex     sync.RWMutex
	semaphore      chan struct{}
	readOnlyTools  map[string]bool
}

// NewAutoEvaluationMiddleware creates a new middleware instance
func NewAutoEvaluationMiddleware(taskManager *task.Manager, config AutoEvaluationConfig) *AutoEvaluationMiddleware {
	middleware := &AutoEvaluationMiddleware{
		taskManager: taskManager,
		config:      config,
		cache:       make(map[string]*EvaluationResult),
		semaphore:   make(chan struct{}, config.MaxConcurrent),
		readOnlyTools: map[string]bool{
			"get_next_task":                true,
			"get_task_dependencies":        true,
			"get_tasks_needing_attention":  true,
			"suggest_next_actions":         true,
			"debug_info":                   true,
		},
	}

	// Start cache cleanup goroutine
	go middleware.cleanupCache()

	return middleware
}

// WrapHandler wraps a tool handler with automatic evaluation
func (m *AutoEvaluationMiddleware) WrapHandler(toolName string, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Skip evaluation if disabled
		if !m.config.Enabled {
			return handler(ctx, request)
		}

		// Skip evaluation for read-only tools if configured
		if m.config.SkipReadOnlyTools && m.readOnlyTools[toolName] {
			return handler(ctx, request)
		}

		// Extract project name from request
		projectName := m.extractProjectName(request)
		if projectName == "" {
			// No project name found, proceed without evaluation
			return handler(ctx, request)
		}

		// Perform automatic evaluation
		evaluationResult, err := m.evaluateProject(ctx, projectName)
		if err != nil && m.config.VerboseLogging {
			// Log error but don't fail the original request
			fmt.Printf("Auto-evaluation failed for project %s: %v\n", projectName, err)
		}

		// Execute the original handler
		result, err := handler(ctx, request)
		if err != nil {
			return result, err
		}

		// Enhance result with evaluation information if successful
		if evaluationResult != nil && result != nil {
			enhancedResult := m.enhanceResultWithEvaluation(result, evaluationResult)
			return enhancedResult, nil
		}

		return result, nil
	}
}

// extractProjectName extracts project name from various tool requests
func (m *AutoEvaluationMiddleware) extractProjectName(request mcp.CallToolRequest) string {
	args := request.GetArguments()
	
	// Try common parameter names
	if projectName, ok := args["project_name"].(string); ok && projectName != "" {
		return projectName
	}
	
	// For tools that might auto-detect project, try to detect it
	// This would require access to the detection logic
	return ""
}

// evaluateProject performs comprehensive project evaluation
func (m *AutoEvaluationMiddleware) evaluateProject(ctx context.Context, projectName string) (*EvaluationResult, error) {
	// Check cache first
	if cached := m.getCachedResult(projectName); cached != nil {
		return cached, nil
	}

	// Acquire semaphore to limit concurrent evaluations
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	startTime := time.Now()

	// Check if project exists
	if !m.taskManager.ProjectExists(projectName) {
		return nil, fmt.Errorf("project %s does not exist", projectName)
	}

	// Load project
	project, err := m.taskManager.LoadProject(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to load project %s: %w", projectName, err)
	}

	// Perform automatic updates
	updates, hasChanges := task.AutoUpdateTaskStatuses(project)
	
	// Save project if changes were made
	if hasChanges {
		if err := m.taskManager.SaveProject(project); err != nil {
			return nil, fmt.Errorf("failed to save project updates: %w", err)
		}
	}

	// Get tasks needing attention
	attentionItems := task.GetTasksNeedingAttention(project)

	// Create evaluation result
	result := &EvaluationResult{
		ProjectName:    projectName,
		UpdatesApplied: updates,
		AttentionItems: attentionItems,
		EvaluationTime: startTime,
		ProcessingTime: time.Since(startTime),
		CacheHit:       false,
	}

	// Cache the result
	m.cacheResult(projectName, result)

	return result, nil
}

// getCachedResult retrieves cached evaluation result if still valid
func (m *AutoEvaluationMiddleware) getCachedResult(projectName string) *EvaluationResult {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	if cached, exists := m.cache[projectName]; exists {
		if time.Since(cached.EvaluationTime) < m.config.CacheTimeout {
			// Mark as cache hit
			cachedCopy := *cached
			cachedCopy.CacheHit = true
			return &cachedCopy
		}
	}

	return nil
}

// cacheResult stores evaluation result in cache
func (m *AutoEvaluationMiddleware) cacheResult(projectName string, result *EvaluationResult) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	m.cache[projectName] = result
}

// cleanupCache periodically removes expired cache entries
func (m *AutoEvaluationMiddleware) cleanupCache() {
	ticker := time.NewTicker(m.config.CacheTimeout)
	defer ticker.Stop()

	for range ticker.C {
		m.cacheMutex.Lock()
		now := time.Now()
		for projectName, result := range m.cache {
			if now.Sub(result.EvaluationTime) > m.config.CacheTimeout {
				delete(m.cache, projectName)
			}
		}
		m.cacheMutex.Unlock()
	}
}

// enhanceResultWithEvaluation adds evaluation information to tool results
func (m *AutoEvaluationMiddleware) enhanceResultWithEvaluation(originalResult *mcp.CallToolResult, evaluation *EvaluationResult) *mcp.CallToolResult {
	if originalResult.Content == nil {
		return originalResult
	}

	// Try to parse existing content as JSON and enhance it
	for i, content := range originalResult.Content {
		if content.Type == "text" && content.Text != nil {
			var resultData map[string]interface{}
			if err := json.Unmarshal([]byte(*content.Text), &resultData); err == nil {
				// Successfully parsed as JSON, enhance it
				resultData["auto_evaluation"] = map[string]interface{}{
					"project_name":     evaluation.ProjectName,
					"updates_applied":  evaluation.UpdatesApplied,
					"attention_count":  len(evaluation.AttentionItems),
					"processing_time":  evaluation.ProcessingTime.String(),
					"cache_hit":        evaluation.CacheHit,
					"evaluation_time":  evaluation.EvaluationTime.Format(time.RFC3339),
				}

				// Include attention items if any
				if len(evaluation.AttentionItems) > 0 {
					attentionSummary := make([]map[string]interface{}, len(evaluation.AttentionItems))
					for j, item := range evaluation.AttentionItems {
						attentionSummary[j] = map[string]interface{}{
							"task_title": item.Task.Title,
							"reason":     item.Reason,
							"type":       string(item.Type),
						}
					}
					resultData["auto_evaluation"].(map[string]interface{})["attention_items"] = attentionSummary
				}

				// Convert back to JSON
				if enhancedJSON, err := json.Marshal(resultData); err == nil {
					enhancedText := string(enhancedJSON)
					originalResult.Content[i].Text = &enhancedText
				}
			} else {
				// Not JSON, append evaluation summary as text
				evaluationSummary := m.formatEvaluationSummary(evaluation)
				enhancedText := *content.Text + "\n\n" + evaluationSummary
				originalResult.Content[i].Text = &enhancedText
			}
		}
	}

	return originalResult
}

// formatEvaluationSummary creates a human-readable evaluation summary
func (m *AutoEvaluationMiddleware) formatEvaluationSummary(evaluation *EvaluationResult) string {
	var summary strings.Builder
	
	summary.WriteString("🔄 **Auto-Evaluation Summary**\n")
	summary.WriteString(fmt.Sprintf("Project: %s\n", evaluation.ProjectName))
	summary.WriteString(fmt.Sprintf("Processing Time: %s\n", evaluation.ProcessingTime))
	
	if evaluation.CacheHit {
		summary.WriteString("Source: Cache\n")
	} else {
		summary.WriteString("Source: Fresh evaluation\n")
	}

	if len(evaluation.UpdatesApplied) > 0 {
		summary.WriteString(fmt.Sprintf("\n✅ **Updates Applied (%d):**\n", len(evaluation.UpdatesApplied)))
		for _, update := range evaluation.UpdatesApplied {
			summary.WriteString(fmt.Sprintf("- %s\n", update))
		}
	}

	if len(evaluation.AttentionItems) > 0 {
		summary.WriteString(fmt.Sprintf("\n⚠️  **Tasks Needing Attention (%d):**\n", len(evaluation.AttentionItems)))
		for _, item := range evaluation.AttentionItems {
			summary.WriteString(fmt.Sprintf("- %s: %s\n", item.Task.Title, item.Reason))
		}
	}

	if len(evaluation.UpdatesApplied) == 0 && len(evaluation.AttentionItems) == 0 {
		summary.WriteString("\n✨ All tasks are up-to-date and no attention needed.\n")
	}

	return summary.String()
}
