# Terraform Provider for Docmost

A Terraform provider for managing [Docmost](https://docmost.com) workspaces as code — spaces, groups, memberships, and user lookups.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for building from source)
- A running Docmost instance with admin credentials

## Installation

### Terraform Registry

```hcl
terraform {
  required_providers {
    docmost = {
      source = "edwinguo/docmost"
    }
  }
}
```

### Build from Source

```bash
make install
```

This compiles the provider and places it in `~/.terraform.d/plugins/` for local use.

## Authentication

The provider supports two authentication methods:

| Method | Config Attributes | Environment Variables |
|--------|------------------|-----------------------|
| Email/Password | `email`, `password` | `DOCMOST_EMAIL`, `DOCMOST_PASSWORD` |
| Token | `token` | `DOCMOST_TOKEN` |

The `host` attribute (or `DOCMOST_HOST`) is always required.

```hcl
provider "docmost" {
  host     = "https://docs.example.com"
  email    = "admin@example.com"
  password = "your-password"
}

# Or with a token:
provider "docmost" {
  host  = "https://docs.example.com"
  token = "your-auth-token"
}
```

## Resources and Data Sources

### Resources

| Resource | Description |
|----------|-------------|
| `docmost_space` | Manages a Docmost space (name, slug, description, sharing settings) |
| `docmost_group` | Manages a Docmost group |
| `docmost_space_member` | Assigns a user or group to a space with a role (`admin`, `writer`, `reader`) |
| `docmost_group_member` | Adds a user to a group |

### Data Sources

| Data Source | Description |
|-------------|-------------|
| `docmost_user` | Looks up an existing workspace user by email |

## Usage Example

```hcl
# Look up SSO-provisioned users
data "docmost_user" "alice" {
  email = "alice@example.com"
}

# Create a space
resource "docmost_space" "engineering" {
  name        = "Engineering"
  slug        = "engineering"
  description = "Engineering team documentation"
}

# Create a group
resource "docmost_group" "backend_team" {
  name        = "Backend Team"
  description = "Backend engineers"
}

# Add a user to a group
resource "docmost_group_member" "alice_backend" {
  group_id = docmost_group.backend_team.id
  user_id  = data.docmost_user.alice.id
}

# Grant the group write access to the space
resource "docmost_space_member" "backend_team_engineering" {
  space_id = docmost_space.engineering.id
  group_id = docmost_group.backend_team.id
  role     = "writer"
}

# Grant an individual user admin access
resource "docmost_space_member" "alice_engineering" {
  space_id = docmost_space.engineering.id
  user_id  = data.docmost_user.alice.id
  role     = "admin"
}
```

See [`examples/main.tf`](examples/main.tf) for a more complete example.

## Resource Reference

### docmost_space

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Space name (2-100 characters) |
| `slug` | string | yes | URL slug (2-100 characters, alphanumeric) |
| `description` | string | no | Space description |
| `disable_public_sharing` | bool | no | Disable public page sharing |
| `allow_viewer_comments` | bool | no | Allow viewers to comment |
| `id` | string | computed | Space UUID |

Supports `terraform import` by space ID.

### docmost_group

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Group name |
| `description` | string | no | Group description |
| `id` | string | computed | Group UUID |
| `is_default` | bool | computed | Whether this is the default workspace group |

Supports `terraform import` by group ID.

### docmost_space_member

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `space_id` | string | yes | Space ID (forces replacement on change) |
| `user_id` | string | one of | User ID (mutually exclusive with `group_id`) |
| `group_id` | string | one of | Group ID (mutually exclusive with `user_id`) |
| `role` | string | yes | Role: `admin`, `writer`, or `reader` |
| `id` | string | computed | Composite ID (`space_id:user:user_id` or `space_id:group:group_id`) |

### docmost_group_member

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `group_id` | string | yes | Group ID (forces replacement on change) |
| `user_id` | string | yes | User ID (forces replacement on change) |
| `id` | string | computed | Composite ID (`group_id:user_id`) |

### docmost_user (data source)

| Attribute | Type | Description |
|-----------|------|-------------|
| `email` | string | Email to look up (required) |
| `id` | string | User UUID |
| `name` | string | Display name |
| `role` | string | Workspace role (`owner`, `admin`, `member`) |
| `locale` | string | User locale |
| `timezone` | string | User timezone |

## Development

```bash
make build    # compile the provider binary
make test     # run tests
make fmt      # format Go source
make vet      # run go vet
make install  # build and install to local Terraform plugins dir
```

## License

See [LICENSE](LICENSE) for details.
