package constants

type TAG_KEY_TYPE string
type TAG_VALUE_TYPE string

const (
	TAG_KEY_JSON TAG_KEY_TYPE = "json" // json标签
	TAG_KEY_GORM TAG_KEY_TYPE = "gorm" // gorm标签

	TAG_VALUE_keep_ignore TAG_VALUE_TYPE = `"-"`  // 忽略
	TAG_VALUE_keep_delete TAG_VALUE_TYPE = `"#-"` // 删除标签

	TAG_VALUE_keep_prefix   TAG_VALUE_TYPE = `"#`                                // 标签值前缀
	TAG_VALUE_keep_toSnake  TAG_VALUE_TYPE = TAG_VALUE_keep_prefix + `toSnake"`  // 转成蛇形-->xx_yy
	TAG_VALUE_keep_toCamel  TAG_VALUE_TYPE = TAG_VALUE_keep_prefix + `toCamel"`  // 转成驼峰-->XxYy
	TAG_VALUE_keep_toCamel2 TAG_VALUE_TYPE = TAG_VALUE_keep_prefix + `toCamel2"` // 转成驼峰,首字符小写-->xxYy
)
