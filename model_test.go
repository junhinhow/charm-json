package charmjson

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func testKeyMsg(text string) tea.KeyPressMsg {
	runes := []rune(text)
	var code rune
	if len(runes) > 0 {
		code = runes[0]
	}
	return tea.KeyPressMsg{Code: code, Text: text}
}

func testSpecialKeyMsg(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func testApplyKey(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update retornou %T, quer Model", updated)
	}
	return next, cmd
}

func TestNavigationKeys(t *testing.T) {
	root := mustParseJSONForTest(t, `{"k0":0,"k1":1,"k2":2,"k3":3,"k4":4}`)
	model := New(root)
	model.SetSize(80, 10)

	model, _ = testApplyKey(t, model, testKeyMsg("j"))
	if got := model.SelectedPath(); got != "/k0" {
		t.Errorf("após j: %q, quer /k0", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyDown))
	if got := model.SelectedPath(); got != "/k1" {
		t.Errorf("após ↓: %q, quer /k1", got)
	}
	model, _ = testApplyKey(t, model, testKeyMsg("k"))
	if got := model.SelectedPath(); got != "/k0" {
		t.Errorf("após k: %q, quer /k0", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyUp))
	if got := model.SelectedPath(); got != "" {
		t.Errorf("após ↑: %q, quer raiz", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyUp))
	if got := model.SelectedPath(); got != "" {
		t.Errorf("↑ no topo saiu da raiz: %q", got)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("G"))
	if got := model.SelectedPath(); got != "/k4" {
		t.Errorf("após G: %q, quer /k4", got)
	}
	model, _ = testApplyKey(t, model, testKeyMsg("j"))
	if got := model.SelectedPath(); got != "/k4" {
		t.Errorf("j no fim saiu da última linha: %q", got)
	}
	model, _ = testApplyKey(t, model, testKeyMsg("g"))
	if got := model.SelectedPath(); got != "" {
		t.Errorf("após g: %q, quer raiz", got)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("x"))
	if got := model.SelectedPath(); got != "" {
		t.Errorf("tecla desconhecida mudou a seleção: %q", got)
	}
}

func TestToggleAndCollapseKeys(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":{"b":1},"c":2}`)
	model := New(root)
	model.SetSize(80, 10)

	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEnter))
	if got := len(model.Rows()); got != 1 {
		t.Fatalf("enter na raiz: %d linhas, quer 1", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeySpace))
	if got := len(model.Rows()); got != 4 {
		t.Fatalf("space na raiz: %d linhas, quer 4", got)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("j"))
	model, _ = testApplyKey(t, model, testKeyMsg("h"))
	if got := len(model.Rows()); got != 3 {
		t.Fatalf("h em /a: %d linhas, quer 3", got)
	}
	model, _ = testApplyKey(t, model, testKeyMsg("l"))
	if got := len(model.Rows()); got != 4 {
		t.Fatalf("l em /a: %d linhas, quer 4", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeySpace))
	if got := len(model.Rows()); got != 3 {
		t.Fatalf("space em /a: %d linhas, quer 3", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEnter))
	if got := len(model.Rows()); got != 4 {
		t.Fatalf("enter em /a: %d linhas, quer 4", got)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("G"))
	path := model.SelectedPath()
	model, _ = testApplyKey(t, model, testKeyMsg("h"))
	if got := len(model.Rows()); got != 4 || model.SelectedPath() != path {
		t.Errorf("h em folha alterou a árvore: %d linhas", got)
	}
	model, _ = testApplyKey(t, model, testKeyMsg("l"))
	if got := len(model.Rows()); got != 4 {
		t.Errorf("l em folha alterou a árvore: %d linhas", got)
	}
}

