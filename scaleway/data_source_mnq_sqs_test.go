package scaleway

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceMNQSQS_Basic(t *testing.T) {
	tt := NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayMNQSQSDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_account_project main {
						name = "tf_tests_ds_mnq_sqs_basic"
					}

					resource scaleway_mnq_sqs main {
						project_id = scaleway_account_project.main.id
					}

					data scaleway_mnq_sqs main {
						project_id = scaleway_mnq_sqs.main.project_id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayMNQSQSExists(tt, "scaleway_mnq_sqs.main"),

					resource.TestCheckResourceAttrPair("scaleway_mnq_sqs.main", "id", "data.scaleway_mnq_sqs.main", "id"),
				),
			},
		},
	})
}
