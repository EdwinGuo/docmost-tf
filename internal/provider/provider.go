package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &DocmostProvider{}

type DocmostProvider struct {
	version string
}

type DocmostProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Email    types.String `tfsdk:"email"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DocmostProvider{version: version}
	}
}

func (p *DocmostProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "docmost"
	resp.Version = p.version
}

func (p *DocmostProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Docmost spaces, groups, and memberships.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Docmost instance URL (e.g. https://docs.example.com). Can also be set via DOCMOST_HOST env var.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "Admin email for authentication. Can also be set via DOCMOST_EMAIL env var.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Admin password for authentication. Can also be set via DOCMOST_PASSWORD env var.",
				Optional:    true,
				Sensitive:   true,
			},
			"token": schema.StringAttribute{
				Description: "Auth token (alternative to email/password). Can also be set via DOCMOST_TOKEN env var.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *DocmostProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config DocmostProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := envOrValue(config.Host, "DOCMOST_HOST")
	email := envOrValue(config.Email, "DOCMOST_EMAIL")
	password := envOrValue(config.Password, "DOCMOST_PASSWORD")
	token := envOrValue(config.Token, "DOCMOST_TOKEN")

	if host == "" {
		resp.Diagnostics.AddError("Missing host", "The 'host' attribute or DOCMOST_HOST environment variable must be set.")
		return
	}

	if token == "" && (email == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing credentials",
			"Either 'token' (DOCMOST_TOKEN) or both 'email' (DOCMOST_EMAIL) and 'password' (DOCMOST_PASSWORD) must be set.",
		)
		return
	}

	client, err := NewDocmostClient(host, email, password, token)
	if err != nil {
		resp.Diagnostics.AddError("Authentication failed", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DocmostProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSpaceResource,
		NewSpaceMemberResource,
		NewGroupResource,
		NewGroupMemberResource,
	}
}

func (p *DocmostProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
	}
}

func envOrValue(val types.String, envKey string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envKey)
}
