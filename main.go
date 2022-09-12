package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/dghubble/sling"
	"github.com/vlab-research/trans"
	"log"
	"net/http"
	"os"
	"strings"
)

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func get(row []string, i int) string {
	if len(row) >= (i + 1) {
		return row[i]
	}
	return ""
}

func ExtractParagraphs(text string) []string {
	return strings.Split(strings.TrimSpace(text), "\n")
}

func extractChoices(options string) []trans.FieldChoice {
	s := ExtractParagraphs(options)
	choices := make([]trans.FieldChoice, len(s))
	for i, ss := range s {
		choices[i] = trans.FieldChoice{
			ID:    "",
			Label: ss,
			Ref:   "",
		}
	}
	return choices
}

func BuildField(row []string) (interface{}, error) {
	if len(row) < 3 {
		return nil, fmt.Errorf("This row doesn't have the right number of columns: %s", row)
	}

	ref := row[0]
	questionType := row[1]
	q := row[2]

	if q == "" || ref == "" {
		return nil, fmt.Errorf("This row has empty columns and will be skipped: %s", row)
	}

	choices := []trans.FieldChoice{}
	var title string

	options := get(row, 3)
	description := get(row, 4)

	title = q

	if questionType == "multiple_choice" {
		if options == "" {
			return nil, fmt.Errorf("multiple_choice question without options! Skipping. Row: %s", row)
		}

		answers, err := trans.ExtractLabels(options)

		if err != nil {
			return nil, err
		}

		if len(answers) == 0 {
			choices = extractChoices(options)
		} else {
			title = fmt.Sprintf("%s\n\n%s", strings.TrimSpace(q), strings.TrimSpace(options))
			for _, answer := range answers {
				label := answer.Response
				choices = append(choices, trans.FieldChoice{Label: label})
			}
		}
	}

	if questionType == "thankyou_screen" {
		f := &ThankyouScreen{
			Ref:   ref,
			Title: title,
		}
		return f, nil
	}

	f := &trans.Field{
		Type:  questionType,
		Title: title,
		Ref:   ref,
		Properties: trans.FieldProperties{
			Choices:     choices,
			Description: description,
		},
	}
	return f, nil
}

func BuildForm(title string, records [][]string) (*Form, error) {
	fields := []*trans.Field{}
	thankyouScreens := []*ThankyouScreen{}

	for _, record := range records {
		f, err := BuildField(record)
		if err != nil {
			fmt.Println(err)
			// hrm...
			continue
		}

		switch f.(type) {
		case *trans.Field:
			fields = append(fields, f.(*trans.Field))
		case *ThankyouScreen:
			thankyouScreens = append(thankyouScreens, f.(*ThankyouScreen))
		}

	}

	return &Form{Title: title, Fields: fields, ThankYouScreens: thankyouScreens}, nil
}

type ErrorDetail struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Field       string `json:"field"`
	In          string `json:"in"`
}

type TypeformError struct {
	Code        string        `json:"code"`
	Description string        `json:"description"`
	Details     []ErrorDetail `json:"details"`
}

func (e *TypeformError) Error() string {
	return fmt.Sprintf("%s. %s. Details: %s", e.Code, e.Description, e.Details)
}

func (e *TypeformError) Empty() bool {
	return e.Code == ""
}

type ThankyouScreen struct {
	Ref   string `json:"ref"`
	Title string `json:"title"`
}

type CreateFormResponse struct {
}

type Workspace struct {
	Href string `json:"href,omitempty"`
}

type Form struct {
	// add workspace and other things
	ID              string            `json:"id,omitempty"`
	Workspace       Workspace         `json:"workspace,omitempty"`
	Title           string            `json:"title"`
	Fields          []*trans.Field    `json:"fields"`
	ThankYouScreens []*ThankyouScreen `json:"thankyou_screens,omitempty"`
	Logic           json.RawMessage   `json:"logic,omitempty"`
}

type TypeformUploader struct {
	BaseUrl       string `env:"TYPEFORM_BASE_URL,required"`
	TypeformToken string `env:"TYPEFORM_TOKEN,required"`
	api           *sling.Sling
}

func (t *TypeformUploader) LoadEnv() {
	err := env.Parse(t)
	handle(err)
}

