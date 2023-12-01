package app_test

import (
	"testing"

	"github.com/phux/apijc/app"
	"github.com/stretchr/testify/assert"
)

func TestParser_ParsePath(t *testing.T) {
	t.Parallel()
	type args struct {
		rawPath string
		opts    app.Options
	}
	tests := []struct {
		name        string
		args        args
		want        []string
		errorString string
	}{
		{
			name: "No pattern found",
			args: args{
				rawPath: "/foo/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{"/foo/bar"},
		},
		{
			name: "2 standalone comma-separated values",
			args: args{
				rawPath: "{1,2}",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{"1", "2"},
		},
		{
			name: "2 standalone comma-separated values with custom prefix and suffix",
			args: args{
				rawPath: "%1,2%",
				opts: app.Options{
					PatternPrefix: "%",
					PatternSuffix: "%",
				},
			},
			want: []string{"1", "2"},
		},
		{
			name: "2 standalone comma-separated values with empty prefix and suffix",
			args: args{
				rawPath: "1,2",
				opts: app.Options{
					PatternPrefix: "",
					PatternSuffix: "",
				},
			},
			want:        []string{},
			errorString: "ParsePath: {PatternPrefix: PatternSuffix:}: PatternPrefix and PatternSuffix cannot be empty",
		},
		{
			name: "2 comma-separated values",
			args: args{
				rawPath: "/foo/{1,2}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{"/foo/1/bar", "/foo/2/bar"},
		},
		{
			name: "2 patterns with 2 comma-separated values",
			args: args{
				rawPath: "/foo/{1,2}/{a,b}",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/1/a",
				"/foo/1/b",
				"/foo/2/a",
				"/foo/2/b",
			},
		},
		{
			name: "1 comma-separated range",
			args: args{
				rawPath: "/foo/{1-20}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/1/bar",
				"/foo/2/bar",
				"/foo/3/bar",
				"/foo/4/bar",
				"/foo/5/bar",
				"/foo/6/bar",
				"/foo/7/bar",
				"/foo/8/bar",
				"/foo/9/bar",
				"/foo/10/bar",
				"/foo/11/bar",
				"/foo/12/bar",
				"/foo/13/bar",
				"/foo/14/bar",
				"/foo/15/bar",
				"/foo/16/bar",
				"/foo/17/bar",
				"/foo/18/bar",
				"/foo/19/bar",
				"/foo/20/bar",
			},
		},
		{
			name: "2 comma-separated ranges",
			args: args{
				rawPath: "/foo/{1-2,3-5}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/1/bar",
				"/foo/2/bar",
				"/foo/3/bar",
				"/foo/4/bar",
				"/foo/5/bar",
			},
		},
		{
			name: "Single value and a comma-separated range",
			args: args{
				rawPath: "/foo/{0,1-2}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/0/bar",
				"/foo/1/bar",
				"/foo/2/bar",
			},
		},
		{
			name: "A comma-separated range and a single value",
			args: args{
				rawPath: "/foo/{1-2,0}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/1/bar",
				"/foo/2/bar",
				"/foo/0/bar",
			},
		},
		{
			name: "Mixed comma-separated ranges and single values",
			args: args{
				rawPath: "/foo/{0,1-3,5,7-9}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/0/bar",
				"/foo/1/bar",
				"/foo/2/bar",
				"/foo/3/bar",
				"/foo/5/bar",
				"/foo/7/bar",
				"/foo/8/bar",
				"/foo/9/bar",
			},
		},
		{
			name: "Range first part is not numerical",
			args: args{
				rawPath: "/foo/{a-2}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want:        []string{},
			errorString: "\"a-2\": first number: not a valid number range",
		},
		{
			name: "Range second part is not numerical",
			args: args{
				rawPath: "/foo/{1-b}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want:        []string{},
			errorString: "\"1-b\": second number: not a valid number range",
		},
		{
			name: "Range first part is bigger than second part",
			args: args{
				rawPath: "/foo/{2-1}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want:        []string{},
			errorString: "\"2-1\": first number cannot be bigger than second number: not a valid number range",
		},
		{
			name: "Range too many ranges",
			args: args{
				rawPath: "/foo/{1-2-5}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want:        []string{},
			errorString: "\"1-2-5\": number of elements != 2, is 3: invalid number range",
		},
		{
			name: "Range negative first number",
			args: args{
				rawPath: "/foo/{-2-0}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/-2/bar",
				"/foo/-1/bar",
				"/foo/0/bar",
			},
		},
		{
			name: "Range negative both numbers",
			args: args{
				rawPath: "/foo/{-2--1}/bar",
				opts: app.Options{
					PatternPrefix: "{",
					PatternSuffix: "}",
				},
			},
			want: []string{
				"/foo/-2/bar",
				"/foo/-1/bar",
			},
		},
	}

	for i := range tests {
		tt := tests[i]

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := app.Parser{}

			got, err := p.ParsePath(tt.args.rawPath, tt.args.opts)

			assert.Equal(t, tt.want, got)
			if tt.errorString != "" {
				assert.EqualError(t, err, tt.errorString)
			}
		})
	}
}
