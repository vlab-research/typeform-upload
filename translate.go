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

func TranslateForm(src *Form, translated *Form) (*Form, error) {
	// Copy over logic and hidden fields from source
	translated.Logic = src.Logic
	translated.Hidden = src.Hidden

	// Copy of the choice Refs form the source (not specified)
	for _, f := range translated.Fields {
		tf, err := findField(f.Ref, src)

		if err != nil {
			return nil, err
		}

		if len(f.Properties.Choices) != len(tf.Properties.Choices) {
			fmt.Println(f.Properties.Choices)

			return nil, fmt.Errorf("Number of choices not the same for field ref: %v. There are %d choices in the target and %d choices in the source", f.Ref, len(f.Properties.Choices), len(tf.Properties.Choices))
		}

		for j := range f.Properties.Choices {
			f.Properties.Choices[j].Ref = tf.Properties.Choices[j].Ref
		}
	}

	// Note: test w/ default thank you screen, maybe funkiness?
	return translated, nil
}
