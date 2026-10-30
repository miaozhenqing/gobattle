package model

type ItemCategory int

const (
	ItemCategoryResource ItemCategory = 1 // 资源：金币等货币
	ItemCategoryWeapon   ItemCategory = 2 // 武器
)

// 道具类型的统一定义
type ItemCategoryDef struct {
	Category  ItemCategory
	Name      string
	Stackable bool
}

var itemCategoryDefByCategory = map[ItemCategory]*ItemCategoryDef{
	ItemCategoryResource: {Category: ItemCategoryResource, Name: "resource", Stackable: true},
	ItemCategoryWeapon:   {Category: ItemCategoryWeapon, Name: "weapon", Stackable: false},
}

func GetItemCategoryDef(category ItemCategory) (*ItemCategoryDef, bool) {
	def, ok := itemCategoryDefByCategory[category]
	return def, ok
}

func ItemCategoryDefs() []*ItemCategoryDef {
	defs := make([]*ItemCategoryDef, 0, len(itemCategoryDefByCategory))
	for _, def := range itemCategoryDefByCategory {
		defs = append(defs, def)
	}
	return defs
}
