package githttp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type GitHttp struct {
	// Root directory to serve repos from
	ProjectRoot string

	// Path to git binary
	GitBinPath string

	// Access rules
	UploadPack  bool
	ReceivePack bool

	// Event handling functions
	EventHandler func(ev Event)
}

// Implement the http.Handler interface
func (g *GitHttp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.requestHandler(w, r)
	return
}

// Shorthand constructor for most common scenario
func New(root string) *GitHttp {
	return &GitHttp{
		ProjectRoot: root,
		GitBinPath:  "/usr/bin/git",
		UploadPack:  true,
		ReceivePack: true,
	}
}

type Service struct {
	Method  string
	Handler func(HandlerReq)
	Rpc     string
}

type HandlerReq struct {
	w    http.ResponseWriter
	r    *http.Request
	Rpc  string
	Dir  string
	File string
}

func (g *GitHttp) services() map[string]Service {
	return map[string]Service{
		"(.*?)/git-upload-pack$":                       Service{"POST", g.serviceRpc, "upload-pack"},
		"(.*?)/git-receive-pack$":                      Service{"POST", g.serviceRpc, "receive-pack"},
		"(.*?)/info/refs$":                             Service{"GET", g.getInfoRefs, ""},
		"(.*?)/HEAD$":                                  Service{"GET", g.getTextFile, ""},
		"(.*?)/objects/info/alternates$":               Service{"GET", g.getTextFile, ""},
		"(.*?)/objects/info/http-alternates$":          Service{"GET", g.getTextFile, ""},
		"(.*?)/objects/info/packs$":                    Service{"GET", g.getInfoPacks, ""},
		"(.*?)/objects/info/[^/]*$":                    Service{"GET", g.getTextFile, ""},
		"(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$":      Service{"GET", g.getLooseObject, ""},
		"(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$": Service{"GET", g.getPackFile, ""},
		"(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$":  Service{"GET", g.getIdxFile, ""},
	}
}

// Request handling function

func (g *GitHttp) requestHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, r.Proto)
	for match, service := range g.services() {
		// Ensure that regex mathces
		re, err := regexp.Compile(match)
		if err != nil {
			log.Print(err)
		}

		if m := re.FindStringSubmatch(r.URL.Path); m != nil {
			if service.Method != r.Method {
				renderMethodNotAllowed(w, r)
				return
			}

			rpc := service.Rpc
			// Get specific file
			file := strings.Replace(r.URL.Path, m[1]+"/", "", 1)
			// Resolve directory
			dir, err := g.getGitDir(m[1])

			if err != nil {
				log.Print(err)
				renderNotFound(w)
				return
			}

			hr := HandlerReq{w, r, rpc, dir, file}
			service.Handler(hr)
			return
		}
	}
	renderNotFound(w)
	return
}

// Regexes to detect types of actions (fetch, push, etc ...)
var (
	receivePackRegex = regexp.MustCompile("([0-9a-fA-F]{40}) ([0-9a-fA-F]{40}) refs\\/(heads|tags)\\/(.*?)( |00|\u0000)|^(0000)$")
	uploadPackRegex  = regexp.MustCompile("^\\S+ ([0-9a-fA-F]{40})")
)

// Publish event if EventHandler is set
func (g *GitHttp) event(e Event) {
	if g.EventHandler != nil {
		g.EventHandler(e)
	} else {
		fmt.Printf("EVENT: %q\n", e)
	}
}

// Actual command handling functions

