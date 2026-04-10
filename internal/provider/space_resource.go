package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SpaceResource{}
var _ resource.ResourceWithImportState = &SpaceResource{}

type SpaceResource struct {
	client *DocmostClient
}

type SpaceResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Slug                 types.String `tfsdk:"slug"`
	Description          types.String `tfsdk:"description"`
	DisablePublicSharing types.Bool   `tfsdk:"disable_public_sharing"`
	AllowViewerComments  types.Bool   `tfsdk:"allow_viewer_comments"`
}

func NewSpaceResource() resource.Resource {
	return &SpaceResource{}
}

func (r *SpaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (r *SpaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Docmost space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Space UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Space name (2-100 characters).",
			},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "Space slug, alphanumeric (2-100 characters).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Space description.",
			},
			"disable_public_sharing": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Disable public page sharing for this space.",
			},
			"allow_viewer_comments": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow viewers to add comments.",
			},
		},
	}
}

func (r *SpaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*DocmostClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *DocmostClient, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *SpaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space, err := r.client.CreateSpace(
		plan.Name.ValueString(),
		plan.Slug.ValueString(),
		plan.Description.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create space", err.Error())
		return
	}

	// Re-read after create to get the authoritative server state
	freshSpace, err := r.client.GetSpace(space.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read space after creation", err.Error())
		return
	}

	// Update boolean settings if specified — these aren't supported by the create API
	needsUpdate := false
	updates := map[string]interface{}{}
	if !plan.DisablePublicSharing.IsNull() && !plan.DisablePublicSharing.IsUnknown() &&
		plan.DisablePublicSharing.ValueBool() != freshSpace.DisablePublicSharing {
		updates["disablePublicSharing"] = plan.DisablePublicSharing.ValueBool()
		needsUpdate = true
	}
	if !plan.AllowViewerComments.IsNull() && !plan.AllowViewerComments.IsUnknown() &&
		plan.AllowViewerComments.ValueBool() != freshSpace.AllowViewerComments {
		updates["allowViewerComments"] = plan.AllowViewerComments.ValueBool()
		needsUpdate = true
	}

	if needsUpdate {
		_, err := r.client.UpdateSpace(freshSpace.ID, updates)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update space settings after creation", err.Error())
			return
		}
		// Re-read again to get final state
		freshSpace, err = r.client.GetSpace(freshSpace.ID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read space after update", err.Error())
			return
		}
	}

	r.mapSpaceToState(freshSpace, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space, err := r.client.GetSpace(state.ID.ValueString())
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read space", err.Error())
		return
	}

	r.mapSpaceToState(space, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SpaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SpaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updates := map[string]interface{}{}
	if plan.Name.ValueString() != state.Name.ValueString() {
		updates["name"] = plan.Name.ValueString()
	}
	if plan.Slug.ValueString() != state.Slug.ValueString() {
		updates["slug"] = plan.Slug.ValueString()
	}
	if plan.Description.ValueString() != state.Description.ValueString() {
		updates["description"] = plan.Description.ValueString()
	}
	if !plan.DisablePublicSharing.IsNull() && !plan.DisablePublicSharing.IsUnknown() {
		updates["disablePublicSharing"] = plan.DisablePublicSharing.ValueBool()
	}
	if !plan.AllowViewerComments.IsNull() && !plan.AllowViewerComments.IsUnknown() {
		updates["allowViewerComments"] = plan.AllowViewerComments.ValueBool()
	}

	space, err := r.client.UpdateSpace(state.ID.ValueString(), updates)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update space", err.Error())
		return
	}

	r.mapSpaceToState(space, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SpaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSpace(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete space", err.Error())
		return
	}
}

func (r *SpaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	space, err := r.client.GetSpace(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import space", err.Error())
		return
	}

	var state SpaceResourceModel
	r.mapSpaceToState(space, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SpaceResource) mapSpaceToState(space *Space, state *SpaceResourceModel) {
	state.ID = types.StringValue(space.ID)
	state.Name = types.StringValue(space.Name)
	state.Slug = types.StringValue(space.Slug)
	state.Description = types.StringValue(space.Description)
	state.DisablePublicSharing = types.BoolValue(space.DisablePublicSharing)
	state.AllowViewerComments = types.BoolValue(space.AllowViewerComments)
}
