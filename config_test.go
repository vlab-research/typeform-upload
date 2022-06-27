package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitialForms_HappyPath(t *testing.T) {
	cf := NewSurveyFile("workey", "test/Survey Translation Example.xlsx")
	forms, err := cf.InitialForms()
	assert.Nil(t, err)

	assert.Equal(t, forms["Baseline"].Name, "Survey Translation Example - Baseline")
	assert.Equal(t, forms["Baseline"].Form.Workspace.Href, "https://api.typeform.com/workspaces/workey")
	assert.Equal(t, len(forms), 1)
	assert.Equal(t, "label.error.mustSelect", forms["Baseline"].MessagesData[1][0])

	assert.Equal(t, "Hello! We would like to take your time!", forms["Baseline"].Form.Fields[0].Title)
}
