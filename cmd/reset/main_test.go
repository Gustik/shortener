package main

import (
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

// -update перезаписывает golden-файлы актуальным выводом генератора.
// Запуск: go test -update
var update = flag.Bool("update", false, "update golden files")

// TestGenerator - главный тест: запускаем генератор на testdata/input,
// сравниваем результат с testdata/golden.
func TestGenerator(t *testing.T) {
	cases := []struct {
		name        string // имя поддиректории в testdata/input и testdata/golden
		wantGenFile bool   // ожидаем ли появление reset.gen.go
	}{
		{
			name:        "simple", // примитивы, слайс, мапа
			wantGenFile: true,
		},
		{
			name:        "pointers", // указатели на примитивы и структуры
			wantGenFile: true,
		},
		{
			name:        "ignored", // структура без маркера — файл не должен создаться
			wantGenFile: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inputDir := filepath.Join("testdata", "input", tc.name)
			goldenFile := filepath.Join("testdata", "golden", tc.name, "reset.gen.go")
			outFile := filepath.Join(inputDir, "reset.gen.go")

			// Убираем файл от предыдущего запуска, чтобы не мешал.
			_ = os.Remove(outFile)
			t.Cleanup(func() { _ = os.Remove(outFile) })

			// Запускаем генератор на конкретной директории.
			if err := runGenerator(t, inputDir); err != nil {
				t.Fatalf("generator error: %v", err)
			}

			// Проверяем случай когда файл не должен появиться.
			if !tc.wantGenFile {
				if _, err := os.Stat(outFile); err == nil {
					t.Errorf("reset.gen.go should not be created for package %q", tc.name)
				}
				return
			}

			// Читаем то что сгенерировал генератор.
			got, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("reading generated file: %v", err)
			}

			// Режим обновления: перезаписываем golden-файл и выходим.
			if *update {
				if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
					t.Fatalf("creating golden dir: %v", err)
				}
				if err := os.WriteFile(goldenFile, got, 0644); err != nil {
					t.Fatalf("writing golden file: %v", err)
				}
				t.Logf("updated golden file: %s", goldenFile)
				return
			}

			// Читаем эталон.
			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("reading golden file %s: %v\nhint: run with -update to create it", goldenFile, err)
			}

			// Сравниваем байт в байт.
			if string(got) != string(want) {
				t.Errorf("generated output does not match golden file %s\n\n--- want ---\n%s\n--- got ---\n%s",
					goldenFile, want, got)
			}
		})
	}
}

// TestHasGenerateReset проверяет парсинг маркера из комментариев.
func TestHasGenerateReset(t *testing.T) {
	cases := []struct {
		comment string
		want    bool
	}{
		{"// generate:reset", true},
		{"// generate:reset\n// other comment", true},
		{"// other comment", false},
		{"//generate:reset", false},   // без пробела — не совпадает
		{"// generate:reset ", false}, // лишний пробел в конце — не совпадает
	}

	for _, tc := range cases {
		t.Run(tc.comment, func(t *testing.T) {
			src := "package p\n" + tc.comment + "\ntype T struct{}"
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			structs := findResetStructs([]*ast.File{f})
			got := len(structs) > 0
			if got != tc.want {
				t.Errorf("hasGenerateReset(%q) = %v, want %v", tc.comment, got, tc.want)
			}
		})
	}
}

// TestPrimitiveZero проверяет что все примитивные типы получают правильный ноль.
func TestPrimitiveZero(t *testing.T) {
	cases := []struct {
		typeName string
		want     string
	}{
		{"int", "0"},
		{"int8", "0"},
		{"int64", "0"},
		{"uint", "0"},
		{"float32", "0"},
		{"float64", "0"},
		{"byte", "0"},
		{"rune", "0"},
		{"string", `""`},
		{"bool", "false"},
		{"MyStruct", ""}, // не примитив — должна вернуть пустую строку
		{"error", ""},    // интерфейс — не примитив
	}

	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			got := primitiveZero(tc.typeName)
			if got != tc.want {
				t.Errorf("primitiveZero(%q) = %q, want %q", tc.typeName, got, tc.want)
			}
		})
	}
}

// TestNoMarkerNoFile проверяет что пакет без маркера не получает reset.gen.go.
func TestNoMarkerNoFile(t *testing.T) {
	src := `package p
type Plain struct {
    Value string
}`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "plain.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	structs := findResetStructs([]*ast.File{f})
	if len(structs) != 0 {
		t.Errorf("expected no structs, got %d", len(structs))
	}
}

// runGenerator запускает полный цикл генерации для одной директории.
// Повторяет логику main(), но для конкретного пути.
func runGenerator(t *testing.T, dir string) error {
	t.Helper()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles,
		Fset: fset,
		Dir:  absDir,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(fset, filename, src, parser.ParseComments)
		},
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 || len(pkg.GoFiles) == 0 {
			continue
		}
		structs := findResetStructs(pkg.Syntax)
		if len(structs) == 0 {
			continue
		}
		pkgDir := filepath.Dir(pkg.GoFiles[0])
		if err := writeGenFile(pkgDir, pkg.Name, structs); err != nil {
			return err
		}
	}
	return nil
}

// TestGeneratedCodeCompiles проверяет что сгенерированный код валиден с точки зрения go/format.
// Это не полная компиляция, но ловит синтаксические ошибки.
func TestGeneratedCodeCompiles(t *testing.T) {
	src := `package p

// generate:reset
type Sample struct {
	Count  int
	Name   string
	Active bool
	Items  []int
	Index  map[string]int
	Nested *Sample
}`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	structs := findResetStructs([]*ast.File{f})
	if len(structs) == 0 {
		t.Fatal("no structs found")
	}

	// writeGenFile использует format.Source внутри — если код невалиден, вернёт ошибку.
	// Пишем во временную директорию.
	tmpDir := t.TempDir()
	if err := writeGenFile(tmpDir, "p", structs); err != nil {
		t.Fatalf("generated invalid Go code: %v", err)
	}

	// Дополнительно проверяем через format.Source напрямую.
	generated, err := os.ReadFile(filepath.Join(tmpDir, "reset.gen.go"))
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}
	if _, err := format.Source(generated); err != nil {
		t.Errorf("generated code is not valid Go: %v\n%s", err, generated)
	}
}
