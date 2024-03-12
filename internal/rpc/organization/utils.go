package organization

import (
	"container/list"
	"context"
	"encoding/hex"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/google/uuid"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/OpenIMSDK/tools/utils"
)

func genDepartmentID() string {
	//r := utils.Md5(strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.FormatUint(rand.Uint64(), 10))
	//bi := big.NewInt(0)
	//bi.SetString(r[0:8], 16)
	//return bi.String()
	id := uuid.New()
	return hex.EncodeToString(id[:])
}

func GenUserID() string {
	r := utils.Md5(strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.FormatUint(rand.Uint64(), 10))
	bi := big.NewInt(0)
	bi.SetString(r[0:8], 16)
	return bi.String()
}

func GenDepartmentID() string {
	r := utils.Md5(strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.FormatUint(rand.Uint64(), 10))
	bi := big.NewInt(0)
	bi.SetString(r[0:8], 16)
	return bi.String()
}

func (o *organizationSvr) GetDepartmentMemberNum(ctx context.Context, departmentID string) (int64, error) {
	var departmentIDs []string
	if departmentID != "" {
		departmentIDSet := make(map[string]struct{})
		queue := list.New()
		queue.PushBack(departmentID)
		for queue.Len() > 0 {
			id := queue.Remove(queue.Front()).(string)
			if _, ok := departmentIDSet[id]; ok {
				return 0, errs.ErrInternalServer.Wrap("ring department " + id)
			}
			departmentIDSet[id] = struct{}{}
			departments, err := o.Database.GetParentDepartment(ctx, id)
			if err != nil {
				return 0, err
			}
			for _, department := range departments {
				queue.PushBack(department.DepartmentID)
			}
		}
		departmentIDs = utils.Keys(departmentIDSet)
	}
	return o.Database.GetMemberNum(ctx, departmentIDs)
}
