# Upload Typeform

## Installation

First, ensure you have Go installed on your computer.

Then:

``` shell
go install github.com/vlab-research/upload-typeform
```

You will now have an executable on your path (assuming your GOPATH is on your path) called `upload-typeform`.

## Usage

You'll need two environment variables:
``` shell
export TYPEFORM_BASE_URL=https://api.typeform.com
export TYPEFORM_TOKEN=someveryverysecrettoken
```

Then you can use the tools as follows.


### Creating forms


Create forms from all sheets except for "Messages":
``` shell
upload-typeform --workspace "foo" --base "path/to-excel-file.xlsx"
```


Create forms from single sheet
``` shell
upload-typeform --workspace "foo" --base "path/to-excel-file.xlsx" --sheet "Baseline"
```


Update a form that already exists
``` shell
upload-typeform --workspace "foo" --base "path/to-excel-file.xlsx" --sheet "Baseline" --update
```

### Creating translations

Create a new translation from an existing form (NOTE: you give the path to the base excel but the already needed to have created the form in Typeform for this step to work)
``` shell
upload-typeform --workspace "foo" --base "path/to-excel-file.xlsx" --translation "path/to-translation.xlsx"
```

Update a translation!
``` shell
upload-typeform --workspace "foo" --base "path/to-excel-file.xlsx" --translation "path/to-translation.xlsx" --update
```
