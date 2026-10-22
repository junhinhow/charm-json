package charmjson

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Códigos ANSI fixos: não dependem do perfil de cor do terminal, o que mantém
// a renderização determinística em testes e em ambientes sem TTY.
const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiReverse = "\x1b[7m"
	ansiCyan    = "\x1b[36m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiMagenta = "\x1b[35m"
)

// YankedMsg é emitido pelo comando devolvido quando o usuário pressiona "y".
// Path contém o JSON Pointer do nó selecionado no momento da cópia.
type YankedMsg struct {
	Path string
}

// Option configura um Model na construção.
type Option func(*Model)

// WithMaxDepth limita a profundidade exibida pelo modelo. Nós contêineres no
// limite aparecem como "…" e não podem ser expandidos na visualização. Zero
// (o padrão) significa profundidade ilimitada; valores negativos são tratados
// como zero.
func WithMaxDepth(depth int) Option {
	return func(model *Model) {
		if depth < 0 {
			depth = 0
		}
		model.maxDepth = depth
	}
}

// WithSyntaxColors liga ou desliga a coloração ANSI da árvore. O padrão é
// ligado.
func WithSyntaxColors(enabled bool) Option {
	return func(model *Model) {
		model.colors = enabled
	}
}

// Model é o explorador interativo de uma árvore JSON/YAML para Bubble Tea.
//
// O estado de colapso pertence aos nós (Node.Collapsed) e é compartilhado entre
// cópias do modelo; seleção, filtro e tamanho pertencem ao Model.
type Model struct {
	root      *Node
	width     int
	height    int
	cursor    int
	filter    string
	searching bool
	maxDepth  int
	colors    bool
}

// New cria um explorador para a raiz informada. Uma raiz nil produz um modelo
// vazio e seguro. As opções são aplicadas na ordem recebida.
func New(root *Node, opts ...Option) Model {
	model := Model{root: root, colors: true}
	for _, option := range opts {
		if option != nil {
			option(&model)
		}
	}
	return model
}

// SetSize define a área de renderização em células. Valores negativos são
// tratados como zero; largura ou altura zero fazem View devolver conteúdo
// vazio.
func (m *Model) SetSize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	m.width = width
	m.height = height
}

// SetFilter define o filtro atual e volta a seleção para a primeira linha.
// A busca é case-insensitive e considera caminho, chave e valor.
func (m *Model) SetFilter(filter string) {
	m.filter = filter
	m.searching = false
	m.cursor = 0
}

// Filter devolve o filtro atual.
func (m Model) Filter() string {
	return m.filter
}

// Rows devolve as linhas visíveis, já considerando colapso, filtro e
// profundidade máxima.
func (m Model) Rows() []Row {
	return flattenRows(m.root, m.filter, m.maxDepth)
}

// SelectedPath devolve o JSON Pointer do nó selecionado ou "" quando não há
// seleção. A raiz também usa "".
func (m Model) SelectedPath() string {
	if node, ok := m.selectedNode(); ok {
		return node.Path
	}
	return ""
}

// SelectedValue devolve o valor escalar do nó selecionado. O segundo retorno é
// falso para contêineres ou quando não há seleção.
func (m Model) SelectedValue() (string, bool) {
	node, ok := m.selectedNode()
	if !ok || node.isContainer() {
		return "", false
	}
	return node.Value, true
}

func (m Model) selectedNode() (*Node, bool) {
	rows := m.Rows()
	if len(rows) == 0 || m.cursor < 0 || m.cursor >= len(rows) {
		return nil, false
	}
	return rows[m.cursor].Node, true
}

func (m Model) clampCursor() Model {
	rows := m.Rows()
	if len(rows) == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	return m
}

