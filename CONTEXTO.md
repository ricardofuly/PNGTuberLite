# PNGTuber Lite — Documento de Contexto do Projeto

> **Objetivo deste documento**: Registrar detalhadamente o estado atual do projeto, a arquitetura, todas as decisões técnicas tomadas, os arquivos implementados, os testes e o histórico de evolução, garantindo continuidade e preservação de contexto em qualquer sessão de desenvolvimento.

---

## 1. Visão Geral

O **PNGTuber Lite** é uma aplicação nativa e ultra-leve em **Go** criada para renderizar e animar avatares 2D com suporte nativo aos arquivos `.save` do **PNGTuber-Plus** (originalmente feito em Godot 4.1.2 por `kaiakairos`).

### Principais Benefícios em Relação ao Original
- **Consumo Mínimo de Recursos**: Footprint de memória RAM < 30 MB (contra centenas de MB de uma engine completa como Godot).
- **Inicialização Instantânea**: Binário único compilado nativamente.
- **Overlay Transparente para OBS**: Janela sem bordas com canal Alpha nativo, permitindo captura de janela no OBS Studio sem necessidade de chroma key (tela verde).
- **Desempenho Estável**: Loop de física desacoplado em fixed timestep (60 Hz) e processamento de áudio em thread/goroutine isolada.

---

## 2. Stack Tecnológica

