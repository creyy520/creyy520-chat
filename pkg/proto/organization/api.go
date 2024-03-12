package organization

import "github.com/OpenIMSDK/tools/utils"

func (x *GetSubDepartmentResp) ApiFormat() {
	utils.InitSlice(&x.Departments)
	utils.InitSlice(&x.Members)
	utils.InitSlice(&x.Parents)
}

func (x *GetOrganizationDepartmentResp) ApiFormat() {
	utils.InitSlice(&x.Departments)
}

func (x *GetDepartmentResp) ApiFormat() {
	utils.InitSlice(&x.Departments)
}

func (x *GetUserInDepartmentResp) ApiFormat() {
	utils.InitSlice(&x.Users)
	for i, user := range x.Users {
		if user.Members == nil {
			utils.InitSlice(&x.Users[i].Members)
		}
	}
}
