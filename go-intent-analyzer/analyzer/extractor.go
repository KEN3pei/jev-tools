package analyzer

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/questions"
)

type ExtractedUnit struct{ ctx UnitContext }

func (u ExtractedUnit) Ctx() UnitContext { return u.ctx }
func (u ExtractedUnit) ID() string       { return u.ctx.ID }
func (u ExtractedUnit) URI() string      { return u.ctx.URI }
func (u ExtractedUnit) SrcRange() Range  { return u.ctx.Range }
func (u ExtractedUnit) Kind() string     { return u.ctx.Kind }
func (u ExtractedUnit) Name() string     { return u.ctx.Name }

type parsedPackage struct {
	dir      string
	name     string
	fset     *token.FileSet
	files    []*ast.File
	sources  map[string]string
	info     *types.Info
	typesPkg *types.Package
}

// ExtractUnits recursively analyzes Go packages under dir. Vendor, hidden,
// generated, and test-only packages are ignored as analysis targets.
func ExtractUnits(dir string, definitions []questions.Definition) ([]ExtractedUnit, error) {
	absRoot, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	dirs, err := goPackageDirs(absRoot)
	if err != nil {
		return nil, err
	}
	var units []ExtractedUnit
	for _, packageDir := range dirs {
		pkgs, err := parsePackages(packageDir)
		if err != nil {
			return nil, err
		}
		for _, pkg := range pkgs {
			extracted := extractPackageUnits(pkg, absRoot, definitions)
			units = append(units, extracted...)
		}
	}
	sort.Slice(units, func(i, j int) bool {
		if units[i].ctx.URI != units[j].ctx.URI {
			return units[i].ctx.URI < units[j].ctx.URI
		}
		if units[i].ctx.Range.Start.Line != units[j].ctx.Range.Start.Line {
			return units[i].ctx.Range.Start.Line < units[j].ctx.Range.Start.Line
		}
		return units[i].ctx.ID < units[j].ctx.ID
	})
	return units, nil
}

func goPackageDirs(root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "vendor" || entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			seen[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs, nil
}

func parsePackages(dir string) ([]*parsedPackage, error) {
	fset := token.NewFileSet()
	sets, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool { return strings.HasSuffix(info.Name(), ".go") }, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}
	names := make([]string, 0, len(sets))
	for name := range sets {
		if !strings.HasSuffix(name, "_test") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := make([]*parsedPackage, 0, len(names))
	for _, name := range names {
		set := sets[name]
		filenames := make([]string, 0, len(set.Files))
		for filename := range set.Files {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		files := make([]*ast.File, 0, len(filenames))
		sources := make(map[string]string, len(filenames))
		for _, filename := range filenames {
			files = append(files, set.Files[filename])
			data, readErr := os.ReadFile(filename)
			if readErr != nil {
				return nil, readErr
			}
			sources[filename] = string(data)
		}
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
		conf := types.Config{Importer: importer.Default(), Error: func(error) {}}
		typesPkg, _ := conf.Check(name, fset, files, info)
		result = append(result, &parsedPackage{dir: dir, name: name, fset: fset, files: files, sources: sources, info: info, typesPkg: typesPkg})
	}
	return result, nil
}

func extractPackageUnits(pkg *parsedPackage, root string, definitions []questions.Definition) []ExtractedUnit {
	var scope []string
	for _, file := range pkg.files {
		filename := pkg.fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				scope = append(scope, "func "+qualifiedFuncName(d))
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						scope = append(scope, "type "+ts.Name.Name)
					}
				}
			}
		}
	}
	sort.Strings(scope)
	var units []ExtractedUnit
	for _, file := range pkg.files {
		filename := pkg.fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		src := pkg.sources[filename]
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				units = append(units, ExtractedUnit{ctx: buildFuncContext(pkg, file, d, src, root, scope, definitions)})
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						units = append(units, ExtractedUnit{ctx: buildTypeContext(pkg, file, d, ts, src, root, scope, definitions)})
					}
				}
			}
		}
	}
	if len(pkg.files) > 0 {
		units = append(units, ExtractedUnit{ctx: buildPackageContext(pkg, root, scope, definitions)})
	}
	return units
}

