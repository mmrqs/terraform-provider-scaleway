package scaleway

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func dataSourceScalewayBaremetalOption() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceScalewayBaremetalOptionRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Exact label of the desired option",
				ConflictsWith: []string{"option_id"},
			},
			"option_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The ID of the option",
				ValidateFunc:  validationUUIDorUUIDWithLocality(),
				ConflictsWith: []string{"name"},
			},
			"manageable": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Is false if the option could not be added or removed",
			},
			"zone": zoneSchema(),
		},
	}
}

func dataSourceScalewayBaremetalOptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, zone, err := baremetalAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var optionName string
	var optionManageable bool
	optionID, ok := d.GetOk("option_id")
	if ok {
		optionID = d.Get("option_id")
		res, err := api.GetOption(&baremetal.GetOptionRequest{
			Zone:     zone,
			OptionID: optionID.(string),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		optionManageable = res.Manageable
		optionName = res.Name
	} else {
		res, err := api.ListOptions(&baremetal.ListOptionsRequest{
			Zone: zone,
		}, scw.WithAllPages(), scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		if len(res.Options) == 0 {
			return diag.FromErr(fmt.Errorf("no option found with the name %s", d.Get("name")))
		}
		for _, option := range res.Options {
			if option.Name == d.Get("name") {
				optionID, optionManageable, optionName = option.ID, option.Manageable, option.Name
				break
			}
		}
	}

	zoneID := datasourceNewZonedID(optionID, zone)
	d.SetId(zoneID)

	_ = d.Set("option_id", zoneID)
	_ = d.Set("zone", zone)
	_ = d.Set("name", optionName)
	_ = d.Set("manageable", optionManageable)

	return nil
}
