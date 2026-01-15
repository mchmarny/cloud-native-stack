// Package argocd provides ArgoCD-based deployment artifact generation.
// Generates ArgoCD Application manifests for GitOps deployment.
package argocd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

//go:embed templates/application.yaml.tmpl
var applicationTemplate string

//go:embed templates/app-of-apps.yaml.tmpl
var appOfAppsTemplate string

//go:embed templates/README.md.tmpl
var readmeTemplate string

func init() {
	// Self-register with the global deployer registry
	registry.MustRegister(types.DeployerTypeArgoCD, NewDeployer)
}

// Deployer generates ArgoCD deployment artifacts.
type Deployer struct{}

// NewDeployer creates a new ArgoCD deployer instance.
func NewDeployer() registry.Deployer {
	return &Deployer{}
}

// ApplicationData contains data for ArgoCD Application template.
type ApplicationData struct {
	Name         string
	Source       string
	Version      string
	Namespace    string
	SyncWave     int    // Deployment order (0 = first, 1 = second, etc.)
	ValuesFile   string // Path to values.yaml relative to Git repo root
	HasManifests bool   // Whether component has manifests/ directory
	ManifestsDir string // Path to manifests directory relative to Git repo root
}

// AppOfAppsData contains data for the app-of-apps template.
type AppOfAppsData struct {
	Components []AppOfAppsComponent
}

// AppOfAppsComponent represents a component reference in app-of-apps.
type AppOfAppsComponent struct {
	Name      string
	Namespace string
	Path      string // Path to component's argocd directory in Git repo
}

// Generate creates ArgoCD Application manifests and deployment README.
func (d *Deployer) Generate(ctx context.Context, recipeResult *recipe.RecipeResult,
	bundleDir string) (*result.Artifacts, error) {

	startTime := time.Now()
	artifacts := result.New(string(types.DeployerTypeArgoCD))

	// Build a map from component name to its position in DeploymentOrder
	orderMap := make(map[string]int)
	for i, name := range recipeResult.DeploymentOrder {
		orderMap[name] = i
	}

	// Generate Application manifest for each component in deployment order
	orderedComponents := orderComponentsByDeployment(recipeResult.ComponentRefs, recipeResult.DeploymentOrder)
	appOfAppsComponents := make([]AppOfAppsComponent, 0, len(orderedComponents))

	for _, componentRef := range orderedComponents {
		// Create argocd subdirectory within the component directory
		componentArgoDir := filepath.Join(bundleDir, componentRef.Name, "argocd")
		if err := os.MkdirAll(componentArgoDir, 0755); err != nil {
			artifacts.Success = false
			artifacts.Error = fmt.Sprintf("failed to create argocd directory for %s: %v", componentRef.Name, err)
			return artifacts, err
		}

		syncWave := orderMap[componentRef.Name] // Default to 0 if not in order

		// Check if component has manifests directory
		manifestsDir := filepath.Join(bundleDir, componentRef.Name, "manifests")
		hasManifests := false
		if info, err := os.Stat(manifestsDir); err == nil && info.IsDir() {
			hasManifests = true
		}

		appData := ApplicationData{
			Name:         componentRef.Name,
			Source:       componentRef.Source,
			Version:      componentRef.Version,
			Namespace:    internal.GetNamespaceForComponent(componentRef.Name),
			SyncWave:     syncWave,
			ValuesFile:   fmt.Sprintf("%s/values.yaml", componentRef.Name),
			HasManifests: hasManifests,
			ManifestsDir: fmt.Sprintf("%s/manifests", componentRef.Name),
		}

		appManifest, err := internal.RenderTemplate(applicationTemplate, appData)
		if err != nil {
			artifacts.Success = false
			artifacts.Error = fmt.Sprintf("failed to render Application template: %v", err)
			return artifacts, err
		}

		// Write application.yaml to component's argocd directory
		appPath := filepath.Join(componentArgoDir, "application.yaml")
		if err := os.WriteFile(appPath, []byte(appManifest), 0600); err != nil {
			artifacts.Success = false
			artifacts.Error = fmt.Sprintf("failed to write Application manifest: %v", err)
			return artifacts, err
		}

		artifacts.Files = append(artifacts.Files, appPath)

		// Track component for app-of-apps
		appOfAppsComponents = append(appOfAppsComponents, AppOfAppsComponent{
			Name:      componentRef.Name,
			Namespace: internal.GetNamespaceForComponent(componentRef.Name),
			Path:      fmt.Sprintf("%s/argocd", componentRef.Name),
		})
	}

	// Generate parent app-of-apps Application at bundle root
	appOfAppsData := AppOfAppsData{
		Components: appOfAppsComponents,
	}
	appOfAppsManifest, err := internal.RenderTemplate(appOfAppsTemplate, appOfAppsData)
	if err != nil {
		artifacts.Success = false
		artifacts.Error = fmt.Sprintf("failed to render app-of-apps template: %v", err)
		return artifacts, err
	}

	// Write app-of-apps.yaml to bundle root (next to recipe.yaml)
	appOfAppsPath := filepath.Join(bundleDir, "app-of-apps.yaml")
	if writeErr := os.WriteFile(appOfAppsPath, []byte(appOfAppsManifest), 0600); writeErr != nil {
		artifacts.Success = false
		artifacts.Error = fmt.Sprintf("failed to write app-of-apps manifest: %v", writeErr)
		return artifacts, writeErr
	}

	artifacts.Files = append(artifacts.Files, appOfAppsPath)

	// Generate README
	readmeData := internal.ReadmeData{
		Timestamp:  time.Now().Format(time.RFC3339),
		Components: make([]internal.ComponentInfo, 0, len(recipeResult.ComponentRefs)),
	}

	for _, componentRef := range recipeResult.ComponentRefs {
		readmeData.Components = append(readmeData.Components, internal.ComponentInfo{
			Name:    componentRef.Name,
			Version: componentRef.Version,
		})
	}

	readme, err := internal.RenderTemplate(readmeTemplate, readmeData)
	if err != nil {
		artifacts.Success = false
		artifacts.Error = fmt.Sprintf("failed to render README template: %v", err)
		return artifacts, err
	}

	readmePath := filepath.Join(bundleDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0600); err != nil {
		artifacts.Success = false
		artifacts.Error = fmt.Sprintf("failed to write README: %v", err)
		return artifacts, err
	}

	artifacts.Files = append(artifacts.Files, readmePath)
	artifacts.ReadmeContent = readme
	artifacts.Duration = time.Since(startTime)

	return artifacts, nil
}

// orderComponentsByDeployment returns components sorted according to deployment order.
// If deployment order is empty or a component is not in the order, it will appear after
// the ordered components in its original position.
func orderComponentsByDeployment(components []recipe.ComponentRef, order []string) []recipe.ComponentRef {
	if len(order) == 0 {
		return components
	}

	// Build position map
	posMap := make(map[string]int, len(order))
	for i, name := range order {
		posMap[name] = i
	}

	// Create result slice
	result := make([]recipe.ComponentRef, len(components))
	copy(result, components)

	// Sort by deployment order
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			posI, okI := posMap[result[i].Name]
			posJ, okJ := posMap[result[j].Name]

			// Components in order come before components not in order
			if !okI && okJ {
				result[i], result[j] = result[j], result[i]
			} else if okI && okJ && posJ < posI {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
