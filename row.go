package charmjson

import (
	"strconv"
	"strings"
)

// Row é uma linha visível da árvore, pronta para renderização.
type Row struct {
	// Node é o nó correspondente.
	Node *Node
	// Depth é a profundidade do nó (a raiz tem profundidade 0).
	Depth int
	// Prefix é o marcador de expansão: "▾ ", "▸ " ou "  ".
	Prefix string
	// Display é o texto já formatado, no formato "chave: valor".
	Display string

	// keyText e valText guardam as partes de Display para coloração.
	keyText string
	valText string
}

// Flatten devolve as linhas visíveis da subárvore, em pré-ordem, respeitando o
// estado Collapsed de cada nó. Itens de array exibem o índice e a raiz usa o
// caminho "". O resultado é determinístico e segue a ordem original dos filhos.
func (n *Node) Flatten() []Row {
	return flattenRows(n, "", 0)
}

// flattenRows percorre a árvore em pré-ordem. filter vazio desativa o filtro;
// maxDepth 0 significa profundidade ilimitada.
func flattenRows(root *Node, filter string, maxDepth int) []Row {
	if root == nil {
		return nil
	}

	query := strings.ToLower(filter)
	var visible map[*Node]bool
	if query != "" {
		visible = make(map[*Node]bool)
		var mark func(*Node) bool
		mark = func(node *Node) bool {
			match := nodeMatches(node, query)
			childMatch := false
			for _, child := range node.Children {
				if mark(child) {
					childMatch = true
				}
			}
			if match || childMatch {
				visible[node] = true
			}
			return match || childMatch
		}
		if !mark(root) {
			return nil
		}
	}

	rows := make([]Row, 0, len(root.Children)+1)
	var walk func(node *Node, depth int, label string)
	walk = func(node *Node, depth int, label string) {
		type childRef struct {
			node  *Node
			index int
		}
		children := make([]childRef, 0, len(node.Children))
		for index, child := range node.Children {
			if visible == nil || visible[child] {
				children = append(children, childRef{node: child, index: index})
			}
		}

		truncated := false
		if maxDepth > 0 && depth >= maxDepth && len(children) > 0 {
			truncated = true
		}

		hasChildren := len(children) > 0 && !truncated
		expanded := hasChildren && (visible != nil || !node.Collapsed)
		prefix := "  "
		if hasChildren {
			if expanded {
				prefix = "▾ "
			} else {
				prefix = "▸ "
			}
		}

		value := displayValue(node, truncated)
		display := value
		if label != "" {
			display = label + ": " + value
		}
		rows = append(rows, Row{
			Node:    node,
			Depth:   depth,
			Prefix:  prefix,
			Display: display,
			keyText: label,
			valText: value,
		})

		if truncated || !expanded {
			return
		}
		for _, child := range children {
			childLabel := formatKey(child.node.Key)
			if node.Kind == Array {
				childLabel = strconv.Itoa(child.index)
			}
			walk(child.node, depth+1, childLabel)
		}
	}
	walk(root, 0, "")
	return rows
}

// nodeMatches verifica se o caminho, a chave ou o valor do nó contém a
// consulta (já convertida para minúsculas).
func nodeMatches(node *Node, query string) bool {
	if strings.Contains(strings.ToLower(node.Path), query) {
		return true
	}
	if node.Key != "" && strings.Contains(strings.ToLower(node.Key), query) {
		return true
	}
	return node.Value != "" && strings.Contains(strings.ToLower(node.Value), query)
}

// displayValue formata o valor de um nó. Strings recebem aspas e escapes;
// contêineres mostram a quantidade de filhos; truncated indica que a
// profundidade máxima foi atingida.
func displayValue(node *Node, truncated bool) string {
	if truncated {
		return "…"
	}
	switch node.Kind {
	case Object:
		if len(node.Children) == 0 {
			return "{}"
		}
		return "{" + strconv.Itoa(len(node.Children)) + "}"
	case Array:
		if len(node.Children) == 0 {
			return "[]"
		}
		return "[" + strconv.Itoa(len(node.Children)) + "]"
	case String:
		return strconv.Quote(node.Value)
	case Number:
		return node.Value
	case Bool:
		return node.Value
	case Null:
		return "null"
	default:
		return node.Value
	}
}
