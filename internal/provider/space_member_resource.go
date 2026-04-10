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

var _ resource.Resource = &SpaceMemberResource{}

type SpaceMemberResource struct {
	client *DocmostClient
}

type SpaceMemberResourceModel struct {
	ID      types.String `tfsdk:"id"`
	SpaceID types.String `tfsdk:"space_id"`
	UserID  types.String `tfsdk:"user_id"`
	GroupID types.String `tfsdk:"group_id"`
	Role    types.String `tfsdk:"role"`
}

func NewSpaceMemberResource() resource.Resource {
	return &SpaceMemberResource{}
}

func (r *SpaceMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_member"
}

func (r *SpaceMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages membership of a user or group in a Docmost space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier (space_id:user_id or space_id:group_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"space_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the space.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the user to add. Exactly one of user_id or group_id must be set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the group to add. Exactly one of user_id or group_id must be set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Role in the space: admin, writer, or reader.",
			},
		},
	}
}

func (r *SpaceMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SpaceMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := plan.UserID.ValueString()
	groupID := plan.GroupID.ValueString()

	if userID == "" && groupID == "" {
		resp.Diagnostics.AddError("Validation error", "Exactly one of user_id or group_id must be set.")
		return
	}
	if userID != "" && groupID != "" {
		resp.Diagnostics.AddError("Validation error", "Only one of user_id or group_id can be set, not both.")
		return
	}

	var userIDs, groupIDs []string
	if userID != "" {
		userIDs = []string{userID}
		groupIDs = []string{}
	} else {
		userIDs = []string{}
		groupIDs = []string{groupID}
	}

	err := r.client.AddSpaceMember(plan.SpaceID.ValueString(), userIDs, groupIDs, plan.Role.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to add space member", err.Error())
		return
	}

	plan.ID = types.StringValue(r.compositeID(plan.SpaceID.ValueString(), userID, groupID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpaceMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.IsSpaceMember(
		state.SpaceID.ValueString(),
		state.UserID.ValueString(),
		state.GroupID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check space membership", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SpaceMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateSpaceMemberRole(
		plan.SpaceID.ValueString(),
		plan.UserID.ValueString(),
		plan.GroupID.ValueString(),
		plan.Role.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update space member role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpaceMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveSpaceMember(
		state.SpaceID.ValueString(),
		state.UserID.ValueString(),
		state.GroupID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove space member", err.Error())
		return
	}
}

func (r *SpaceMemberResource) compositeID(spaceID, userID, groupID string) string {
	if userID != "" {
		return spaceID + ":user:" + userID
	}
	return spaceID + ":group:" + groupID
}
