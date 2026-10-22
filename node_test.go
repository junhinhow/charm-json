package charmjson

import (
	"reflect"
	"testing"
)

func rowPaths(rows []Row) []string {
	paths := make([]string, len(rows))
	for index, row := range rows {
		paths[index] = row.Node.Path
	}
	return paths
}

func TestFindJSONPointer(t *testing.T) {
	root := mustParseJSON(t, `{
  "a": [{"b": 1}],
  "a~b": 2,
  "a/b": 3,
  "": 4,
  "arr": [10, 20]
}`)

	tests := []struct {
		path  string
		value string
		found bool
	}{
		{path: "", value: "", found: true},
		{path: "/a/0/b", value: "1", found: true},
		{path: "/a~0b", value: "2", found: true},
		{path: "/a~1b", value: "3", found: true},
		{path: "/", value: "4", found: true},
		{path: "/arr/1", value: "20", found: true},
		{path: "/arr/2", found: false},
		{path: "/arr/01", found: false},
		{path: "/arr/-", found: false},
		{path: "/arr/+1", found: false},
		{path: "a/0/b", found: false},
		{path: "/a~2b", found: false},
		{path: "/a~", found: false},
		{path: "/missing", found: false},
		{path: "/a/0/b/c", found: false},
	}
	for _, test := range tests {
		node, found := root.Find(test.path)
		if found != test.found {
			t.Errorf("Find(%q) encontrado = %v, quer %v", test.path, found, test.found)
			continue
		}
		if !found {
			continue
		}
		if test.path == "" {
			if node != root {
				t.Errorf("Find(%q) retornou outro nó", test.path)
			}
			continue
		}
		if node.Value != test.value {
			t.Errorf("Find(%q).Value = %q, quer %q", test.path, node.Value, test.value)
		}
	}
}

func TestFindIgnoresCollapse(t *testing.T) {
	root := mustParseJSON(t, `{"a":{"b":1}}`)
	root.Collapse("/a")
	if node, found := root.Find("/a/b"); !found || node.Value != "1" {
		t.Errorf("Find após colapso = (%v, %v), quer 1", node, found)
	}
}

func TestFlattenRespectsCollapse(t *testing.T) {
	root := mustParseJSON(t, `{"a":{"b":{"c":1,"d":2}},"e":3}`)

	rows := root.Flatten()
	wantPaths := []string{"", "/a", "/a/b", "/a/b/c", "/a/b/d", "/e"}
	if got := rowPaths(rows); !reflect.DeepEqual(got, wantPaths) {
		t.Fatalf("paths = %v, quer %v", got, wantPaths)
	}
	wantDepths := []int{0, 1, 2, 3, 3, 1}
	for index, want := range wantDepths {
		if rows[index].Depth != want {
			t.Errorf("Depth[%d] = %d, quer %d", index, rows[index].Depth, want)
		}
	}
	if rows[0].Display != "{2}" || rows[0].Prefix != "▾ " {
		t.Errorf("raiz = (%q, %q)", rows[0].Display, rows[0].Prefix)
	}
	if rows[3].Display != "c: 1" || rows[3].Prefix != "  " {
		t.Errorf("folha = (%q, %q)", rows[3].Display, rows[3].Prefix)
	}

	root.Collapse("/a")
	rows = root.Flatten()
	wantPaths = []string{"", "/a", "/e"}
	if got := rowPaths(rows); !reflect.DeepEqual(got, wantPaths) {
		t.Fatalf("após colapsar /a: %v, quer %v", got, wantPaths)
	}
	if rows[1].Prefix != "▸ " {
		t.Errorf("prefixo de /a colapsado = %q, quer ▸ ", rows[1].Prefix)
	}

	root.Toggle("/a")
	if got := len(root.Flatten()); got != 6 {
		t.Errorf("após toggle /a: %d linhas, quer 6", got)
	}

	root.Toggle("/a/b")
	rows = root.Flatten()
	wantPaths = []string{"", "/a", "/a/b", "/e"}
	if got := rowPaths(rows); !reflect.DeepEqual(got, wantPaths) {
		t.Fatalf("após colapsar /a/b: %v, quer %v", got, wantPaths)
	}
	if rows[2].Prefix != "▸ " {
		t.Errorf("prefixo de /a/b colapsado = %q, quer ▸ ", rows[2].Prefix)
	}
}

func TestCollapseMethodsAreSafe(t *testing.T) {
	root := mustParseJSON(t, `{"a":{"b":1},"c":2}`)
	var nilNode *Node
	nilNode.Collapse("")
	nilNode.Expand("")
	nilNode.Toggle("")
	nilNode.CollapseAll()
	nilNode.ExpandAll()
	if nilNode.Flatten() != nil {
		t.Error("Flatten de nó nil deveria ser nil")
	}

	root.Collapse("/nope")
	root.Expand("/nope")
	root.Toggle("/nope")
	root.Collapse("/c")
	if leaf, _ := root.Find("/c"); leaf.Collapsed {
		t.Error("folha não deveria ser colapsada")
	}

	root.CollapseAll()
	rows := root.Flatten()
	if len(rows) != 1 || rows[0].Prefix != "▸ " {
		t.Fatalf("CollapseAll = %d linhas, prefixo %q", len(rows), rows[0].Prefix)
	}
	root.CollapseAll()
	if got := len(root.Flatten()); got != 1 {
		t.Errorf("CollapseAll não é idempotente: %d linhas", got)
	}

	root.ExpandAll()
	if got := len(root.Flatten()); got != 4 {
		t.Errorf("ExpandAll = %d linhas, quer 4", got)
	}
	root.ExpandAll()
	if got := len(root.Flatten()); got != 4 {
		t.Errorf("ExpandAll não é idempotente: %d linhas", got)
	}
}

