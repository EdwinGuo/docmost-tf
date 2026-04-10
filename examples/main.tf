terraform {
  required_providers {
    docmost = {
      source = "registry.terraform.io/gokite/docmost"
    }
  }
}

provider "docmost" {
  host     = "https://docs.example.com"
  email    = "admin@example.com"
  password = "your-password"
  # Or use token-based auth:
  # token = "your-auth-token"
}

# ---------------------------------------------------------------------------
# Look up SSO users by email (users already provisioned via GSuite SSO)
# ---------------------------------------------------------------------------

data "docmost_user" "alice" {
  email = "alice@example.com"
}

data "docmost_user" "bob" {
  email = "bob@example.com"
}

data "docmost_user" "charlie" {
  email = "charlie@example.com"
}

# ---------------------------------------------------------------------------
# Spaces
# ---------------------------------------------------------------------------

resource "docmost_space" "engineering" {
  name        = "Engineering"
  slug        = "engineering"
  description = "Engineering team documentation"
}

resource "docmost_space" "product" {
  name        = "Product"
  slug        = "product"
  description = "Product specs and roadmaps"
}

# ---------------------------------------------------------------------------
# Groups
# ---------------------------------------------------------------------------

resource "docmost_group" "backend_team" {
  name        = "Backend Team"
  description = "Backend engineers"
}

resource "docmost_group" "frontend_team" {
  name        = "Frontend Team"
  description = "Frontend engineers"
}

# ---------------------------------------------------------------------------
# Assign SSO users to groups
# ---------------------------------------------------------------------------

resource "docmost_group_member" "alice_backend" {
  group_id = docmost_group.backend_team.id
  user_id  = data.docmost_user.alice.id
}

resource "docmost_group_member" "bob_frontend" {
  group_id = docmost_group.frontend_team.id
  user_id  = data.docmost_user.bob.id
}

# ---------------------------------------------------------------------------
# Assign individual users to spaces with specific roles
# ---------------------------------------------------------------------------

# Alice is an admin of the engineering space
resource "docmost_space_member" "alice_engineering" {
  space_id = docmost_space.engineering.id
  user_id  = data.docmost_user.alice.id
  role     = "admin"
}

# Bob gets write access to product space
resource "docmost_space_member" "bob_product" {
  space_id = docmost_space.product.id
  user_id  = data.docmost_user.bob.id
  role     = "writer"
}

# Charlie gets read-only access to engineering space
resource "docmost_space_member" "charlie_engineering" {
  space_id = docmost_space.engineering.id
  user_id  = data.docmost_user.charlie.id
  role     = "reader"
}

# ---------------------------------------------------------------------------
# Assign groups to spaces (all group members inherit the role)
# ---------------------------------------------------------------------------

# Backend team gets write access to engineering space
resource "docmost_space_member" "backend_team_engineering" {
  space_id = docmost_space.engineering.id
  group_id = docmost_group.backend_team.id
  role     = "writer"
}

# Frontend team gets read access to product space
resource "docmost_space_member" "frontend_team_product" {
  space_id = docmost_space.product.id
  group_id = docmost_group.frontend_team.id
  role     = "reader"
}
