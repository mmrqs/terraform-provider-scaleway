package scaleway

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayDocumentDBInstancePrivateNetworkEndpoint() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayDocumentDBInstanceEndpointCreate,
		ReadContext:   resourceScalewayDocumentDBInstanceEndpointRead,
		UpdateContext: resourceScalewayDocumentDBInstanceEndpointUpdate,
		DeleteContext: resourceScalewayDocumentDBInstanceEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultRdbInstanceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Instance on which the endpoint is attached",
			},
			"private_network_id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validationUUIDorUUIDWithLocality(),
				DiffSuppressFunc: diffSuppressFuncLocality,
				Description:      "The private network ID",
				ForceNew:         true,
			},
			// Computed
			"ip_net": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsCIDR,
				Description:  "The IP with the given mask within the private subnet",
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP of your private network service",
			},
			"port": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsPortNumber,
				Description:  "The port of your private service",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of your private service",
			},
			"hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The hostname of your endpoint",
			},
			"zone":   zoneSchema(),
			"region": regionSchema(),
		},
	}
}

func resourceScalewayDocumentDBInstanceEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := documentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := expandID(d.Get("instance_id"))
	endpointSpecPN := &documentdb.EndpointSpecPrivateNetwork{}
	createEndpointRequest := &documentdb.CreateEndpointRequest{
		Region:       region,
		InstanceID:   instanceID,
		EndpointSpec: &documentdb.EndpointSpec{},
	}

	endpointSpecPN.PrivateNetworkID = expandID(d.Get("private_network_id").(string))
	ipNet := d.Get("ip_net").(string)
	if len(ipNet) > 0 {
		ip, err := expandIPNet(ipNet)
		if err != nil {
			return diag.FromErr(err)
		}
		endpointSpecPN.ServiceIP = &ip
	} else {
		endpointSpecPN.IpamConfig = &documentdb.EndpointSpecPrivateNetworkIpamConfig{}
	}

	createEndpointRequest.EndpointSpec.PrivateNetwork = endpointSpecPN
	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		if is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	endpoint, err := api.CreateEndpoint(createEndpointRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		if is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, endpoint.ID))

	return resourceScalewayDocumentDBInstanceEndpointRead(ctx, d, meta)
}

func resourceScalewayDocumentDBInstanceEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoint, err := api.GetEndpoint(&documentdb.GetEndpointRequest{
		EndpointID: id,
		Region:     region,
	}, scw.WithContext(ctx))
	if err != nil {
		if is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	pnID := newRegionalIDString(region, endpoint.PrivateNetwork.PrivateNetworkID)
	serviceIP, err := flattenIPNet(endpoint.PrivateNetwork.ServiceIP)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("private_network_id", pnID)
	_ = d.Set("ip_net", serviceIP)
	_ = d.Set("zone", endpoint.PrivateNetwork.Zone)
	_ = d.Set("port", int(endpoint.Port))
	_ = d.Set("name", endpoint.Name)
	_ = d.Set("hostname", endpoint.Hostname)
	_ = d.Set("ip", flattenIPPtr(endpoint.IP))
	_ = d.Set("region", region.String())

	return nil
}

func resourceScalewayDocumentDBInstanceEndpointUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &documentdb.MigrateEndpointRequest{
		EndpointID: id,
		Region:     region,
	}

	if d.HasChange("instance_id") {
		req.InstanceID = expandID(d.Get("instance_id"))

		if _, err := api.MigrateEndpoint(req, scw.WithContext(ctx)); err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForDocumentDBInstance(ctx, api, region, req.InstanceID, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceScalewayDocumentDBInstanceEndpointRead(ctx, d, meta)
}

func resourceScalewayDocumentDBInstanceEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.DeleteEndpoint(&documentdb.DeleteEndpointRequest{
		Region:     region,
		EndpointID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