func buildFuncContext(pkg *parsedPackage, file *ast.File, decl *ast.FuncDecl, src, root string, scope []string, definitions []questions.Definition) UnitContext {
	filename := pkg.fset.Position(file.Pos()).Filename
	name := qualifiedFuncName(decl)
	rng := nodeRange(pkg.fset, src, decl)
	ctx := UnitContext{Kind: "function", Name: name, URI: filename, Range: rng, Declaration: sourceSlice(pkg.fset, src, decl), PackageScope: scope}
	if decl.Recv != nil {
		ctx.Kind = "method"
	}
	if decl.Doc != nil {
		ctx.DocComment = decl.Doc.Text()
	}
	obj := pkg.info.Defs[decl.Name]
	ctx.Callees = callees(pkg, decl)
	ctx.Callers = callers(pkg, obj, decl)
	ctx.Interfaces = implementedInterfaces(pkg, decl)
	ctx.RelatedTests = relatedTests(pkg, obj, decl.Name.Name)
	ctx.GitHistory = gitHistoryForFile(root, filename, 12)
	ctx.ID = unitID(root, filename, ctx.Kind, name, rng)
	ctx.EvidenceByIntent = buildEvidenceIndex(ctx, definitions)
	return ctx
}

func buildTypeContext(pkg *parsedPackage, file *ast.File, declaration *ast.GenDecl, spec *ast.TypeSpec, src, root string, scope []string, definitions []questions.Definition) UnitContext {
	filename := pkg.fset.Position(file.Pos()).Filename
	rng := nodeRange(pkg.fset, src, spec)
	ctx := UnitContext{Kind: "type", Name: spec.Name.Name, URI: filename, Range: rng, Declaration: sourceSlice(pkg.fset, src, spec), PackageScope: scope}
	if declaration.Doc != nil {
		ctx.DocComment = declaration.Doc.Text()
	} else if spec.Comment != nil {
		ctx.DocComment = spec.Comment.Text()
	}
	ctx.Callers = references(pkg, pkg.info.Defs[spec.Name], spec)
	ctx.Interfaces = implementedInterfacesForType(pkg, spec.Name.Name)
	ctx.RelatedTests = relatedTests(pkg, pkg.info.Defs[spec.Name], spec.Name.Name)
	ctx.GitHistory = gitHistoryForFile(root, filename, 12)
	ctx.ID = unitID(root, filename, ctx.Kind, ctx.Name, rng)
	ctx.EvidenceByIntent = buildEvidenceIndex(ctx, definitions)
	return ctx
}

func buildPackageContext(pkg *parsedPackage, root string, scope []string, definitions []questions.Definition) UnitContext {
	files := append([]*ast.File(nil), pkg.files...)
	sort.Slice(files, func(i, j int) bool {
		return pkg.fset.Position(files[i].Pos()).Filename < pkg.fset.Position(files[j].Pos()).Filename
	})
	file := files[0]
	filename := pkg.fset.Position(file.Pos()).Filename
	src := pkg.sources[filename]
	rng := nodeRange(pkg.fset, src, file.Name)
	ctx := UnitContext{Kind: "package", Name: pkg.name, URI: filename, Range: rng, PackageScope: scope, Declaration: "package " + pkg.name}
	for _, candidate := range files {
		if candidate.Doc != nil {
			ctx.DocComment = candidate.Doc.Text()
			break
		}
	}
	ctx.GitHistory = gitHistoryForDirectory(root, pkg.dir, 20)
	ctx.ID = unitID(root, filename, ctx.Kind, ctx.Name, rng)
	ctx.EvidenceByIntent = buildEvidenceIndex(ctx, definitions)
	return ctx
}

func callees(pkg *parsedPackage, decl *ast.FuncDecl) []CallRef {
	var refs []CallRef
	seen := map[string]bool{}
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		obj, name := calledObject(pkg.info, call.Fun)
		if name == "" {
			return true
		}
		key := name
		if obj != nil {
			key = obj.Id()
		}
		if seen[key] {
			return true
		}
		seen[key] = true
		refs = append(refs, CallRef{Name: name})
		return true
	})
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs
}

func callers(pkg *parsedPackage, target types.Object, exclude ast.Node) []CallRef {
	return references(pkg, target, exclude)
}

