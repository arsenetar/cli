package label

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		output  listOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:   "no argument",
			input:  "",
			output: listOptions{Limit: 30},
		},
		{
			name:   "limit flag",
			input:  "--limit 10",
			output: listOptions{Limit: 10},
		},
		{
			name:    "invalid limit flag",
			input:   "--limit 0",
			wantErr: true,
			errMsg:  "invalid limit: 0",
		},
		{
			name:   "web flag",
			input:  "--web",
			output: listOptions{Limit: 30, WebMode: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			argv, err := shlex.Split(tt.input)
			assert.NoError(t, err)
			var gotOpts *listOptions
			cmd := newCmdList(f, func(opts *listOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantErr {
				assert.EqualError(t, err, tt.errMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.output.Limit, gotOpts.Limit)
			assert.Equal(t, tt.output.WebMode, gotOpts.WebMode)
		})
	}
}

func TestListRun(t *testing.T) {
	tests := []struct {
		name       string
		tty        bool
		opts       *listOptions
		httpStubs  func(*httpmock.Registry)
		wantErr    bool
		wantStdout string
		wantStderr string
	}{
		{
			name: "lists labels",
			tty:  true,
			opts: &listOptions{},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query LabelList\b`),
					httpmock.StringResponse(`
						{
							"data": {
								"repository": {
									"labels": {
										"totalCount": 2,
										"nodes": [
											{
												"name": "bug",
												"color": "d73a4a",
												"description": "This is a bug label"
											},
											{
												"name": "docs",
												"color": "ffa8da",
												"description": "This is a docs label"
											}
										],
										"pageInfo": {
											"hasNextPage": false,
											"endCursor": "Y3Vyc29yOnYyOpK5MjAxOS0xMC0xMVQwMTozODowMyswODowMM5f3HZq"
										}
									}
								}
							}
						}`,
					),
				)
			},
			wantStdout: "\nShowing 2 of 2 labels in OWNER/REPO\n\nbug   This is a bug label   #d73a4a\ndocs  This is a docs label  #ffa8da\n",
		},
		{
			name: "lists labels notty",
			tty:  false,
			opts: &listOptions{},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query LabelList\b`),
					httpmock.StringResponse(`
						{
							"data": {
								"repository": {
									"labels": {
										"totalCount": 2,
										"nodes": [
											{
												"name": "bug",
												"color": "d73a4a",
												"description": "This is a bug label"
											},
											{
												"name": "docs",
												"color": "ffa8da",
												"description": "This is a docs label"
											}
										],
										"pageInfo": {
											"hasNextPage": false,
											"endCursor": "Y3Vyc29yOnYyOpK5MjAxOS0xMC0xMVQwMTozODowMyswODowMM5f3HZq"
										}
									}
								}
							}
						}`,
					),
				)
			},
			wantStdout: "bug\tThis is a bug label\t#d73a4a\ndocs\tThis is a docs label\t#ffa8da\n",
		},
		{
			name: "empty label list",
			tty:  true,
			opts: &listOptions{},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`query LabelList\b`),
					httpmock.StringResponse(`
						{"data":{"repository":{"labels":{"totalCount":0,"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}`,
					),
				)
			},
			wantErr: true,
		},
		{
			name:       "web mode",
			tty:        true,
			opts:       &listOptions{WebMode: true},
			wantStderr: "Opening github.com/OWNER/REPO/labels in your browser.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			if tt.httpStubs != nil {
				tt.httpStubs(reg)
			}
			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			tt.opts.IO = ios
			tt.opts.Browser = &cmdutil.TestBrowser{}
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.New("OWNER", "REPO"), nil
			}
			defer reg.Verify(t)
			err := listRun(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}
