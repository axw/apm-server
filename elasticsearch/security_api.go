// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearch

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CreateAPIKey requires manage_security cluster privilege
func CreateAPIKey(client Client, apikeyReq CreateApiKeyRequest) (CreateApiKeyResponse, error) {
	response := client.JSONRequest(http.MethodPut, "/_security/api_key", apikeyReq)

	var apikey CreateApiKeyResponse
	err := response.DecodeTo(&apikey)
	return apikey, err
}

// GetAPIKeys requires manage_security cluster privilege
func GetAPIKeys(client Client, apikeyReq GetApiKeyRequest) (GetApiKeyResponse, error) {
	u := url.URL{Path: "/_security/api_key"}
	params := url.Values{}
	params.Set("owner", strconv.FormatBool(apikeyReq.Owner))
	if apikeyReq.Id != nil {
		params.Set("id", *apikeyReq.Id)
	} else if apikeyReq.Name != nil {
		params.Set("name", *apikeyReq.Name)
	}
	u.RawQuery = params.Encode()

	response := client.JSONRequest(http.MethodGet, u.String(), nil)

	var apikey GetApiKeyResponse
	err := response.DecodeTo(&apikey)
	return apikey, err
}

// CreatePrivileges requires manage_security cluster privilege
func CreatePrivileges(client Client, privilegesReq CreatePrivilegesRequest) (CreatePrivilegesResponse, error) {
	response := client.JSONRequest(http.MethodPut, "/_security/privilege", privilegesReq)

	var privileges CreatePrivilegesResponse
	err := response.DecodeTo(&privileges)
	return privileges, err
}

// InvalidateAPIKey requires manage_security cluster privilege
func InvalidateAPIKey(client Client, apikeyReq InvalidateApiKeyRequest) (InvalidateApiKeyResponse, error) {
	response := client.JSONRequest(http.MethodDelete, "/_security/api_key", apikeyReq)

	var confirmation InvalidateApiKeyResponse
	err := response.DecodeTo(&confirmation)
	return confirmation, err
}

// DeletePrivileges requires manage_security cluster privilege
func DeletePrivileges(client Client, privilegesReq DeletePrivilegeRequest) (DeletePrivilegeResponse, error) {
	path := fmt.Sprintf("/_security/privilege/%v/%v", privilegesReq.Application, privilegesReq.Privilege)
	response := client.JSONRequest(http.MethodDelete, path, nil)

	var confirmation DeletePrivilegeResponse
	err := response.DecodeTo(&confirmation)
	return confirmation, err
}

func HasPrivileges(client Client, privileges HasPrivilegesRequest, credentials string) (HasPrivilegesResponse, error) {
	h := fmt.Sprintf("Authorization: ApiKey %s", credentials)
	response := client.JSONRequest(http.MethodGet, "/_security/user/_has_privileges", privileges, h)

	var info HasPrivilegesResponse
	err := response.DecodeTo(&info)
	return info, err
}

type CreateApiKeyRequest struct {
	Name            string         `json:"name"`
	Expiration      *string        `json:"expiration,omitempty"`
	RoleDescriptors RoleDescriptor `json:"role_descriptors"`
}

type CreateApiKeyResponse struct {
	ApiKey
	Key string `json:"api_key"`
}

type GetApiKeyRequest struct {
	ApiKeyQuery
	Owner bool `json:"owner"`
}

type GetApiKeyResponse struct {
	ApiKeys []ApiKeyResponse `json:"api_keys"`
}

type CreatePrivilegesRequest map[AppName]PrivilegeGroup

type CreatePrivilegesResponse map[AppName]PrivilegeResponse

type HasPrivilegesRequest struct {
	// can't reuse the `Applications` type because here the JSON attribute must be singular
	Applications []Application `json:"application"`
}
type HasPrivilegesResponse struct {
	Username    string                            `json:"username"`
	HasAll      bool                              `json:"has_all_requested"`
	Application map[AppName]PrivilegesPerResource `json:"application"`
}

type InvalidateApiKeyRequest struct {
	ApiKeyQuery
}

type InvalidateApiKeyResponse struct {
	Invalidated []string `json:"invalidated_api_keys"`
	ErrorCount  int      `json:"error_count"`
}

type DeletePrivilegeRequest struct {
	Application AppName       `json:"application"`
	Privilege   PrivilegeName `json:"privilege"`
}

type DeletePrivilegeResponse map[AppName](map[PrivilegeName]DeleteResponse)

type RoleDescriptor map[AppName]Applications

type Applications struct {
	Applications []Application `json:"applications"`
}

type Application struct {
	Name       AppName           `json:"application"`
	Privileges []PrivilegeAction `json:"privileges"`
	Resources  []Resource        `json:"resources"`
}

type ApiKeyResponse struct {
	ApiKey
	Creation    int64  `json:"creation"`
	Invalidated bool   `json:"invalidated"`
	Username    string `json:"username"`
}

type ApiKeyQuery struct {
	// normally the Elasticsearch API will require either Id or Name, but not both
	Id   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type ApiKey struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	ExpirationMs *int64 `json:"expiration,omitempty"`
	// This attribute does not come from Elasticsearch, but is filled in by APM Server
	Credentials *string `json:"credentials,omitempty"`
}

type PrivilegeResponse map[PrivilegeAction]PutResponse

type PrivilegeGroup map[PrivilegeName]Actions

type Permissions map[PrivilegeAction]bool

type PrivilegesPerResource map[Resource]Permissions

type Actions struct {
	Actions []PrivilegeAction `json:"actions"`
}

type PutResponse struct {
	Created bool `json:"created"`
}

type DeleteResponse struct {
	Found bool `json:"found"`
}

type AppName string

type Resource string

// in Elasticsearch a "privilege" represents both an "action" that a user might/might not have authorization to
// perform; and a tuple consisting of a name and an action
// for differentiation, we call the tuple NamedPrivilege
// in apm-server, each name is associated with one action, but that needs not to be the case (see PrivilegeGroup)
type NamedPrivilege struct {
	Name   PrivilegeName
	Action PrivilegeAction
}

type PrivilegeAction string

type PrivilegeName string

func NewPrivilege(name, action string) NamedPrivilege {
	return NamedPrivilege{
		Name:   PrivilegeName(name),
		Action: PrivilegeAction(action),
	}
}
