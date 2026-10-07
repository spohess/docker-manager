# docker-manager

CLI em Go para operar o `docker compose` em vários projetos ao mesmo tempo:
subir só os serviços básicos de cada projeto, subir/derrubar grupos de projetos
(backend + fronts) e executar comandos no container da aplicação sem digitar o
`docker exec -ti <container>` toda vez.

## Instalação

```bash
make build           # gera ./bin/docker-manager
make install         # instala em ~/.local/bin e cria o atalho dm
```

Requisitos: Go 1.26+ e `docker` com o plugin `compose` no PATH.

## Configuração

```yaml
default: backend

projects:
  - name: backend
    path: ~/projects/backend
    app: backend-app
    profiles:
      basic: [backend-app, backend-db]
      queue: [backend-app, backend-db, backend-redis, backend-horizon]

  - name: admin
    path: ~/projects/frontend-admin

  - name: client
    path: ~/projects/frontend-client
    files: [docker-compose.dev.yml]

groups:
  system: [backend, admin, client]
```

| chave | obrigatória | descrição |
| --- | --- | --- |
| `default` | não | projeto(s) ou grupo usado quando não há `-p` nem projeto no diretório atual; aceita lista separada por vírgula |
| `projects[].name` | sim | nome usado em `-p nome` |
| `projects[].path` | sim | diretório do `docker-compose.yml` (aceita `~` e variáveis de ambiente) |
| `projects[].app` | não | nome do **container** usado pelo comando `app` |
| `projects[].files` | não | arquivos compose passados com `-f`, relativos ao `path` |
| `projects[].profiles` | não | listas de serviços; o perfil `basic` é o padrão de `up`, `stop` e `restart` |
| `groups` | não | nome → lista ordenada de projetos |

Ordem de busca do arquivo quando `--config` não é informado:

1. variável de ambiente `DOCKER_MANAGER_CONFIG`;
2. `./docker-manager.yml` (e `.yaml`);
3. `~/.config/docker-manager/config.yml` (e `.yaml`);
4. `~/.docker-manager.yml` (e `.yaml`).

## Seleção de projetos

1. `-p`/`--project`, quando informado: aceita projetos e grupos separados por
   vírgula (`-p backend,admin`, `-p system`), na ordem informada e sem
   repetição;
2. o projeto cujo `path` contém o diretório atual (o mais específico vence);
3. a chave `default`.

## Comandos

| comando | executa em cada projeto selecionado |
| --- | --- |
| `up` | `docker compose up -d <serviços do perfil>` |
| `down` | `docker compose down`, em ordem reversa |
| `stop` | `docker compose stop <serviços do perfil>`, em ordem reversa |
| `restart` | `docker compose restart <serviços do perfil>` |
| `ps` | `docker compose ps` |
| `logs [args]` | `docker compose logs -f [args]` (um único projeto) |
| `compose <args>` | `docker compose <args>` no diretório do projeto (um único projeto) |
| `app <cmd>` | `docker exec -it <app> <cmd>` |

`up`, `stop` e `restart` usam o perfil `basic`; projetos sem o perfil usam
todos os serviços. `-f`/`--full` ignora o perfil e `--profile=nome` escolhe
outro (precisa existir em pelo menos um dos projetos selecionados).

`up` segue a ordem da seleção (o backend sobe antes dos fronts que dependem da
rede dele); `down` e `stop` seguem a ordem inversa. Os projetos são processados
em sequência, o resumo final lista o resultado de cada um e o processo sai com
código 1 se algum falhar.

`app` exige que exatamente um dos projetos selecionados tenha a chave `app`;
assim `dm -p system app ...` usa o container do backend. O `-t` só é usado
quando stdin e stdout são terminal, permitindo `dm app php artisan x | grep y`.

`app`, `logs` e `compose` substituem o processo do `dm` pelo `docker`, então
TTY, Ctrl+C e o código de saída são os do próprio comando.

### Parâmetros

| parâmetro | efeito |
| --- | --- |
| `-p`, `--project=nome` | projeto(s) ou grupo(s), separados por vírgula |
| `-f`, `--full` | todos os serviços (`up`, `stop`, `restart`) |
| `--profile=nome` | outro perfil em vez do `basic` (`up`, `stop`, `restart`) |
| `--build` | acrescenta `--build` ao `up` |
| `--config=arquivo.yml` | caminho do arquivo de configuração |
| `--dry-run` | mostra os comandos sem executar |

Nos comandos `app`, `logs` e `compose`, os parâmetros do `dm` vêm antes do
comando ou logo após ele; do primeiro argumento em diante tudo é repassado ao
docker. Logo após o comando só são aceitos `-p`/`--project`, `--config` e
`--dry-run`; qualquer outra flag já é repassada (`dm compose -f x.yml up`).
Use `--` para repassar um argumento que coincida com esses parâmetros.

## Exemplos

```bash
dm up                                 # default, perfil basic
dm up -p backend -f                   # todos os serviços do backend
dm up -p backend,admin --build
dm up -p system                       # backend, admin e client
dm down -p system                     # client, admin e backend
dm restart --profile=queue
dm app php artisan migrate
dm -p backend app php artisan tinker
dm app -p backend composer install
dm logs backend-app --tail=100
dm compose pull
dm up -p system --dry-run
```

## Desenvolvimento

```bash
make fmt
make vet
make test
```
