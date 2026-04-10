package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SpaceDataSource{}

type SpaceDataSource struct {
	client *DocmostClient
}

type SpaceDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Slug                 types.String `tfsdk:"slug"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	DisablePublicSharing types.Bool   `tfsdk:"disable_public_sharing"`
	AllowViewerComments  types.Bool   `tfsdk:"allow_viewer_comments"`
}

func NewSpaceDataSource() datasource.DataSource {
	return &SpaceDataSource{}
}

func (d *SpaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (d *SpaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Docmost space by slug.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "Space slug to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Space UUID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Space name.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Space description.",
			},
			"disable_public_sharing": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether public page sharing is disabled.",
			},
			"allow_viewer_comments": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether viewers can add comments.",
			},
		},
	}
}

func (d *SpaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SpaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SpaceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space, err := d.client.GetSpaceBySlug(config.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Space not found",
			fmt.Sprintf("Could not find space with slug %q: %s", config.Slug.ValueString(), err.Error()),
		)
		return
	}

	config.ID = types.StringValue(space.ID)
	config.Name = types.StringValue(space.Name)
	config.Slug = types.StringValue(space.Slug)
	config.Description = types.StringValue(space.Description)
	config.DisablePublicSharing = types.BoolValue(space.DisablePublicSharing)
	config.AllowViewerComments = types.BoolValue(space.AllowViewerComments)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
