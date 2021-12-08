package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/dghubble/sling"
	"github.com/vlab-research/trans"
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

func BuildForm(title string, records [][]string) (Form, error) {
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

	return Form{Title: title, Fields: fields, ThankYouScreens: thankyouScreens}, nil
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
	Workspace       Workspace         `json:"workspace,omitempty"`
	Title           string            `json:"title"`
	Fields          []*trans.Field    `json:"fields"`
	ThankYouScreens []*ThankyouScreen `json:"thankyou_screens,omitempty"`
	Logic           json.RawMessage   `json:"logic,omitempty"`
}

type Config struct {
	TypeformToken string `env:"TYPEFORM_TOKEN,required"`
}

func getConfig() *Config {
	cfg := Config{}
	err := env.Parse(&cfg)
	handle(err)
	return &cfg
}

func Api(cnf *Config) *sling.Sling {
	client := &http.Client{}
	sli := sling.New().Client(client).Base("https://api.typeform.com")

	auth := fmt.Sprintf("%v %v", "Bearer", cnf.TypeformToken)
	sli = sli.Set("Authorization", auth)

	return sli
}

func postForm(api *sling.Sling, form Form) {
	apiError := TypeformError{}
	resp := new(CreateFormResponse)

	httpResponse, err := api.Post("forms").BodyJSON(&form).Receive(resp, &apiError)
	if err != nil {
		fmt.Println(err)
	}

	if httpResponse.StatusCode == 201 {
		fmt.Printf("Success! Created form in Typeform with %d questions", len(form.Fields))
	} else {
		fmt.Println("FAIL")
		fmt.Println(apiError)
		fmt.Println(httpResponse)
	}
}

func getForm(api *sling.Sling, id string) string {
	// apiError := TypeformError{}
	// resp := new(Form)

	req, err := api.Path("forms/").Get(id).Request()
	handle(err)

	client := &http.Client{}
	res, err := client.Do(req)
	handle(err)

	b, err := io.ReadAll(res.Body)
	handle(err)

	return string(b)
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

func UpdateMessages(api *sling.Sling, id string, messages Messages) {
	apiError := TypeformError{}
	resp := new(CreateFormResponse)

	httpResponse, err := api.Path("forms/").Path(id+"/").Put("messages").BodyJSON(messages).Receive(resp, &apiError)
	handle(err)

	if httpResponse.StatusCode == 204 {
		fmt.Println("Success!")
	} else {
		fmt.Println(apiError)
	}

}

func main() {
	cnf := getConfig()
	api := Api(cnf)

	records := readCsvFile("Bangla DIME Curious Learning Survey - Messages.csv")
	UpdateMessages(api, "kUxQWvKg", ParseMessages(records))

	// records := readCsvFile("endline.csv")
	// form, err := BuildForm("Routine Immunization - Endline", records[1:])
	// form.Workspace = Workspace{"https://api.typeform.com/workspaces/yPd8ZS"}

	// handle(err)
	// postForm(api, form)

	// fmt.Println(getForm(api, "m1BE0cgH"))
}
