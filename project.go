package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ProjectsResponse struct {
	Pagination Pagination `json:"pagination"`
	Projects   []Project  `json:"data"`
}

type EmailPreferencesResponse struct {
	Pagination           Pagination       `json:"pagination"`
	UserEmailPreferences []UserPreference `json:"data"`
}

type UserSettingResponse struct {
	Pagination   Pagination    `json:"pagination"`
	UserSettings []UserSetting `json:"data"`
}

func (c *client) projects(page int) (*ProjectsResponse, error) {
	req, err := c.newRequest(http.MethodGet, "projects", nil)
	if err != nil {
		return nil, err
	}
	rsp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	var r ProjectsResponse
	if err := json.NewDecoder(rsp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("while decoding projects response: %w", err)
	}
	return &r, nil
}

func (c *client) ListProjects() ([]Project, error) {
	var projects []Project
	for page := 1; ; page++ {
		rsp, err := c.projects(page)
		if err != nil {
			return nil, err
		}
		projects = append(projects, rsp.Projects...)
		if rsp.Pagination.TotalPages <= page {
			return projects, nil
		}
	}
}

func (c *client) ProjectEmailPreferences(projectId int) (UserPreference, error) {
	return c.notificationPreferences(projectId, "projects", "user_email_preferences")
}

func (c *client) WorkspaceEmailPreferences(workspaceId int) (UserPreference, error) {
	return c.notificationPreferences(workspaceId, "workspaces", "user_email_preferences")
}

func (c *client) ProjectInAppPreferences(projectId int) (UserPreference, error) {
	return c.notificationPreferences(projectId, "projects", "user_in_app_preferences")
}

func (c *client) WorkspaceInAppPreferences(workspaceId int) (UserPreference, error) {
	return c.notificationPreferences(workspaceId, "workspaces", "user_in_app_preferences")
}

func (c *client) notificationPreferences(objectId int, object, prefType string) (UserPreference, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/%d/%s", object, objectId, prefType), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	var r EmailPreferencesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("while decoding project user email prefs response: %w", err)
	}
	prefs := r.UserEmailPreferences
	if len(prefs) > 1 {
		log.Printf("unexpected project user email preferences response: %#v", r)
	}
	return prefs[0], nil
}

func (c *client) ProjectTriageUserSettings(projectId int) (*UserSetting, error) {
	return c.userSettings(projectId, "projects", true)
}

func (c *client) ProjectUserSettings(projectId int) (*UserSetting, error) {
	return c.userSettings(projectId, "projects", false)
}

func (c *client) WorkspaceUserSettings(workspaceId int) (*UserSetting, error) {
	return c.userSettings(workspaceId, "workspaces", false)
}

func (c *client) userSettings(objectId int, object string, triage bool) (*UserSetting, error) {
	method := "user_settings"
	if triage {
		method = "triage_user_settings"
	}
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/%d/%s", object, objectId, method), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	var r UserSettingResponse
	if err := json.NewDecoder(rsp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("while decoding project triage user settings: %w", err)
	}
	settings := r.UserSettings
	if len(settings) > 1 {
		log.Printf("unexpected project triage user settings: %#v", r)
	}
	return &settings[0], nil
}

func (c *client) DisableProjectEmailNotifications(projectId, prefId int, body io.Reader) error {
	return c.disableNotifications(projectId, "projects", prefId, "user_email_preferences", body)
}

func (c *client) DisableProjectInAppNotifications(projectId, prefId int, body io.Reader) error {
	return c.disableNotifications(projectId, "projects", prefId, "user_in_app_preferences", body)
}

func (c *client) DisableWorkspaceEmailNotifications(workspaceId, prefId int, body io.Reader) error {
	return c.disableNotifications(workspaceId, "workspaces", prefId, "user_email_preferences", body)
}

func (c *client) DisableWorkspaceInAppNotifications(workspaceId, prefId int, body io.Reader) error {
	return c.disableNotifications(workspaceId, "workspaces", prefId, "user_in_app_preferences", body)
}

func (c *client) disableNotifications(objectId int, object string, prefId int, prefType string, body io.Reader) error {
	req, err := c.newRequest(http.MethodPut,
		fmt.Sprintf("%s/%d/%s/%d", object, objectId, prefType, prefId), body)
	if err != nil {
		return err
	}
	rsp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	var i interface{}
	if err := json.NewDecoder(rsp.Body).Decode(&i); err != nil {
		return err
	}
	if m, ok := i.(map[string]interface{}); ok {
		if msg, present := m["error"]; present {
			return fmt.Errorf("error disabling notifications for %s: %s", req.URL.String(), msg)
		}
	}

	return nil
}
