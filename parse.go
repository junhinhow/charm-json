package charmjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseError descreve um erro de sintaxe com a posição aproximada no
// documento. Line e Column são baseados em 1; Column vale 0 quando a
// biblioteca de origem não informa a coluna.
type ParseError struct {
	Line   int
	Column int
	Msg    string
}

// Error implementa error e inclui a posição quando disponível.
func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	switch {
	case e.Line > 0 && e.Column > 0:
		return fmt.Sprintf("linha %d, coluna %d: %s", e.Line, e.Column, e.Msg)
	case e.Line > 0:
		return fmt.Sprintf("linha %d: %s", e.Line, e.Msg)
	default:
		return e.Msg
	}
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// stripBOM remove a marca de ordem de bytes UTF-8 do início dos dados.
func stripBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, utf8BOM)
}

// Parse decodifica data no formato indicado. Os formatos aceitos são "json",
// "yaml"/"yml" e "" (detecção automática). A detecção automática usa JSON
// quando o conteúdo é JSON válido e tenta YAML nos demais casos; entradas que
// começam com "{" ou "[" e não são válidas em nenhum dos dois retornam o erro
// do JSON, que costuma ser mais preciso.
func Parse(data []byte, format string) (*Node, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "auto":
		return parseAuto(data)
	case "json":
		return ParseJSON(data)
	case "yaml", "yml":
		return ParseYAML(data)
	default:
		return nil, fmt.Errorf("formato desconhecido %q: use \"json\", \"yaml\" ou \"\"", format)
	}
}

func parseAuto(data []byte) (*Node, error) {
	trimmed := bytes.TrimSpace(stripBOM(data))
	if len(trimmed) == 0 {
		return ParseYAML(data)
	}
	if json.Valid(trimmed) {
		return ParseJSON(data)
	}
	yamlNode, yamlErr := ParseYAML(data)
	if yamlErr == nil {
		return yamlNode, nil
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		if jsonNode, jsonErr := ParseJSON(data); jsonErr != nil {
			return nil, jsonErr
		} else {
			return jsonNode, nil
		}
	}
	return nil, yamlErr
}

// newParseError calcula a linha e a coluna de um deslocamento em bytes.
func newParseError(data []byte, offset int, msg string) *ParseError {
	line, column := lineColumn(data, offset)
	return &ParseError{Line: line, Column: column, Msg: msg}
}

// lineColumn converte um deslocamento em bytes para linha e coluna baseadas em
// 1. A coluna é contada em runas.
func lineColumn(data []byte, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	line := 1
	lineStart := 0
	for i := 0; i < offset; i++ {
		if data[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	return line, utf8.RuneCount(data[lineStart:offset]) + 1
}
