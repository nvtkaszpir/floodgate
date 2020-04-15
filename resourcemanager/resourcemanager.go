package resourcemanager

import (
	log "github.com/sirupsen/logrus"

	c "github.com/codilime/floodgate/config"
	"github.com/codilime/floodgate/gateclient"
	"github.com/codilime/floodgate/parser"
	spr "github.com/codilime/floodgate/spinnakerresource"
)

// SpinnakerResources Spinnaker resources collection
type SpinnakerResources struct {
	Applications      []*spr.Application
	Pipelines         []*spr.Pipeline
	PipelineTemplates []*spr.PipelineTemplate
}

// ResourceChange store resource change
type ResourceChange struct {
	Type    string
	ID      string
	Name    string
	Changes string
}

// ResourceManager stores Spinnaker resources and has methods for access, syncing etc.
type ResourceManager struct {
	resources         SpinnakerResources
	desyncedResources SpinnakerResources
}

// Init initialize sync
func (rm *ResourceManager) Init(configPath string) error {
	config, err := c.LoadConfig(configPath)
	if err != nil {
		return err
	}
	client := gateclient.NewGateapiClient(config)
	p := parser.CreateParser(config.Libraries)
	if err := p.LoadObjectsFromDirectories(config.Resources); err != nil {
		return err
	}
	resourceData := &p.Resources
	for _, localData := range resourceData.Applications {
		application := &spr.Application{}
		if err := application.Init(client, localData); err != nil {
			return err
		}
		rm.resources.Applications = append(rm.resources.Applications, application)
		changed, err := application.IsChanged()
		if err != nil {
			return err
		}
		if changed {
			rm.desyncedResources.Applications = append(rm.desyncedResources.Applications, application)
		}
	}
	for _, localData := range resourceData.Pipelines {
		pipeline := &spr.Pipeline{}
		if err := pipeline.Init(client, localData); err != nil {
			return err
		}
		rm.resources.Pipelines = append(rm.resources.Pipelines, pipeline)
		changed, err := pipeline.IsChanged()
		if err != nil {
			return err
		}
		if changed {
			rm.desyncedResources.Pipelines = append(rm.desyncedResources.Pipelines, pipeline)
		}
	}
	for _, localData := range resourceData.PipelineTemplates {
		pipelineTemplate := &spr.PipelineTemplate{}
		if err := pipelineTemplate.Init(client, localData); err != nil {
			return err
		}
		rm.resources.PipelineTemplates = append(rm.resources.PipelineTemplates, pipelineTemplate)
		changed, err := pipelineTemplate.IsChanged()
		if err != nil {
			return err
		}
		if changed {
			rm.desyncedResources.PipelineTemplates = append(rm.desyncedResources.PipelineTemplates, pipelineTemplate)
		}
	}
	return nil
}

// GetChanges get resources' changes
func (rm ResourceManager) GetChanges() (changes []ResourceChange) {
	for _, application := range rm.resources.Applications {
		var change string
		changed, err := application.IsChanged()
		if err != nil {
			log.Fatal(err)
		}
		if changed {
			change = application.GetFullDiff()
			changes = append(changes, ResourceChange{Type: "application", ID: "", Name: application.Name(), Changes: change})
		}
	}
	for _, pipeline := range rm.resources.Pipelines {
		var change string
		changed, err := pipeline.IsChanged()
		if err != nil {
			log.Fatal(err)
		}
		if changed {
			change = pipeline.GetFullDiff()
			changes = append(changes, ResourceChange{Type: "pipeline", ID: pipeline.ID(), Name: pipeline.Name(), Changes: change})
		}
	}
	for _, pipelineTemplate := range rm.resources.PipelineTemplates {
		var change string
		changed, err := pipelineTemplate.IsChanged()
		if err != nil {
			log.Fatal(err)
		}
		if changed {
			change = pipelineTemplate.GetFullDiff()
			changes = append(changes, ResourceChange{Type: "pipelinetemplate", ID: pipelineTemplate.ID(), Name: pipelineTemplate.Name(), Changes: change})
		}
	}
	return
}

// SyncResources synchronize resources with Spinnaker
func (rm *ResourceManager) SyncResources() error {
	if err := rm.syncApplications(); err != nil {
		log.Fatal(err)
	}
	if err := rm.syncPipelines(); err != nil {
		log.Fatal(err)
	}
	if err := rm.syncPipelineTemplates(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (rm ResourceManager) syncResource(resource spr.Resourcer) (bool, error) {
	needToSave, err := resource.IsChanged()
	if err != nil {
		return false, err
	}
	if !needToSave {
		return false, nil
	}
	if err := resource.SaveLocalState(); err != nil {
		return false, err
	}
	return true, nil
}

func (rm ResourceManager) syncApplications() error {
	log.Print("Syncing applications")
	for _, application := range rm.resources.Applications {
		synced, err := rm.syncResource(application)
		if err != nil {
			log.Warn("failed to sync application: ", application)
			return err
		}
		if !synced {
			log.Printf("No need to save application %v", application)
		} else {
			log.Printf("Successfully synced application %v", application)
		}
	}
	return nil
}

func (rm ResourceManager) syncPipelines() error {
	log.Print("Syncing pipelines")
	for _, pipeline := range rm.resources.Pipelines {
		synced, err := rm.syncResource(pipeline)
		if err != nil {
			log.Warn("failed to sync pipeline: ", pipeline)
			return err
		}
		if !synced {
			log.Printf("No need to save pipeline %v", pipeline)
		}
	}
	return nil
}

func (rm ResourceManager) syncPipelineTemplates() error {
	log.Print("Syncing pipeline templates")
	for _, pipelineTemplate := range rm.resources.PipelineTemplates {
		synced, err := rm.syncResource(pipelineTemplate)
		if err != nil {
			log.Warn("failed to sync pipeline template: ", pipelineTemplate)
			return err
		}
		if !synced {
			log.Printf("No need to save pipeline template %v", pipelineTemplate)
		}
	}
	return nil
}

// GetAllApplicationsRemoteState returns a concatenated string of applications JSONs.
func (rm *ResourceManager) GetAllApplicationsRemoteState() (state string) {
	for _, application := range rm.resources.Applications {
		state += string(application.GetRemoteState())
	}
	return
}

// GetAllPipelinesRemoteState returns a concatenated string of pipelines JSONs.
func (rm *ResourceManager) GetAllPipelinesRemoteState() (state string) {
	for _, pipeline := range rm.resources.Pipelines {
		state += string(pipeline.GetRemoteState())
	}
	return
}

// GetAllPipelineTemplatesRemoteState returns a concatenated string of pipeline templates JSONs.
func (rm *ResourceManager) GetAllPipelineTemplatesRemoteState() (state string) {
	for _, pipelineTemplate := range rm.resources.Applications {
		state += string(pipelineTemplate.GetRemoteState())
	}
	return
}
