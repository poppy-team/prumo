// Package localintel holds the Local Intelligence contracts (GAP-025):
// KnowledgeTask routing, ResourceManager admission, WorkerSupervisor
// lifecycle and model package manifests. Workers (llama.cpp/GGUF, ONNX)
// live outside the Core Go binary; this package is the boundary they
// implement. Concrete models stay benchmark-driven; No-LLM is first-class.
package localintel

import (
	"fmt"
	"sort"
)

// TaskKind classifies local work.
type TaskKind string

const (
	TaskGeneration TaskKind = "generation"
	TaskRerank     TaskKind = "rerank"
	TaskEmbedding  TaskKind = "embedding"
)

// KnowledgeTask is one routable unit of local work.
type KnowledgeTask struct {
	ID     string         `json:"id"`
	Kind   TaskKind       `json:"kind"`
	Budget map[string]any `json:"budget,omitempty"`
}

// Profile names a capability/resource policy (never a model list).
type Profile string

const (
	ProfileNoLLM      Profile = "no-llm"
	ProfileUltraLight Profile = "ultra-light"
	ProfileLight      Profile = "light"
	ProfileBalanced   Profile = "balanced"
	ProfileUltra      Profile = "ultra"
	ProfileCustom     Profile = "custom"
)

// ModelPackage is a versioned, staged artifact (side-by-side capable).
type ModelPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Runtime string `json:"runtime"` // llama.cpp | onnx | openvino
}

// ResourceStatus reports pressure for admission decisions.
type ResourceStatus struct {
	MemoryPressure string   `json:"memory_pressure"` // low|medium|high|critical
	ResidentModels []string `json:"resident_models,omitempty"`
}

// ResourceManager owns load/admission/eviction (see page 30).
type ResourceManager interface {
	Admit(task KnowledgeTask, status ResourceStatus) (string, error)
	Release(model string)
	Pressure() ResourceStatus
}

// WorkerSupervisor owns worker process lifecycle.
type WorkerSupervisor interface {
	Start(pkg ModelPackage) error
	Stop(name string) error
	Health(name string) (string, error)
}

// Router picks the minimum-sufficient model for a task.
type Router interface {
	Route(task KnowledgeTask, pkgs []ModelPackage) (ModelPackage, error)
}

// MinSufficientRouter implements Router deterministically: first package
// (sorted by name) whose runtime matches the task kind mapping.
func MinSufficientRouter() Router { return minRouter{} }

type minRouter struct{}

var kindRuntime = map[TaskKind][]string{
	TaskGeneration: {"llama.cpp"},
	TaskRerank:     {"llama.cpp"},
	TaskEmbedding:  {"onnx"},
}

func (minRouter) Route(task KnowledgeTask, pkgs []ModelPackage) (ModelPackage, error) {
	if task.ID == "" {
		return ModelPackage{}, fmt.Errorf("task requires id")
	}
	runtimes := kindRuntime[task.Kind]
	if len(runtimes) == 0 {
		return ModelPackage{}, fmt.Errorf("unknown task kind %q", task.Kind)
	}
	sorted := append([]ModelPackage{}, pkgs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, p := range sorted {
		for _, rt := range runtimes {
			if p.Runtime == rt {
				return p, nil
			}
		}
	}
	return ModelPackage{}, fmt.Errorf("no package for kind %q (profiles are policies, not model lists)", task.Kind)
}
