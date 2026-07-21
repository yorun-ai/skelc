package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yorun.ai/skelc/internal/util/checkutil"
)

const DomainFileName = "domain.skel"

type SourceFile struct {
	FilePath string
	Content  []byte
}

type Result struct {
	Files    []*SourceFile
	Warnings []string
	IsDir    bool
}

type _Loader struct {
	skelIn string

	directory  bool
	domainFile string
	skelFiles  []string
	warnings   []string
}

func Load(skelIn string) Result {
	sourceLoader := new(_Loader{skelIn: skelIn})
	sourceLoader.discoverFiles()
	return Result{
		Files:    sourceLoader.loadSourceFiles(),
		Warnings: append([]string{}, sourceLoader.warnings...),
		IsDir:    sourceLoader.directory,
	}
}

func classifyFile(fileName string) string {
	switch {
	case strings.HasPrefix(fileName, "."):
		return "hidden"
	case fileName == DomainFileName:
		return "domain"
	case strings.HasSuffix(fileName, ".skel"):
		return "skel"
	default:
		return "other"
	}
}

func isSkelFile(filePath string) bool {
	return strings.HasSuffix(filePath, ".skel")
}

func (l *_Loader) discoverFiles() {
	skelInPath, err := filepath.Abs(l.skelIn)
	checkutil.CheckNilError(err, "resolve skel input failed")
	info, err := os.Stat(skelInPath)
	checkutil.CheckNilError(err, "stat skel input failed")

	l.directory = info.IsDir()
	if !l.directory {
		l.discoverSingleFile(skelInPath)
		return
	}
	l.discoverDirectory(skelInPath)
}

func (l *_Loader) discoverSingleFile(filePath string) {
	checkutil.Check(isSkelFile(filePath), "%s is not a .skel file", filePath)
	l.domainFile = filePath
	l.skelFiles = []string{filePath}
}

func (l *_Loader) discoverDirectory(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	checkutil.CheckNilError(err, "read directory %s", dirPath)

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		fileKind := classifyFile(entry.Name())
		switch {
		case entry.IsDir():
			l.warnings = append(l.warnings, fmt.Sprintf("%s ignored (DIRECTORY)", fullPath))
		case fileKind == "hidden":
			l.warnings = append(l.warnings, fmt.Sprintf("%s ignored (HIDDEN_FILE)", fullPath))
		case fileKind == "domain":
			l.domainFile = fullPath
			l.skelFiles = append(l.skelFiles, fullPath)
		case fileKind == "skel":
			l.skelFiles = append(l.skelFiles, fullPath)
		default:
			l.warnings = append(l.warnings, fmt.Sprintf("%s ignored", fullPath))
		}
	}
	sort.Strings(l.skelFiles)
	checkutil.Check(l.domainFile != "", "%s not found under %s", DomainFileName, dirPath)
}

func (l *_Loader) loadSourceFiles() []*SourceFile {
	files := make([]*SourceFile, 0, len(l.skelFiles))
	for _, skelFile := range l.skelFiles {
		content, err := os.ReadFile(skelFile)
		checkutil.CheckNilError(err, "read %s", skelFile)
		files = append(files, &SourceFile{FilePath: skelFile, Content: content})
	}
	return files
}
