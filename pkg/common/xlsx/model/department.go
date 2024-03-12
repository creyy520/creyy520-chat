package model

type Department struct {
	ID      string `column:"ID"`
	Name    string `column:"名字"` // 开发/后端/Go
	FaceURL string `column:"头像"`
	Order   string `column:"排序"`
	Ignore  bool   `column:"忽略"`
}

func (Department) SheetName() string {
	return "部门"
}
