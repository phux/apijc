package app_test

import (
	"errors"
	"testing"

	"github.com/phux/apijc/app"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestApp_Run(t *testing.T) {
	t.Parallel()
	type fields struct {
		BaseDomain string
		NewDomain  string
		URLs       app.URLs
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "no targets defined",
			fields: fields{
				BaseDomain: "",
				NewDomain:  "",
				URLs:       app.URLs{},
			},
			wantErr: app.ErrNoTargetsDefined,
		},
		{
			name: "no BaseDomain defined",
			fields: fields{
				BaseDomain: "",
				NewDomain:  "http://localhost:123",
				URLs:       app.URLs{},
			},
			wantErr: app.ErrNoTargetsDefined,
		},
		{
			name: "no NewDomain defined",
			fields: fields{
				BaseDomain: "http://localhost:123",
				NewDomain:  "",
				URLs:       app.URLs{},
			},
			wantErr: app.ErrNoTargetsDefined,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := &app.App{
				BaseDomain: tt.fields.BaseDomain,
				NewDomain:  tt.fields.NewDomain,
				URLs:       tt.fields.URLs,
			}

			if err := a.Run(); err != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("App.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_CheckTarget(t *testing.T) {
	type fields struct {
		BaseDomain string
		NewDomain  string
		URLs       app.URLs
		Results    *app.Results
	}
	type args struct {
		httpMethod  string
		relativeURL string
		statusCode  int
	}
	type httpResponse struct {
		statusCode int
		body       interface{}
	}
	type httpResponses struct {
		baseResponse httpResponse
		newResponse  httpResponse
	}
	type expectedFindings []app.Finding

	tests := []struct {
		name                 string
		fields               fields
		args                 args
		mockedHTTPResponses  httpResponses
		expectedFindings     expectedFindings
		expectedCheckedPaths int
	}{
		{
			name: "Wrong status code for base url",
			fields: fields{
				BaseDomain: "http://localhost:1234",
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 500,
					body:       `{}`,
				},
			},
			expectedFindings: []app.Finding{
				{
					URL:   "http://localhost:1234/foobar",
					Error: "unexpected status code: expected 200, got 500",
					Diff:  "",
				},
			},
			expectedCheckedPaths: 0,
		},
		{
			name: "Wrong status code for new url",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{}`,
				},
				newResponse: httpResponse{
					statusCode: 500,
					body:       `{}`,
				},
			},
			expectedFindings: []app.Finding{
				{
					URL:   "http://localhost:5678/foobar",
					Error: "unexpected status code: expected 200, got 500",
					Diff:  "",
				},
			},
			expectedCheckedPaths: 0,
		},
		{
			name: "Happy Path - matching empty bodies",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{}`,
				},
			},
			expectedFindings:     []app.Finding{},
			expectedCheckedPaths: 1,
		},
		{
			name: "JSON response missmatch 1st level",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": 123}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": 456}`,
				},
			},
			expectedFindings: []app.Finding{
				{
					URL:   "/foobar",
					Error: "JSON mismatch",
					Diff:  "@ [\"foo\"]\n- 123\n+ 456\n",
				},
			},
			expectedCheckedPaths: 1,
		},
		{
			name: "JSON response missmatch 2nd level",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": {"bar": 123}}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": {"bar": "baz"}}`,
				},
			},
			expectedFindings: []app.Finding{
				{
					URL:   "/foobar",
					Error: "JSON mismatch",
					Diff:  "@ [\"foo\",\"bar\"]\n- 123\n+ \"baz\"\n",
				},
			},
			expectedCheckedPaths: 1,
		},
		{
			name: "Happy Path - matching 1 level JSON",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
				URLs:       app.URLs{},
				Results:    &app.Results{},
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": "bar"}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": "bar"}`,
				},
			},
			expectedFindings:     []app.Finding{},
			expectedCheckedPaths: 1,
		},
		{
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
				URLs:       app.URLs{},
				Results:    &app.Results{},
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": "bar"}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": "bar"}`,
				},
			},
			expectedFindings:     []app.Finding{},
			expectedCheckedPaths: 1,
		},
		{
			name: "Happy Path - matching 2 level JSON",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
				URLs:       app.URLs{},
				Results:    &app.Results{},
			},
			args: args{
				httpMethod:  "GET",
				relativeURL: "/foobar",
				statusCode:  200,
			},
			mockedHTTPResponses: httpResponses{
				baseResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": {"bar": 123}}`,
				},
				newResponse: httpResponse{
					statusCode: 200,
					body:       `{"foo": {"bar": 123}}`,
				},
			},
			expectedFindings:     []app.Finding{},
			expectedCheckedPaths: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Off()

			expectedBaseURL := tt.fields.BaseDomain + tt.args.relativeURL
			gock.New(expectedBaseURL).
				Reply(tt.mockedHTTPResponses.baseResponse.statusCode).
				JSON(tt.mockedHTTPResponses.baseResponse.body)

			if tt.mockedHTTPResponses.baseResponse.statusCode == 200 {
				expectedNewURL := tt.fields.NewDomain + tt.args.relativeURL
				gock.New(expectedNewURL).
					Reply(tt.mockedHTTPResponses.newResponse.statusCode).
					JSON(tt.mockedHTTPResponses.newResponse.body)
			}

			a := app.NewApp(
				tt.fields.BaseDomain,
				tt.fields.NewDomain,
				app.NewURLParser(),
				1000,
				nil,
			)

			checkedPaths, totalPaths, err := a.CheckTarget(
				tt.args.httpMethod,
				app.Target{
					RelativePath:       tt.args.relativeURL,
					ExpectedStatusCode: tt.args.statusCode,
				},
			)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCheckedPaths, checkedPaths)
			assert.Equal(t, 1, totalPaths)

			findings := []app.Finding{}
			if a.Results != nil {
				findings = a.Results.Findings
			}
			assert.Len(t, findings, len(tt.expectedFindings))

			for i := range tt.expectedFindings {
				assert.Equal(t, tt.expectedFindings[i].URL, a.Results.Findings[i].URL)
				assert.Equal(t, tt.expectedFindings[i].Error, a.Results.Findings[i].Error)
				assert.Equal(t, tt.expectedFindings[i].Diff, a.Results.Findings[i].Diff)
			}

			assert.True(t, gock.IsDone())
		})
	}
}

