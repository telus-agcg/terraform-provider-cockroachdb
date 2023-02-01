package provider

import (
	"context"
	"fmt"
	"regexp"
	"telusag/terraform-provider-cockroachdb/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jackc/pgx/v5"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &cockroachdbProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &cockroachdbProvider{
			version: version,
		}
	}
}

// cockroachdbProvider is the provider implementation.
type cockroachdbProvider struct {
	configured bool

	config cockroachdbProviderModel

	version string
}

func getConnStr(config cockroachdbProviderModel, database string) string {
	if utils.IsNilOrEmpty(&database) {
		database = "defaultdb"
	}

	return regexp.MustCompile(`\"`).ReplaceAllString(fmt.Sprintf("postgresql://%s@%s:%d/%s?sslmode=%s&sslrootcert=%s&sslcert=%s&sslkey=%s", config.User.ValueString(), config.Host.ValueString(), config.Port.ValueInt64(), database, config.SslConfig.Attributes()["mode"], config.SslConfig.Attributes()["rootcert"], config.SslConfig.Attributes()["cert"], config.SslConfig.Attributes()["key"]), "")
}

func (p *cockroachdbProvider) Conn(ctx context.Context, database string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, getConnStr(p.config, database))
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}

// cockroachdbProviderModel maps provider schema data to a Go type.
type cockroachdbProviderModel struct {
	Host      types.String `tfsdk:"host" env:"COCKROACH_HOST"`
	User      types.String `tfsdk:"user" env:"COCKROACH_USER"`
	Port      types.Int64  `tfsdk:"port" env:"COCKROACH_PORT"`
	SslConfig types.Object `tfsdk:"sslconfig" env:"COCKROACH_SSLCONFIG"`
}

// Metadata returns the provider type name.
func (p *cockroachdbProvider) Metadata(ctx context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cockroachdb"
}

// Schema defines the provider-level schema for configuration data.
func (p *cockroachdbProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Cockroach host name",
			},
			"user": schema.StringAttribute{
				Required:    true,
				Description: "Cockroach user name",
			},
			"port": schema.Int64Attribute{
				Required:    true,
				Description: "Cockroach port number",
			},
			"sslconfig": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"mode":     types.StringType,
					"rootcert": types.StringType,
					"cert":     types.StringType,
					"key":      types.StringType,
				},
				Required:    true,
				Description: "Cockroach SSL config",
			},
		},
	}
}

// Configure prepares a cockroachdb client for data sources and resources.
func (p *cockroachdbProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring cockroachdb client")

	// Retrieve provider data from configuration
	var config cockroachdbProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown CockroachDb Host",
			"The provider cannot create the CockroachDb client as there is an unknown configuration value for the CockroachDb host. "+
				"Target apply the source of the value first and set the value statically in the configuration.",
		)
	}

	if config.User.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("user"),
			"Unknown CockroachDb user",
			"The provider cannot create the CockroachDb client as there is an unknown configuration value for the CockroachDb user. "+
				"Target apply the source of the value first and set the value statically in the configuration.",
		)
	}

	if config.Port.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Unknown CockroachDb port",
			"The provider cannot create the CockroachDb client as there is an unknown configuration value for the CockroachDb port. "+
				"Target apply the source of the value first and set the value statically in the configuration.",
		)
	}

	if config.SslConfig.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("sslconfig"),
			"Unknown CockroachDb sslconfig",
			"The provider cannot create the CockroachDb client as there is an unknown configuration value for the CockroachDb sslconfig. "+
				"Target apply the source of the value first and set the value statically in the configuration.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "cockroachdb_host", config.Host.ValueString())
	ctx = tflog.SetField(ctx, "cockroachdb_user", config.User.ValueString())
	ctx = tflog.SetField(ctx, "cockroachdb_port", config.Port.ValueInt64())

	tflog.Debug(ctx, "Creating cockroachdb client")

	// Set client dsn
	p.config = config
	p.configured = true

	// Make the cockroachdb client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = p
	resp.ResourceData = p

	tflog.Info(ctx, "Configured cockroachdb client", map[string]any{"success": true})
}

func (p *cockroachdbProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatabaseResource,
		NewGrantRoleResource,
		NewGrantResource,
		NewRoleResource,
	}
}

func (p *cockroachdbProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
