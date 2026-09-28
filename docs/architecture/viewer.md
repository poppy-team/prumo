# Prumo IDE / Native Workspace Viewer (`prumo native`)

O **Prumo IDE** (alimentado pelo *Native Workspace Viewer* / `prumo-viewer`) é um componente de visualização gráfica desktop implementado em Rust para navegação instantânea em repositórios massivos.

## Por que Rust?

Em projetos com dezenas de milhares de arquivos, centenas de Metas e grafos complexos de dependências, interfaces baseadas em Electron ou scripts interpretados sofrem com consumo excessivo de memória e latência de renderização. O Prumo IDE entrega:

- **Tempo de Inicialização Sub-Milissegundo**: Acesso direto ao sistema de arquivos com cache em memória compartilhada.
- **Renderização Nativa com Freya & Skia**: Construído em Rust sobre Freya 0.4+ (declarativo, layout Torin, rasterização Skia) com realce via Tree-sitter.
- **Ecosistema de Extensões**: Barramento seguro via `prumo-extension-sdk` para extensões LSP/DAP.
- **Integração com o Terminal**: Funciona em total harmonia e sincronia com o **Prumo Agent** (`prumo agent` / `prumo tui`).

## Invocando o Prumo IDE

```bash
# Iniciar o Prumo IDE no repositório atual via CLI oficial
prumo native

# Invocação direta via binário visualizador
prumo-viewer .

# Iniciar focado no grafo de uma Meta específica
prumo native --goal P01-G01
```
