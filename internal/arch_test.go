package internal_test

import (
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// internalPackages dynamically discovers all internal subpackages (excluding
// domain and the root internal package) and returns them as full module paths.
func internalPackages(t *testing.T) []string {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedName}
	pkgs, err := packages.Load(cfg, "github.com/Cyclone1070/iav/internal/...")
	if err != nil {
		t.Fatalf("failed to load internal packages: %v", err)
	}
	const domain = "github.com/Cyclone1070/iav/internal/domain"
	const root = "github.com/Cyclone1070/iav/internal"
	var result []string
	for _, pkg := range pkgs {
		if pkg.PkgPath == root || pkg.PkgPath == domain {
			continue
		}
		result = append(result, pkg.PkgPath)
	}
	return result
}

func TestArchitecture_Dependencies(t *testing.T) {
	const modulePrefix = "github.com/Cyclone1070/iav/"
	const internalPrefix = modulePrefix + "internal/"
	const domainPackage = internalPrefix + "domain"

	for _, pkgPath := range internalPackages(t) {
		t.Run(pkgPath, func(t *testing.T) {
			cfg := &packages.Config{
				Mode:  packages.NeedImports,
				Tests: true,
			}
			pkgs, err := packages.Load(cfg, pkgPath)
			if err != nil {
				t.Fatalf("failed to load package %s: %v", pkgPath, err)
			}

			for _, pkg := range pkgs {
				for imp := range pkg.Imports {
					verifyImport(t, pkg.PkgPath, imp, internalPrefix, domainPackage, modulePrefix)
				}
			}
		})
	}
}

func verifyImport(t *testing.T, fromPkg, imp, internalPrefix, domainPackage, modulePrefix string) {
	t.Helper()
	// Allow importing the domain package
	if imp == domainPackage {
		return
	}
	// Allow test packages/executables to import their own package under test
	if fromPkg == "" || fromPkg == "command-line-arguments" || strings.HasSuffix(fromPkg, ".test") || fromPkg == imp || strings.HasPrefix(fromPkg, imp+".") || fromPkg == imp+"_test" {
		return
	}
	if strings.HasPrefix(imp, internalPrefix) {
		t.Errorf("Package %s is not allowed to import internal package %q. Only %q is allowed.", fromPkg, imp, domainPackage)
	}
	if strings.HasPrefix(imp, modulePrefix+"cmd") {
		t.Errorf("Package %s is not allowed to import cmd package %q.", fromPkg, imp)
	}
}

func TestArchitecture_DomainIsPureData(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, "./domain")
	if err != nil {
		t.Fatalf("failed to load domain package: %v", err)
	}

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.GoFiles[i]
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.InterfaceType:
					t.Errorf("Forbidden interface definition found in domain package: %s", filename)
				case *ast.FuncDecl:
					if x.Recv == nil {
						t.Errorf("Forbidden package-level function %q found in domain package: %s", x.Name.Name, filename)
					}
				}
				return true
			})
		}
	}
}

func TestArchitecture_NoAnyReturnTypes(t *testing.T) {
	for _, pkgPath := range internalPackages(t) {
		t.Run(pkgPath, func(t *testing.T) {
			checkNoAnyInPackage(t, pkgPath)
		})
	}
}

func checkNoAnyInPackage(t *testing.T, pkgPath string) {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		t.Fatalf("failed to load %s package: %v", pkgPath, err)
	}

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.GoFiles[i]
			inspectNoAny(t, filename, file)
		}
	}
}

func inspectNoAny(t *testing.T, filepathKey string, file *ast.File) {
	t.Helper()
	ast.Inspect(file, func(n ast.Node) bool {
		funcType, ok := n.(*ast.FuncType)
		if !ok {
			return true
		}
		if funcType.Results == nil {
			return true
		}
		for _, field := range funcType.Results.List {
			checkNoAny(t, filepathKey, field.Type)
		}
		return true
	})
}

func checkNoAny(t *testing.T, filename string, expr ast.Expr) {
	t.Helper()
	switch x := expr.(type) {
	case *ast.Ident:
		if x.Name == "any" {
			t.Errorf("Forbidden 'any' return type found in %s", filename)
		}
	case *ast.InterfaceType:
		if x.Methods == nil || len(x.Methods.List) == 0 {
			t.Errorf("Forbidden empty interface{} return type found in %s", filename)
		}
	}
}

func TestArchitecture_NoGlobals(t *testing.T) {
	for _, pkgPath := range internalPackages(t) {
		if pkgPath == "github.com/Cyclone1070/iav/internal/fs" {
			continue
		}
		t.Run(pkgPath, func(t *testing.T) {
			checkNoGlobalsInPackage(t, pkgPath)
		})
	}
}

func checkNoGlobalsInPackage(t *testing.T, pkgPath string) {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		t.Fatalf("failed to load %s package: %v", pkgPath, err)
	}

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.GoFiles[i]
			inspectNoGlobals(t, filename, file)
		}
	}
}

func inspectNoGlobals(t *testing.T, filepathKey string, file *ast.File) {
	t.Helper()
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fn.Recv == nil && ast.IsExported(fn.Name.Name) {
			if !strings.HasPrefix(fn.Name.Name, "New") {
				t.Errorf("Forbidden public package-level function %q found in %s. Only constructors starting with 'New' are allowed.", fn.Name.Name, filepathKey)
			}
		}
		return true
	})
}

