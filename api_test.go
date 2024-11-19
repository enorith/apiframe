package apiframe_test

import (
	"fmt"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func TestLoadRelation(t *testing.T) {
	// apiframe.WithLoadRelations([]apiframe.QueryField{
	// 	{
	// 		Name: "user.name",
	// 	},
	// 	{
	// 		Name: "user.id",
	// 	},
	// 	{
	// 		Name: "profile_user.id",
	// 	},
	// 	{
	// 		Name: "profile_user.age",
	// 	},
	// })

	resl := groupSlice([]string{"user.name", "user.id", "profile_user.id", "profile_user.age", "user.group.id", "user.group.name"})

	printJson(resl)
}
func printJson(v any) {
	data, _ := jsoniter.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func groupSlice(slice []string) map[string][]string {
	result := make(map[string][]string)
	for _, str := range slice {
		// 从最顶层开始递归分组
		groupRecursive(str, result)
	}
	return result
}

// groupRecursive 递归处理每个字符串，根据层级分组
func groupRecursive(str string, result map[string][]string) {
	// 分割字符串为层级
	parts := strings.Split(str, ".")
	// 用来构建当前的前缀
	var prefix string

	// 遍历所有层级
	for i := 0; i < len(parts); i++ {
		// 构建当前层级的前缀
		if i > 0 {
			prefix += "."
		}
		prefix += parts[i]

		// 如果是最后一层（叶子节点），则直接加入该字段，不再递归
		if i == len(parts)-1 {
			// 把当前字段的值（即parts[i]）加入到对应的分组
			result[prefix] = append(result[prefix], parts[i])
		} else {
			// 如果不是最后一层，继续递归调用
			if _, exists := result[prefix]; !exists {
				// 只在第一次遇到该前缀时创建空切片
				result[prefix] = []string{}
			}
		}
	}
}