func TestYankEmitsMessage(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":1}`)
	model := New(root)

	_, cmd := testApplyKey(t, model, testKeyMsg("y"))
	if cmd == nil {
		t.Fatal("y na raiz não devolveu comando")
	}
	message := cmd()
	yanked, ok := message.(YankedMsg)
	if !ok {
		t.Fatalf("comando devolveu %T, quer YankedMsg", message)
	}
	if yanked.Path != "" {
		t.Errorf("YankedMsg.Path = %q, quer vazio (raiz)", yanked.Path)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("j"))
	model, cmd = testApplyKey(t, model, testKeyMsg("y"))
	if cmd == nil {
		t.Fatal("y em /a não devolveu comando")
	}
	yanked, ok = cmd().(YankedMsg)
	if !ok || yanked.Path != "/a" {
		t.Errorf("YankedMsg = %#v, quer Path /a", cmd())
	}

	model.SetFilter("zzz")
	if _, cmd := testApplyKey(t, model, testKeyMsg("y")); cmd != nil {
		t.Error("y sem linhas deveria devolver comando nil")
	}
}

func TestSearchMode(t *testing.T) {
	root := mustParseJSONForTest(t, `{"alpha":{"beta":1,"gamma":2},"beta":3}`)
	model := New(root)
	model.SetSize(80, 10)

	model, _ = testApplyKey(t, model, testKeyMsg("/"))
	if !model.searching || model.Filter() != "" {
		t.Fatalf("após /: searching=%v filtro=%q", model.searching, model.Filter())
	}

	for _, char := range []string{"b", "e", "t", "a"} {
		model, _ = testApplyKey(t, model, testKeyMsg(char))
	}
	if got := model.Filter(); got != "beta" {
		t.Fatalf("filtro digitado = %q, quer beta", got)
	}
	if got := model.SelectedPath(); got != "" {
		t.Errorf("digitação moveu a seleção para %q", got)
	}
	if len(model.Rows()) != 4 {
		t.Errorf("linhas filtradas = %d, quer 4", len(model.Rows()))
	}

	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyBackspace))
	if got := model.Filter(); got != "bet" {
		t.Errorf("após backspace = %q, quer bet", got)
	}

	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEnter))
	if model.searching {
		t.Error("enter deveria encerrar o modo de busca")
	}
	if got := model.Filter(); got != "bet" {
		t.Errorf("enter deveria manter o filtro, obteve %q", got)
	}

	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEsc))
	if model.Filter() != "" || model.searching {
		t.Errorf("esc deveria limpar o filtro, obteve %q", model.Filter())
	}

	model.SetFilter("beta")
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEsc))
	if model.Filter() != "" {
		t.Errorf("esc no modo normal não limpou o filtro: %q", model.Filter())
	}
}

func TestSearchTypingDoesNotNavigate(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":1,"b":2}`)
	model := New(root)

	model, _ = testApplyKey(t, model, testKeyMsg("/"))
	model, _ = testApplyKey(t, model, testKeyMsg("j"))
	if got := model.Filter(); got != "j" {
		t.Errorf("j na busca = %q, quer j", got)
	}
	if got := model.SelectedPath(); got != "" {
		t.Errorf("j na busca moveu a seleção para %q", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeySpace))
	if got := model.Filter(); got != "j " {
		t.Errorf("espaço na busca = %q, quer \"j \"", got)
	}
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyUp))
	if got := model.Filter(); got != "j " {
		t.Errorf("seta na busca alterou o filtro para %q", got)
	}

	model, _ = testApplyKey(t, model, testKeyMsg("ã"))
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyBackspace))
	if got := model.Filter(); got != "j " {
		t.Errorf("backspace com unicode = %q, quer \"j \"", got)
	}

	model, _ = testApplyKey(t, model, tea.KeyPressMsg{Code: 'j', Text: "j", Mod: tea.ModCtrl})
	if got := model.Filter(); got != "j " {
		t.Errorf("ctrl+j alterou o filtro para %q", got)
	}

	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEnter))
	model, _ = testApplyKey(t, model, testKeyMsg("/"))
	if got := model.Filter(); got != "" {
		t.Errorf("nova busca deveria começar vazia, obteve %q", got)
	}
}

