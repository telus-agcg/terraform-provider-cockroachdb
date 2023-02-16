package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cockroachdb": providerserver.NewProtocol6WithError(New("test")()),
}

type SslConfig struct {
	mode     string
	rootcert string
	cert     string
	key      string
}

type ProviderConfig struct {
	host      string
	port      int
	user      string
	sslconfig SslConfig
}

var providerConfig = `
provider "cockroachdb" {
	host = "%s"
	port = %d
	user = "%s"
	sslconfig = {
		mode = "%s"
		rootcert = "%s"
		cert = "%s"
		key = "%s"
	}
}
`

func getProviderVals() ProviderConfig {
	return ProviderConfig{
		host: "localhost",
		port: 26257,
		user: "root",
		sslconfig: SslConfig{
			mode:     "verify-ca",
			rootcert: "/Users/bblazer/git/terraform-provider-cockroachdb/certs/ca.crt",
			cert:     "/Users/bblazer/git/terraform-provider-cockroachdb/certs/client.root.crt",
			key:      "/Users/bblazer/git/terraform-provider-cockroachdb/certs/client.root.key",
		},
	}
}

func prefixProvider(resourceConfig string) string {
	pc := getProviderVals()

	return fmt.Sprintf(providerConfig, pc.host, pc.port, pc.user, pc.sslconfig.mode, pc.sslconfig.rootcert, pc.sslconfig.cert, pc.sslconfig.key) + resourceConfig
}
