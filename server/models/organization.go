package models

import (
	"github.com/meshery/schemas/models/v1beta2/organization"
)

// TODO: Move to schemas
type OrganizationsPage struct {
	Organizations []*organization.Organization `json:"organizations"`
	TotalCount    int                          `json:"totalCount"`
	Page          uint64                       `json:"page"`
	PageSize      uint64                       `json:"pageSize"`
}
