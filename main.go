package main

import "flag"

type program struct {
	baseURL    string
	versions   []string
	listFilter string
	dirPrefix  string
}

var linux = &program{
	baseURL: "https://mirrors.edge.kernel.org/pub/linux/kernel",
	versions: []string{"v1.1", "v1.2", "v1.3", "v2.0", "v2.2", "v2.3",
		"v2.4", "v2.5", "v2.6", "v3.0", "v3.x", "v4.x", "v5.x", "v6.x"},
	listFilter: " | grep tar.xz | grep https | grep -v bdflush" +
		"| grep -vi changelog | grep -vi modules | grep -v patches " +
		"| grep -v v1.1.0 | grep -vi drm | grep -vi dontuse | grep -vi pre " +
		"| grep -vi badsig | awk '{print $2}'",
	dirPrefix: "linux",
}

func main() {
	dataFlag := flag.Bool("data", false,
		"generate data (files with graph description inside data/)")
	flag.Parse()

	if *dataFlag == true {
		generateData(linux)
	} else {
		computeComponents()
	}
}
