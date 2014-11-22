package auth

import (
	"net/http"
	"regexp"
	"strings"
)

type AuthInfo struct {
	// Usernane or email
	Username string
	// Plaintext password or token
	Password string

	// repo component of URL
	// Usually: "username/repo_name"
	// But could also be: "some_repo.git"
	Repo string

	// Are we pushing or fetching ?
	Push  bool
	Fetch bool
}

var (
	repoNameRegex = regexp.MustCompile("^/?(.*?)/(HEAD|git-upload-pack|git-receive-pack|info/refs|objects/.*)$")
)

func Authenticator(authf func(AuthInfo) (bool, error)) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			auth, err := parseAuthHeader(req.Header.Get("Authorization"))
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="git server"`)
				http.Error(w, err.Error(), 401)
				return
			}

			// Build up info from request headers and URL
			info := AuthInfo{
				Username: auth.Name,
				Password: auth.Pass,
				Repo:     repoName(req.URL.Path),
				Push:     isPush(req),
				Fetch:    isFetch(req),
			}

			// Call authentication function
			authenticated, err := authf(info)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			// Deny access to repo
			if !authenticated {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Forbidden"))
				return
			}

			// Access granted
			handler.ServeHTTP(w, req)
		})
	}
}

func isFetch(req *http.Request) bool {
	return isService("upload-pack", req)
}

func isPush(req *http.Request) bool {
	return isService("receive-pack", req)
}

func isService(service string, req *http.Request) bool {
	return getServiceType(req) == service || strings.HasSuffix(req.URL.Path, service)
}

func repoName(urlPath string) string {
	matches := repoNameRegex.FindStringSubmatch(urlPath)
	if matches == nil {
		return ""
	}
	return matches[1]
}

func getServiceType(r *http.Request) string {
	service_type := r.FormValue("service")

	if s := strings.HasPrefix(service_type, "git-"); !s {
		return ""
	}

	return strings.Replace(service_type, "git-", "", 1)
}
