package githttp_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/AaronO/go-git-http"
)

func TestRpcReader(t *testing.T) {
	tests := []struct {
		rpc  string
		file string

		want []githttp.Event
	}{
		{
			rpc:  "receive-pack",
			file: "receive-pack.0",

			want: []githttp.Event{
				(githttp.Event)(githttp.Event{
					Type:    (githttp.EventType)(githttp.PUSH),
					Commit:  (string)("92eef6dcb9cc198bc3ac6010c108fa482773f116"),
					Dir:     (string)(""),
					Tag:     (string)(""),
					Last:    (string)("0000000000000000000000000000000000000000"),
					Branch:  (string)("master"),
					Error:   (error)(nil),
					Request: (*http.Request)(nil),
				}),
			},
		},

		{
			rpc:  "upload-pack",
			file: "upload-pack.0",

			want: []githttp.Event{
				(githttp.Event)(githttp.Event{
					Type:    (githttp.EventType)(githttp.FETCH),
					Commit:  (string)("a647ec2ea40ee9ca35d32232dc28de22b1537e00"),
					Dir:     (string)(""),
					Tag:     (string)(""),
					Last:    (string)(""),
					Branch:  (string)(""),
					Error:   (error)(nil),
					Request: (*http.Request)(nil),
				}),
			},
		},

		{
			rpc:  "upload-pack",
			file: "upload-pack.1",

			want: []githttp.Event{
				(githttp.Event)(githttp.Event{
					Type:    (githttp.EventType)(githttp.FETCH),
					Commit:  (string)("92eef6dcb9cc198bc3ac6010c108fa482773f116"),
					Dir:     (string)(""),
					Tag:     (string)(""),
					Last:    (string)(""),
					Branch:  (string)(""),
					Error:   (error)(nil),
					Request: (*http.Request)(nil),
				}),
			},
		},
	}

	for _, tt := range tests {
		f, err := os.Open(filepath.Join("testdata", tt.file))
		if err != nil {
			t.Fatal(err)
		}

		r := f

		rr := &githttp.RpcReader{
			Reader: r,
			Rpc:    tt.rpc,
		}

		_, err = io.Copy(ioutil.Discard, rr)
		if err != nil {
			t.Errorf("io.Copy: %v", err)
		}

		f.Close()

		if got := rr.Events; !reflect.DeepEqual(got, tt.want) {
			t.Errorf("test %q/%q:\n got: %#v\nwant: %#v\n", tt.rpc, tt.file, got, tt.want)
		}
	}
}
