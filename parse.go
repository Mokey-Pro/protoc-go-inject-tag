package main

import (
	"fmt"
	"github.com/Monkey-Pro/protoc-go-inject-tag/utils"
	"sort"
	"strings"
)

func tagFromComment(comment string) (tag string) {
	match := rComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

type tagItem struct {
	key   string
	value string
}

func (thiz tagItem) isEmpty() bool {
	return utils.IsEmpty(thiz.key) && utils.IsEmpty(thiz.value)
}

type tagItems []tagItem

func (ti tagItems) format() string {
	tags := []string{}
	for _, item := range ti {
		tags = append(tags, fmt.Sprintf(`%s:%s`, item.key, item.value))
	}
	return strings.Join(tags, " ")
}

func (ti tagItems) override(nti tagItems) tagItems {
	overrided := []tagItem{}
	for i := range ti {
		var dup = -1
		for j := range nti {
			if ti[i].key == nti[j].key {
				dup = j
				break
			}
		}
		if dup == -1 {
			overrided = append(overrided, ti[i])
		} else {
			overrided = append(overrided, nti[dup])
			nti = append(nti[:dup], nti[dup+1:]...)
		}
	}
	overrided = append(overrided, nti...)
	return overrided
}

func newTagItems(tags ...string) tagItems {
	items := []tagItem{}
	keyValueMap := make(map[string]string, 0)

	for _, tag := range tags {
		splitted := rTags.FindAllString(tag, -1)

		for _, t := range splitted {
			sepPos := strings.Index(t, ":")
			key := t[:sepPos]
			value := t[sepPos+1:]

			keyValueMap[key] = value
		}
	}

	for key, value := range keyValueMap {
		items = append(items, tagItem{
			key:   key,
			value: value,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].key < items[j].key {
			return false
		}
		if items[i].key > items[j].key {
			return true
		}

		return false
	})

	return items
}

// filedAreas 一个字段的所有标签
func injectTag(contents []byte, filedAreas []textArea, offset int) (injected []byte, finalOffset int) {
	if len(filedAreas) <= 0 {
		return
	}

	temp := filedAreas[0]
	expr := make([]byte, temp.End-temp.Start)

	copy(expr, contents[temp.Start-1+offset:temp.End-1+offset])

	getStr := func(areas []textArea, vType int) []string {
		retList := make([]string, 0)
		for _, each := range areas {
			//eachStr := ""
			var eachStr string
			switch vType {
			case 1:
				eachStr = each.CurrentTag
			case 2:
				eachStr = each.InjectTag
			default:
				fmt.Println("没有对应的类型, vType=" + string(vType))
			}
			retList = append(retList, eachStr)
		}
		return retList
	}

	cti := newTagItems(getStr(filedAreas, 1)...)
	iti := newTagItems(getStr(filedAreas, 2)...)
	ti := cti.override(iti)

	oldLen := len(expr)
	expr = rInject.ReplaceAll(expr, []byte(fmt.Sprintf("`%s`", ti.format())))
	finalOffset = len(expr) - oldLen + offset

	injected = append(injected, contents[:temp.Start-1+offset]...)
	injected = append(injected, expr...)
	injected = append(injected, contents[temp.End-1+offset:]...)
	return
}