func (g *GitHttp) serviceRpc(hr HandlerReq) {
	w, r, rpc, dir := hr.w, hr.r, hr.Rpc, hr.Dir
	access := g.hasAccess(r, dir, rpc, true)

	if access == false {
		renderNoAccess(w)
		return
	}

	reader, err := requestReader(r)
	if err != nil {
		fmt.Printf("Error getting reader: %s\n", err)
		return
	}

	input, _ := ioutil.ReadAll(reader)

	if rpc == "upload-pack" {
		matches := uploadPackRegex.FindAllStringSubmatch(string(input[:]), -1)
		if matches != nil {
			for _, m := range matches {
				g.event(Event{
					Dir:    dir,
					Type:   FETCH,
					Commit: m[1],
				})
			}
		}
	} else if rpc == "receive-pack" {
		matches := receivePackRegex.FindAllStringSubmatch(string(input[:]), -1)
		if matches != nil {
			for _, m := range matches {
				e := Event{
					Dir:    dir,
					Last:   m[1],
					Commit: m[2],
				}

				// Handle pushes to branches and tags differently
				if m[3] == "heads" {
					e.Type = PUSH
					e.Branch = m[4]
				} else {
					e.Type = TAG
					e.Tag = m[4]
				}

				g.event(e)
			}
		}
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	w.WriteHeader(http.StatusOK)

	args := []string{rpc, "--stateless-rpc", dir}
	cmd := exec.Command(g.GitBinPath, args...)
	cmd.Dir = dir
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Print(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Print(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Print(err)
	}

	in.Write(input)
	io.Copy(w, stdout)
	cmd.Wait()
}

func (g *GitHttp) getInfoRefs(hr HandlerReq) {
	w, r, dir := hr.w, hr.r, hr.Dir
	service_name := getServiceType(r)
	access := g.hasAccess(r, dir, service_name, false)

	if access {
		args := []string{service_name, "--stateless-rpc", "--advertise-refs", "."}
		refs := g.gitCommand(dir, args...)

		hdrNocache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service_name))
		w.WriteHeader(http.StatusOK)
		w.Write(packetWrite("# service=git-" + service_name + "\n"))
		w.Write(packetFlush())
		w.Write(refs)
	} else {
		g.updateServerInfo(dir)
		hdrNocache(w)
		sendFile("text/plain; charset=utf-8", hr)
	}
}

func (g *GitHttp) getInfoPacks(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("text/plain; charset=utf-8", hr)
}

func (g *GitHttp) getLooseObject(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-loose-object", hr)
}

func (g *GitHttp) getPackFile(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-packed-objects", hr)
}

func (g *GitHttp) getIdxFile(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-packed-objects-toc", hr)
}

func (g *GitHttp) getTextFile(hr HandlerReq) {
	hdrNocache(hr.w)
	sendFile("text/plain", hr)
}

// Logic helping functions

func sendFile(content_type string, hr HandlerReq) {
	w, r := hr.w, hr.r
	req_file := path.Join(hr.Dir, hr.File)

	f, err := os.Stat(req_file)
	if os.IsNotExist(err) {
		renderNotFound(w)
		return
	}

	w.Header().Set("Content-Type", content_type)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Size()))
	w.Header().Set("Last-Modified", f.ModTime().Format(http.TimeFormat))
	http.ServeFile(w, r, req_file)
}

func (g *GitHttp) getGitDir(file_path string) (string, error) {
	root := g.ProjectRoot

	if root == "" {
		cwd, err := os.Getwd()

		if err != nil {
			log.Print(err)
			return "", err
		}

		root = cwd
	}

	f := path.Join(root, file_path)
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return "", err
	}

	return f, nil
}

func (g *GitHttp) hasAccess(r *http.Request, dir string, rpc string, check_content_type bool) bool {
	if check_content_type {
		if r.Header.Get("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
			return false
		}
	}

	if !(rpc == "upload-pack" || rpc == "receive-pack") {
		return false
	}
	if rpc == "receive-pack" {
		return g.ReceivePack
	}
	if rpc == "upload-pack" {
		return g.UploadPack
	}

	return g.getConfigSetting(rpc, dir)
}

func (g *GitHttp) getConfigSetting(service_name string, dir string) bool {
	service_name = strings.Replace(service_name, "-", "", -1)
	setting := g.getGitConfig("http."+service_name, dir)

	if service_name == "uploadpack" {
		return setting != "false"
	}

	return setting == "true"
}

func (g *GitHttp) getGitConfig(config_name string, dir string) string {
	args := []string{"config", config_name}
	out := string(g.gitCommand(dir, args...))
	return out[0 : len(out)-1]
}

func (g *GitHttp) updateServerInfo(dir string) []byte {
	args := []string{"update-server-info"}
	return g.gitCommand(dir, args...)
}

func (g *GitHttp) gitCommand(dir string, args ...string) []byte {
	command := exec.Command(g.GitBinPath, args...)
	command.Dir = dir
	out, err := command.Output()

	if err != nil {
		log.Print(err)
	}

	return out
}
