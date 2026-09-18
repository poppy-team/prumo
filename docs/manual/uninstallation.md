# Manual de Desinstalação

## Desinstalação com uma linha

Baixe e revise o removedor. O modo `pure` remove somente estado gerenciado e preserva todos os arquivos de projetos:

```bash
curl --fail --location https://raw.githubusercontent.com/raillen/prumo/main/scripts/uninstall.sh -o uninstall.sh
less uninstall.sh
sh uninstall.sh --mode pure --dry-run
sh uninstall.sh --mode pure
```

Para remover também o executável encontrado em `PATH`:

```bash
sh uninstall.sh --mode full --remove-binary --dry-run
sh uninstall.sh --mode full --remove-binary
```

## Regra de segurança (Uninstall safety)

A desinstalação do Prumo nunca remove dados de projetos.

Permanecem intocados:

- `.ai/`;
- `prumo.json`;
- `docs/`;
- Goals, Plans, Tasks, Evidence;
- histórico Git;
- arquivos criados pelo usuário.

Remoção de projeto continua sendo decisão explícita do usuário com as ferramentas normais do sistema operacional e do Git.

## Remoção básica

```bash
prumo-agent uninstall
prumo-agent --home ~/.prumo uninstall
```

Esse comando atualiza o manifest global e remove somente estado de instalação diretamente rastreado pelo Prumo.

## Remover conectores

```bash
prumo-agent uninstall --connectors
```

Um conector só é removido quando possui cleanup manifest com caminhos criados pelo próprio Prumo. Caminhos fora do manifest, arquivos modificados pelo usuário, ou ownership incerto são reportados como leftovers em vez de apagados.

## Limpar cache e estado derivado

```bash
prumo-agent uninstall --purge-cache
```

Remove cache, logs locais, bases derivadas e arquivos temporários sob `PRUMO_HOME`. Esse comando não altera o manifest principal a menos que `--purge-global-config` também seja usado.

## Remover configuração global

```bash
prumo-agent uninstall --purge-global-config
```

Remove `PRUMO_HOME/config`. A execução seguinte cria um estado limpo através de `prumo-agent setup`. Dados de projetos continuam preservados.

## Remoção completa do binário

O CLI não apaga seu próprio executável. Remova o binário instalado pelo procedimento normal do sistema:

```bash
rm /usr/local/bin/prumo
```

Remove também o diretório home somente quando ele não contiver mais nada necessário:

```bash
rm -rf ~/.prumo
```

Confirme antes que o diretório não contenha segredos, chaves, backups locais ou runtime packages ainda necessários.

## Verificação pós-remoção

```bash
command -v prumo || echo "binário removido"
test ! -e ~/.prumo/config/installation.json && echo "configuração global removida"
git -C ./my-project status --short
```

O último comando deve mostrar que o projeto permanece funcional e versionado, mesmo sem o Prumo instalado globalmente.

## Recuperação de erro

Se uma remoção for interrompida:

1. Execute `prumo-agent setup` em um home temporário e novo.
2. Compare os manifests gerados.
3. Restaure apenas backups registrados no cleanup manifest.
4. Nunca restaure um backup sobre um arquivo modificado pelo usuário sem confirmar conteúdo e checksum.
