package githttp

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
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

	// Reader that decompresses if necessary
	reader, err := requestReader(r)
	if err != nil {
		fmt.Printf("Error getting reader: %s\n", err)
		return
	}

	// Reader that scans for events
	rpcReader := &RpcReader{
		ReadCloser: reader,
		Rpc:        rpc,
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	w.WriteHeader(http.StatusOK)

	args := []string{rpc, "--stateless-rpc", dir}
	cmd := exec.Command(g.GitBinPath, args...)
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
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

	// Copy input to git binary
	io.Copy(stdin, rpcReader)

	// Write git binary's output to http response
	io.Copy(w, stdout)

	// Wait till command has completed
	cmd.Wait()

	// Fire events
	for _, e := range rpcReader.Events {
		// Set directory to current repo
		e.Dir = dir

		// Fire event
		g.event(e)
	}
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