func (m Model) move(delta int) Model {
	rows := m.Rows()
	if len(rows) == 0 {
		m.cursor = 0
		return m
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	return m
}

func (m Model) selectFirst() Model {
	m.cursor = 0
	return m
}

func (m Model) selectLast() Model {
	rows := m.Rows()
	if len(rows) == 0 {
		m.cursor = 0
		return m
	}
	m.cursor = len(rows) - 1
	return m
}

func (m Model) toggleSelected() Model {
	node, ok := m.selectedNode()
	if !ok || !node.isContainer() {
		return m
	}
	node.Collapsed = !node.Collapsed
	return m.clampCursor()
}

func (m Model) collapseSelected() Model {
	node, ok := m.selectedNode()
	if !ok || !node.isContainer() {
		return m
	}
	node.Collapsed = true
	return m.clampCursor()
}

func (m Model) expandSelected() Model {
	node, ok := m.selectedNode()
	if !ok || !node.isContainer() {
		return m
	}
	node.Collapsed = false
	return m.clampCursor()
}

func (m Model) clearFilter() Model {
	m.filter = ""
	m.searching = false
	m.cursor = 0
	return m
}

// Init implementa tea.Model e não agenda comandos iniciais.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implementa tea.Model e trata as teclas do explorador:
// j/k e setas navegam, enter/espaço alternam o colapso, h/l recolhem e
// expandem, g/G vão ao topo e ao fim, "/" inicia a busca, esc limpa o filtro
// e "y" emite YankedMsg com o caminho selecionado.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	key := tea.Key(keyMsg)
	if m.searching {
		return m.updateSearch(key)
	}

	switch key.Code {
	case tea.KeyUp:
		return m.move(-1), nil
	case tea.KeyDown:
		return m.move(1), nil
	case tea.KeyEnter, tea.KeySpace:
		return m.toggleSelected(), nil
	case tea.KeyEsc:
		return m.clearFilter(), nil
	case tea.KeyLeft:
		return m.collapseSelected(), nil
	case tea.KeyRight:
		return m.expandSelected(), nil
	}

	if key.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModMeta) != 0 {
		return m, nil
	}
	switch key.Text {
	case "j":
		return m.move(1), nil
	case "k":
		return m.move(-1), nil
	case "h":
		return m.collapseSelected(), nil
	case "l":
		return m.expandSelected(), nil
	case "g":
		return m.selectFirst(), nil
	case "G":
		return m.selectLast(), nil
	case "/":
		m.filter = ""
		m.searching = true
		m.cursor = 0
		return m, nil
	case "y":
		if _, ok := m.selectedNode(); !ok {
			return m, nil
		}
		path := m.SelectedPath()
		return m, func() tea.Msg {
			return YankedMsg{Path: path}
		}
	}
	return m, nil
}

func (m Model) updateSearch(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		return m.clearFilter(), nil
	case tea.KeyEnter:
		m.searching = false
		m.cursor = 0
		return m, nil
	case tea.KeyBackspace:
		m.filter = trimLastRune(m.filter)
		m.cursor = 0
		return m, nil
	case tea.KeySpace:
		m.filter += " "
		m.cursor = 0
		return m, nil
	}
	if key.Mod&(tea.ModCtrl|tea.ModAlt|tea.ModMeta) != 0 {
		return m, nil
	}
	if key.Text != "" {
		m.filter += key.Text
		m.cursor = 0
	}
	return m, nil
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

// View implementa tea.Model. O conteúdo nunca excede a largura configurada:
// cada linha é truncada com ansi.Truncate e o número de linhas respeita a
// altura. Largura ou altura não positivas devolvem um View vazio.
func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.View{}
	}

	rows := m.Rows()
	showPrompt := m.searching || m.filter != ""
	listHeight := m.height
	if showPrompt {
		listHeight--
	}

	lines := make([]string, 0, m.height)
	if listHeight > 0 && len(rows) > 0 {
		offset := 0
		if m.cursor >= listHeight {
			offset = m.cursor - listHeight + 1
		}
		if maxOffset := len(rows) - listHeight; offset > maxOffset {
			offset = maxOffset
		}
		if offset < 0 {
			offset = 0
		}
		end := offset + listHeight
		if end > len(rows) {
			end = len(rows)
		}
		for index := offset; index < end; index++ {
			lines = append(lines, m.renderRow(rows[index], index == m.cursor))
		}
	}
	if showPrompt {
		lines = append(lines, m.renderPrompt())
	}
	return tea.View{Content: strings.Join(lines, "\n")}
}

func (m Model) renderRow(row Row, selected bool) string {
	var builder strings.Builder
	if selected {
		if m.colors {
			builder.WriteString(ansiReverse + "> " + ansiReset)
		} else {
			builder.WriteString("> ")
		}
	} else {
		builder.WriteString("  ")
	}
	depth := row.Depth
	if depth < 0 {
		depth = 0
	}
	builder.WriteString(strings.Repeat("  ", depth))

	if !m.colors {
		builder.WriteString(row.Prefix)
		builder.WriteString(row.Display)
		return ansi.Truncate(builder.String(), m.width, "…")
	}

	kind := Null
	if row.Node != nil {
		kind = row.Node.Kind
	}
	builder.WriteString(styleText(row.Prefix, ansiCyan))
	if row.keyText != "" {
		builder.WriteString(styleText(row.keyText, ansiBold+ansiCyan))
		builder.WriteString(": ")
		builder.WriteString(styleText(row.valText, kindANSIColor(kind)))
	} else {
		builder.WriteString(styleText(row.valText, kindANSIColor(kind)))
	}
	return ansi.Truncate(builder.String(), m.width, "…")
}

func (m Model) renderPrompt() string {
	prompt := "/" + m.filter
	if m.searching {
		prompt += "█"
	}
	return ansi.Truncate(sanitizePrompt(prompt), m.width, "…")
}

// sanitizePrompt troca controles por espaço para que um filtro colado com
// quebras de linha não aumente a quantidade de linhas da View.
func sanitizePrompt(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return ' '
		}
		return r
	}, text)
}

func styleText(text, style string) string {
	if text == "" {
		return ""
	}
	return style + text + ansiReset
}

func kindANSIColor(kind Kind) string {
	switch kind {
	case String:
		return ansiGreen
	case Number:
		return ansiYellow
	case Bool:
		return ansiMagenta
	case Null:
		return ansiDim
	default:
		return ansiDim
	}
}
