package main

import (
	"fmt"
	"github.com/favadi/protoc-go-inject-tag/constants"
	"github.com/favadi/protoc-go-inject-tag/utils"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
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

		builder := strings.Builder{}
		if len(xxxSkip) > 0 {
			for i, skip := range xxxSkip {
				builder.WriteString(fmt.Sprintf("%s:\"-\"", skip))
				if i > 0 {
					builder.WriteString(",")
				}
			}
		}

		for _, field := range structDecl.Fields.List {
			// skip if field has no doc
			if len(field.Names) > 0 {
				name := field.Names[0].Name
				if len(xxxSkip) > 0 && strings.HasPrefix(name, "XXX") {
					currentTag := field.Tag.Value
					area := textArea{
						Start:      int(field.Pos()),
						End:        int(field.End()),
						CurrentTag: currentTag[1 : len(currentTag)-1],
						InjectTag:  builder.String(),
					}
					areas = append(areas, area)
				}
			}
			if field.Doc == nil {
				if genDecl.Doc == nil {
					// 结构体上没有注解 && 字段上没有注解
					continue
				} else {
					// 结构体上有注解，字段上没有注解 ==>> 根据字段名称构建注解
					for _, structComment := range genDecl.Doc.List {
						structTag := tagFromComment(structComment.Text) // 结构体注解上的标签
						if structTag == "" {
							continue
						} else {
							structTags := newTagItems(structTag)
							for _, eachStructTag := range structTags {
								currAreas := buildTagByFieldName(eachStructTag.key, eachStructTag.value, field)
								if currAreas != nil {
									areas = append(areas, currAreas...)
								}
							}
						}
					}
				}
			} else {
				// 字段上有注解 (忽略结构体上的注解)
				for _, comment := range field.Doc.List {
					tag := tagFromComment(comment.Text)
					if tag == "" {
						continue
					}

					fieldTags := newTagItems(tag)
					for _, eachFieldTag := range fieldTags {
						if strings.HasPrefix(eachFieldTag.value, string(constants.TAG_VALUE_keep_prefix)) {
							currAreas := buildTagByFieldName(eachFieldTag.key, eachFieldTag.value, field)
							if currAreas != nil {
								areas = append(areas, currAreas...)
							}
						} else {
							currentTag := field.Tag.Value
							area := textArea{
								Start:      int(field.Pos()),
								End:        int(field.End()),
								CurrentTag: currentTag[1 : len(currentTag)-1],
								InjectTag:  tag,
							}
							areas = append(areas, area)
						}
					}
				}
			}
		}
	}
	logf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(areas))
	return
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
	if string(constants.TAG_VALUE_keep_toCamel) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.Format2Camel(fieldName))
	} else if string(constants.TAG_VALUE_keep_toCamel2) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.LcFirst(utils.Format2Camel(fieldName)))
	} else if string(constants.TAG_VALUE_keep_toSnake) == tagValue {
		buildTag = fmt.Sprintf("%s:\"%s\"", tagKey, utils.Format2Snake(fieldName))
	}

	currentTag := field.Tag.Value
	currentTags := newTagItems(field.Tag.Value)
	for _, eachCurrentTag := range currentTags {
		if string(constants.TAG_VALUE_keep_ignore) != eachCurrentTag.value {
			area := textArea{
				Start:      int(field.Pos()),
				End:        int(field.End()),
				CurrentTag: currentTag[1 : len(currentTag)-1],
				InjectTag:  buildTag,
			}
			areas = append(areas, area)
		}
	}
	return
}
