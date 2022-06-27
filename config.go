package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"path/filepath"
	"strings"
)

// workspace --base file
// workspace --base file --translation file

type SurveyFile struct {
	Workspace string
	BaseName  string
	Path      string
}

func NewSurveyFile(workspace, path string) *SurveyFile {

	// create base name of form from filename itself
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.ReplaceAll(base, ext, "")

	return &SurveyFile{
		workspace, name, path,
	}
}

func (c *SurveyFile) InitialForms() (map[string]*FormConf, error) {

	f, err := excelize.OpenFile(c.Path)
	if err != nil {
		return nil, err
	}

	messageRecords, err := f.GetRows("Messages")
	if err != nil {
		return nil, err
	}

	sheets := f.GetSheetList()
	forms := map[string]*FormConf{}

	for _, s := range sheets {
		if s != "Messages" {
			finalName := fmt.Sprintf("%s - %s", c.BaseName, s)
			formRecords, err := f.GetRows(s)
			if err != nil {
				return nil, err
			}

			conf, err := NewFormConf(c.Workspace, finalName, formRecords, messageRecords)
			if err != nil {
				return nil, err
			}
			forms[s] = conf
		}
	}

	return forms, nil
}
