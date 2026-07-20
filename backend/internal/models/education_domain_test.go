package models

import "testing"

func TestNormalizeEducationDomain(t *testing.T) {
	cases := map[string]string{
		"k12":        EducationDomainK12,
		"VOCATIONAL": EducationDomainVocational,
		" adult ":    EducationDomainAdult,
		"mixed":      EducationDomainMixed,
		"":           EducationDomainK12,
		"unknown":    EducationDomainK12,
	}

	for input, expected := range cases {
		if actual := NormalizeEducationDomain(input); actual != expected {
			t.Fatalf("NormalizeEducationDomain(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestEducationProfileForDomain(t *testing.T) {
	k12 := EducationProfileForDomain(EducationDomainK12)
	if k12.SubjectLabel != "学科" || !k12.CurriculumEnabled || !k12.PublisherEnabled {
		t.Fatalf("K12画像异常：%+v", k12)
	}

	vocational := EducationProfileForDomain(EducationDomainVocational)
	if vocational.SubjectLabel != "课程" ||
		vocational.CurriculumEnabled ||
		vocational.PublisherEnabled ||
		!vocational.MajorEnabled ||
		!vocational.PracticalTrainingEnabled {
		t.Fatalf("职业教育画像异常：%+v", vocational)
	}

	adult := EducationProfileForDomain(EducationDomainAdult)
	if adult.TopicLabel != "培训主题" ||
		adult.CurriculumEnabled ||
		adult.PublisherEnabled {
		t.Fatalf("成人教育画像异常：%+v", adult)
	}

	mixed := EducationProfileForDomain(EducationDomainMixed)
	if mixed.Code != EducationDomainMixed {
		t.Fatalf("mixed画像异常：%+v", mixed)
	}
}
