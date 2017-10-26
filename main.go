package main

import (
	"encoding/json"
	"flag"
	"github.com/go-openapi/spec"
	"github.com/jackmanlabs/errors"
	"go/build"
	"log"
	"os"
	"runtime/pprof"
	"strings"
)

var (
	// Command-line parameters
	pkgPath     *string = flag.String("pkg", "", "The main package of your application.")
	profilePath *string = flag.String("profile", "", "The path where you'd like to store profiling results.")
	ignore      *string = flag.String("ignore", "", "The comma seperated package paths that you want to ignore.")
	naming      *string = flag.String("naming", "full", "One of 'full', 'partial', or 'simple' to describe the amount of the package path on the resulting JSON models.")
)

var (
	// Global variables
	// Normally, I don't like global variables. The fact is, however, that if we
	// were to pass these three things around, it would get very tedious very
	// fast. This is not a multi-threaded program, and we've been careful to
	// avoid modifying maps during iterations.
	definitionStore DefinitionStore        = make(map[string]*DefinitionIntermediate)
	pkgInfos        map[string]PackageInfo = make(map[string]PackageInfo)
	srcPath         string
	ignoredPackages []string = make([]string, 0)
)

func main() {
	flag.Parse()

	if *profilePath != "" {
		f, err := os.Create(*profilePath)
		if err != nil {
			log.Fatal(errors.Stack(err))
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *pkgPath == "" {
		flag.Usage()
		log.Fatal("Package path is required.")
	}

	if !(*naming == "full" || *naming == "partial" || *naming == "simple") {
		flag.Usage()
		log.Fatal("Unrecognized value provided for naming convention: " + *naming)
	}

	ignores := strings.Split(*ignore, ",")
	for _, i := range ignores {
		if i != "" {
			ignoredPackages = append(ignoredPackages, i)
		}
	}

	var err error

	// Determine the source path of the package specified.
	srcPath, err = getPackageSourceDir(*pkgPath)
	if err != nil {
		log.Fatal(errors.Stack(err))
	}

	// Which packages need to be analyzed? Get a list of all pkgInfos.
	pkgInfos, err = getPackageInfoRecursive(*pkgPath)
	if err != nil {
		log.Fatal(errors.Stack(err))
	}

	// What pkgComments need to be parsed?
	// Find all pkgComments with keywords.
	pkgComments := make(map[string][]string, 0)
	for importPath := range pkgInfos {
		newBlocks, err := getRelevantComments(importPath)
		if err != nil {
			log.Fatal(errors.Stack(err))
		}

		pkgComments[importPath] = newBlocks
	}

	// Now, we need to organize the pkgComments and parse them.

	apiComments := make([]string, 0)
	for _, commentBlocks := range pkgComments {
		newApiComments := extractApiComments(commentBlocks)
		apiComments = append(apiComments, newApiComments...)
	}
	apiIntermediate := intermediatateApi(apiComments)

	// We need to know the package so we know where to look for the types.
	operationPkgComments := make(map[string][]string)
	for importPath, comments := range pkgComments {
		operationPkgComments[importPath] = extractOperationComments(comments)
	}

	operationIntermediates := make([]OperationIntermediate, 0)
	for importPath, commentBlocks := range operationPkgComments {
		for _, commentBlock := range commentBlocks {
			operationIntermediate := intermediatateOperation(commentBlock)
			operationIntermediate.PackagePath = importPath
			operationIntermediates = append(operationIntermediates, operationIntermediate)
		}
	}

	operationIntermediates = tagOperations(apiIntermediate, operationIntermediates)

	err = deriveDefinitionsFromOperations(operationIntermediates)
	if err != nil {
		log.Fatal(errors.Stack(err))
	}

	// Transform the extractions above and combine them into a single Swagger Spec.

	swagger := swaggerizeApi(apiIntermediate)
	pathItems := swaggerizeOperations(operationIntermediates)
	definitions := swaggerizeDefinitions()

	swagger.Paths = &spec.Paths{
		Paths: pathItems,
	}

	swagger.Definitions = definitions

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	err = enc.Encode(swagger)
	if err != nil {
		log.Fatal(errors.Stack(err))
	}
}

func getPackageSourceDir(pkgPath string) (string, error) {

	var (
		bpkg *build.Package
		err  error
	)

	// I should hope there's an easier way of resolving the source path.
	srcDirs := build.Default.SrcDirs()
	for _, srcDir := range srcDirs {
		bpkg, err = build.Import(pkgPath, srcDir, 0)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", errors.Stack(err)
	}

	return bpkg.Dir, nil
}
