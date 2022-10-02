package main

import (
	"fmt"

	"github.com/vlab-research/trans"
)

func findField(ref string, form *Form) (*trans.Field, error) {
	for _, f := range form.Fields {
		if f.Ref == ref {
			return f, nil
		}
	}
	return nil, fmt.Errorf("Could not find field ref %v in form titled %v", ref, form.Title)
}

func copyChoiceRefs(f *trans.Field, dest *Form) (*trans.Field, error) {
	tf, err := findField(f.Ref, dest)

	if err != nil {
		return nil, err
	}

	if len(f.Properties.Choices) != len(tf.Properties.Choices) {
		return nil, fmt.Errorf("Number of choices not the same for field ref: %v. There are %d choices in the target and %d choices in the source", f.Ref, len(f.Properties.Choices), len(tf.Properties.Choices))
	}

	for j := range f.Properties.Choices {
		tf.Properties.Choices[j].Ref = f.Properties.Choices[j].Ref
	}

	return tf, nil
}

func CopyChoiceRefs(src *Form, dest *Form, skipErrors bool) ([]*trans.Field, error) {
	fields := make([]*trans.Field, len(src.Fields))

	for i, f := range src.Fields {
		field, err := copyChoiceRefs(f, dest)

		if (err != nil) && (skipErrors) {
			fields[i] = f
			continue
		}

		if err != nil {
			return nil, err
		}

		fields[i] = field
	}
	return fields, nil
}

func TranslateForm(src *Form, translated *Form) (*Form, error) {
	// Note: mutates translated

	res := new(Form)

	// Keep logic and hidden fields from source
	res.Logic = src.Logic
	res.Hidden = src.Hidden

	// translate workspace/title/thanksyouscreens directly
	res.Workspace = translated.Workspace
	res.Title = translated.Title
	res.ThankYouScreens = translated.ThankYouScreens

	fields, err := CopyChoiceRefs(src, translated, false)

	if err != nil {
		return nil, err
	}

	res.Fields = fields

	// Note: test w/ default thank you screen, maybe funkiness?
	return res, nil
}
