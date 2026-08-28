# PNGTuber Lite

**PNGTuber Lite** é uma engine nativa, leve e de alto desempenho para avatares 2D compatível com os arquivos `.save` do **PNGTuber-Plus**. Desenvolvida em **Go** com renderização acelerada por GPU via **Raylib** e captura de áudio com baixa latência via **Miniaudio** (`malgo`).

---

## ✨ Recursos

- 🚀 **Ultra-leve e Instantâneo**: Consumo mínimo de memória RAM (< 30MB) e uso desprezível de CPU.
- 🌐 **Suporte a Múltiplos Idiomas (i18n)**: Suporte nativo a Português (`pt-BR`) e Inglês (`en-US`), com troca instantânea em tempo real e descoberta automática de novos idiomas adicionados em `pkg/i18n/locales/`.
- 🎨 **Compatibilidade Nativa com `.save`**: Lê e exporta arquivos `.save` do PNGTuber-Plus sem necessidade de conversão manual.
- 🗂️ **Ícones Nativos & Ícone Oficial do App**: Renderização de ícones embutidos via GPU e ícone oficial integrado para Windows (`.ico`/`.syso`) e Linux (`rl.SetWindowIcon`).
- 📌 **Bandeja do Sistema (System Tray)**: Permite minimizar o app para a bandeja com menu rápido ("Abrir PNGTuber Lite" e "Sair") e confirmação inteligente ao fechar.
- 📊 **Telemetria & Profiler em Tempo Real (`F3`)**: HUD com amostragem de CPU (% do processo e sistema), RAM Física (RSS), Go Heap, VRAM GPU, frametimes com gráficos de sparkline em tempo real e barras de progresso com interpolação suave.
- ⚙️ **Painel de Controle Flutuante (Menu UI)**: Pressione `M`, `TAB`, `ESC` ou clique no botão flutuante no canto da tela para abrir o menu com abas:
  - 📁 **Avatar**: Seleção e troca de avatares `.save`, criação de novo avatar, importação de pastas PNG, alternador de idioma e espelhamento horizontal.
  - 🎙️ **Áudio**: VU Meter com volume ao vivo, seletor de microfones do SO e calibração de sensibilidade.
  - 👔 **Roupas**: Botões dos 10 figurinos com alternância instantânea.
  - 🌀 **Física**: Sliders de força do pulo, gravidade, inércia (wobble), respiração e velocidade do piscar.
  - ⌨️ **Teclas**: Visualização e remapeamento interativo de todos os atalhos.
  - 🪟 **OBS**: Ativação do modo overlay em 1 clique, seletor de fundos (Transparente, Verde Chroma Key, Magenta, Azul) e guia passo a passo.
- 🌊 **Física Hierárquica Completa**: Propagação de inércia do corpo para a cabeça e da cabeça para os cabelos e roupas com spring-damper, arrasto e esticamento dinâmico.

---

## 🚀 Compilação e Execução

### Executar diretamente:
```bash
go run .
```

### Compilar o executável nativo:
```bash
go build -ldflags="-s -w" -o pngtuber-lite main.go
```

### Ou via Makefile:
```bash
make
```

---

### 🎮 Executando o Avatar:
```bash
./pngtuber-lite -avatar assets/samples/defaultAvatar.save
# ou
./run.sh -avatar assets/samples/defaultAvatar.save
```

---

## 🎨 Editor Visual de Avatares Integrado

O PNGTuber Lite inclui um **Editor Visual Completo**, permitindo criar novos avatares ou customizar avatares existentes com facilidade:

- **Como abrir o Editor**: Clique no botão flutuante **`[ ✏ EDITOR ]`** no topo, no botão da aba de avatares, ou pressione a tecla **`E`** (ou **`F2`**).
- **Arrastar e Soltar (Drag & Drop)**:
  - Arraste qualquer arquivo **`.png`** para a janela para adicionar uma nova camada instantaneamente.
  - Arraste qualquer arquivo **`.save`** para carregar um avatar diretamente no editor.
- **Hierarquia de Camadas**:
  - Ajuste de ordem de renderização (`Z-Index + / -`).
  - Vinculação de nós pais/filhos (`Parent: Corpo -> Cabeça -> Olhos/Boca`).
  - Duplicar e remover camadas.
- **Painel de Propriedades da Camada**:
  - **Posicionamento e Pivô**: Ajuste de posição (X, Y) e offset de rotação via sliders ou **arrastando o gizmo diretamente no canvas com o mouse**!
  - **Visibilidade**: Configure se a camada aparece em repouso (silêncio), falando (boca aberta), com olhos normais ou piscando.
  - **Figurinos**: Selecione em quais dos 10 figurinos a camada está ativa.
  - **Física**: Inércia angular (`rotDrag`), limites de rotação (`rLimitMin`/`rLimitMax`), respiração senoidal (`YAmp`), elasticidade (`stretch`) e pulo.
  - **SpriteSheets**: Suporte a animações em spritesheets com configuração de número de quadros e velocidade (FPS).