func TestViewRespectsSizeAndWidth(t *testing.T) {
	root := mustParseJSONForTest(t, `{
  "long": "abcdefghijklmnopqrstuvwxyz0123456789",
  "emoji": "😀😀😀😀😀",
  "nested": {"ação": {"日本語": "テキスト"}},
  "n": 12345678901234567890,
  "ok": true,
  "nil": null
}`)

	model := New(root)
	model.SetSize(0, 10)
	if content := model.View().Content; content != "" {
		t.Errorf("largura 0 = %q, quer vazio", content)
	}
	model.SetSize(80, 0)
	if content := model.View().Content; content != "" {
		t.Errorf("altura 0 = %q, quer vazio", content)
	}
	model.SetSize(-3, -3)
	if content := model.View().Content; content != "" {
		t.Errorf("tamanho negativo = %q, quer vazio", content)
	}

	for _, width := range []int{1, 20, 80, 200} {
		model.SetSize(width, 8)
		content := model.View().Content
		if !utf8.ValidString(content) {
			t.Errorf("largura %d gerou UTF-8 inválido", width)
		}
		lines := strings.Split(content, "\n")
		if len(lines) > 8 {
			t.Errorf("largura %d: %d linhas, máximo 8", width, len(lines))
		}
		for index, line := range lines {
			if got := ansi.StringWidth(line); got > width {
				t.Errorf("largura %d: linha %d tem %d células: %q", width, index, got, line)
			}
		}
	}

	model.SetSize(20, 8)
	content := model.View().Content
	if !strings.Contains(content, "…") {
		t.Errorf("conteúdo truncado deveria conter …: %q", content)
	}
	if !strings.Contains(content, "\x1b[") {
		t.Error("cores deveriam estar ligadas por padrão")
	}

	plain := New(root, WithSyntaxColors(false))
	plain.SetSize(20, 8)
	plainContent := plain.View().Content
	if strings.Contains(plainContent, "\x1b[") {
		t.Errorf("WithSyntaxColors(false) ainda tem ANSI: %q", plainContent)
	}
	if !utf8.ValidString(plainContent) {
		t.Error("conteúdo sem cores gerou UTF-8 inválido")
	}
	for index, line := range strings.Split(plainContent, "\n") {
		if got := ansi.StringWidth(line); got > 20 {
			t.Errorf("sem cores: linha %d tem %d células", index, got)
		}
	}

	model.SetSize(20, 1)
	if lines := strings.Split(model.View().Content, "\n"); len(lines) != 1 {
		t.Errorf("altura 1 = %d linhas, quer 1", len(lines))
	}

	model.SetSize(20, 3)
	model.SetFilter(strings.Repeat("a", 50))
	lines := strings.Split(model.View().Content, "\n")
	if len(lines) > 3 {
		t.Errorf("prompt de busca: %d linhas, máximo 3", len(lines))
	}
	for _, line := range lines {
		if got := ansi.StringWidth(line); got > 20 {
			t.Errorf("prompt largo: %d células", got)
		}
	}

	model.SetFilter("a\nb\tc")
	lines = strings.Split(model.View().Content, "\n")
	if len(lines) > 3 {
		t.Errorf("prompt com controles: %d linhas, máximo 3", len(lines))
	}
	for _, line := range lines {
		if got := ansi.StringWidth(line); got > 20 {
			t.Errorf("prompt com controles: %d células", got)
		}
	}

	empty := New(nil)
	empty.SetSize(80, 24)
	if content := empty.View().Content; content != "" {
		t.Errorf("raiz nil = %q, quer vazio", content)
	}
	if rows := empty.Rows(); rows != nil {
		t.Errorf("raiz nil produziu %d linhas", len(rows))
	}
}

func TestViewScrollsToSelection(t *testing.T) {
	var source strings.Builder
	source.WriteString("{")
	for index := 0; index < 30; index++ {
		if index > 0 {
			source.WriteString(",")
		}
		fmt.Fprintf(&source, `"k%02d":%d`, index, index)
	}
	source.WriteString("}")
	root := mustParseJSONForTest(t, source.String())

	model := New(root, WithSyntaxColors(false))
	model.SetSize(200, 5)
	model, _ = testApplyKey(t, model, testKeyMsg("G"))
	content := model.View().Content
	if !strings.Contains(content, "k29: 29") {
		t.Errorf("G não trouxe a última linha para a view: %q", content)
	}
	if lines := strings.Split(content, "\n"); len(lines) > 5 {
		t.Errorf("scroll gerou %d linhas, máximo 5", len(lines))
	}

	model, _ = testApplyKey(t, model, testKeyMsg("g"))
	content = model.View().Content
	if !strings.Contains(content, "k00: 0") || strings.Contains(content, "k29: 29") {
		t.Errorf("g não voltou ao topo: %q", content)
	}
}

