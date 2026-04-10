package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// APIError represents a non-2xx response from the Docmost API.
type APIError struct {
	StatusCode int
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API %s returned status %d: %s", e.Path, e.StatusCode, e.Body)
}

// IsNotFoundError returns true if the error is a 404 API response.
func IsNotFoundError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// DocmostClient handles HTTP communication with the Docmost API.
type DocmostClient struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

// NewDocmostClient authenticates with Docmost and returns a configured client.
func NewDocmostClient(baseURL, email, password, token string) (*DocmostClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	client := &DocmostClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}

	if token != "" {
		client.AuthToken = token
		return client, nil
	}

	loginBody := map[string]string{
		"email":    email,
		"password": password,
	}

	bodyBytes, err := json.Marshal(loginBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/auth/login", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "authToken" {
			client.AuthToken = cookie.Value
			break
		}
	}

	if client.AuthToken == "" {
		return nil, fmt.Errorf("login succeeded but no authToken cookie was returned")
	}

	return client, nil
}

// DoRequest sends a POST request to the given API path with the provided body.
func (c *DocmostClient) DoRequest(path string, body interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "authToken", Value: c.AuthToken})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Path: path, Body: string(respBody)}
	}

	return respBody, nil
}

// Space represents a Docmost space.
type Space struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Slug                 string `json:"slug"`
	Description          string `json:"description"`
	DisablePublicSharing bool   `json:"disablePublicSharing"`
	AllowViewerComments  bool   `json:"allowViewerComments"`
	CreatorID            string `json:"creatorId"`
	WorkspaceID          string `json:"workspaceId"`
}

// CreateSpace creates a new space.
func (c *DocmostClient) CreateSpace(name, slug, description string) (*Space, error) {
	body := map[string]string{
		"name": name,
		"slug": slug,
	}
	if description != "" {
		body["description"] = description
	}

	respBody, err := c.DoRequest("/api/spaces/create", body)
	if err != nil {
		return nil, err
	}

	var space Space
	if err := json.Unmarshal(respBody, &space); err != nil {
		return nil, fmt.Errorf("failed to parse create space response: %w", err)
	}
	return &space, nil
}

// GetSpace retrieves space info by ID.
func (c *DocmostClient) GetSpace(spaceID string) (*Space, error) {
	body := map[string]string{"spaceId": spaceID}

	respBody, err := c.DoRequest("/api/spaces/info", body)
	if err != nil {
		return nil, err
	}

	var space Space
	if err := json.Unmarshal(respBody, &space); err != nil {
		return nil, fmt.Errorf("failed to parse space info response: %w", err)
	}
	return &space, nil
}

// UpdateSpace updates an existing space.
func (c *DocmostClient) UpdateSpace(spaceID string, updates map[string]interface{}) (*Space, error) {
	updates["spaceId"] = spaceID

	respBody, err := c.DoRequest("/api/spaces/update", updates)
	if err != nil {
		return nil, err
	}

	var space Space
	if err := json.Unmarshal(respBody, &space); err != nil {
		return nil, fmt.Errorf("failed to parse update space response: %w", err)
	}
	return &space, nil
}

// DeleteSpace deletes a space by ID.
func (c *DocmostClient) DeleteSpace(spaceID string) error {
	body := map[string]string{"spaceId": spaceID}
	_, err := c.DoRequest("/api/spaces/delete", body)
	return err
}

// SpaceMember represents a member entry in a space.
type SpaceMember struct {
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	GroupID string `json:"groupId"`
	Role    string `json:"role"`
}

// SpaceMembersResponse is the paginated response from the members endpoint.
type SpaceMembersResponse struct {
	Items []json.RawMessage `json:"items"`
}

// AddSpaceMember adds a user or group to a space with a role.
func (c *DocmostClient) AddSpaceMember(spaceID string, userIDs []string, groupIDs []string, role string) error {
	body := map[string]interface{}{
		"spaceId":  spaceID,
		"role":     role,
		"userIds":  userIDs,
		"groupIds": groupIDs,
	}
	_, err := c.DoRequest("/api/spaces/members/add", body)
	return err
}

// RemoveSpaceMember removes a user or group from a space.
func (c *DocmostClient) RemoveSpaceMember(spaceID, userID, groupID string) error {
	body := map[string]string{"spaceId": spaceID}
	if userID != "" {
		body["userId"] = userID
	}
	if groupID != "" {
		body["groupId"] = groupID
	}
	_, err := c.DoRequest("/api/spaces/members/remove", body)
	return err
}

// UpdateSpaceMemberRole changes the role of a user or group in a space.
func (c *DocmostClient) UpdateSpaceMemberRole(spaceID, userID, groupID, role string) error {
	body := map[string]string{
		"spaceId": spaceID,
		"role":    role,
	}
	if userID != "" {
		body["userId"] = userID
	}
	if groupID != "" {
		body["groupId"] = groupID
	}
	_, err := c.DoRequest("/api/spaces/members/change-role", body)
	return err
}

// GetSpaceMembers lists members of a space (paginated).
func (c *DocmostClient) GetSpaceMembers(spaceID string, page, limit int) ([]byte, error) {
	body := map[string]interface{}{
		"spaceId": spaceID,
		"page":    page,
		"limit":   limit,
	}
	return c.DoRequest("/api/spaces/members", body)
}

