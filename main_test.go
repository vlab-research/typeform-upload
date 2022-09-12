package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dghubble/sling"
	"github.com/stretchr/testify/assert"
)

func testServer(handler func(http.ResponseWriter, *http.Request)) (*httptest.Server, *sling.Sling) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	sli := sling.New().Client(&http.Client{}).Base(ts.URL)
	return ts, sli
}

func TestCreateForm_CreatesAndUpdatesMessages(t *testing.T) {
	call := 0

	ts, _ := testServer(func(w http.ResponseWriter, r *http.Request) {
		call++

		if call == 1 {
			assert.Equal(t, "/forms", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(200)
			fmt.Fprintf(w, `{"items": [{"id": "foo", "title": "Foo"}]}`)
		}

		if call == 2 {
			assert.Equal(t, "/forms", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			bodyBytes, _ := ioutil.ReadAll(r.Body)
			body := strings.TrimSpace(string(bodyBytes))

			expected := `{"workspace":{"href":"https://api.typeform.com/workspaces/workspace"},"title":"form name","fields":[{"type":"multiple_choice","title":"hello\n\n- A. Foo\n- B. Bar","ref":"var1","properties":{"choices":[{"label":"A"},{"label":"B"}]}}]}`

			assert.Equal(t, expected, body)

			w.Header().Set("Location", "https://api.typeform.com/forms/foobar")
			w.WriteHeader(201)
		}

		if call == 3 {
			assert.Equal(t, "/forms/foobar/messages", r.URL.Path)
			assert.Equal(t, "PUT", r.Method)

			bodyBytes, _ := ioutil.ReadAll(r.Body)
			body := strings.TrimSpace(string(bodyBytes))

			expected := `{"var1":"message1","var2":"message2"}`
			assert.Equal(t, expected, body)

			w.WriteHeader(204)
		}
	})

	uploader := TypeformUploader{
		BaseUrl:       ts.URL,
		TypeformToken: "secret",
	}

	formData := [][]string{
		{"variable", "question_type", "question", "answers", "description"},
		{"var1", "multiple_choice", "hello", "- A. Foo\n- B. Bar", ""},
	}

	messageData := [][]string{
		{"variable", "message"},
		{"var1", "message1"},
		{"var2", "message2"},
	}

	conf, _ := NewFormConf("workspace", "form name", formData, messageData)
	err := uploader.CreateForm(conf)
	assert.Nil(t, err)
	assert.Equal(t, 3, call)
}

func TestCreateForm_FailsIfFormWithSameNameExists(t *testing.T) {
	call := 0

	ts, _ := testServer(func(w http.ResponseWriter, r *http.Request) {
		call++

		if call == 1 {
			w.WriteHeader(200)
			fmt.Fprintf(w, `{"items": [{"id": "foo", "title": "form name"}]}`)
		}
	})

	uploader := TypeformUploader{
		BaseUrl:       ts.URL,
		TypeformToken: "secret",
	}

	formData := [][]string{
		{"variable", "question_type", "question", "answers", "description"},
		{"var1", "multiple_choice", "hello", "- A. Foo\n- B. Bar", ""},
	}

	messageData := [][]string{
		{"variable", "message"},
		{"var1", "message1"},
	}

	conf, _ := NewFormConf("workspace", "form name", formData, messageData)
	err := uploader.CreateForm(conf)
	assert.Contains(t, err.Error(), "form name")
	assert.Equal(t, 1, call)
}

func TestCreateForm_ReturnsApiErrors(t *testing.T) {
	call := 0

	ts, _ := testServer(func(w http.ResponseWriter, r *http.Request) {
		call++

		if call == 1 {
			w.WriteHeader(200)
			fmt.Fprintf(w, `{"items": [{"id": "foo", "title": "Foo"}]}`)
		}

		if call == 2 {
			assert.Equal(t, "/forms", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			w.WriteHeader(400)
			fmt.Fprintf(w, `{"code": "SOME_CODE"}`)

		}

	})

	uploader := TypeformUploader{
		BaseUrl:       ts.URL,
		TypeformToken: "secret",
	}

	formData := [][]string{
		{"variable", "question_type", "question", "answers", "description"},
		{"var1", "multiple_choice", "hello", "- A. Foo\n- B. Bar", ""},
	}

	messageData := [][]string{
		{"variable", "message"},
		{"var1", "message1"},
	}

	conf, _ := NewFormConf("workspace", "form name", formData, messageData)
	err := uploader.CreateForm(conf)
	e := err.(*TypeformError)

	assert.Equal(t, "SOME_CODE", e.Code)
}

func TestUploaderTranslations_GetsTranslationsBasedOnFilesAndExistingForm(t *testing.T) {
	call := 0

	// note implicitly testing NewSurveyFile here
	baseForms, _ := NewSurveyFile("workey", "test/Survey Translation Example.xlsx").InitialForms()
	baseForm := baseForms["Baseline"]

	ts, _ := testServer(func(w http.ResponseWriter, r *http.Request) {
		call++

		if call == 1 {
			assert.Equal(t, "/forms", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(200)
			fmt.Fprintf(w, `{"items": [{"id": "foo", "title": "Survey Translation Example - Baseline"}]}`)
		}

		if call == 2 {
			assert.Equal(t, "/forms/foo", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			w.WriteHeader(200)

			// send back base form as parsed
			b, err := json.Marshal(baseForm.Form)
			handle(err)

			fmt.Println(string(b))
			fmt.Fprintf(w, string(b))
		}
	})

	uploader := TypeformUploader{
		BaseUrl:       ts.URL,
		TypeformToken: "secret",
	}

	translations, err := uploader.Translations("workey", "test/Survey Translation Example.xlsx", "test/Survey Translation Example Spanish.xlsx")
	assert.Nil(t, err)

	assert.Equal(t, 2, call)

	assert.Equal(t, 1, len(translations))

	// Form has a new name
	assert.Equal(t, "Survey Translation Example Spanish - Baseline", translations["Baseline"].Name)

	// Form has correct workspace
	assert.Equal(t, "https://api.typeform.com/workspaces/workey", translations["Baseline"].Form.Workspace.Href)

	// Form has translated messages data
	assert.Equal(t, "Perdona, solo entiendo ciertas cosas. ", translations["Baseline"].MessagesData[1][1])

	// Form has translated fields
	assert.Equal(t, "Hola! Quieremos robar tu tiempo. ", translations["Baseline"].Form.Fields[0].Title)
	assert.Equal(t, "Te parece?", translations["Baseline"].Form.Fields[2].Title)
	assert.Equal(t, "Si", translations["Baseline"].Form.Fields[2].Properties.Choices[0].Label)

}

// You added stupid messages-only, now test that...
