package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	g "github.com/aholanda/graphs"
	gio "github.com/aholanda/graphs/io"
)

const (
	// GraphFormatExtension is the suffix used in the files
	// containing graph description, e.g., ".dot", ".net".
	GraphFormatExtension string = gio.PajekFormatExtension
	// GraphDataRelativePath is the relative path in the current
	// directory where the main program is running. Inside it data with
	// graph description are saved.
	GraphDataRelativePath   string = "data"
	termBrowser             string = "`which lynx` -dump "
	downloader              string = "`which wget`"
	compressedFileExtension string = ".tar.xz"
)

func check(e error) {
	if e != nil {
		log.Fatalf("failed with %s", e)
	}
}

func extractFilePrefixFromFileURL(fileURL string) string {
	return strings.TrimRight(filepath.Base(fileURL),
		compressedFileExtension)
}

func listRemoteDir(url string, filter string) []string {
	cmdStr := termBrowser + url + " " + filter
	log.Println(cmdStr)
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()

	// TODO: check error due broken connection
	check(err)
	return strings.Fields(string(out))
}

func downloadFile(filepath string, fileURL string) {
	// Get the data
	resp, err := http.Get(fileURL)
	check(err)
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	check(err)
	defer out.Close()

	log.Println("downloading", filepath)
	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	check(err)
}

func unpackFile(file string) string {
	// Get the temporary absolute path where the file
	// is unpacked.
	dir, err := filepath.Abs(filepath.Dir(file))
	check(err)

	// Unpack the file
	cmdStr := "`which tar` xfJ " + file + " -C " + dir
	log.Println(cmdStr)
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	err = cmd.Run()
	check(err)

	// Get the name of the program with the version appended as suffix.
	versionName := extractFilePrefixFromFileURL(file)
	progName := strings.Split(versionName, "-")[0]

	// Fix when directory name is without the version as suffix.
	// e.g. $ [ -d /tmp/linux1234/linux ] \
	// && mv /tmp/linux1234/linux /tmp/linux1234/linux-v1.0
	cmdStr = fmt.Sprintf("[ -d  %s ] && mv %s %s",
		path.Join(dir, progName),
		path.Join(dir, progName),
		path.Join(dir, versionName))
	log.Println(cmdStr)
	cmd = exec.Command("/bin/bash", "-c", cmdStr)
	err = cmd.Run()
	// Ignore error here because when the the version name
	// is ok, contains the version number appended, the
	// bash command returns non-zero, that is interpreted as
	// error.

	return versionName
}

func addVertex(v2Adj map[string][]string, vertex string) map[string][]string {
	if _, ok := v2Adj[vertex]; !ok {
		v2Adj[vertex] = []string{}
	}
	return v2Adj
}

func addAdjVertex(v2Adj map[string][]string, from, to string) map[string][]string {
	if _, ok := v2Adj[from]; ok {
		v2Adj[from] = append(v2Adj[from], to)
	} else {
		v2Adj[from] = []string{to}
	}

	return v2Adj
}

func buildGraph(v2Adj map[string][]string) *g.Digraph {
	var digraph = g.NewDigraph(len(v2Adj))
	var vcount g.VertexID

	// The digraph is built in two passes. In the first
	// the vertices' names are indexed and in the second
	// the adjacencies are assigned.

	// First pass: vertices' names
	vcount = 0
	for key := range v2Adj {
		digraph.NameVertex(vcount, key)
		vcount++
	}

	// Second pass: adjacencies
	for from, adj := range v2Adj {
		v, err := digraph.VertexIndex(from)
		if err != nil {
			log.Fatalf("%v", err)
		}
		for _, to := range adj {
			w, err := digraph.VertexIndex(to)
			if err != nil {
				log.Fatalf("%v", err)
			}
			digraph.AddArc(v, w)
		}
	}
	return digraph
}

