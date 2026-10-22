package charmjson

import (
	"errors"
	"strings"
	"testing"
)

func mustParseJSON(t *testing.T, source string) *Node {
	t.Helper()
	node, err := ParseJSON([]byte(source))
	if err != nil {
		t.Fatalf("ParseJSON falhou: %v", err)
	}
	return node
}

func mustParseYAML(t *testing.T, source string) *Node {
	t.Helper()
	node, err := ParseYAML([]byte(source))
	if err != nil {
		t.Fatalf("ParseYAML falhou: %v", err)
	}
	return node
}

func TestParseJSONPreservesOrderAndTypes(t *testing.T) {
	source := `{
  "name": "charm \"json\"\n",
  "big": 12345678901234567890,
  "ratio": 1.234567890123456789,
  "ok": true,
  "nothing": null,
  "list": [1, "two", false, null],
  "nested": {"x": 1}
}`
	root := mustParseJSON(t, source)

	if root.Kind != Object {
		t.Fatalf("Kind da raiz = %v, quer %v", root.Kind, Object)
	}
	if root.Path != "" {
		t.Fatalf("Path da raiz = %q, quer vazio", root.Path)
	}
	wantKeys := []string{"name", "big", "ratio", "ok", "nothing", "list", "nested"}
	if len(root.Children) != len(wantKeys) {
		t.Fatalf("filhos = %d, quer %d", len(root.Children), len(wantKeys))
	}
	for index, want := range wantKeys {
		if got := root.Children[index].Key; got != want {
			t.Errorf("chave %d = %q, quer %q (ordem de inserção)", index, got, want)
		}
	}

	name := root.Children[0]
	if name.Kind != String || name.Value != "charm \"json\"\n" {
		t.Errorf("string = (%v, %q), quer (String, %q)", name.Kind, name.Value, "charm \"json\"\n")
	}
	if name.Path != "/name" {
		t.Errorf("Path = %q, quer /name", name.Path)
	}

	big := root.Children[1]
	if big.Kind != Number || big.Value != "12345678901234567890" {
		t.Errorf("número grande = (%v, %q), perdeu precisão", big.Kind, big.Value)
	}
	if ratio := root.Children[2]; ratio.Value != "1.234567890123456789" {
		t.Errorf("decimal = %q, quer texto original", ratio.Value)
	}
	if ok := root.Children[3]; ok.Kind != Bool || ok.Value != "true" {
		t.Errorf("bool = (%v, %q), quer (Bool, true)", ok.Kind, ok.Value)
	}
	if nothing := root.Children[4]; nothing.Kind != Null || nothing.Value != "null" {
		t.Errorf("null = (%v, %q), quer (Null, null)", nothing.Kind, nothing.Value)
	}

	list, found := root.Find("/list")
	if !found || list.Kind != Array {
		t.Fatalf("Find(/list) = (%v, %v), quer array", list, found)
	}
	wantList := []Kind{Number, String, Bool, Null}
	for index, want := range wantList {
		child := list.Children[index]
		if child.Kind != want {
			t.Errorf("item %d = %v, quer %v", index, child.Kind, want)
		}
		if child.Key != "" {
			t.Errorf("item de array %d tem Key = %q, quer vazio", index, child.Key)
		}
		if wantPath := "/list/" + string(rune('0'+index)); child.Path != wantPath {
			t.Errorf("Path do item %d = %q, quer %q", index, child.Path, wantPath)
		}
	}
}

func TestParseJSONEscapedKeysAndUnicode(t *testing.T) {
	root := mustParseJSON(t, `{"a\u00e7\u00e3o":"olá 😀","quote\"key":1}`)
	if len(root.Children) != 2 {
		t.Fatalf("filhos = %d, quer 2", len(root.Children))
	}
	if got := root.Children[0].Key; got != "ação" {
		t.Errorf("chave = %q, quer ação", got)
	}
	if got := root.Children[0].Value; got != "olá 😀" {
		t.Errorf("valor = %q, quer olá 😀", got)
	}
	if got := root.Children[1].Key; got != `quote"key` {
		t.Errorf("chave escapada = %q", got)
	}
}

