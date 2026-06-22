package upcloud

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func NewProviderServerFactory() func() tfprotov6.ProviderServer {
	return providerserver.NewProtocol6(New())
}
