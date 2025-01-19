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

func copyChoiceRefs(f *trans.Field, src *Form) (*trans.Field, error) {
	srcField, err := findField(f.Ref, src)

	if err != nil {
		return nil, err
	}

	if len(f.Properties.Choices) != len(srcField.Properties.Choices) {
		return nil, fmt.Errorf("Number of choices not the same for field ref: %v. There are %d choices in the target and %d choices in the source", f.Ref, len(f.Properties.Choices), len(srcField.Properties.Choices))
	}

	for j := range f.Properties.Choices {
		f.Properties.Choices[j].Ref = srcField.Properties.Choices[j].Ref
	}

	return f, nil
}

func CopyChoiceRefs(src *Form, dest *Form, skipErrors bool) ([]*trans.Field, error) {
	fields := make([]*trans.Field, len(dest.Fields))

	for i, f := range dest.Fields {
		field, err := copyChoiceRefs(f, src)

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

func CheckFields(src *Form, dest *Form) error {
	for _, f := range src.Fields {
		destField, err := findField(f.Ref, dest)
		if err != nil {
			return err
		}

		if len(f.Properties.Choices) != len(destField.Properties.Choices) {
			return fmt.Errorf("Number of choices not the same for field ref: %v. There are %d choices in the source and %d choices in the target", f.Ref, len(f.Properties.Choices), len(destField.Properties.Choices))
		}

	}

	return nil
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

	// copy choice refs from source to translation
	formattedTranslated, err := CopyChoiceRefs(src, translated, true)
	if err != nil {
		return nil, err
	}
	translated.Fields = formattedTranslated

	// Check to make sure fields and choices look the same
	// in both forms
	err = CheckFields(src, translated)
	if err != nil {
		return nil, err
	}
	res.Fields = translated.Fields

	// Note: test w/ default thank you screen, maybe funkiness?
	return res, nil
}
