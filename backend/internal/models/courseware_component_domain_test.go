package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCWComponentResourceJSONIncludesEducationDomain(
	t *testing.T,
) {
	resource := &CWComponentResource{
		CoursewareComponent: &CoursewareComponent{
			Name:          "示例组件",
			ComponentType: "layout",
		},
		EducationDomain: EducationDomainK12,
	}

	rawJSON, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf(
			"序列化组件资源失败：%v",
			err,
		)
	}

	var payload map[string]interface{}

	if err := json.Unmarshal(
		rawJSON,
		&payload,
	); err != nil {
		t.Fatalf(
			"解析组件资源JSON失败：%v",
			err,
		)
	}

	if payload["education_domain"] !=
		EducationDomainK12 {
		t.Fatalf(
			"响应缺少可信教育域：%v",
			payload["education_domain"],
		)
	}

	if payload["name"] != "示例组件" {
		t.Fatalf(
			"嵌入的基础组件字段丢失：%v",
			payload["name"],
		)
	}
}

func TestCreateCWComponentDomainRequestJSONIncludesEducationDomain(
	t *testing.T,
) {
	request := &CreateCWComponentDomainRequest{
		CreateCWComponentRequest: CreateCWComponentRequest{
			Name:          "创建组件",
			ComponentType: "layout",
			CodeContent:   "<div></div>",
		},
		EducationDomain: EducationDomainVocational,
	}

	rawJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatalf(
			"序列化创建请求失败：%v",
			err,
		)
	}

	var payload map[string]interface{}

	if err := json.Unmarshal(
		rawJSON,
		&payload,
	); err != nil {
		t.Fatalf(
			"解析创建请求JSON失败：%v",
			err,
		)
	}

	if payload["education_domain"] !=
		EducationDomainVocational {
		t.Fatalf(
			"创建请求未携带教育域：%v",
			payload["education_domain"],
		)
	}

	if payload["name"] != "创建组件" {
		t.Fatalf(
			"创建请求基础字段丢失：%v",
			payload["name"],
		)
	}
}

func TestUpdateCWComponentRequestDoesNotExposeEducationDomain(
	t *testing.T,
) {
	requestType := reflect.TypeOf(
		UpdateCWComponentRequest{},
	)

	for index := 0; index < requestType.NumField(); index++ {
		field := requestType.Field(index)

		jsonName := strings.Split(
			field.Tag.Get("json"),
			",",
		)[0]

		if field.Name == "EducationDomain" ||
			jsonName == "education_domain" {
			t.Fatalf(
				"更新协议不能暴露education_domain字段：%s",
				field.Name,
			)
		}
	}
}
