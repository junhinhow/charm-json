// Package charmjson oferece um explorador interativo de JSON e YAML para o
// terminal, com parser que preserva a ordem das chaves, caminhos JSON Pointer,
// colapso de nós, filtro e um modelo Bubble Tea.
package charmjson

import (
	"strconv"
	"strings"
	"unicode"
)

// Kind identifica o tipo de um nó da árvore.
type Kind int

const (
	// Object representa um objeto JSON ou um mapa YAML.
	Object Kind = iota
	// Array representa uma lista JSON ou uma sequência YAML.
	Array
	// String representa um escalar de texto.
	String
	// Number representa um escalar numérico, preservado como texto original.
	Number
	// Bool representa um escalar booleano.
	Bool
	// Null representa a ausência de valor.
	Null
)

// String devolve o nome do tipo em minúsculas, útil para depuração.
func (k Kind) String() string {
	switch k {
	case Object:
		return "object"
	case Array:
		return "array"
	case String:
		return "string"
	case Number:
		return "number"
	case Bool:
		return "bool"
	case Null:
		return "null"
	default:
		return "unknown"
	}
}

// Node é um nó da árvore JSON/YAML.
//
// Key é a chave no nó pai (vazia para a raiz e para itens de array). Value
// guarda a representação escalar: strings são armazenadas já decodificadas,
// números mantêm o texto original e Null vale "null". Path é o JSON Pointer
// RFC 6901 do nó. Collapsed indica se um nó contêiner está recolhido.
type Node struct {
	Key       string
	Kind      Kind
	Value     string
	Children  []*Node
	Path      string
	Collapsed bool
}

// isContainer informa se o nó pode ter filhos.
func (n *Node) isContainer() bool {
	return n != nil && (n.Kind == Object || n.Kind == Array)
}

// Find procura o nó no caminho JSON Pointer informado.
//
// O caminho da raiz é "" (string vazia); "/" refere-se ao filho de chave
// vazia. Os escapes RFC 6901 "~0" (til) e "~1" (barra) são aceitos. Caminhos
// sem barra inicial, com escape inválido, índice fora do intervalo ou chave
// inexistente retornam (nil, false). O colapso não interfere na busca.
func (n *Node) Find(path string) (*Node, bool) {
	if n == nil {
		return nil, false
	}
	if path == "" {
		return n, true
	}
	if path[0] != '/' {
		return nil, false
	}
	current := n
	for _, raw := range strings.Split(path[1:], "/") {
		token, ok := decodePointerToken(raw)
		if !ok {
			return nil, false
		}
		switch current.Kind {
		case Object:
			var next *Node
			for _, child := range current.Children {
				if child.Key == token {
					next = child
					break
				}
			}
			if next == nil {
				return nil, false
			}
			current = next
		case Array:
			index, ok := parseArrayIndex(token)
			if !ok || index >= len(current.Children) {
				return nil, false
			}
			current = current.Children[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// Collapse recolhe o nó contêiner no caminho informado. Caminhos inexistentes
// ou nós escalares são ignorados.
func (n *Node) Collapse(path string) {
	if node, ok := n.Find(path); ok && node.isContainer() {
		node.Collapsed = true
	}
}

// Expand expande o nó contêiner no caminho informado. Caminhos inexistentes ou
// nós escalares são ignorados.
func (n *Node) Expand(path string) {
	if node, ok := n.Find(path); ok && node.isContainer() {
		node.Collapsed = false
	}
}

// Toggle alterna o estado de colapso do nó contêiner no caminho informado,
// mantendo o estado anterior se o caminho não existir ou não for contêiner.
func (n *Node) Toggle(path string) {
	if node, ok := n.Find(path); ok && node.isContainer() {
		node.Collapsed = !node.Collapsed
	}
}

// CollapseAll recolhe todos os nós contêineres, inclusive a raiz.
func (n *Node) CollapseAll() {
	if n == nil {
		return
	}
	if n.isContainer() {
		n.Collapsed = true
	}
	for _, child := range n.Children {
		child.CollapseAll()
	}
}

// ExpandAll expande todos os nós contêineres.
func (n *Node) ExpandAll() {
	if n == nil {
		return
	}
	if n.isContainer() {
		n.Collapsed = false
	}
	for _, child := range n.Children {
		child.ExpandAll()
	}
}

var pathEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// joinPath acrescenta um token ao caminho, aplicando os escapes RFC 6901.
func joinPath(parent, token string) string {
	return parent + "/" + pathEscaper.Replace(token)
}

// decodePointerToken decodifica um token JSON Pointer, rejeitando escapes
// desconhecidos e "~" no fim da string.
func decodePointerToken(token string) (string, bool) {
	if !strings.ContainsRune(token, '~') {
		return token, true
	}
	var b strings.Builder
	b.Grow(len(token))
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) {
			return "", false
		}
		switch token[i+1] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", false
		}
		i++
	}
	return b.String(), true
}

// parseArrayIndex aceita apenas o formato de índice de array do RFC 6901:
// zero ou um número decimal sem zeros à esquerda.
func parseArrayIndex(token string) (int, bool) {
	if token == "" {
		return 0, false
	}
	if len(token) > 1 && token[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(token); i++ {
		if token[i] < '0' || token[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(token)
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}

// formatKey deixa chaves simples legíveis e cita as demais, evitando
// ambiguidade com o separador ": ".
func formatKey(key string) string {
	if key == "" {
		return `""`
	}
	for _, r := range key {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			return strconv.Quote(key)
		}
	}
	return key
}