func TestFlattenArrayIndicesAndEmptyContainers(t *testing.T) {
	root := mustParseJSON(t, `{"lista":["á","😀"],"vazio":{},"vazia":[]}`)
	rows := root.Flatten()
	want := []string{
		`{3}`,
		`lista: [2]`,
		`0: "á"`,
		`1: "😀"`,
		`vazio: {}`,
		`vazia: []`,
	}
	if len(rows) != len(want) {
		t.Fatalf("linhas = %d, quer %d", len(rows), len(want))
	}
	for index, display := range want {
		if rows[index].Display != display {
			t.Errorf("Display[%d] = %q, quer %q", index, rows[index].Display, display)
		}
	}
}

func TestFilterKeepsMatchesAndAncestors(t *testing.T) {
	root := mustParseJSON(t, `{
  "alpha": {"beta": 1, "gamma": 2},
  "delta": {"beta": 3},
  "str": "BETA"
}`)
	model := New(root)

	model.SetFilter("beta")
	wantPaths := []string{"", "/alpha", "/alpha/beta", "/delta", "/delta/beta", "/str"}
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Fatalf("filtro beta = %v, quer %v", got, wantPaths)
	}
	if model.Rows()[0].Prefix != "▾ " {
		t.Errorf("raiz filtrada deveria estar expandida")
	}

	model.SetFilter("BETA")
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("filtro case-insensitive = %v, quer %v", got, wantPaths)
	}

	model.SetFilter("2")
	wantPaths = []string{"", "/alpha", "/alpha/gamma"}
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("filtro por valor = %v, quer %v", got, wantPaths)
	}

	model.SetFilter("/alpha/gamma")
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("filtro por caminho = %v, quer %v", got, wantPaths)
	}

	model.SetFilter("zzz")
	if rows := model.Rows(); len(rows) != 0 {
		t.Errorf("filtro sem match = %v, quer vazio", rowPaths(rows))
	}

	model.SetFilter("")
	if got := len(model.Rows()); got != 7 {
		t.Errorf("filtro vazio = %d linhas, quer 7", got)
	}
}

func TestFilterIgnoresCollapseWhileActive(t *testing.T) {
	root := mustParseJSON(t, `{"alpha":{"beta":1,"gamma":2},"delta":3}`)
	root.Collapse("/alpha")
	model := New(root)

	model.SetFilter("gamma")
	wantPaths := []string{"", "/alpha", "/alpha/gamma"}
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Fatalf("filtro com nó colapsado = %v, quer %v", got, wantPaths)
	}
	if model.Rows()[1].Prefix != "▾ " {
		t.Errorf("ancestral do match deveria aparecer expandido")
	}

	model.SetFilter("")
	if rows := model.Rows(); len(rows) != 3 || rows[1].Prefix != "▸ " {
		t.Errorf("colapso deveria voltar após limpar o filtro: %v", rowPaths(rows))
	}
}

func TestFilterEscapedKey(t *testing.T) {
	root := mustParseJSON(t, `{"a/b":1}`)
	model := New(root)
	model.SetFilter("a/b")
	wantPaths := []string{"", "/a~1b"}
	if got := rowPaths(model.Rows()); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("filtro com chave escapada = %v, quer %v", got, wantPaths)
	}
}

func TestSelectedHelpers(t *testing.T) {
	root := mustParseJSON(t, `{"name":"charm","obj":{},"nil":null}`)
	model := New(root)
	model.SetSize(80, 10)

	if got := model.SelectedPath(); got != "" {
		t.Errorf("SelectedPath inicial = %q, quer vazio (raiz)", got)
	}
	if value, ok := model.SelectedValue(); ok || value != "" {
		t.Errorf("SelectedValue na raiz = (%q, %v), quer (\", false)", value, ok)
	}

	model, _ = applyKey(t, model, keyMsg("j"))
	if got := model.SelectedPath(); got != "/name" {
		t.Errorf("SelectedPath = %q, quer /name", got)
	}
	if value, ok := model.SelectedValue(); !ok || value != "charm" {
		t.Errorf("SelectedValue = (%q, %v), quer (charm, true)", value, ok)
	}

	model, _ = applyKey(t, model, keyMsg("j"))
	if _, ok := model.SelectedValue(); ok {
		t.Error("SelectedValue de contêiner deveria ser falso")
	}

	model, _ = applyKey(t, model, keyMsg("j"))
	if value, ok := model.SelectedValue(); !ok || value != "null" {
		t.Errorf("SelectedValue de null = (%q, %v), quer (null, true)", value, ok)
	}

	model.SetFilter("zzz")
	if got := model.SelectedPath(); got != "" {
		t.Errorf("SelectedPath sem linhas = %q, quer vazio", got)
	}
	if _, ok := model.SelectedValue(); ok {
		t.Error("SelectedValue sem linhas deveria ser falso")
	}
}
