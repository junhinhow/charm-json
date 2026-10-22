# charm-json

Explorador interativo de árvores JSON/YAML para o terminal: faça o parse, recolha nós, filtre e copie caminhos JSON Pointer com um modelo Bubble Tea.

## Instalação

```bash
go get github.com/junhinhow/charm-json
```

## Uso

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	charmjson "github.com/junhinhow/charm-json"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root, err := charmjson.Parse(data, "") // "" detecta JSON ou YAML
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model := charmjson.New(root, charmjson.WithMaxDepth(8))
	model.SetSize(80, 24)

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Teclas

| Tecla | Ação |
| --- | --- |
| `j` / `↓` | Vai para a próxima linha |
| `k` / `↑` | Vai para a linha anterior |
| `enter` / `espaço` | Expande ou recolhe o contêiner selecionado |
| `h` / `←` | Recolhe o contêiner selecionado |
| `l` / `→` | Expande o contêiner selecionado |
| `g` | Vai para a primeira linha |
| `G` | Vai para a última linha |
| `/` | Inicia uma busca |
| `esc` | Limpa a busca/filtro |
| `y` | Emite `YankedMsg` com o caminho selecionado |

Durante a busca, caracteres imprimíveis editam a consulta, `backspace` remove o último rune e `enter` confirma o filtro.

## API

| Símbolo | Descrição |
| --- | --- |
| `Parse(data, format)` | Decodifica `"json"`, `"yaml"`/`"yml"` ou detecta automaticamente com `""` |
| `ParseJSON(data)` | Decodifica JSON preservando a ordem das chaves e o texto original dos números |
| `ParseYAML(data)` | Decodifica YAML preservando a ordem de definição; resolve âncoras/aliases |
| `Node` | Nó da árvore com `Key`, `Kind`, `Value`, `Children`, `Path` e `Collapsed` |
| `(*Node).Find(path)` | Procura um nó por JSON Pointer RFC 6901 |
| `(*Node).Flatten()` | Linhas visíveis em pré-ordem, respeitando `Collapsed` |
| `(*Node).Collapse/Expand/Toggle(path)` | Altera o colapso de um nó |
| `(*Node).CollapseAll()/ExpandAll()` | Altera o colapso da árvore inteira |
| `Row` | Linha renderizável com `Node`, `Depth`, `Prefix` e `Display` |
| `New(root, opts...)` | Cria o explorador Bubble Tea |
| `(*Model).SetSize(w, h)` | Define a área de renderização |
| `(*Model).SetFilter(s)` / `Filter()` | Define/lê o filtro case-insensitive |
| `(*Model).Rows()` | Linhas visíveis após colapso, filtro e limite de profundidade |
| `(*Model).SelectedPath()` | JSON Pointer do nó selecionado |
| `(*Model).SelectedValue()` | Valor escalar do nó selecionado |
| `WithMaxDepth(n)` | Limita a profundidade exibida; contêineres mais fundos aparecem como `…` |
| `WithSyntaxColors(bool)` | Liga/desliga as cores ANSI (ligadas por padrão) |
| `YankedMsg` | Mensagem emitida pela tecla `y` |
| `ParseError` | Erro de sintaxe com `Line`, `Column` e `Msg` |

## Notas de projeto

- **Nome do pacote.** O módulo é `charm-json`; o pacote importado é `charmjson`.
- **Caminhos.** A raiz é `""`; `/` endereça o filho de chave vazia. `~` e `/` são escapados como `~0` e `~1`, conforme o RFC 6901.
- **Valores escalares.** `Node.Value` guarda strings já decodificadas e números exatamente como escritos (`json.Decoder` com `UseNumber`). As aspas aparecem apenas na exibição de strings.
- **Ordenação.** JSON mantém a ordem de inserção e YAML mantém a ordem de definição; nada é ordenado alfabeticamente. A renderização nunca itera mapas.
- **Filtro.** A consulta compara caminho, chave e valor, sem diferenciar maiúsculas de minúsculas. Um nó fica visível quando casa ou é ancestral de um nó que casa. O colapso é ignorado enquanto o filtro está ativo, para manter os matches alcançáveis; sem nenhum match, a lista de linhas fica vazia.
- **Limite de profundidade.** `WithMaxDepth` só trunca a visualização: os nós abaixo do limite continuam no modelo e voltam a aparecer quando o limite muda.
- **YAML.** `yaml.Node` preserva a ordem. Âncoras e aliases são resolvidos. Chaves de mesclagem (`<<`) são mantidas como chaves comuns, sem mesclar. O `yaml.v3` informa apenas a linha, então `ParseError.Column` vale `1` quando a coluna é desconhecida.
- **Renderização.** A largura é medida e truncada com `github.com/charmbracelet/x/ansi`, então runes largos e emoji permanecem alinhados. Uma calha de duas colunas marca a seleção. As cores ANSI são constantes fixas (sem detecção de perfil do terminal), o que mantém a saída determinística; `WithSyntaxColors(false)` remove todos os códigos de escape.
- **Cópia.** `y` não acessa a área de transferência: apenas devolve um `tea.Cmd` que emite `YankedMsg{Path}` para que a aplicação hospedeira copie o valor.

## Licença

MIT