func references(pkg *parsedPackage, target types.Object, exclude ast.Node) []CallRef {
	if target == nil {
		return nil
	}
	var refs []CallRef
	seen := map[string]bool{}
	for _, file := range pkg.files {
		filename := pkg.fset.Position(file.Pos()).Filename
		src := pkg.sources[filename]
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn == exclude {
				continue
			}
			found := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if found {
					return false
				}
				switch n := node.(type) {
				case *ast.Ident:
					found = pkg.info.Uses[n] == target
				case *ast.SelectorExpr:
					if selection := pkg.info.Selections[n]; selection != nil {
						found = selection.Obj() == target
					}
				}
				return !found
			})
			if found {
				name := qualifiedFuncName(fn)
				key := filename + ":" + name
				if !seen[key] {
					seen[key] = true
					refs = append(refs, CallRef{Name: name, URI: filename, Range: nodeRange(pkg.fset, src, fn), Snippet: truncate(sourceSlice(pkg.fset, src, fn), 500)})
				}
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].URI != refs[j].URI {
			return refs[i].URI < refs[j].URI
		}
		return refs[i].Range.Start.Line < refs[j].Range.Start.Line
	})
	return refs
}

func relatedTests(pkg *parsedPackage, target types.Object, targetName string) []TestRef {
	var refs []TestRef
	for _, file := range pkg.files {
		filename := pkg.fset.Position(file.Pos()).Filename
		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}
		src := pkg.sources[filename]
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			matches := strings.Contains(strings.ToLower(fn.Name.Name), strings.ToLower(targetName))
			if !matches && target != nil {
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if id, ok := node.(*ast.Ident); ok && pkg.info.Uses[id] == target {
						matches = true
						return false
					}
					return !matches
				})
			}
			if matches {
				refs = append(refs, TestRef{Name: fn.Name.Name, URI: filename, Range: nodeRange(pkg.fset, src, fn), Snippet: truncate(sourceSlice(pkg.fset, src, fn), 1200)})
			}
		}
	}
	return refs
}

func implementedInterfaces(pkg *parsedPackage, decl *ast.FuncDecl) []InterfaceRef {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return nil
	}
	return implementedInterfacesForType(pkg, receiverName(decl.Recv.List[0].Type))
}

func implementedInterfacesForType(pkg *parsedPackage, typeName string) []InterfaceRef {
	if pkg.typesPkg == nil {
		return nil
	}
	object := pkg.typesPkg.Scope().Lookup(typeName)
	if object == nil {
		return nil
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil
	}
	var refs []InterfaceRef
	for _, file := range pkg.files {
		filename := pkg.fset.Position(file.Pos()).Filename
		src := pkg.sources[filename]
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, rawSpec := range gen.Specs {
				spec, ok := rawSpec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				interfaceObject := pkg.info.Defs[spec.Name]
				if interfaceObject == nil {
					continue
				}
				iface, ok := interfaceObject.Type().Underlying().(*types.Interface)
				if !ok {
					continue
				}
				if types.Implements(named, iface) || types.Implements(types.NewPointer(named), iface) {
					refs = append(refs, InterfaceRef{Name: spec.Name.Name, URI: filename, Range: nodeRange(pkg.fset, src, spec), Snippet: truncate(sourceSlice(pkg.fset, src, spec), 800)})
				}
			}
		}
	}
	return refs
}

func buildEvidenceIndex(ctx UnitContext, definitions []questions.Definition) map[string][]Evidence {
	result := make(map[string][]Evidence, len(definitions))
	baseText := strings.ToLower(ctx.DocComment + "\n" + ctx.Declaration)
	for _, definition := range definitions {
		evidence := []Evidence{sourceEvidence(ctx, "判定対象の宣言")}
		if containsAny(baseText, definition.Keywords) {
			evidence = append(evidence, sourceEvidence(ctx, "対象宣言またはdoc commentに意図に関連する記述・構造がある"))
		}
		if definition.ID == "extensibility" || definition.ID == "change_localization" || definition.ID == "testability" {
			for _, iface := range ctx.Interfaces {
				rng := iface.Range
				evidence = append(evidence, Evidence{ID: evidenceID("source", iface.URI, iface.Name), Kind: "source", URI: iface.URI, Range: &rng, Signal: "関連interface: " + iface.Name})
			}
		}
		if definition.ID == "testability" || definition.ID == "backward_compatibility" || definition.ID == "reliability" || definition.ID == "migration_safety" {
			for _, test := range ctx.RelatedTests {
				if definition.ID == "testability" || containsAny(strings.ToLower(test.Name+" "+test.Snippet), definition.Keywords) {
					rng := test.Range
					evidence = append(evidence, Evidence{ID: evidenceID("test", test.URI, test.Name), Kind: "test", URI: test.URI, Range: &rng, Signal: "関連テスト: " + test.Name})
				}
			}
		}
		for _, entry := range ctx.GitHistory {
			if containsAny(strings.ToLower(entry.Message), definition.Keywords) {
				evidence = append(evidence, Evidence{ID: evidenceID("commit", entry.Revision, definition.ID), Kind: "commit", Revision: entry.Revision, Signal: entry.Message})
			}
		}
		result[definition.ID] = dedupeEvidence(evidence)
	}
	return result
}

