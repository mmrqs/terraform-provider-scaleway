package scaleway

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	jobs "github.com/scaleway/scaleway-sdk-go/api/jobs/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// jobsAPIWithRegion returns a new jobs API and the region for a Create request
func jobsAPIWithRegion(d *schema.ResourceData, m interface{}) (*jobs.API, scw.Region, error) {
	meta := m.(*Meta)
	jobsAPI := jobs.NewAPI(meta.scwClient)

	region, err := extractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}

	return jobsAPI, region, nil
}

// jobsAPIWithRegionalAndID returns a new jobs API with region and ID extracted from the state
func jobsAPIWithRegionAndID(m interface{}, regionalID string) (*jobs.API, scw.Region, string, error) {
	meta := m.(*Meta)
	jobsAPI := jobs.NewAPI(meta.scwClient)

	region, ID, err := parseRegionalID(regionalID)
	if err != nil {
		return nil, "", "", err
	}

	return jobsAPI, region, ID, nil
}

type JobDefinitionCron struct {
	Schedule string
	Timezone string
}

func (c *JobDefinitionCron) ToCreateRequest() *jobs.CreateJobDefinitionRequestCronScheduleConfig {
	if c == nil {
		return nil
	}

	return &jobs.CreateJobDefinitionRequestCronScheduleConfig{
		Schedule: c.Schedule,
		Timezone: c.Timezone,
	}
}

func (c *JobDefinitionCron) ToUpdateRequest() *jobs.UpdateJobDefinitionRequestCronScheduleConfig {
	if c == nil {
		return &jobs.UpdateJobDefinitionRequestCronScheduleConfig{
			Schedule: nil,
			Timezone: nil,
		} // Send an empty update request to delete cron
	}

	return &jobs.UpdateJobDefinitionRequestCronScheduleConfig{
		Schedule: &c.Schedule,
		Timezone: &c.Timezone,
	}
}

func expandJobDefinitionCron(i any) *JobDefinitionCron {
	rawList := i.([]any)
	if len(rawList) == 0 {
		return nil
	}
	rawCron := rawList[0].(map[string]any)

	return &JobDefinitionCron{
		Schedule: rawCron["schedule"].(string),
		Timezone: rawCron["timezone"].(string),
	}
}

func flattenJobDefinitionCron(cron *jobs.CronSchedule) []any {
	if cron == nil {
		return []any{}
	}

	return []any{
		map[string]any{
			"schedule": cron.Schedule,
			"timezone": cron.Timezone,
		},
	}
}