func TestParseJSONInvalidIncludesPosition(t *testing.T) {
	source := "{\n  \"a\": 1,\n  \"b\": }\n"
	_, err := ParseJSON([]byte(source))
	if err == nil {
		t.Fatal("ParseJSON aceitou JSON inválido")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("erro = %T (%v), quer *ParseError", err, err)
	}
	if parseErr.Line != 3 {
		t.Errorf("linha = %d, quer 3", parseErr.Line)
	}
	if parseErr.Column < 1 {
		t.Errorf("coluna = %d, quer >= 1", parseErr.Column)
	}
	if !strings.Contains(err.Error(), "linha 3") {
		t.Errorf("mensagem = %q, quer conter \"linha 3\"", err.Error())
	}
}

func TestParseJSONEmpty(t *testing.T) {
	_, err := ParseJSON(nil)
	if err == nil {
		t.Fatal("ParseJSON aceitou entrada vazia")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("erro = %T, quer *ParseError", err)
	}
	if parseErr.Line != 1 || parseErr.Column != 1 {
		t.Errorf("posição = %d:%d, quer 1:1", parseErr.Line, parseErr.Column)
	}
}

func TestParseJSONTrailingContent(t *testing.T) {
	for _, source := range []string{`{"a":1} {"b":2}`, `{"a":1} lixo`, `[1] 2`} {
		if _, err := ParseJSON([]byte(source)); err == nil {
			t.Errorf("ParseJSON(%q) aceitou conteúdo extra", source)
		}
	}
}

func TestParseYAMLAnchorsOrderAndNesting(t *testing.T) {
	source := `
base: &base
  name: charm
  version: 3
copy: *base
list:
  - one
  - two: 2
    three: 3
big: 12345678901234567890
quoted: "true"
flag: true
empty:
`
	root := mustParseYAML(t, source)
	if root.Kind != Object {
		t.Fatalf("raiz = %v, quer Object", root.Kind)
	}
	wantKeys := []string{"base", "copy", "list", "big", "quoted", "flag", "empty"}
	if len(root.Children) != len(wantKeys) {
		t.Fatalf("filhos = %d, quer %d", len(root.Children), len(wantKeys))
	}
	for index, want := range wantKeys {
		if got := root.Children[index].Key; got != want {
			t.Errorf("chave %d = %q, quer %q (ordem de definição)", index, got, want)
		}
	}

	base := root.Children[0]
	if base.Kind != Object || len(base.Children) != 2 {
		t.Fatalf("base = (%v, %d filhos)", base.Kind, len(base.Children))
	}
	if base.Children[0].Key != "name" || base.Children[1].Key != "version" {
		t.Errorf("ordem das chaves de base = %q, %q", base.Children[0].Key, base.Children[1].Key)
	}

	copied, found := root.Find("/copy/name")
	if !found || copied.Value != "charm" {
		t.Errorf("alias /copy/name = (%v, %v), quer charm", copied, found)
	}
	if copied.Path != "/copy/name" {
		t.Errorf("Path do alias = %q, quer /copy/name", copied.Path)
	}

	list := root.Children[2]
	if list.Kind != Array || len(list.Children) != 2 {
		t.Fatalf("list = (%v, %d filhos)", list.Kind, len(list.Children))
	}
	if list.Children[0].Value != "one" || list.Children[1].Kind != Object {
		t.Errorf("itens da lista inesperados")
	}
	if got, _ := root.Find("/list/1/three"); got == nil || got.Value != "3" {
		t.Errorf("Find(/list/1/three) = %v, quer 3", got)
	}

	if big := root.Children[3]; big.Kind != Number || big.Value != "12345678901234567890" {
		t.Errorf("big = (%v, %q), perdeu o texto original", big.Kind, big.Value)
	}
	if quoted := root.Children[4]; quoted.Kind != String || quoted.Value != "true" {
		t.Errorf("quoted = (%v, %q), quer String true", quoted.Kind, quoted.Value)
	}
	if flag := root.Children[5]; flag.Kind != Bool {
		t.Errorf("flag = %v, quer Bool", flag.Kind)
	}
	if empty := root.Children[6]; empty.Kind != Null {
		t.Errorf("empty = %v, quer Null", empty.Kind)
	}
}

