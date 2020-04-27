package main

import (
	"fmt"
	"github.com/Mokey-Pro/protoc-go-inject-tag/constants"
	"github.com/Mokey-Pro/protoc-go-inject-tag/utils"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
)

var (
	rComment = regexp.MustCompile(`^//\s*@inject_tag:\s*(.*)$`)
	rInject  = regexp.MustCompile("`.+`$")
	rTags    = regexp.MustCompile(`[\w_]+:"[^"]+"`)
)

type textArea struct {
	Start      int
	End        int
	CurrentTag string
	InjectTag  string
}

func parseFile(inputPath string, xxxSkip []string) (areas []textArea, err error) {
	logf("parsing file %q for inject tag comments", inputPath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for _, decl := range f.Decls {
		// check if is generic declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		var typeSpec *ast.TypeSpec
		for _, spec := range genDecl.Specs {
			if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
				typeSpec = ts
				break
			}
		}

		// skip if can't get type spec
		if typeSpec == nil {
			continue
		}

		// not a struct, skip
		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		/*builder := strings.Builder{}
		if len(xxxSkip) > 0 {
			for i, skip := range xxxSkip {
				builder.WriteString(fmt.Sprintf("%s:\"-\"", skip))
				if i > 0 {
					builder.WriteString(",")
				}
			}
		}*/

		var structTags tagItems // 结构体注解上的标签
		if genDecl.Doc != nil {
			structTags = getTagsFromComment(genDecl.Doc.List)
		}

		for _, field := range structDecl.Fields.List {
			var fieldTags tagItems // 字段注解上的标签
			if field.Doc != nil {
				fieldTags = getTagsFromComment(field.Doc.List)
			}

			// 根据tagKey获取结构体注释中对应的标签
			getFieldTag := func(tagKey string) tagItem {
				for _, each := range fieldTags {
					if tagKey == each.key {
						return each
					}
				}
				return tagItem{}
			}

			// 处理结构体注释中的标签
			for _, eachStructTag := range structTags {
				// 字段上对应key的标签
				fieldTag := getFieldTag(eachStructTag.key)

				if !fieldTag.isEmpty() {
					// 字段标签有值 --> 根据字段标签构造
					if currAreas := buildTagByFieldName(fieldTag.key, fieldTag.value, field); currAreas != nil {
						areas = append(areas, currAreas...)
					}
					continue
				} else {
					// 字段标签没有值 --> 根据结构体标签构造
					if currAreas := buildTagByFieldName(eachStructTag.key, eachStructTag.value, field); currAreas != nil {
						areas = append(areas, currAreas...)
					}
					continue
				}
			}

			// 处理字段注释中出现而结构体注释中没有出现的标签
			for _, eachFieldTag := range fieldTags {

				// 根据tagKey获取字段注释中对应的标签
				getStructTag := func(tagKey string) tagItem {
					for _, each := range structTags {
						if tagKey == each.key {
							return each
						}
					}
					return tagItem{}
				}

				structTag := getStructTag(eachFieldTag.key)
				if !structTag.isEmpty() {
					// 已经处理过了
					continue
				}

				if currAreas := buildTagByFieldName(eachFieldTag.key, eachFieldTag.value, field); currAreas != nil {
					areas = append(areas, currAreas...)
				}
			}
		}
	}
	logf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(areas))
	return
}

// 获取注解上的标签
func getTagsFromComment(commentList []*ast.Comment) tagItems {
	var structTags []tagItem
	// 结构体上有注解，字段上没有注解 ==>> 根据字段名称构建注解
	for _, structComment := range commentList {
		structTag := tagFromComment(structComment.Text) // 结构体注解上的标签
		if structTag == "" {
			continue
		} else {
			structTags = append(structTags, newTagItems(structTag)...)
		}
	}

	return structTags
}

func writeFile(inputPath string, areas []textArea) (err error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	if err = f.Close(); err != nil {
		return
	}

	// inject custom tags from tail of file first to preserve order
	for i := range areas {
		area := areas[len(areas)-i-1]
		logf("inject custom tag %q to expression %q", area.InjectTag, string(contents[area.Start-1:area.End-1]))
		contents = injectTag(contents, area)
	}
	if err = ioutil.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}

	if len(areas) > 0 {
		logf("file %q is injected with custom tags", inputPath)
	}
	return
}

// 根据字段名构建标签
func buildTagByFieldName(tagKey, tagValue string, field *ast.Field) (areas []textArea) {
	if len(field.Names) <= 0 {
		return nil
	}
	fieldName := field.Names[0].Name

	var buildTag string
	if string(constants.TAG_VALUE_keep_ignore) == tagValue {
		buildTag = fmt.Sprintf("%s:%s", tagKey, tagValue)
	} else if string(constants.TAG_VALUE_keep_toCamel) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.Format2Camel(fieldName))
	} else if string(constants.TAG_VALUE_keep_toCamel2) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.LcFirst(utils.Format2Camel(fieldName)))
	} else if string(constants.TAG_VALUE_keep_toSnake) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.Format2Snake(fieldName))
	} else {
		buildTag = fmt.Sprintf("%s:%s", tagKey, tagValue)
	}

	currentTag := field.Tag.Value
	area := textArea{
		Start:      int(field.Pos()),
		End:        int(field.End()),
		CurrentTag: currentTag[1 : len(currentTag)-1],
		InjectTag:  buildTag,
	}
	areas = append(areas, area)
	return
}
