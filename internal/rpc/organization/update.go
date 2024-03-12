package organization

import (
	"github.com/OpenIMSDK/chat/pkg/proto/organization"
	"github.com/OpenIMSDK/tools/errs"
	"time"
)

func UpdateDepartment(req *organization.UpdateDepartmentReq) (map[string]any, error) {
	if req.DepartmentID == "" {
		return nil, errs.ErrArgs.Wrap("departmentID is empty")
	}
	update := make(map[string]any)
	if req.Name != nil {
		if req.Name.Value == "" {
			return nil, errs.ErrArgs.Wrap("name is empty")
		}
		update["name"] = req.Name.Value
	}
	if req.FaceURL != nil {
		update["face_url"] = req.FaceURL.Value
	}
	if req.ParentDepartmentID != nil {
		update["parent_department_id"] = req.ParentDepartmentID.Value
	}
	if req.Order != nil {
		update["order"] = req.Order.Value
	}
	if len(update) == 0 {
		return nil, errs.ErrArgs.Wrap("no update to update")
	}
	return update, nil
}

func UpdateDepartmentMember(req *organization.UpdateUserInDepartmentReq) (map[string]any, error) {
	if req.DepartmentID == "" {
		return nil, errs.ErrArgs.Wrap("departmentID is empty")
	}
	if req.UserID == "" {
		return nil, errs.ErrArgs.Wrap("userID is empty")
	}
	update := make(map[string]any)
	if req.Position != nil {
		update["position"] = req.Position.Value
	}
	if req.Station != nil {
		update["station"] = req.Station.Value
	}
	if req.Order != nil {
		update["order"] = req.Order.Value
	}
	if req.EntryTime != nil {
		update["entry_time"] = time.UnixMilli(req.EntryTime.Value)
	}
	if req.TerminationTime != nil {
		if req.TerminationTime.Value == 0 {
			update["termination_time"] = nil
		} else {
			update["termination_time"] = time.UnixMilli(req.TerminationTime.Value)
		}
	}
	if len(update) == 0 {
		return nil, errs.ErrArgs.Wrap("no update to update")
	}
	return update, nil
}

func UpdateOrganization(req *organization.SetOrganizationReq) (map[string]any, error) {
	update := make(map[string]any)
	if req.LogoURL != nil {
		update["logo_url"] = req.LogoURL.Value
	}
	if req.Name != nil {
		if req.Name.Value == "" {
			return nil, errs.ErrArgs.Wrap("name is empty")
		}
		update["name"] = req.Name.Value
	}
	if req.Homepage != nil {
		update["homepage"] = req.Homepage.Value
	}
	if req.Introduction != nil {
		update["introduction"] = req.Introduction.Value
	}
	if len(update) == 0 {
		return nil, errs.ErrArgs.Wrap("no update to update")
	}
	return update, nil
}