// Group represents a Docmost group.
type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"isDefault"`
	WorkspaceID string `json:"workspaceId"`
}

// CreateGroup creates a new group.
func (c *DocmostClient) CreateGroup(name, description string) (*Group, error) {
	body := map[string]string{"name": name}
	if description != "" {
		body["description"] = description
	}

	respBody, err := c.DoRequest("/api/groups/create", body)
	if err != nil {
		return nil, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return nil, fmt.Errorf("failed to parse create group response: %w", err)
	}
	return &group, nil
}

// GetGroup retrieves group info by ID.
func (c *DocmostClient) GetGroup(groupID string) (*Group, error) {
	body := map[string]string{"groupId": groupID}

	respBody, err := c.DoRequest("/api/groups/info", body)
	if err != nil {
		return nil, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return nil, fmt.Errorf("failed to parse group info response: %w", err)
	}
	return &group, nil
}

// UpdateGroup updates an existing group.
func (c *DocmostClient) UpdateGroup(groupID, name, description string) (*Group, error) {
	body := map[string]string{
		"groupId":     groupID,
		"name":        name,
		"description": description,
	}

	respBody, err := c.DoRequest("/api/groups/update", body)
	if err != nil {
		return nil, err
	}

	var group Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return nil, fmt.Errorf("failed to parse update group response: %w", err)
	}
	return &group, nil
}

// DeleteGroup deletes a group by ID.
func (c *DocmostClient) DeleteGroup(groupID string) error {
	body := map[string]string{"groupId": groupID}
	_, err := c.DoRequest("/api/groups/delete", body)
	return err
}

// AddGroupMembers adds users to a group.
func (c *DocmostClient) AddGroupMembers(groupID string, userIDs []string) error {
	body := map[string]interface{}{
		"groupId": groupID,
		"userIds": userIDs,
	}
	_, err := c.DoRequest("/api/groups/members/add", body)
	return err
}

// RemoveGroupMember removes a user from a group.
func (c *DocmostClient) RemoveGroupMember(groupID, userID string) error {
	body := map[string]string{
		"groupId": groupID,
		"userId":  userID,
	}
	_, err := c.DoRequest("/api/groups/members/remove", body)
	return err
}

// GetGroupMembers lists members of a group (paginated).
func (c *DocmostClient) GetGroupMembers(groupID string, page, limit int) ([]byte, error) {
	body := map[string]interface{}{
		"groupId": groupID,
		"page":    page,
		"limit":   limit,
	}
	return c.DoRequest("/api/groups/members", body)
}

// WorkspaceUser represents a user in a Docmost workspace.
type WorkspaceUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatarUrl"`
	Locale    string `json:"locale"`
	Timezone  string `json:"timezone"`
}

// CursorPaginatedResponse is the response shape for cursor-paginated endpoints.
type CursorPaginatedResponse struct {
	Items      []json.RawMessage `json:"items"`
	HasMore    bool              `json:"hasMore"`
	NextCursor string            `json:"nextCursor"`
}

// FindWorkspaceUserByEmail searches workspace members by email and returns the first exact match.
// It paginates through all pages using cursor-based pagination.
func (c *DocmostClient) FindWorkspaceUserByEmail(email string) (*WorkspaceUser, error) {
	var cursor string
	for {
		body := map[string]interface{}{
			"query": email,
			"limit": 100,
		}
		if cursor != "" {
			body["cursor"] = cursor
		}

		respBody, err := c.DoRequest("/api/workspace/members", body)
		if err != nil {
			return nil, err
		}

		var paginated CursorPaginatedResponse
		if err := json.Unmarshal(respBody, &paginated); err != nil {
			return nil, fmt.Errorf("failed to parse workspace members response: %w", err)
		}

		for _, raw := range paginated.Items {
			var user WorkspaceUser
			if err := json.Unmarshal(raw, &user); err != nil {
				continue
			}
			if strings.EqualFold(user.Email, email) {
				return &user, nil
			}
		}

		if !paginated.HasMore || paginated.NextCursor == "" {
			break
		}
		cursor = paginated.NextCursor
	}

	return nil, fmt.Errorf("user with email %q not found in workspace", email)
}

// IsSpaceMember checks whether a user or group is still a member of a space
// by paginating through the space members list.
func (c *DocmostClient) IsSpaceMember(spaceID, userID, groupID string) (bool, error) {
	for page := 1; ; page++ {
		respBody, err := c.GetSpaceMembers(spaceID, page, 100)
		if err != nil {
			return false, err
		}

		var resp struct {
			Items []SpaceMember `json:"items"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return false, fmt.Errorf("failed to parse space members response: %w", err)
		}

		for _, m := range resp.Items {
			if userID != "" && m.UserID == userID {
				return true, nil
			}
			if groupID != "" && m.GroupID == groupID {
				return true, nil
			}
		}

		if len(resp.Items) < 100 {
			break
		}
	}
	return false, nil
}

// IsGroupMember checks whether a user is still a member of a group
// by paginating through the group members list.
func (c *DocmostClient) IsGroupMember(groupID, userID string) (bool, error) {
	for page := 1; ; page++ {
		respBody, err := c.GetGroupMembers(groupID, page, 100)
		if err != nil {
			return false, err
		}

		var resp struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return false, fmt.Errorf("failed to parse group members response: %w", err)
		}

		for _, m := range resp.Items {
			if m.ID == userID {
				return true, nil
			}
		}

		if len(resp.Items) < 100 {
			break
		}
	}
	return false, nil
}