func sourceEvidence(ctx UnitContext, signal string) Evidence {
	rng := ctx.Range
	return Evidence{ID: evidenceID("source", ctx.URI, ctx.ID), Kind: "source", URI: ctx.URI, Range: &rng, Signal: signal}
}
func evidenceID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("e-%x", sum[:8])
}
func dedupeEvidence(items []Evidence) []Evidence {
	seen := map[string]bool{}
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			out = append(out, item)
		}
	}
	return out
}
func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func gitHistoryForFile(root, filename string, limit int) []GitEntry {
	rel, err := filepath.Rel(root, filename)
	if err != nil {
		return nil
	}
	return gitHistory(root, rel, limit)
}
func gitHistoryForDirectory(root, dir string, limit int) []GitEntry {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return nil
	}
	return gitHistory(root, rel, limit)
}
func gitHistory(root, path string, limit int) []GitEntry {
	output, err := exec.Command("git", "-C", root, "log", fmt.Sprintf("-n%d", limit), "--format=%H%x09%s", "--", path).Output()
	if err != nil {
		return nil
	}
	var entries []GitEntry
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			entries = append(entries, GitEntry{Revision: parts[0], Message: parts[1]})
		}
	}
	return entries
}

func RepositoryRevision(root string) string {
	output, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
func unitID(root, filename, kind, name string, rng Range) string {
	rel, _ := filepath.Rel(root, filename)
	return fmt.Sprintf("%s#%s:%s:%d", filepath.ToSlash(rel), kind, name, rng.Start.Line+1)
}
func qualifiedFuncName(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return decl.Name.Name
	}
	return receiverName(decl.Recv.List[0].Type) + "." + decl.Name.Name
}
func receiverName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	case *ast.IndexExpr:
		return receiverName(value.X)
	case *ast.IndexListExpr:
		return receiverName(value.X)
	}
	return ""
}
func calledObject(info *types.Info, expression ast.Expr) (types.Object, string) {
	switch value := expression.(type) {
	case *ast.Ident:
		return info.Uses[value], value.Name
	case *ast.SelectorExpr:
		if selection := info.Selections[value]; selection != nil {
			return selection.Obj(), selection.Obj().Name()
		}
		return info.Uses[value.Sel], value.Sel.Name
	}
	return nil, ""
}
func sourceSlice(fset *token.FileSet, src string, node ast.Node) string {
	start, end := fset.Position(node.Pos()), fset.Position(node.End())
	if start.Offset < 0 || end.Offset > len(src) || start.Offset >= end.Offset {
		return ""
	}
	return src[start.Offset:end.Offset]
}
func nodeRange(fset *token.FileSet, src string, node ast.Node) Range {
	return Range{Start: lspPosition(fset.Position(node.Pos()), src), End: lspPosition(fset.Position(node.End()), src)}
}
func lspPosition(position token.Position, src string) Position {
	lineStart := position.Offset - (position.Column - 1)
	if lineStart < 0 || position.Offset > len(src) {
		return Position{Line: position.Line - 1, Character: position.Column - 1}
	}
	units := 0
	for offset := lineStart; offset < position.Offset; {
		r, size := utf8.DecodeRuneInString(src[offset:position.Offset])
		if size == 0 {
			break
		}
		units += utf16.RuneLen(r)
		offset += size
	}
	return Position{Line: position.Line - 1, Character: units}
}
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