| Componente | Tecnologia | Detalhes |
|---|---|---|
| **Linguagem** | Go (v1.26+) | Binário nativo com alta eficiência e baixo garbage collection overhead |
| **Renderização & Janela** | [`raylib-go`](https://github.com/gen2brain/raylib-go/raylib) | Aceleração por GPU via OpenGL, suporte a janelas transparentes e sem bordas |
| **Captura de Áudio** | [`malgo`](https://github.com/gen2brain/malgo) | Binding Go para Miniaudio, baixa latência (< 25ms) e multiplataforma |
| **Parsing & Formato** | `encoding/json`, `encoding/base64`, `regexp` | Biblioteca padrão Go para deserialização defensiva do `.save` |
| **Configuração** | `pkg/config` | Persistência em arquivo JSON local (`config.json`) |
| **Internacionalização (i18n)** | `pkg/i18n` | Dicionários JSON embutidos (`embed.FS`), suporte nativo a `pt-BR` e `en-US`, detecção automática de novos idiomas e troca a quente sem reiniciar |

---

## 3. Matriz de Progresso das Fases (Conforme Seção 7 do `pngtuber-lite-projeto.md`)

| # | Fase Oficial do Projeto | Status | O que foi feito / O que falta |
|---|---|---|---|
| **1** | **Parser + modelo de dados** | ✅ **Concluído** | Structs `Vector2`, `Layer`, `Avatar`, parser defensivo JSON/Base64/IHDR, testes com `defaultAvatar.save` e `slugcat.save`. |
| **2** | **Renderer estático** | ✅ **Concluído** | `TextureCache` GPU com filtro bilinear e wrap clamp, `ComputeWorldTransforms` hierárquico (pais $\rightarrow$ filhos), ordenação por `zindex` e profundidade. |
| **3** | **Sprite-sheet** | ✅ **Concluído** | Avanço de quadros conforme `frames` e `animSpeed` no `SpriteSheetAnimator`. |
| **4** | **Blink (Piscar)** | ✅ **Concluído** | Temporizador randômico com cooldown, alternância de `showBlink` (0=sempre, 1=piscando, 2=olhos abertos). |
| **5** | **Wobble físico** | ✅ **Concluído** | Spring-damper com `rotDrag`, limites `rLimitMin/Max`, ondas senoidais idle (`xAmp/xFrq`, `yAmp/yFrq`) e stretch. |
| **6** | **Costumes (Figurinos)** | ✅ **Concluído** | `CostumeManager` para 10 slots com `costumeLayers`, hotkeys locais (`1` a `0`) e bounce opcional. |
| **7** | **Áudio/talk detection** | ✅ **Concluído** | Captura de microfone com `malgo`, VAD com RMS, histerese anti-flicker e debounce de silêncio. |
| **8** | **Janela overlay** | ✅ **Concluído** | Transparência nativa Alpha, always-on-top dinâmico (`F11`), modo borderless (`F10`), click-through (`F9`), arraste de avatar compatível com Wayland/X11, zoom (`Scroll`) e persistência em `config.json`. |
| **9** | **Empacotamento & CI** | ✅ **Concluído** | GitHub Actions CI/CD (`.github/workflows/release.yml`) com builds nativos automatizados para Linux (`amd64`) e Windows (`amd64.exe`), checksums SHA256 e release automático. |
| **10** | **Editor Visual de Avatar** | ✅ **Concluído** | Editor integrado para manipulação de nós, gizmo no canvas, árvore de camadas, física, visibilidade, spritesheets e exportação `.save`. |
| **11** | **Telemetria & Profiler F3** | ✅ **Concluído** | HUD completo com amostragem de CPU (/proc/self/stat), RAM Física (RSS), Go Heap, VRAM GPU, FPS, frametime e gráficos de sparkline em tempo real. |
| **12** | **Auto-Update & Hotfix** | ✅ **Concluído** | Verificação assíncrona no GitHub Releases, substituição atômica in-place, barra de progresso e auto-restart desacoplado no Linux e Windows. |
| **13** | **System Tray & Fechamento** | ✅ **Concluído** | Ícone na bandeja do sistema com restauração rápida e modal de confirmação para minimizar ao fechar. |
| **14** | **Internacionalização (i18n)** | ✅ **Concluído** | Suporte multi-idiomas nativo (`pt-BR`, `en-US`), descoberta dinâmica de idiomas, medição de texto e auto-dimensionamento de botões/modais. |

---

## 4. Estrutura de Arquivos e Módulos Implementados

```
/run/media/ricardo-fuly/SSD/Dev/PNGTuberLite/
├── go.mod                              # Definição do módulo Go e dependências
├── go.sum                              # Checksums de dependências
├── main.go                             # Entrypoint, controle de janela, loop principal e HUD F3
├── README.md                           # Guia rápido de uso, compilação e atalhos
├── CONTEXTO.md                         # Este documento de contexto permanente
├── pngtuber-lite-projeto.md            # Especificação original do projeto
├── assets/
│   ├── assets.go                       # Embed FS de fontes e ícones
│   ├── fonts/                          # Fontes TrueType (regular e bold)
│   ├── icons/                          # Ícones e texturas embutidas (PNG, ICO)
│   │   ├── language.png                # Ícone de idiomas para o menu
│   │   └── ...
│   └── samples/
│       ├── defaultAvatar.save          # Avatar padrão de 9 camadas extraído do PNGTuber-Plus
│       └── slugcat.save                # Avatar complexo de 22 camadas
└── pkg/
    ├── i18n/                           # [Fase 14] Internacionalização e Localização
    │   ├── i18n.go                     # Gerenciador i18n, carregamento de bundles JSON e fallback
    │   ├── i18n_test.go                # Testes unitários de bundles e alternância de idioma
    │   └── locales/                    # Dicionários de tradução embutidos
    │       ├── pt-BR.json              # Português do Brasil
    │       └── en-US.json              # Inglês Americano
    ├── model/                          # [Fase 1] Estruturas de dados e parser do .save
    │   ├── vector2.go                  # Struct Vector2 e parser de strings "Vector2(x, y)"
    │   ├── layer.go                    # Struct Layer, parâmetros físicos e regras de visibilidade
    │   ├── avatar.go                   # Struct Avatar, árvore hierárquica e ordenação por ZIndex
    │   ├── parser.go                   # Parser defensivo do JSON, decodificação Base64 e dimensões PNG
    │   ├── builder.go                  # Utilitário para construir avatares .save a partir de pastas
    │   └── parser_test.go              # Testes unitários do parser e validação do defaultAvatar.save
    ├── render/                         # [Fase 2] Motor gráfico Raylib
    │   ├── texture_cache.go            # Carregamento GPU com filtro bilinear e WrapClamp (eliminação de artefatos de borda)
    │   ├── transform.go                # Transformadas hierárquicas e rotações acumuladas de pais para filhos
    │   └── renderer.go                 # Pipeline de desenho por ZIndex, recorte de frames e ancoragem
    ├── anim/                           # [Fase 3] Física e Animações
    │   ├── wobble.go                   # Spring-damper de rotação/arrasto e oscilação senoidal (idle)
    │   ├── blink.go                    # Temporizador aleatório de piscar com cooldown anti-repetição
    │   ├── bounce.go                   # Salto global com gravidade ao falar e ao trocar figurino
    │   ├── spritesheet.go              # Avanço de frames de spritesheets conforme animSpeed
    │   ├── animator.go                 # Orquestrador central de animações em 60 Hz
    │   └── anim_test.go                # Testes unitários da física, blink, bounce e spritesheet
    ├── audio/                          # [Fase 4] Microfone e Detecção de Fala (VAD)
    │   ├── vad.go                      # Cálculo de RMS com histerese e debounce contra flickering da boca
    │   ├── capture.go                  # Stream de microfone em goroutine com malgo e flags atômicas
    │   └── vad_test.go                 # Testes unitários do cálculo RMS e histerese do VAD
    ├── costume/                        # [Fase 5] Sistema de Figurinos
    │   ├── costume.go                  # Gerenciador dos 10 slots de figurino (costumeLayers)
    │   └── costume_test.go             # Testes unitários de troca de figurino
    ├── config/                         # [Fase 5] Configurações Persistentes
    │   ├── config.go                   # Leitura e gravação de config.json com valores padrão
    │   ├── keybinds.go                 # Gerenciador de atalhos remapeáveis e tradução de códigos Raylib
    │   └── config_test.go              # Testes unitários de persistência e atalhos
    ├── window/                         # [Fase 6] Janela Overlay Transparente
    │   └── window.go                   # Configuração de flags Raylib (Alpha, Topmost, Undecorated, Passthrough)
    ├── ui/                             # [Fase 7+] Interface de Usuário / Menu de Configurações
    │   ├── ui.go                       # Menu com abas (Avatar, Áudio, Roupas, Física, Teclas, OBS), layouts responsivos com auto-ajuste e fontes TrueType
    │   ├── icons.go                    # Atlas e carregador de ícones GPU nativos
    │   └── ui_test.go                  # Testes unitários de renderização de ícones e assets
    ├── profiler/                       # [Fase 11] Telemetria e Profiler de Recursos
    │   ├── profiler.go                 # Amostragem em tempo real de CPU (ticks /proc), RAM Física (RSS), Go Heap e VRAM GPU
    │   └── profiler_test.go            # Testes unitários do profiler
    ├── tray/                           # [Fase 13] Integração com System Tray (Bandeja do Sistema)
    │   ├── tray.go                     # Suporte cross-platform (Windows e Linux) com menu de contexto e restauração rápida
    │   └── tray_test.go                # Testes unitários de callbacks do system tray
    ├── updater/                        # [Fase 12] Auto-Update e Sistema de Hotfix
    │   ├── updater.go                  # Verificação assíncrona no GitHub Releases, in-place update e extração de binários
    │   ├── restart_unix.go             # Execução de processo desacoplado (Setsid) no Linux/Unix
    │   ├── restart_windows.go          # Execução de processo desacoplado (CREATE_NEW_PROCESS_GROUP) no Windows
    │   └── updater_test.go             # Testes unitários do parser de versão e extração de tar.gz
    └── editor/                         # [Fase 10] Editor Visual de Avatares
        ├── editor.go                   # Editor completo de camadas, hierarquia, propriedades, gizmo no canvas, drag & drop e exportação .save
        └── editor_test.go              # Testes unitários do editor de avatares
```

---

## 4. Decisões Técnicas e Lógica de Negócio

### 4.1 Parser do Formato `.save`
- O arquivo `.save` é um JSON com chaves em string (`"0"`, `"1"`, ...).
- **`Vector2(x, y)`**: valores serializados como string pelo Godot são extraídos via expressão regular `Vector2\(\s*([-\d.eE+]+)\s*,\s*([-\d.eE+]+)\s*\)`.
- **`costumeLayers`**: array serializado em string como `"[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]"` é decomposto em `[10]int`.
- **`imageData`**: string Base64 é decodificada uma única vez para `[]byte` no carregamento. O cabeçalho PNG (IHDR) é lido para obter `width` e `height` instantaneamente sem overhead.
- **Hierarquia**: camadas com `parentId` válido são associadas aos nós pais. Camadas sem pai tornam-se `RootLayers`.
- **Ordem de Desenho (DrawOrder)**:
  1. Primeiro pelo campo `zindex` (menor = atrás, maior = na frente).
  2. Em caso de empate no `zindex`, a ordenação avalia a **profundidade na árvore (TreeDepth)**: os pais são desenhados antes dos filhos, garantindo que olhos, boca e acessórios fiquem renderizados **sobre a face da cabeça**, e não cobertos por ela.
  3. Desempate final determinístico por `identification`.

### 4.2 Regras de Visibilidade de Camadas
Uma camada só é desenhada se todas as 3 condições forem verdadeiras:
1. **Figurino**: `layer.CostumeLayers[slot - 1] == 1`
2. **Piscar (`showBlink`)**:
   - `0`: Sempre visível.
   - `1`: Visível apenas quando o avatar **não está piscando** (`isBlinking == false`, olhos normais abertos).
   - `2`: Visível apenas quando o avatar **está piscando** (`isBlinking == true`, olhos fechados).
3. **Fala (`showTalk`)**:
   - `0`: Sempre visível.
   - `1`: Visível apenas quando o avatar **está em silêncio** (`isTalking == false`, boca fechada).
   - `2`: Visível apenas quando o avatar **está falando** (`isTalking == true`, boca aberta).

### 4.3 Sistema de Física (Wobble, Oscilação e Bounce)
- **Massa-Mola-Amortecedor (`rotDrag`)**:
  $$\text{angAccel} = (0 - \theta) \cdot k_{\text{stiffness}} - \omega \cdot \text{rotDrag}$$
  O ângulo resultante é limitado estritamente entre `rLimitMin` e `rLimitMax`.
- **Oscilação Senoidal Idle (Flutuação / Respiração)**:
  $$\text{offset}_X = x_{\text{amp}} \cdot \text{bobbingIntensity} \cdot \sin(2\pi \cdot x_{\text{frq}} \cdot \text{ticks})$$
  $$\text{offset}_Y = y_{\text{amp}} \cdot \text{bobbingIntensity} \cdot \sin(2\pi \cdot y_{\text{frq}} \cdot \text{ticks})$$
- **Controle de Intensidade de Flutuação e Inércia**:
  - `bobbingIntensity`: multiplicador ajustável de $0.0\text{x}$ (completamente estático) a $2.0\text{x}$ (padrão suave $0.4\text{x}$ - $0.5\text{x}$).
  - `wobbleIntensity`: multiplicador de inércia angular e amortecimento.
- **Ciclo de Piscar Humano (Blink)**: Duração rápida e natural de **0.09s** com intervalo médio entre piscadas de **~14 segundos** (randomizado entre 10s e 18s).
- **Bounce Global**: Impulso vertical com gravidade configurável ($g = 1000$) acionado ao começar a falar ou trocar de figurino.

### 4.4 Detecção de Fala (VAD) com Histerese e Debounce
- **Remoção de Offset DC**: Subtração da média do sinal PCM antes do cálculo de RMS para eliminar ruído estático de placas de som e microfones embutidos.
- **Histerese e Debounce**: Limiar de desativação a $80\%$ e debounce de 6 quadros (~100 ms) para fechamento ágil e imediato da boca ao parar de falar.
- **Suporte a Fontes Virtuais e DSP (Easy Effects)**: Enumeração e priorização automática de fontes virtuais (`Easy Effects Source`) sobre monitores loopback, permitindo troca dinâmica do dispositivo de áudio em tempo real pelo menu.

### 4.5 Ancoragem Proporcional e Auto-Clamping de Janela
- **Coordenadas Relativas Normalizadas (`AvatarRelX`, `AvatarRelY`)**: O avatar armazena sua posição como razão entre $0.0$ e $1.0$ em relação à largura e altura da janela (padrão $0.5, 0.5$ no centro).
- **Reancoragem Dinâmica em Tempo Real**: Ao maximizar, restaurar ou redimensionar a janela, o aplicativo recalcula as coordenadas absolutas na proporção exata da nova dimensão, mantendo o avatar sempre visível e no mesmo ponto relativo.
- **Auto-Clamping com Margem Segura**: Limita o avatar dentro das margens visíveis da tela para impedir que ele seja arrastado ou empurrado para fora do viewport visível.

### 4.6 Sistema de Teclas Remapeáveis (Keybinds)
- **Mapeamento Flexível**: Todos os comandos do aplicativo (abrir menu, editor, HUD, transparência, overlays, reset de posição, sensibilidade de mic) são definidos na struct `Keybinds` e salvos no `config.json`.
- **Interface Interativa de Rebind**: A aba **`Teclas`** permite clicar em qualquer atalho e pressionar uma nova tecla no teclado para capturar e persistir a nova configuração instantaneamente, com botão para restaurar os padrões.

### 4.7 Telemetria e Profiler em Tempo Real (`F3`)
- **CPU**: Amostragem periódica de ticks de CPU do processo via `/proc/self/stat` normalizada pela quantidade de cores lógicos com barra de progresso suave e sparkline histórico com scroll contínuo.
- **RAM Física (RSS)**: Leitura de páginas residentes via `/proc/self/statm` combinada com métricas de heap Go (`runtime.MemStats`).
- **GPU e Render**: Monitoramento de tempo de renderização em milissegundos por quadro, contagem de texturas ativas e cálculo de VRAM dedicada.

### 4.8 Auto-Update In-Place, Hotfix e Auto-Restart
- **Repositório Oficial**: [`https://github.com/ricardofuly/PNGTuberLite`](https://github.com/ricardofuly/PNGTuberLite)
- **Verificação Assíncrona no Startup**: O pacote `pkg/updater` consulta as releases do GitHub em background e alerta o usuário na UI com o botão flutuante `[ ATUALIZAR ]`.
- **Substituição Atômica In-Place**: O atualizador baixa o `.tar.gz` ou `.zip` do sistema operacional correspondente, extrai o novo binário e renomeia o executável ativo com permissões `0755`.
- **Reinício Automático Multiplataforma**: Spawning desacoplado do novo processo via `Setsid` (Linux/Unix) e `CREATE_NEW_PROCESS_GROUP` (Windows).

### 4.9 Internacionalização e Layouts Adaptativos (`pkg/i18n`)
- **Arquitetura i18n Desacoplada**: Dicionários JSON embutidos (`embed.FS`) com carregamento automático de metadados (`LanguageMeta`), bandeiras e títulos nativos.
- **Descoberta Automática de Idiomas**: Adicionar qualquer novo arquivo `pkg/i18n/locales/{code}.json` faz o novo idioma aparecer instantaneamente na interface e no seletor de idiomas sem nenhuma alteração no código Go.
- **Layouts e Botões com Auto-Ajuste**: Medição de fontes em tempo real (`MeasureTextBold`) para que botões, títulos de abas, cápsulas de idiomas e modais nunca fiquem espremidos ou com texto cortado.

---

## 5. Status dos Testes Automatizados

Todos os pacotes possuem testes unitários implementados e aprovados:

```bash
go test -v ./...
```

| Pacote | Testes | Status |
|---|---|---|
| `pkg/model` | `TestParseVector2`, `TestParseCostumeLayers`, `TestPNGDimensionsExtraction`, `TestParseSaveDataAndHierarchy`, `TestParseRealDefaultAvatar`, `TestSaveAndReloadAvatar`, `TestBuildAvatarFromSlugcatDirectory` | ✅ PASS |
| `pkg/anim` | `TestBlinkController`, `TestBounceController`, `TestSpriteSheetAnimator`, `TestWobbleAngleLimits` | ✅ PASS |
| `pkg/audio` | `TestCalculateRMS`, `TestVADHysteresisAndDebounce` | ✅ PASS |
| `pkg/costume` | `TestCostumeManager` | ✅ PASS |
| `pkg/config` | `TestConfigLoadSave`, `TestKeybinds` | ✅ PASS |
| `pkg/render` | `TestComputeWorldTransforms` (transformadas hierárquicas e rotação de nós) | ✅ PASS |
| `pkg/ui` | `TestEmbeddedAssets` (validação de todos os ícones PNG, ICO e Logo) | ✅ PASS |
| `pkg/tray` | `TestTrayManagerSignals` (registro de callbacks e eventos do tray) | ✅ PASS |
| `pkg/editor` | `TestEditorStateNewAndModify` (adição e modificação de camadas) | ✅ PASS |
| `pkg/updater` | `TestIsNewerVersion`, `TestExtractExecutableFromArchive` | ✅ PASS |
| `pkg/profiler` | `TestSystemProfiler` | ✅ PASS |
| `pkg/i18n` | `TestI18nBundlesAndSwitching` (validação de chaves e alternância dinâmica) | ✅ PASS |

---

## 6. Guia de Compilação e Execução

### Compilação do Binário
```bash
go build -ldflags="-s -w" -o pngtuber-lite main.go
```

### Execução
```bash
# Execução padrão com avatar de exemplo
./pngtuber-lite -avatar assets/samples/defaultAvatar.save

# Execução especificando limiar de sensibilidade do microfone
./pngtuber-lite -avatar assets/samples/defaultAvatar.save -threshold 0.04
```

### Controles no App
- `1` a `9`, `0`: Alternar figurinos 1 a 10.
- `Espaço`: Testar piscar forçado e salto (bounce).
- `Botão Esquerdo ou Direito do Mouse`: Arrastar e posicionar o avatar livremente na tela (compatível com Wayland e X11).
- `Scroll do Mouse`: Ajustar zoom/escala do avatar.
- `R`: Resetar posição e escala para o centro.
- `F9`: Alternar modo **Click-Through** (cliques atravessam o avatar).
- `F10`: Alternar modo **Sem Bordas (Borderless)** / Com Bordas.
- `F11`: Alternar modo **Sempre no Topo (Always-On-Top)**.
- `F3`: Ativar/desativar painel de telemetria e profiler HUD.
- `M`: Abrir/fechar gaveta de configurações.
- `E`: Abrir/fechar editor visual de avatar.

---

## 7. Changelog de Evolução

- **v1.2.0**:
  - Implementação do módulo completo de internacionalização (`pkg/i18n`) com suporte nativo a Português (`pt-BR`) e Inglês (`en-US`).
  - Descoberta automática de novos idiomas adicionados na pasta `locales/`.
  - Botões, abas, modais e cápsulas com medição de fontes e layout dinâmico sem clipping de texto.
  - Correção e polimento na rolagem dos cards de configurações.
  - Tradução integral do Editor de Avatares, Telemetria F3 e modais de confirmação.
- **v1.1.0**:
  - Implementação da telemetria de CPU/RAM/VRAM/FPS em tempo real no HUD F3 com barras de progresso e sparklines com interpolação suave.
  - Otimizações de segurança (limite de memória e proteção contra zip bombs, sanitização de caminhos contra path traversal).
  - Correção da posição do gizmo no editor e desvio de camadas filhas no canvas.
- **v1.0.0 a v1.0.9**:
  - Lançamento inicial, auto-update in-place, ícones integrados e suporte a system tray.

