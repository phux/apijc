package app_test

import (
	"testing"

	"github.com/phux/apijc/app"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestApp_Run(t *testing.T) {
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
		{
			name: "PatternPrefix defined but PatternSuffix not",
			fields: fields{
				BaseDomain: "http://localhost:123",
				NewDomain:  "http://localhost:456",
				URLs: app.URLs{
					Targets: []app.Target{
						{
							RelativePath:       "/foo",
							HTTPMethod:         "GET",
							ExpectedStatusCode: 0,
							PatternPrefix:      stringPointer("{"),
							PatternSuffix:      nil,
						},
					},
				},
			},
			wantErr: app.ErrPrefixFilledButSuffixNot,
		},
		{
			name: "PatternSuffix defined but PatternPrefix not",
			fields: fields{
				BaseDomain: "http://localhost:123",
				NewDomain:  "http://localhost:456",
				URLs: app.URLs{
					Targets: []app.Target{
						{
							RelativePath:       "/foo",
							HTTPMethod:         "GET",
							ExpectedStatusCode: 0,
							PatternPrefix:      nil,
							PatternSuffix:      stringPointer("}"),
						},
					},
				},
			},
			wantErr: app.ErrSuffixFilledButPrefixNot,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			a := app.NewApp(
				tt.fields.BaseDomain,
				tt.fields.NewDomain,
				app.NewURLParser(),
				1.0,
				app.Headers{},
			)
			a.URLs = tt.fields.URLs

			err := a.Run()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
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
		headers    app.Headers
	}
	type args struct {
		httpMethod      string
		requestBody     *string
		requestBodyFile *string
		relativeURL     string
		statusCode      int
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
		{
			name: "Happy Path - global headers are applied",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
				URLs:       app.URLs{},
				Results:    &app.Results{},
				headers: app.Headers{
					Global: map[string]string{"foo": "bar"},
				},
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
			name: "Happy Path - baseDomain headers and newDomain headers are applied",
			fields: fields{
				BaseDomain: "http://localhost:1234",
				NewDomain:  "http://localhost:5678",
				URLs:       app.URLs{},
				Results:    &app.Results{},
				headers: app.Headers{
					Global:     map[string]string{"foo": "bar"},
					BaseDomain: map[string]string{"baseHeader": "baseValue"},
					NewDomain:  map[string]string{"newHeader": "newValue"},
				},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Off()

			expectedBaseURL := tt.fields.BaseDomain + tt.args.relativeURL
			gock.New(expectedBaseURL).
				MatchHeaders(tt.fields.headers.Global).
				MatchHeaders(tt.fields.headers.BaseDomain).
				Reply(tt.mockedHTTPResponses.baseResponse.statusCode).
				JSON(tt.mockedHTTPResponses.baseResponse.body)

			if tt.mockedHTTPResponses.baseResponse.statusCode == 200 {
				expectedNewURL := tt.fields.NewDomain + tt.args.relativeURL
				gock.New(expectedNewURL).
					MatchHeaders(tt.fields.headers.Global).
					MatchHeaders(tt.fields.headers.NewDomain).
					Reply(tt.mockedHTTPResponses.newResponse.statusCode).
					JSON(tt.mockedHTTPResponses.newResponse.body)
			}

			a := app.NewApp(
				tt.fields.BaseDomain,
				tt.fields.NewDomain,
				app.NewURLParser(),
				1000,
				tt.fields.headers,
			)

			checkedPaths, totalPaths, err := a.CheckTarget(
				app.Target{
					RelativePath:       tt.args.relativeURL,
					HTTPMethod:         tt.args.httpMethod,
					ExpectedStatusCode: tt.args.statusCode,
					RequestBody:        tt.args.requestBody,
					RequestBodyFile:    tt.args.requestBodyFile,
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
	str := ""
	tests := []struct {
		name            string
		mockedResponses []mockedResponse
		target          app.Target
	}{
		{
			name: "Single range from 1 to 2",
			mockedResponses: []mockedResponse{
				{
					targetURL:    "/foo/1/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: map[string]string{},
				},
				{
					targetURL:    "/foo/2/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
			},
			target: app.Target{
				RelativePath:       "/foo/{1-2}/bar",
				HTTPMethod:         "GET",
				ExpectedStatusCode: 200,
				RequestBody:        &str,
			},
		},
		{
			name: "Mixed ranges and single values",
			mockedResponses: []mockedResponse{
				{
					targetURL:    "/foo/0/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/1/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/2/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/3/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/5/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/7/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/8/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
				{
					targetURL:    "/foo/9/bar",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
			},
			target: app.Target{
				RelativePath:       "/foo/{0,1-3,5,7-9}/bar",
				HTTPMethod:         "GET",
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
				app.Headers{},
			)

			defer gock.Off()
			for _, mockedResponse := range tt.mockedResponses {
				mockGock(baseDomain, mockedResponse)
				mockGock(newDomain, mockedResponse)
			}

			checkedPaths, countPaths, err := a.CheckTarget(tt.target)

			assert.NoError(t, err)
			assert.Empty(t, a.Results.Findings)
			assert.Equal(t, len(tt.mockedResponses), countPaths)
			assert.Equal(t, len(tt.mockedResponses), checkedPaths)
			assert.True(t, gock.IsDone())
		})
	}
}

type mockedResponse struct {
	targetURL    string
	httpMethod   string
	statusCode   int
	responseBody interface{}
}

func TestRun_WithSequentialTargets(t *testing.T) {
	// str := ""
	tests := []struct {
		name              string
		mockedResponses   []mockedResponse
		sequentialTargets map[string][]app.Target
	}{
		{
			name: "POST and GET",
			mockedResponses: []mockedResponse{
				{
					targetURL:    "/order",
					httpMethod:   "POST",
					statusCode:   201,
					responseBody: map[string]string{},
				},
				{
					targetURL:    "/order-get",
					httpMethod:   "GET",
					statusCode:   200,
					responseBody: `{}`,
				},
			},
			sequentialTargets: map[string][]app.Target{
				"First POST, then GET": {
					{
						RelativePath:       "/order",
						HTTPMethod:         "POST",
						ExpectedStatusCode: 201,
					},
					{
						RelativePath:       "/order-get",
						HTTPMethod:         "GET",
						ExpectedStatusCode: 200,
					},
				},
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
				app.Headers{},
			)
			a.URLs.SequentialTargets = tt.sequentialTargets

			defer gock.Off()
			for _, mockedResponse := range tt.mockedResponses {
				mockGock(baseDomain, mockedResponse)
				mockGock(newDomain, mockedResponse)
			}

			err := a.Run()

			assert.NoError(t, err)
			assert.Empty(t, a.Results.Findings)
			assert.True(t, gock.IsDone())
		})
	}
}

func mockGock(domain string, resp mockedResponse) {
	switch resp.httpMethod {
	case "POST":
		gock.New(domain).
			Post(resp.targetURL).
			Reply(resp.statusCode).
			JSON(resp.responseBody)
	case "GET":
		gock.New(domain).
			Get(resp.targetURL).
			Reply(resp.statusCode).
			JSON(resp.responseBody)
	}
}

func stringPointer(str string) *string {
	return &str
}