- **Salvar**: Clique em **`[ 💾 SALVAR ]`** para gerar o arquivo `.save` 100% compatível tanto com o PNGTuber Lite quanto com o PNGTuber-Plus!

---

## 🎮 Controles e Atalhos (100% Customizáveis na aba `Teclas`)

Todos os atalhos de teclado podem ser visualizados e remapeados facilmente na aba **`Teclas`** do menu de configurações! Os atalhos padrão são:

| Atalho / Tecla | Ação |
| :--- | :--- |
| **`E`** ou **`F2`** | Abrir / Fechar o **Editor Visual de Avatar** |
| **`M`** ou **`TAB`** | Abrir / Fechar o **Menu de Configurações (Drawer)** |
| **`ESC`** | Fechar menu ou editor ativo |
| **`F3`** | Alternar Painel de **Telemetria & Profiler HUD** (CPU, RAM, GPU, Frametimes) |
| **`1` a `9`, `0`** | Trocar entre os Figurinos 1 a 10 |
| **`Botão Esquerdo/Direito` (Arrastar)** | Mover o avatar na tela |
| **`Scroll do Mouse`** | Zoom / Escala do avatar (0.1x a 5.0x) |
| **`F9`** | Alternar Modo **Click-Through** (clicar através do avatar) |
| **`F10`** | Alternar Janela **Sem Bordas (Borderless Overlay)** |
| **`F11`** | Alternar **Sempre no Topo (Always on Top)** |
| **`+` / `-`** | Ajustar sensibilidade do microfone |
| **`Espaço`** | Testar pulo e piscar |
| **`R`** | Resetar posição e escala do avatar para o centro |

---

## 🚀 Sistema de Auto-Update e Auto-Restart Integrado

O PNGTuber Lite inclui um sistema inteligente de atualização e aplicação de hotfixes direto pelo GitHub Releases:

- **Notificação Automática**: Ao iniciar, o app verifica silenciosamente se há novas versões disponíveis e exibe o botão **`[ 🚀 ATUALIZAR ]`** no topo.
- **Atualização In-Place com 1 Clique & Auto-Restart**: Baixa a nova versão com barra de progresso, substitui o executável atomicamente e **reinicia o aplicativo automaticamente**, sem necessidade de intervenção manual!
- **Linha de Comando (CLI)**:
  ```bash
  # Verificar se há atualizações disponíveis
  ./pngtuber-lite -check-update

  # Aplicar atualização/hotfix imediatamente via terminal
  ./pngtuber-lite -update

  # Exibir versão atual
  ./pngtuber-lite -version
  ```

---

## 🏗️ Estrutura do Projeto

- [`pkg/i18n`](pkg/i18n): Sistema de internacionalização, detecção de idiomas e dicionários (`pt-BR`, `en-US`).
- [`pkg/model`](pkg/model): Estruturas do avatar, camadas (`Layer`), vetor 2D, serializador, builder e parser de `.save`.
- [`pkg/render`](pkg/render): Motor de renderização Raylib, cache de texturas GPU e cálculo de matrizes hierárquicas.
- [`pkg/anim`](pkg/anim): Física de mola-amortecedor (wobble), piscar natural (blink), salto (bounce) e sprite-sheets.
- [`pkg/audio`](pkg/audio): Captura de microfone com `malgo`, suporte a Easy Effects / DSP e VAD com histerese.
- [`pkg/costume`](pkg/costume): Gerenciador dos 10 figurinos e visibilidade por slot.
- [`pkg/config`](pkg/config): Configurações persistentes (`config.json`) e atalhos customizáveis (`keybinds.go`).
- [`pkg/ui`](pkg/ui): Interface gráfica com abas, layouts auto-ajustáveis, ícones GPU nativos e fontes TrueType.
- [`pkg/editor`](pkg/editor): Editor visual de criação de avatares com gizmo no canvas, hierarquia e drag-and-drop.
- [`pkg/profiler`](pkg/profiler): Monitoramento em tempo real de CPU (ticks /proc), RAM Física (RSS), Go Heap e VRAM GPU.
- [`pkg/tray`](pkg/tray): Integração nativa com a bandeja do sistema (System Tray) para Windows e Linux.
- [`pkg/updater`](pkg/updater): Verificação de releases no GitHub, in-place auto-updater e auto-restart.
- [`assets/samples`](assets/samples): Arquivos `.save` de exemplo (`defaultAvatar.save`, `slugcat.save`).