func TestParseYAMLInvalidIncludesLine(t *testing.T) {
	_, err := ParseYAML([]byte("a: 1\n  b: 2\n"))
	if err == nil {
		t.Fatal("ParseYAML aceitou YAML inválido")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("erro = %T (%v), quer *ParseError", err, err)
	}
	if parseErr.Line < 2 {
		t.Errorf("linha = %d, quer >= 2", parseErr.Line)
	}
	if parseErr.Column < 1 {
		t.Errorf("coluna = %d, quer >= 1", parseErr.Column)
	}
	if !strings.Contains(err.Error(), "linha") {
		t.Errorf("mensagem = %q, quer conter linha", err.Error())
	}
}

func TestParseAutoDetection(t *testing.T) {
	jsonNode, err := Parse([]byte(`{"a":1,"b":[true]}`), "")
	if err != nil {
		t.Fatalf("auto JSON: %v", err)
	}
	if jsonNode.Kind != Object || jsonNode.Children[1].Kind != Array {
		t.Errorf("auto JSON retornou árvore inesperada")
	}

	yamlNode, err := Parse([]byte("a: 1\nb: 2\n"), "")
	if err != nil {
		t.Fatalf("auto YAML: %v", err)
	}
	if yamlNode.Kind != Object || len(yamlNode.Children) != 2 {
		t.Errorf("auto YAML retornou árvore inesperada")
	}

	flowNode, err := Parse([]byte(`{a: 1, b: 2}`), "")
	if err != nil {
		t.Fatalf("auto YAML flow: %v", err)
	}
	if flowNode.Kind != Object || len(flowNode.Children) != 2 {
		t.Errorf("mapping de fluxo YAML não foi reconhecido")
	}

	emptyNode, err := Parse(nil, "")
	if err != nil || emptyNode.Kind != Null {
		t.Errorf("entrada vazia = (%v, %v), quer Null sem erro", emptyNode, err)
	}

	ymlNode, err := Parse([]byte("a: 1\n"), "yml")
	if err != nil || ymlNode.Kind != Object {
		t.Errorf("formato yml = (%v, %v)", ymlNode, err)
	}

	if _, err := Parse([]byte("{}"), "xml"); err == nil || !strings.Contains(err.Error(), "formato desconhecido") {
		t.Errorf("formato inválido retornou %v", err)
	}

	_, err = Parse([]byte(`{"a": [}`), "")
	if err == nil {
		t.Fatal("auto aceitou conteúdo inválido em JSON e YAML")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) || parseErr.Line != 1 {
		t.Errorf("auto inválido = %T %v, quer *ParseError na linha 1", err, err)
	}
}

func TestParseStripsBOM(t *testing.T) {
	node, err := ParseJSON([]byte("\ufeff{\"a\":1}"))
	if err != nil {
		t.Fatalf("BOM JSON: %v", err)
	}
	if len(node.Children) != 1 {
		t.Errorf("BOM JSON retornou %d filhos", len(node.Children))
	}
	yamlNode, err := ParseYAML([]byte("\ufeffa: 1\n"))
	if err != nil {
		t.Fatalf("BOM YAML: %v", err)
	}
	if yamlNode.Kind != Object {
		t.Errorf("BOM YAML retornou %v", yamlNode.Kind)
	}
}