func createGraphFromCflowsOutput(dir string) *g.Digraph {
	// Map to accumulate vertices and its adjacent lists before
	// adding to graph, this procedure is needed because the
	// number of vertices is not know at first sight
	var vertexToAdj map[string][]string = make(map[string][]string)
	// Levels of indentation in the cflow output;
	// "0": first level, refers to the function caller
	// "1": second level, refers to the function callee
	var indentLevels [2]string = [2]string{"0", "1"}

	// List all C files recursivelly in the specified directory.
	cmdStr := fmt.Sprintf("/usr/bin/find %s -name \\*.c", dir)
	log.Println(cmdStr)
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	check(err)

	cfiles := strings.Fields(string(out))
	for _, cfile := range cfiles {
		// Current function caller during the call flow
		var curCaller string = ""

		// Run cflows on C files to get the function caller
		// at the first level (no indent) and the functions
		// callees at the second level (indented).
		cmdStr := fmt.Sprintf("/usr/bin/cflow --depth 2 "+
			" --omit-arguments --print-level %s", cfile)
		log.Println(cmdStr)
		cmd := exec.Command("/bin/bash", "-c", cmdStr)
		out, err = cmd.CombinedOutput()
		check(err)

		cflowOutLines := strings.Split(string(out), "\n")
	NEXT_LINE:
		for _, line := range cflowOutLines {
			for l, level := range indentLevels {
				exp := fmt.Sprintf(`^\{\s+%s\}.+`, level)
				re := regexp.MustCompile(exp)
				matched := re.MatchString(line)
				if matched {
					exp = fmt.Sprintf(`\{\s+%s\}\s+(?P<function>\w+)\(\)`+
						`\s*(?P<rest>.*)`, level)
					re = regexp.MustCompile(exp)
					ret := re.FindStringSubmatch(line)
					if len(ret) <= 1 { // problems with parsing
						continue NEXT_LINE
					}
					funcName := ret[1]
					// Sometimes a function performs no function call, so we
					// add to hashmap to be counted as vertex after.
					vertexToAdj = addVertex(vertexToAdj, funcName)
					if l == 0 { // level = 0 -> function caller
						curCaller = funcName
					} else { // level = 1 -> function callee
						vertexToAdj = addAdjVertex(vertexToAdj, curCaller, funcName)
					}
					//fmt.Println(level, ret[1])
					break
				}
			}
		}
	}
	return buildGraph(vertexToAdj)
}

func mkTmpDataDir(p *program) string {
	// Set the temporary directory to put downloaded
	// compressed and unpacked files. The files are
	// saved at home directory plus "tmp" to avoid
	// permission and space problems.
	homeDir, err := os.UserHomeDir()
	check(err)
	tmpDir := path.Join(homeDir, "tmp")
	check(err)
	// Create $HOME/tmp if it does not exist
	if _, err = os.Stat(tmpDir); os.IsNotExist(err) {
		err = os.Mkdir(tmpDir, 0755)
		check(err)
	}
	// Append specific information about what's being
	// saved
	tmpDataDir, err := ioutil.TempDir(tmpDir, p.dirPrefix)
	if _, err = os.Stat(tmpDataDir); os.IsNotExist(err) {
		err = os.Mkdir(tmpDataDir, 0755)
		check(err)
	}
	log.Printf("> temporary directory: %s\n", tmpDataDir)
	return tmpDataDir
}

func genData(p *program, version, tmpDataDir string) {
	// Append the program version to its base URL
	// to have access to the files.
	url := p.baseURL + "/" + version

	// List existing remote files reached by url.
	remoteFiles := listRemoteDir(url, p.listFilter)

	for _, remFile := range remoteFiles {
		if alreadyHasData(remFile) == true {
			continue
		}

		// Construct the file path to write the downloaded data.
		filepath := path.Join(tmpDataDir, path.Base(remFile))
		downloadFile(filepath, remFile)

		// Dont check error of unpack file due
		// some problems with shell commando to rename
		// linux files without version.
		// When there is a version
		// sometimes the command returns an error.
		versionName := unpackFile(filepath)

		digraph := createGraphFromCflowsOutput(path.Join(tmpDataDir, versionName))
		digraph.NameIt(versionName)
		fn := path.Join("data", versionName+GraphFormatExtension)
		gio.WritePajek(digraph, fn)
	}
}

func buildDataPath(filePrefix string) string {
	return path.Join(GraphDataRelativePath, filePrefix+
		GraphFormatExtension)
}

func alreadyHasData(fileURL string) bool {
	filePrefix := extractFilePrefixFromFileURL(fileURL)
	dataPath := buildDataPath(filePrefix)

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return false
	}
	log.Printf("Data file %s already exists.\n", dataPath)
	return true
}

func cleanTmpFiles(tmpDir string) {
	// Clean the linux files
	cmd := exec.Command("/bin/bash", "-c", "rm -rf "+tmpDir)
	err := cmd.Run()
	check(err)
	log.Printf(" $ rm -rf %s\n", tmpDir)
}

func generateData(p *program) {
	// Check if the needed programs are installed
	cmds := [...]string{"cflow", "lynx", "wget", "tar"}
	for _, cmd := range cmds {
		path, err := exec.LookPath(cmd)
		if err != nil {
			log.Fatalf("%v: please install %s\n", err, cmd)
		}
		log.Printf("ok> found %s at path %s\n", cmd, path)
	}

	for _, v := range p.versions {
		tmpDataDir := mkTmpDataDir(p)
		genData(p, v, tmpDataDir)
		cleanTmpFiles(tmpDataDir)
	}
}
