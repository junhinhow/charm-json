package charmjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// ParseJSON decodifica um documento JSON preservando a ordem de inserção das
// chaves e o texto original dos números. Erros de sintaxe retornam
// *ParseError com linha e coluna.
func ParseJSON(data []byte) (*Node, error) {
	data = stripBOM(data)
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, &ParseError{Line: 1, Column: 1, Msg: "JSON vazio"}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	root, err := decodeJSONValue(decoder, "", "")
	if err != nil {
		return nil, wrapJSONError(data, decoder, err)
	}

	token, err := decoder.Token()
	switch {
	case err == nil:
		return nil, newParseError(data, int(decoder.InputOffset()),
			fmt.Sprintf("JSON inválido: conteúdo extra após o valor raiz (%v)", token))
	case errors.Is(err, io.EOF):
		return root, nil
	default:
		return nil, wrapJSONError(data, decoder, err)
	}
}

// decodeJSONValue lê um valor completo do decodificador, montando a subárvore.
func decodeJSONValue(decoder *json.Decoder, key, path string) (*Node, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			return decodeJSONObject(decoder, key, path)
		case '[':
			return decodeJSONArray(decoder, key, path)
		default:
			return nil, fmt.Errorf("delimitador inesperado %q", value)
		}
	case string:
		return &Node{Key: key, Kind: String, Value: value, Path: path}, nil
	case json.Number:
		return &Node{Key: key, Kind: Number, Value: value.String(), Path: path}, nil
	case bool:
		return &Node{Key: key, Kind: Bool, Value: strconv.FormatBool(value), Path: path}, nil
	case nil:
		return &Node{Key: key, Kind: Null, Value: "null", Path: path}, nil
	default:
		return nil, fmt.Errorf("token inesperado %v", token)
	}
}

func decodeJSONObject(decoder *json.Decoder, key, path string) (*Node, error) {
	node := &Node{Key: key, Kind: Object, Path: path}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		childKey, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("chave de objeto inválida: %v", token)
		}
		child, err := decodeJSONValue(decoder, childKey, joinPath(path, childKey))
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return node, nil
}

func decodeJSONArray(decoder *json.Decoder, key, path string) (*Node, error) {
	node := &Node{Key: key, Kind: Array, Path: path}
	for index := 0; decoder.More(); index++ {
		child, err := decodeJSONValue(decoder, "", joinPath(path, strconv.Itoa(index)))
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return node, nil
}

// wrapJSONError converte erros do pacote encoding/json em *ParseError com
// posição. Mensagens de detalhe da biblioteca padrão permanecem em inglês.
func wrapJSONError(data []byte, decoder *json.Decoder, err error) error {
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		return parseErr
	}
	message := "JSON inválido: " + err.Error()
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return newParseError(data, int(syntaxErr.Offset)-1, message)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return newParseError(data, len(data), "JSON inválido: fim inesperado")
	}
	return newParseError(data, int(decoder.InputOffset()), message)
}
