package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockForm(j string) *Form {
	f := new(Form)
	err := json.Unmarshal([]byte(j), f)
	handle(err)

	return f
}

func readFile(fi string) *Form {
	b, e := ioutil.ReadFile(fmt.Sprintf("test/%s", fi))
	handle(e)

	return mockForm(string(b))
}

func TestTranslateForm_WithMultipleChoice(t *testing.T) {
	j := `{"workspace":{"href":"https://api.typeform.com/workspaces/workspace"},"title":"form name","fields":[{"type":"multiple_choice","title":"hello\n\n- A. Foo\n- B. Bar","ref":"var1","properties":{"choices":[{"label":"A"},{"label":"B"}]}}]}`

	jt := `{"workspace":{"href":"https://api.typeform.com/workspaces/workspace"},"title":"form name","fields":[{"type":"multiple_choice","title":"hola\n\n- C. Foosp\n- D. Barsp","ref":"var1","properties":{"choices":[{"label":"C"},{"label":"D"}]}}]}`

	f := mockForm(j)
	ft := mockForm(jt)

	res, err := TranslateForm(f, ft)

	assert.Nil(t, err)
	assert.NotNil(t, f)

	assert.Equal(t, "hola\n\n- C. Foosp\n- D. Barsp", res.Fields[0].Title)
	assert.Equal(t, "C", res.Fields[0].Properties.Choices[0].Label)
	assert.Equal(t, "D", res.Fields[0].Properties.Choices[1].Label)
}

func TestTranslateForm_WithPerfectOrder(t *testing.T) {
	f := readFile("translate_test_en.json")  // will be mutated
	f2 := readFile("translate_test_en.json") // original

	ft := readFile("translate_test_sp.json")

	res, err := TranslateForm(f, ft)

	assert.Nil(t, err)
	assert.NotNil(t, f)

	// assert all refs/ids are the same as original
	for i, field := range f2.Fields {
		resField := res.Fields[i]

		assert.Equal(t, field.Ref, resField.Ref)

		// don't copy description

		if len(field.Properties.Choices) != 0 {
			for j, choice := range field.Properties.Choices {
				resChoice := resField.Properties.Choices[j]

				assert.Equal(t, choice.Ref, resChoice.Ref)
			}
		}
	}

	assert.Equal(t, f.Logic, res.Logic)

	assert.Equal(t, ft.Title, res.Title)

	assert.Equal(t, "hola.", res.Fields[0].Title)
	assert.Equal(t, "¿Le gusta éste?", res.Fields[1].Title)

	assert.Equal(t, "sí", res.Fields[1].Properties.Choices[0].Label)
	assert.Equal(t, "no", res.Fields[1].Properties.Choices[1].Label)

	assert.Equal(t, "Gracias por su tiempo!", res.ThankYouScreens[0].Title)
}
