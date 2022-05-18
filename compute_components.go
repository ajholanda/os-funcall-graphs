package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"

	g "github.com/aholanda/graphs"
	gio "github.com/aholanda/graphs/io"
)

func listDigraph(d *g.Digraph) {
	vIter := g.NewVertexIterator(d)
	for vIter.HasNext() {
		v := vIter.Value()
		fmt.Printf("%v:", v)

		aIter := g.NewArcIterator(d, v)
		for aIter.HasNext() {
			w := aIter.Value()
			fmt.Printf(" %v", w)
		}
		fmt.Println()
	}
}

func checkDataDirExists(path string) {
	dirInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Fatalf("fatal: data directory \"%s\" does not exist.", path)
	}
	log.Printf("ok> Directory \"%d\" found.\n", dirInfo)
}

func listDataFiles() []string {
	pathS, err := os.Getwd()
	pathS = path.Join(pathS, "data")
	if err != nil {
		panic(err)
	}

	checkDataDirExists(pathS)

	var files []string
	filepath.Walk(pathS, func(fpath string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(GraphFormatExtension, f.Name())
			if err == nil && r {
				filepath := path.Join("data/", f.Name())
				files = append(files, filepath)
			}
		}
		return nil
	})
	return files
}

func extractVersionNumbering(path string) string {
	exp := fmt.Sprintf(`\blinux-(\d+\.\d+\.*\d*\.*\d*)%s`,
		GraphFormatExtension)
	re := regexp.MustCompile(exp)
	ret := re.FindStringSubmatch(path)

	if len(ret) == 0 {
		log.Fatalf("fatal: error finding version number for \"%s\"", path)
	}
	return ret[1]
}

func computeComponents() {
	var digraph *g.Digraph
	var sep string = "\t"
	var resFn = "scc.dat"

	file, err := os.Create(resFn)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	defer log.Printf("> wrote file %s\n", resFn)

	fmt.Fprintf(file, "version_number%s", sep)
	fmt.Fprintf(file, "#vertices%s", sep)
	fmt.Fprintf(file, "#arcs%s", sep)
	fmt.Fprintf(file, "avg_degree%s", sep)
	fmt.Fprintf(file, "std_dev%s", sep)
	fmt.Fprintf(file, "#components%s", sep)
	fmt.Fprintf(file, "largest_component_size%s", sep)
	fmt.Fprintf(file, "\n")

	filenames := listDataFiles()
	for i, fn := range filenames {
		if i%50 != 0 {
			continue
		}
		log.Printf("> reading %v\n", fn)
		digraph = gio.ReadPajek(fn)
		// To reverse the graph and concentrate
		// on the function usage
		//digraph = g.Reverse(digraph)
		scc := g.NewKosarajuSharirSCC(digraph)
		scc.Compute()
		version := extractVersionNumbering(fn)
		fmt.Fprintf(file, "\"%s\"%s", version, sep)
		fmt.Fprintf(file, "%d%s", digraph.V(), sep)
		fmt.Fprintf(file, "%d%s", digraph.A(), sep)
		avgDeg, stdDev := digraph.AverageDegree()
		fmt.Fprintf(file, "%f%s", avgDeg, sep)
		fmt.Fprintf(file, "%f%s", stdDev, sep)
		fmt.Fprintf(file, "%d%s", scc.Count(), sep)
		//fmt.Fprintf(file, "%d%s", scc.GreatestComponentSize(), sep)
		fmt.Fprintf(file, "\n")

	}
}
