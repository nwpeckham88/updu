package api

import (
	"strings"

	"github.com/updu/updu/internal/models"
)

// canAccessGroups returns true if the user is authorized to view a resource
// associated with the given groups (e.g. from LLDAP).
// Rules:
// 1. Unauthenticated users cannot access any groups.
// 2. Admins have unrestricted access to all resources.
// 3. Resources with no groups assigned (empty) are accessible to all authenticated users.
// 4. Resources with assigned groups require the user to belong to at least one matching group.
func canAccessGroups(user *models.User, groups []string) bool {
	if user == nil {
		return false
	}
	if user.Role == models.RoleAdmin {
		return true
	}
	if len(groups) == 0 {
		return true
	}
	for _, mg := range groups {
		trimmedMg := strings.TrimSpace(mg)
		if strings.EqualFold(trimmedMg, "all") || strings.EqualFold(trimmedMg, "public") {
			return true
		}
		for _, ug := range user.Groups {
			if strings.EqualFold(trimmedMg, strings.TrimSpace(ug)) {
				return true
			}
		}
	}
	return false
}

// canAccessMonitor checks if a user is permitted to view a monitor.
func canAccessMonitor(user *models.User, monitor *models.Monitor) bool {
	if monitor == nil {
		return false
	}
	return canAccessGroups(user, monitor.Groups)
}

// canAccessService checks if a user is permitted to view a service.
func canAccessService(user *models.User, service *models.Service) bool {
	if service == nil {
		return false
	}
	return canAccessGroups(user, service.Groups)
}
