package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct {
	client *DocmostClient
}

type UserDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Email    types.String `tfsdk:"email"`
	Name     types.String `tfsdk:"name"`
	Role     types.String `tfsdk:"role"`
	Locale   types.String `tfsdk:"locale"`
	Timezone types.String `tfsdk:"timezone"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Docmost workspace user by email address.",
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Required:    true,
				Description: "Email address of the user to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "User UUID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "User display name.",
			},
			"role": schema.StringAttribute{
				Computed:    true,
				Description: "User workspace role (owner, admin, member).",
			},
			"locale": schema.StringAttribute{
				Computed:    true,
				Description: "User locale.",
			},
			"timezone": schema.StringAttribute{
				Computed:    true,
				Description: "User timezone.",
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*DocmostClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *DocmostClient, got %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.FindWorkspaceUserByEmail(config.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"User not found",
			fmt.Sprintf("Could not find user with email %q: %s", config.Email.ValueString(), err.Error()),
		)
		return
	}

	config.ID = types.StringValue(user.ID)
	config.Email = types.StringValue(user.Email)
	config.Name = types.StringValue(user.Name)
	config.Role = types.StringValue(user.Role)
	config.Locale = types.StringValue(user.Locale)
	config.Timezone = types.StringValue(user.Timezone)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