func TestCheckTarget_WithRanges(t *testing.T) {
	t.Parallel()
	type mockedResponse struct {
		targetURL    string
		statusCode   int
		responseBody interface{}
	}

	str := ""
	tests := []struct {
		name            string
		mockedResponses []mockedResponse
		target          app.Target
	}{
		{
			name: "Single range from 1 to 5",
			mockedResponses: []mockedResponse{
				{
					targetURL:    "/foo/1/bar",
					statusCode:   200,
					responseBody: map[string]string{},
				},
				{
					targetURL:    "/foo/2/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
			},
			target: app.Target{
				RelativePath:       "/foo/{1-2}/bar",
				ExpectedStatusCode: 200,
				RequestBody:        &str,
			},
		},
		{
			name: "Mixed ranges and single values",
			mockedResponses: []mockedResponse{
				{
					targetURL:    "/foo/0/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/1/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/2/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/3/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/5/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/7/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/8/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/9/bar",
					statusCode:   200,
					responseBody: `{}`,
				},
			},
			target: app.Target{
				RelativePath:       "/foo/{0,1-3,5,7-9}/bar",
				ExpectedStatusCode: 200,
				RequestBody:        &str,
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			baseDomain := "http://localhost:1234"
			newDomain := "http://localhost:5678"
			a := app.NewApp(
				baseDomain,
				newDomain,
				app.NewURLParser(),
				1000,
				nil,
			)

			defer gock.Off()
			for _, mockedResponse := range tt.mockedResponses {
				gock.New(baseDomain).
					Get(mockedResponse.targetURL).
					Reply(200).
					JSON(mockedResponse.responseBody)

				gock.New(newDomain).
					Get(mockedResponse.targetURL).
					Reply(200).
					JSON(mockedResponse.responseBody)
			}

			checkedPaths, countPaths, err := a.CheckTarget("GET", tt.target)

			assert.NoError(t, err)
			assert.Empty(t, a.Results.Findings)
			assert.Equal(t, len(tt.mockedResponses), countPaths)
			assert.Equal(t, len(tt.mockedResponses), checkedPaths)
			assert.True(t, gock.IsDone())
		})
	}
}