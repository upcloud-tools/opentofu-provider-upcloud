package accounttests

import (
	"testing"

	upc "github.com/UpCloudLtd/terraform-provider-upcloud/upcloud"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceUpCloudAccount_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { upc.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: upc.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "upcloud_account" "this" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "id"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "username"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "credits"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "resource_limits.cores"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "resource_limits.memory_mb"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "resource_limits.public_ipv4"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "resource_limits.storage_hdd"),
					resource.TestCheckResourceAttr("data.upcloud_account.this", "account_details.type", "main"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "account_details.currency"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "account_details.first_name"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "account_details.last_name"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "account_details.email"),
					resource.TestCheckResourceAttrSet("data.upcloud_account.this", "account_details.company"),
				),
			},
		},
	})
}
