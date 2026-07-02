package servertests

import (
	"fmt"
	"testing"

	"github.com/UpCloudLtd/terraform-provider-upcloud/upcloud"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceUpCloudServer(t *testing.T) {
	hostname := "tf-acc-test-server-ds"
	plan := "1xCPU-1GB"
	zone := "fi-hel1"

	config := fmt.Sprintf(`
	resource "upcloud_server" "this" {
		hostname = "%s"
		zone     = "%s"
		plan     = "%s"
		metadata = true
		firewall = true

		template {
			storage = "%s"
			size    = 10
		}

		network_interface {
			type = "public"
		}

		labels = {
			env = "test"
		}

		tags = ["acc-test"]
	}

	data "upcloud_server" "this" {
		id = upcloud_server.this.id
	}
	`, hostname, zone, plan, upcloud.DebianTemplateUUID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { upcloud.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: upcloud.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "id"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "hostname", hostname),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "zone", zone),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "plan", plan),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "state", "started"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "cpu", "1"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "mem", "1024"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "metadata", "true"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "firewall", "true"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "title"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "boot_order"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "timezone"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "nic_model"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "video_model"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "host"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "labels.env", "test"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.upcloud_server.this", "tags.*", "acc-test"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "network_interface.#", "1"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "network_interface.0.type", "public"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "network_interface.0.ip_address"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "network_interface.0.ip_address_family", "IPv4"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "network_interface.0.mac"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "network_interface.0.bootable"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "network_interface.0.source_ip_filtering", "true"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "storage_devices.#", "1"),
					resource.TestCheckResourceAttr("data.upcloud_server.this", "storage_devices.0.size", "10"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.uuid"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.type"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.tier"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.title"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.address"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.address_position"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.boot_disk"),
					resource.TestCheckResourceAttrSet("data.upcloud_server.this", "storage_devices.0.encrypted"),
				),
			},
		},
	})
}