func TestViewDeterministic(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":{"b":[1,2,3]},"c":"texto","d":true,"e":null}`)
	model := New(root)
	model.SetSize(80, 10)

	first := model.View().Content
	for render := 0; render < 100; render++ {
		if got := model.View().Content; got != first {
			t.Fatalf("render %d divergiu do primeiro", render)
		}
	}

	firstPaths := testRowPaths(model.Rows())
	for render := 0; render < 100; render++ {
		if got := testRowPaths(model.Rows()); !testEqualStrings(got, firstPaths) {
			t.Fatalf("Rows() divergiu no render %d", render)
		}
	}
}

func TestMaxDepth(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":{"b":{"c":{"d":1}}}}`)

	unlimited := New(root)
	unlimited.SetSize(80, 10)
	if got := len(unlimited.Rows()); got != 5 {
		t.Fatalf("sem maxDepth = %d linhas, quer 5", got)
	}

	model := New(root, WithMaxDepth(-3))
	if got := len(model.Rows()); got != 5 {
		t.Errorf("WithMaxDepth negativo = %d linhas, quer ilimitado", got)
	}

	model = New(root, WithMaxDepth(2))
	model.SetSize(80, 10)
	rows := model.Rows()
	if got := testRowPaths(rows); !testEqualStrings(got, []string{"", "/a", "/a/b"}) {
		t.Fatalf("WithMaxDepth(2) = %v", got)
	}
	if rows[2].Display != "b: …" || rows[2].Prefix != "  " {
		t.Errorf("nó truncado = (%q, %q)", rows[2].Display, rows[2].Prefix)
	}
	if _, found := root.Find("/a/b/c"); !found {
		t.Error("maxDepth não deveria remover nós do modelo")
	}

	model, _ = testApplyKey(t, model, testKeyMsg("G"))
	model, _ = testApplyKey(t, model, testSpecialKeyMsg(tea.KeyEnter))
	if got := len(model.Rows()); got != 3 {
		t.Errorf("expandir nó truncado = %d linhas, quer 3", got)
	}

	one := New(root, WithMaxDepth(1))
	rows = one.Rows()
	if got := testRowPaths(rows); !testEqualStrings(got, []string{"", "/a"}) {
		t.Errorf("WithMaxDepth(1) = %v", got)
	}
	if rows[1].valText != "…" {
		t.Errorf("WithMaxDepth(1) último valor = %q, quer …", rows[1].valText)
	}
}

func TestFilterClampsSelection(t *testing.T) {
	root := mustParseJSONForTest(t, `{"k0":0,"k1":1,"k2":2,"k3":3}`)
	model := New(root)
	model.SetSize(80, 10)
	model, _ = testApplyKey(t, model, testKeyMsg("G"))
	if got := model.SelectedPath(); got != "/k3" {
		t.Fatalf("G = %q, quer /k3", got)
	}
	model.SetFilter("k1")
	if got := model.SelectedPath(); got != "" {
		t.Errorf("SetFilter deveria voltar à primeira linha, obteve %q", got)
	}
	if rows := model.Rows(); len(rows) != 2 {
		t.Errorf("filtro k1 = %d linhas, quer 2", len(rows))
	}
}

func TestUpdateIgnoresOtherMessages(t *testing.T) {
	root := mustParseJSONForTest(t, `{"a":1}`)
	model := New(root)
	model.SetSize(80, 10)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 10, Height: 10})
	if cmd != nil {
		t.Errorf("WindowSizeMsg devolveu comando %v", cmd)
	}
	if _, ok := updated.(Model); !ok {
		t.Errorf("Update devolveu %T", updated)
	}
}

func testEqualStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
