package app_test

import (
	"testing"

	"github.com/phux/apijc/app"

	"github.com/stretchr/testify/assert"
)

func TestLoadURLsFromFile(t *testing.T) {
	t.Parallel()
	patternPrefix := "{"
	patternSuffix := "}"
	patternPrefix2 := "@"
	patternSuffix2 := "@"
	var nostring *string
	postBody := `{"a":"b"}`

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *app.URLs
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				path: "../.testdata/urlfile_example.json",
			},
			want: &app.URLs{
				Targets: []app.Target{
					{
						RelativePath:       "/v1/example",
						HTTPMethod:         "GET",
						ExpectedStatusCode: 200,
						RequestBody:        nostring,
						PatternPrefix:      &patternPrefix,
						PatternSuffix:      &patternSuffix,
					},
					{
						RelativePath:       "/v1/{1-100}",
						HTTPMethod:         "GET",
						ExpectedStatusCode: 200,
						RequestBody:        nostring,
						PatternPrefix:      &patternPrefix,
						PatternSuffix:      &patternSuffix,
					},
					{
						RelativePath:       "/v1/@1-3@",
						HTTPMethod:         "GET",
						ExpectedStatusCode: 200,
						RequestBody:        nostring,
						PatternPrefix:      &patternPrefix2,
						PatternSuffix:      &patternSuffix2,
					},
					{
						RelativePath:       "/v1/expected_jsonmissmatch",
						HTTPMethod:         "GET",
						ExpectedStatusCode: 200,
						RequestBody:        nostring,
						PatternPrefix:      &patternPrefix,
						PatternSuffix:      &patternSuffix,
					},
					{
						RelativePath:       "/v1/example",
						HTTPMethod:         "POST",
						ExpectedStatusCode: 201,
						RequestHeaders: map[string]string{
							"Content-Type": "application/json",
						},
						RequestBody:   &postBody,
						PatternPrefix: &patternPrefix,
						PatternSuffix: &patternSuffix,
					},
					{
						RelativePath:       "/v1/post_with_body_file",
						HTTPMethod:         "POST",
						ExpectedStatusCode: 201,
						RequestBodyFile:    stringPointer(".testdata/request_body.json"),
					},
				},
				SequentialTargets: map[string][]app.Target{
					"First POST, then GET": {
						{
							RelativePath:       "/v1/sequential_post",
							HTTPMethod:         "POST",
							ExpectedStatusCode: 201,
							RequestBody:        &postBody,
						},
						{
							RelativePath:       "/v1/sequential_get",
							HTTPMethod:         "GET",
							ExpectedStatusCode: 200,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for i := range tests {
		testcase := tests[i]
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			got, err := app.LoadURLsFromFile(testcase.args.path)

			if !testcase.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, testcase.want, got)
		})
	}
}
