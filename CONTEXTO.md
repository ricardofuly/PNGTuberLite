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

---

## 3. Matriz de Progresso das Fases (Conforme Seção 7 do `pngtuber-lite-projeto.md`)

| # | Fase Oficial do Projeto | Status | O que foi feito / O que falta |
|---|---|---|---|
| **1** | **Parser + modelo de dados** | ✅ **Concluído** | Structs `Vector2`, `Layer`, `Avatar`, parser defensivo JSON/Base64/IHDR, testes com `defaultAvatar.save`. |
| **2** | **Renderer estático** | ✅ **Concluído** | `TextureCache` GPU, `ComputeWorldTransforms` hierárquico (pais $\rightarrow$ filhos), ordenação por `zindex`. |
| **3** | **Sprite-sheet** | ✅ **Concluído** | Avanço de quadros conforme `frames` e `animSpeed` no `SpriteSheetAnimator`. |
| **4** | **Blink (Piscar)** | ✅ **Concluído** | Temporizador randômico com cooldown, alternância de `showBlink` (0=sempre, 1=piscando, 2=olhos abertos). |
| **5** | **Wobble físico** | ✅ **Concluído** | Spring-damper com `rotDrag`, `drag`, limites `rLimitMin/Max`, ondas senoidais idle (`xAmp/xFrq`, `yAmp/yFrq`) e stretch. |
| **6** | **Costumes (Figurinos)** | ✅ **Concluído** | `CostumeManager` para 10 slots com `costumeLayers`, hotkeys locais (`1` a `0`) e bounce opcional. |
| **7** | **Áudio/talk detection** | ✅ **Concluído** | Captura de microfone com `malgo`, VAD com RMS, histerese anti-flicker e debounce de silêncio. |
| **8** | **Janela overlay** | ✅ **Concluído** | Transparência nativa Alpha, always-on-top dinâmico (`F11`), modo borderless (`F10`), click-through (`F9`), arraste de avatar compatível com Wayland/X11, zoom (`Scroll`) e persistência em `config.json`. |
| **9** | **Empacotamento & CI** | ✅ **Concluído** | `Makefile` automatizado, GitHub Actions CI/CD (`.github/workflows/release.yml`) com builds nativos para Linux (`amd64`) e Windows (`amd64.exe`), e testes de regressão hierárquica. |

---

## 4. Estrutura de Arquivos e Módulos Implementados

```
/run/media/ricardo-fuly/SSD/Dev/PNGTuberLite/
├── go.mod                              # Definição do módulo Go e dependências
├── go.sum                              # Checksums de dependências
├── main.go                             # Entrypoint, controle de janela, loop principal e HUD
├── README.md                           # Guia rápido de uso, compilação e atalhos
├── CONTEXTO.md                         # Este documento de contexto permanente
├── pngtuber-lite-projeto.md            # Especificação original do projeto
├── assets/
│   └── samples/
│       └── defaultAvatar.save          # Avatar padrão de 9 camadas extraído do PNGTuber-Plus
└── pkg/
    ├── model/                          # [Fase 1] Estruturas de dados e parser do .save
    │   ├── vector2.go                  # Struct Vector2 e parser de strings "Vector2(x, y)"
    │   ├── layer.go                    # Struct Layer, parâmetros físicos e regras de visibilidade
    │   ├── avatar.go                   # Struct Avatar, árvore hierárquica e ordenação por ZIndex
    │   ├── parser.go                   # Parser defensivo do JSON, decodificação Base64 e dimensões PNG
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
    │   └── ui.go                       # Menu com abas (Avatar, Áudio, Roupas, Física, Teclas, OBS), TrueType anti-aliased com suporte total a UTF-8 e rebind interativo
    ├── profiler/                       # [Fase 11] Telemetria e Profiler de Recursos
    │   ├── profiler.go                 # Amostragem em tempo real de CPU (ticks /proc), RAM Física (RSS), Go Heap e VRAM GPU
    │   └── profiler_test.go            # Testes unitários do profiler
    ├── updater/                        # [Fase 12] Auto-Update e Sistema de Hotfix
    │   ├── updater.go                  # Verificação assíncrona no GitHub Releases, in-place update e extração de binários
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

### 4.7 Telemetria e Profiler em Tempo Real (`H` / `F1`)
- **CPU**: Amostragem periódica de ticks de CPU do processo via `/proc/self/stat` normalizada pela quantidade de cores lógicos.
- **RAM Física (RSS)**: Leitura de páginas residentes via `/proc/self/statm` (inclui alocações CGo, Raylib e drivers OpenGL/Mesa) combinada com métricas de heap Go (`runtime.MemStats`).
- **GPU e Render**: Monitoramento de tempo de renderização em milissegundos por quadro, contagem de texturas ativas e cálculo de VRAM dedicada.

### 4.8 Auto-Update In-Place, Hotfix e CI/CD
- **Repositório Oficial**: [`https://github.com/ricardofuly/PNGTuberLite`](https://github.com/ricardofuly/PNGTuberLite)
- **Verificação Assíncrona no Startup**: O pacote `pkg/updater` consulta as releases do GitHub em background e alerta o usuário na UI com o botão flutuante `[ 🚀 ATUALIZAR ]`.
- **Substituição Atômica In-Place**: O atualizador baixa o `.tar.gz` ou `.zip` do sistema operacional correspondente, extrai o novo binário e renomeia o executável ativo sem que o usuário precise reinstalar ou baixar manualmente.
- **Workflows GitHub Actions**:
  - `ci.yml`: Validação e execução contínua de toda a suíte de testes em cada push/PR.
  - `release.yml`: Compilação nativa para Linux e Windows com injeção automática de versão e publicação de release com checksums SHA256.
  - `hotfix.yml`: Disparador automatizado de tags para publicação instantânea de patches emergenciais.