func (t *TypeformUploader) Api() *sling.Sling {
	if t.api != nil {
		return t.api
	}
	client := &http.Client{}
	sli := sling.New().Client(client).Base(t.BaseUrl)

	auth := fmt.Sprintf("%v %v", "Bearer", t.TypeformToken)
	sli = sli.Set("Authorization", auth)

	t.api = sli
	return sli
}

func postForm(api *sling.Sling, form *Form) (error, string) {
	apiError := new(TypeformError)
	resp := new(CreateFormResponse)

	httpResponse, err := api.New().Post("forms").BodyJSON(form).Receive(resp, &apiError)
	if err != nil {
		return nil, ""
	}

	if !apiError.Empty() {
		return apiError, ""
	}

	loc := httpResponse.Header.Get("Location")
	fmt.Println(fmt.Sprintf("Success! Created form in Typeform with %d questions", len(form.Fields)))

	return nil, loc
}

func (t *TypeformUploader) GetForm(id string) (*Form, error) {

	api := t.Api()

	apiError := new(TypeformError)
	form := new(Form)

	_, err := api.New().Path("forms/").Get(id).Receive(form, apiError)
	if err != nil {
		return nil, err
	}

	if !apiError.Empty() {
		return nil, apiError
	}
	return form, nil
}

type FormsResponse struct {
	Items []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"items"`
}

func (t *TypeformUploader) GetForms(workspace string) (*FormsResponse, error) {

	api := t.Api()

	apiError := new(TypeformError)
	forms := new(FormsResponse)

	params := struct {
		WorkspaceId string `url:"workspace_id"`
	}{workspace}

	_, err := api.New().Path("forms").QueryStruct(params).Receive(forms, apiError)

	if err != nil {
		return nil, err
	}

	if !apiError.Empty() {
		fmt.Println(fmt.Sprintf("Cannot get forms from workspace %s", workspace))
		return nil, apiError
	}

	return forms, nil
}

var ExistingFormError = errors.New("Form exists already.")

func (t *TypeformUploader) AssertFormDoesNotExist(workspace, name string) error {
	forms, err := t.GetForms(workspace)

	if err != nil {
		return err
	}

	for _, f := range forms.Items {
		if f.Title == name {
			return fmt.Errorf("Form with name %s in workspace %s already exists: %w", name, workspace, ExistingFormError)
		}
	}

	return nil
}

type Messages map[string]string

func ParseMessages(records [][]string) Messages {
	messages := Messages{}

	for _, r := range records[1:] {
		k := r[0]
		v := r[1]

		if k == "" || v == "" {
			fmt.Printf("skipping row: %s", r)
			continue
		}

		messages[k] = strings.TrimSpace(v)
	}

	return messages
}

func UpdateMessages(api *sling.Sling, id string, messages Messages) error {
	apiError := new(TypeformError)
	resp := new(CreateFormResponse)

	httpResponse, err := api.New().Path("forms/").Path(id+"/").Put("messages").BodyJSON(messages).Receive(resp, apiError)

	if err != nil {
		return err
	}

	if !apiError.Empty() {
		return apiError
	}

	if httpResponse.StatusCode != 204 {
		return apiError // may not exist?
	}
	return nil
}

func (t *TypeformUploader) CreateForm(conf *FormConf) error {
	api := t.Api()

	// get workspace from url version
	parts := strings.Split(conf.Form.Workspace.Href, "/")
	workspace := parts[len(parts)-1]

	err := t.AssertFormDoesNotExist(workspace, conf.Name)
	if err != nil {
		return err
	}

	err, loc := postForm(api, conf.Form)
	if err != nil {
		return err
	}

	// get FormId of newly created form
	parts = strings.Split(loc, "/")
	formId := parts[len(parts)-1]

	err = UpdateMessages(api, formId, ParseMessages(conf.MessagesData))
	return err
}

func (t *TypeformUploader) UpdateFormMessages(conf *FormConf) error {
	api := t.Api()

	parts := strings.Split(conf.Form.Workspace.Href, "/")
	workspace := parts[len(parts)-1]

	form, err := t.GetByName(workspace, conf.Form.Title)

	if err != nil {
		return err
	}

	err = UpdateMessages(api, form.ID, ParseMessages(conf.MessagesData))
	return err
}

func (t *TypeformUploader) GetByName(workspace, name string) (*Form, error) {
	forms, err := t.GetForms(workspace)
	if err != nil {
		return nil, err
	}

	for _, form := range forms.Items {
		if form.Title == name {
			return t.GetForm(form.ID)
		}
	}
	return nil, fmt.Errorf("Could not find form with name: %s", name)
}

func (t *TypeformUploader) BaseForms(workspace, basePath string) (map[string]*FormConf, error) {
	return NewSurveyFile(workspace, basePath).InitialForms()
}

func (t *TypeformUploader) Translations(workspace, basePath, translationPath string) (map[string]*FormConf, error) {

	bases, err := NewSurveyFile(workspace, basePath).InitialForms()
	if err != nil {
		return nil, err
	}

	translations, err := NewSurveyFile(workspace, translationPath).InitialForms()
	if err != nil {
		return nil, err
	}

	for sheet, baseConf := range bases {
		actualForm, err := t.GetByName(workspace, baseConf.Name)
		if err != nil {
			return nil, err
		}

		translationConf, ok := translations[sheet]

		if !ok {
			return nil, fmt.Errorf("Could not find translation for form: %s", baseConf.Name)
		}

		newForm, err := TranslateForm(actualForm, translationConf.Form)
		if err != nil {
			return nil, err
		}

		translationConf.Form = newForm
	}

	return translations, nil
}

type FormConf struct {
	Name         string
	Form         *Form
	MessagesData [][]string
}

func NewFormConf(workspace, name string, formData [][]string, messagesData [][]string) (*FormConf, error) {
	form, err := BuildForm(name, formData[1:])
	if err != nil {
		return nil, err
	}
	form.Workspace = Workspace{fmt.Sprintf("https://api.typeform.com/workspaces/%s", workspace)}

	conf := &FormConf{name, form, messagesData}
	return conf, nil
}

func TranslateConf(conf *FormConf, src *Form) (*FormConf, error) {

	// get source form from api?

	newForm, err := TranslateForm(src, conf.Form)
	if err != nil {
		return nil, err
	}

	conf.Form = newForm
	return conf, nil
}

// Step 1
// create forms - english
// with custom messages, etc.
// make logic in Typeform
// test and finalize

// Step 2
// use ref form from Typeform (en) +
// csv with translation
// make translated form in Typeform
// with custom messages

func runCreate(uploader TypeformUploader, formConfs map[string]*FormConf, sheet string, messagesOnly bool) {
	for s, c := range formConfs {

		if sheet != "" && s != sheet {
			continue
		}

		var err error 

		if messagesOnly {
			err = uploader.UpdateFormMessages(c)
		} else {
			err = uploader.CreateForm(c)
		}
		
		if errors.Is(err, ExistingFormError) {
			log.Println("Skipping existing form")
			continue
		}

		log.Println(err)
	}
}

func runBaseCreate(uploader TypeformUploader, workspace, basePath, sheet string, messagesOnly bool) {
	formConfs, err := uploader.BaseForms(workspace, basePath)
	handle(err)

	runCreate(uploader, formConfs, sheet, messagesOnly)
}

func runTranslations(uploader TypeformUploader, workspace, basePath, translations, sheet string, messagesOnly bool) {
	formConfs, err := uploader.Translations(workspace, basePath, translations)
	handle(err)

	runCreate(uploader, formConfs, sheet, messagesOnly)
}

func main() {
	workspace := flag.String("workspace", "", "Typeform workspace id")
	basePath := flag.String("base", "", "path to base file")
	translationPath := flag.String("translation", "", "path to translation file")

	messagesOnly := flag.Bool("messages-only", false, "if you only want to modify messages")

	sheet := flag.String("sheet", "", "sheet to load individual sheet")

	flag.Parse()

	uploader := TypeformUploader{}
	uploader.LoadEnv()

	if *translationPath == "" {
		runBaseCreate(uploader, *workspace, *basePath, *sheet, *messagesOnly)
	} else {
		runTranslations(uploader, *workspace, *basePath, *translationPath, *sheet, *messagesOnly)
	}
}
