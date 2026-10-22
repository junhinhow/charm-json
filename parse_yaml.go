package charmjson

import (
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

var (
	yamlLinePattern   = regexp.MustCompile(`line (\d+):\s*(.*)`)
	yamlColumnPattern = regexp.MustCompile(`column (\d+)`)
)

// ParseYAML decodifica um documento YAML mantendo a ordem de definição das
// chaves. Âncoras e aliases são resolvidos; chaves de mesclagem ("<<") são
// mantidas como chaves comuns. Erros retornam *ParseError com posição
// aproximada (a biblioteca yaml.v3 informa a linha, mas não a coluna).
func ParseYAML(data []byte) (*Node, error) {
	data = stripBOM(data)

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, wrapYAMLError(err)
	}
	if document.Kind == 0 {
		return &Node{Kind: Null, Value: "null"}, nil
	}
	return convertYAMLNode(&document, "", "", make(map[*yaml.Node]bool)), nil
}

// convertYAMLNode converte a árvore de yaml.Node em Node. O mapa cycle evita
// recursão infinita em aliases cíclicos.
func convertYAMLNode(value *yaml.Node, key, path string, cycle map[*yaml.Node]bool) *Node {
	if value == nil || cycle[value] {
		return &Node{Key: key, Kind: Null, Value: "null", Path: path}
	}

	switch value.Kind {
	case yaml.DocumentNode:
		if len(value.Content) == 0 {
			return &Node{Key: key, Kind: Null, Value: "null", Path: path}
		}
		return convertYAMLNode(value.Content[0], key, path, cycle)
	case yaml.AliasNode:
		if value.Alias == nil {
			return &Node{Key: key, Kind: Null, Value: "null", Path: path}
		}
		return convertYAMLNode(value.Alias, key, path, cycle)
	case yaml.SequenceNode:
		node := &Node{Key: key, Kind: Array, Path: path}
		cycle[value] = true
		for index, child := range value.Content {
			node.Children = append(node.Children, convertYAMLNode(
				child, "", joinPath(path, strconv.Itoa(index)), cycle))
		}
		delete(cycle, value)
		return node
	case yaml.MappingNode:
		node := &Node{Key: key, Kind: Object, Path: path}
		cycle[value] = true
		for index := 0; index+1 < len(value.Content); index += 2 {
			childKey := yamlKey(value.Content[index])
			node.Children = append(node.Children, convertYAMLNode(
				value.Content[index+1], childKey, joinPath(path, childKey), cycle))
		}
		delete(cycle, value)
		return node
	case yaml.ScalarNode:
		kind, scalar := yamlScalar(value)
		return &Node{Key: key, Kind: kind, Value: scalar, Path: path}
	default:
		return &Node{Key: key, Kind: Null, Value: "null", Path: path}
	}
}

// yamlKey extrai o texto de uma chave YAML. Chaves compostas recebem um rótulo
// estável e simplificado; chaves escalares preservam o valor original.
func yamlKey(value *yaml.Node) string {
	if value == nil {
		return ""
	}
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Value
	case yaml.AliasNode:
		if value.Alias != nil {
			return yamlKey(value.Alias)
		}
		return ""
	case yaml.MappingNode:
		return "{...}"
	case yaml.SequenceNode:
		return "[...]"
	case yaml.DocumentNode:
		if len(value.Content) > 0 {
			return yamlKey(value.Content[0])
		}
		return ""
	default:
		return ""
	}
}

// yamlScalar classifica um escalar YAML pelo seu tipo resolvido.
func yamlScalar(value *yaml.Node) (Kind, string) {
	switch value.ShortTag() {
	case "!!null":
		return Null, "null"
	case "!!bool":
		return Bool, value.Value
	case "!!int", "!!float":
		return Number, value.Value
	default:
		return String, value.Value
	}
}

// wrapYAMLError converte o erro textual de yaml.v3 em *ParseError. A biblioteca
// não expõe um tipo de erro com posição, então a linha é extraída da mensagem;
// quando a coluna não está disponível, usa 1.
func wrapYAMLError(err error) error {
	text := err.Error()
	message := text
	line := 0
	if match := yamlLinePattern.FindStringSubmatch(text); match != nil {
		line, _ = strconv.Atoi(match[1])
		message = match[2]
	}
	column := 0
	if match := yamlColumnPattern.FindStringSubmatch(text); match != nil {
		column, _ = strconv.Atoi(match[1])
	}
	message = strings.TrimSpace(strings.TrimPrefix(message, "yaml: "))
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	return &ParseError{Line: line, Column: column, Msg: "YAML inválido: " + message}
}