---

## 5. Status dos Testes Automatizados

Todos os pacotes possuem testes unitários implementados e aprovados:

```bash
go test -v ./pkg/model/... ./pkg/anim/... ./pkg/audio/... ./pkg/costume/... ./pkg/config/...
```

| Pacote | Testes | Status |
|---|---|---|
| `pkg/model` | `TestParseVector2`, `TestParseCostumeLayers`, `TestPNGDimensionsExtraction`, `TestParseSaveDataAndHierarchy`, `TestParseRealDefaultAvatar` | ✅ PASS |
| `pkg/anim` | `TestBlinkController`, `TestBounceController`, `TestSpriteSheetAnimator`, `TestWobbleAngleLimits` | ✅ PASS |
| `pkg/audio` | `TestCalculateRMS`, `TestVADHysteresisAndDebounce` | ✅ PASS |
| `pkg/costume` | `TestCostumeManager` | ✅ PASS |
| `pkg/config` | `TestConfigLoadSave` | ✅ PASS |
| `pkg/render` | `TestComputeWorldTransforms` (transformadas hierárquicas e rotação de nós) | ✅ PASS |

---

## 6. Guia de Compilação e Execução

### Dependências de Desenvolvimento no Linux
```bash
sudo apt update && sudo apt install -y libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libwayland-dev libxkbcommon-dev libasound2-dev
```

### Compilação do Binário
```bash
go build -tags noaudio -ldflags="-s -w" -o pngtuber-lite main.go
```
> **Nota técnica**: A flag `-tags noaudio` desativa o módulo de áudio embutido do `raylib-go` para evitar colisão de símbolos com o `miniaudio` do `malgo` durante a linkedição CGO.

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
- `+` / `-` ou `PageUp` / `PageDown`: Ajustar limiar de sensibilidade do microfone.
- `H` ou `F1`: Ativar/desativar painel de depuração HUD.

---

## 7. Próximos Passos e Roadmap

1. **Configuração de Hotkeys Globais**: Suporte a atalhos globais de teclado no sistema operacional (fora de foco) para troca de figurinos durante transmissões ao vivo.
2. **Integração com WebSocket / Elgato Stream Deck**: API leve local para troca remota de figurinos via Stream Deck ou OBS WebSocket.
3. **Texture Atlas (Otimização Avançada)**: Agrupar sprites em uma única textura GPU para avatares com dezenas de camadas.
4. **Interface Gráfica de Configuração (Janela de Controle)**: Menu opcional para seleção de dispositivo de áudio por lista suspensa e carregamento de avatares com file picker nativo.

---

## 8. Changelog de Evolução

- **2026-08-27**:
  - Leitura e análise do documento [pngtuber-lite-projeto.md](pngtuber-lite-projeto.md).
  - Inicialização do módulo Go (`go.mod`) e download das dependências `raylib-go` e `malgo`.
  - Elaboração do plano arquitetural [plano-implementacao.md](plano-implementacao.md).
  - **Fase 1**: Implementação do modelo de dados (`Vector2`, `Layer`, `Avatar`) e parser tolerante a JSON/Base64/Godot com testes.
  - **Fase 2**: Implementação do `TextureCache` GPU, cálculo de transformadas hierárquicas e renderizador 2D `Renderer`.
  - **Fase 3**: Implementação dos controladores de animação e física (`WobbleSystem`, `BlinkController`, `BounceController`, `SpriteSheetAnimator`, `Animator`).
  - **Fase 4**: Implementação da captura de microfone com `malgo` e detector VAD com histerese e debounce.
  - **Fase 5**: Implementação do gerenciador de figurinos (`CostumeManager`) e persistência de configuração (`Config`).
  - **Fase 6**: Implementação da janela overlay transparente (`WindowManager`), HUD interativo e entrypoint completo em `main.go`.
  - Extração e teste end-to-end do arquivo real `defaultAvatar.save` com 9 camadas.
  - Criação da documentação completa em `README.md` e deste arquivo `CONTEXTO.md`.