func TestArchitecture_MockedUnitTestsOnly(t *testing.T) {
	for _, pkgPath := range internalPackages(t) {
		if pkgPath == "github.com/Cyclone1070/iav/internal/fs" {
			continue
		}
		t.Run(pkgPath, func(t *testing.T) {
			checkMockedUnitTestsInPackage(t, pkgPath)
		})
	}
}

func checkMockedUnitTestsInPackage(t *testing.T, pkgPath string) {
	t.Helper()
	cfg := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedImports,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		t.Fatalf("failed to load %s package with tests: %v", pkgPath, err)
	}

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.GoFiles[i]
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}
			inspectMockedUnitTests(t, filename, file)
		}
	}
}

func inspectMockedUnitTests(t *testing.T, filepathKey string, file *ast.File) {
	t.Helper()
	checkMockImports(t, filepathKey, file.Imports)
	checkMockFSCalls(t, filepathKey, file)
}

func checkMockImports(t *testing.T, filepathKey string, imports []*ast.ImportSpec) {
	t.Helper()
	for _, imp := range imports {
		val := strings.Trim(imp.Path.Value, `"`)
		if val == "net" || val == "net/http" || val == "os/exec" || strings.HasPrefix(val, "github.com/docker/docker/client") {
			t.Errorf("Forbidden import %q in test file %s. Unit tests must use mock objects.", val, filepathKey)
		}
	}
}

func checkMockFSCalls(t *testing.T, filepathKey string, file *ast.File) {
	t.Helper()
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "os" {
			fn := sel.Sel.Name
			if fn == "Open" || fn == "ReadFile" || fn == "Create" {
				t.Errorf("Forbidden direct os filesystem call %s.%s found in test file %s. Use mock filesystem instead.", ident.Name, fn, filepathKey)
			}
		}
		return true
	})
}

func TestArchitecture_SharedDomainStructsOnly(t *testing.T) {
	structNames := findDomainStructNames(t)
	structUsage := scanDomainStructUsage(t, structNames)

	for s, packagesUsing := range structUsage {
		if len(packagesUsing) < 2 {
			var pkgsList []string
			for p := range packagesUsing {
				pkgsList = append(pkgsList, p)
			}
			t.Errorf("Domain struct %q is only used in %d package(s) (%v). Structs in domain must be shared across 2 or more packages.", s, len(packagesUsing), pkgsList)
		}
	}
}

func findDomainStructNames(t *testing.T) map[string]bool {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedFiles,
	}
	domainPkgs, err := packages.Load(cfg, "./domain")
	if err != nil {
		t.Fatalf("failed to load domain package: %v", err)
	}

	structNames := make(map[string]bool)
	for _, pkg := range domainPkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				if _, ok := ts.Type.(*ast.StructType); ok {
					structNames[ts.Name.Name] = true
				}
				return true
			})
		}
	}
	return structNames
}

func scanDomainStructUsage(t *testing.T, structNames map[string]bool) map[string]map[string]bool {
	t.Helper()
	cfgAll := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedName,
		Tests: true,
	}
	allPkgs, err := packages.Load(cfgAll, "github.com/Cyclone1070/iav/...")
	if err != nil {
		t.Fatalf("failed to load module packages: %v", err)
	}

	structUsage := make(map[string]map[string]bool)
	for s := range structNames {
		structUsage[s] = make(map[string]bool)
	}

	for _, pkg := range allPkgs {
		if pkg.PkgPath == "github.com/Cyclone1070/iav/internal/domain" || pkg.PkgPath == "github.com/Cyclone1070/iav/internal/domain_test" {
			continue
		}
		pkgPath := pkg.PkgPath
		pkgPath = strings.TrimSuffix(pkgPath, ".test")
		pkgPath = strings.TrimSuffix(pkgPath, "_test")

		for _, file := range pkg.Syntax {
			inspectDomainUsageInFile(file, pkgPath, structNames, structUsage)
		}
	}
	return structUsage
}

func inspectDomainUsageInFile(file *ast.File, pkgPath string, structNames map[string]bool, structUsage map[string]map[string]bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "domain" {
			if structNames[sel.Sel.Name] {
				structUsage[sel.Sel.Name][pkgPath] = true
			}
		}
		return true
	})
}

func TestArchitecture_SymmetricalTestFileNaming(t *testing.T) {
	dirs := []string{"../internal", "../cmd"}
	exceptions := map[string]bool{
		"integration_test.go": true,
		"arch_test.go":        true,
		"mock_test.go":        true,
	}

	for _, startDir := range dirs {
		checkSymmetricalNamingInDir(t, startDir, exceptions)
	}
}

func checkSymmetricalNamingInDir(t *testing.T, startDir string, exceptions map[string]bool) {
	t.Helper()
	err := filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		checkTestFileSymmetry(t, path, d.Name(), exceptions)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory %s: %v", startDir, err)
	}
}

func checkTestFileSymmetry(t *testing.T, path, name string, exceptions map[string]bool) {
	t.Helper()
	if !strings.HasSuffix(path, "_test.go") {
		return
	}
	if exceptions[name] {
		return
	}

	dir := filepath.Dir(path)
	implName := strings.TrimSuffix(name, "_test.go") + ".go"
	implPath := filepath.Join(dir, implName)

	if _, err := os.Stat(implPath); err != nil {
		t.Errorf("Test file %q does not have a corresponding implementation file %q", path, implPath)
	}
}
